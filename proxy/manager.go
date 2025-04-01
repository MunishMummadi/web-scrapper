package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MunishMummadi/web-scrapper/config"
)

// Manager handles proxy rotation and health checking
type Manager struct {
	proxies      []*ProxyServer
	current      int32
	mu           sync.RWMutex
	proxyAPI     string
	apiKey       string
	refreshTimer *time.Ticker
	client       *http.Client
	enabled      bool
}

// ProxyServer represents a proxy server with health status
type ProxyServer struct {
	URL       string
	LastCheck time.Time
	Healthy   bool
	ErrorRate float64
	Failures  int
	Successes int
}

// NewManager creates a new proxy rotation manager
func NewManager(cfg config.ProxyConfig) (*Manager, error) {
	if !cfg.Enabled {
		return &Manager{enabled: false}, nil
	}

	manager := &Manager{
		proxies:  make([]*ProxyServer, 0, len(cfg.URLs)),
		proxyAPI: cfg.APIUrl,
		apiKey:   cfg.APIKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: true,
	}

	// Initialize with the static proxies from config
	for _, proxyURL := range cfg.URLs {
		// Validate the proxy URL
		_, err := url.Parse(proxyURL)
		if err != nil {
			continue // Skip invalid URLs
		}

		manager.proxies = append(manager.proxies, &ProxyServer{
			URL:       proxyURL,
			LastCheck: time.Now(),
			Healthy:   true, // Assume healthy until proven otherwise
		})
	}

	// Start refresh timer if API URL is provided
	if cfg.APIUrl != "" && cfg.APIKey != "" {
		// Refresh proxies every hour
		manager.refreshTimer = time.NewTicker(1 * time.Hour)
		go manager.refreshProxies()
	}

	return manager, nil
}

// GetTransport returns an http.Transport that uses proxies
func (m *Manager) GetTransport() *http.Transport {
	if !m.enabled || len(m.proxies) == 0 {
		// No proxies or disabled, return default transport
		return &http.Transport{
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     30 * time.Second,
		}
	}

	return &http.Transport{
		Proxy:               m.proxyFunc,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	}
}

// proxyFunc is called to determine which proxy to use for a request
func (m *Manager) proxyFunc(req *http.Request) (*url.URL, error) {
	if !m.enabled || len(m.proxies) == 0 {
		return nil, nil // No proxy
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.proxies) == 0 {
		return nil, errors.New("no proxies available")
	}

	// Round-robin selection of proxies
	current := atomic.AddInt32(&m.current, 1) % int32(len(m.proxies))
	proxyServer := m.proxies[current]

	if !proxyServer.Healthy {
		// Try to find a healthy proxy
		for i := 0; i < len(m.proxies); i++ {
			current = (current + 1) % int32(len(m.proxies))
			proxyServer = m.proxies[current]
			if proxyServer.Healthy {
				break
			}
		}
	}

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	return proxyURL, nil
}

// refreshProxies fetches fresh proxies from the API
func (m *Manager) refreshProxies() {
	for range m.refreshTimer.C {
		if m.proxyAPI == "" || m.apiKey == "" {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Here you would implement the API call to your proxy provider
		// This is a placeholder - implement according to your proxy API provider's requirements
		req, err := http.NewRequestWithContext(ctx, "GET", m.proxyAPI, nil)
		if err != nil {
			continue
		}

		// Add authentication (varies by provider)
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

		resp, err := m.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Parse the response and update proxies
		// This implementation depends on your proxy provider's API response format
		// For now, we'll just log that we would parse proxies here
		
		// Example parsing logic (commented out as it depends on provider):
		/*
		var proxyResponse struct {
			Proxies []string `json:"proxies"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&proxyResponse); err != nil {
			continue
		}
		
		m.mu.Lock()
		m.proxies = make([]*ProxyServer, 0, len(proxyResponse.Proxies))
		for _, p := range proxyResponse.Proxies {
			m.proxies = append(m.proxies, &ProxyServer{
				URL:       p,
				LastCheck: time.Now(),
				Healthy:   true,
			})
		}
		m.mu.Unlock()
		*/
	}
}

// RecordSuccess records a successful request through a proxy
func (m *Manager) RecordSuccess(proxyURL string) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, proxy := range m.proxies {
		if proxy.URL == proxyURL {
			proxy.Successes++
			proxy.ErrorRate = float64(proxy.Failures) / float64(proxy.Successes+proxy.Failures)
			proxy.Healthy = proxy.ErrorRate < 0.5 // Mark unhealthy if error rate is too high
			break
		}
	}
}

// RecordFailure records a failed request through a proxy
func (m *Manager) RecordFailure(proxyURL string) {
	if !m.enabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, proxy := range m.proxies {
		if proxy.URL == proxyURL {
			proxy.Failures++
			proxy.ErrorRate = float64(proxy.Failures) / float64(proxy.Successes+proxy.Failures)
			proxy.Healthy = proxy.ErrorRate < 0.5 && proxy.Failures < 5 // Mark unhealthy if error rate is too high
			break
		}
	}
}

// Close stops the refresh timer
func (m *Manager) Close() {
	if m.refreshTimer != nil {
		m.refreshTimer.Stop()
	}
}
