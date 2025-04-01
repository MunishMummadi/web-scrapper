package crawler

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsCache caches robots.txt files and provides access control methods
type RobotsCache struct {
	cache     map[string]*robotsEntry
	userAgent string
	client    *http.Client
	mu        sync.RWMutex
	ttl       time.Duration
}

type robotsEntry struct {
	data       *robotstxt.RobotsData
	fetchedAt  time.Time
	statusCode int
}

// NewRobotsCache creates a new robots.txt cache with the given user agent
func NewRobotsCache(userAgent string, client *http.Client) *RobotsCache {
	return &RobotsCache{
		cache:     make(map[string]*robotsEntry),
		userAgent: userAgent,
		client:    client,
		ttl:       24 * time.Hour, // Cache robots.txt for 24 hours
	}
}

// IsAllowed checks if the given URL is allowed to be scraped
func (rc *RobotsCache) IsAllowed(urlStr string) (bool, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false, err
	}

	host := parsedURL.Hostname()
	robotsURL := (&url.URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   "/robots.txt",
	}).String()

	// Get or fetch robots data
	robotsData, err := rc.getRobotsData(robotsURL, host)
	if err != nil {
		// If we can't fetch robots.txt, err on the side of caution (disallow)
		return false, err
	}

	// If status code was 4xx/5xx, assume allowed if 4xx, disallow if 5xx
	if robotsData.statusCode >= 500 && robotsData.statusCode < 600 {
		return false, nil
	} else if robotsData.statusCode >= 400 && robotsData.statusCode < 500 {
		return true, nil
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	if parsedURL.RawQuery != "" {
		path += "?" + parsedURL.RawQuery
	}

	return robotsData.data.TestAgent(path, rc.userAgent), nil
}

// getRobotsData gets robots data from cache or fetches it
func (rc *RobotsCache) getRobotsData(robotsURL, host string) (*robotsEntry, error) {
	// Check cache first
	rc.mu.RLock()
	entry, exists := rc.cache[host]
	rc.mu.RUnlock()

	if exists && time.Since(entry.fetchedAt) < rc.ttl {
		return entry, nil
	}

	// Fetch robots.txt
	req, err := http.NewRequest("GET", robotsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", rc.userAgent)

	resp, err := rc.client.Do(req)
	statusCode := 0
	if err != nil {
		// Create an empty robots.txt if we can't fetch it
		robotsData, _ := robotstxt.FromStatusAndString(statusCode, "")
		entry := &robotsEntry{
			data:       robotsData,
			fetchedAt:  time.Now(),
			statusCode: statusCode,
		}

		rc.mu.Lock()
		rc.cache[host] = entry
		rc.mu.Unlock()

		return entry, err
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	var robotsData *robotstxt.RobotsData
	
	// Parse response depending on status code
	if statusCode >= 200 && statusCode < 300 {
		robotsData, err = robotstxt.FromResponse(resp)
		if err != nil {
			// Empty robots.txt on parse error
			robotsData, _ = robotstxt.FromStatusAndString(statusCode, "")
		}
	} else {
		// Create an empty robots.txt for non-2xx responses
		robotsData, _ = robotstxt.FromStatusAndString(statusCode, "")
	}

	entry = &robotsEntry{
		data:       robotsData,
		fetchedAt:  time.Now(),
		statusCode: statusCode,
	}

	rc.mu.Lock()
	rc.cache[host] = entry
	rc.mu.Unlock()

	return entry, nil
}
