package crawler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/MunishMummadi/web-scrapper/config"
	"github.com/MunishMummadi/web-scrapper/database"
	"github.com/MunishMummadi/web-scrapper/metrics"
	"github.com/MunishMummadi/web-scrapper/proxy"
	"github.com/MunishMummadi/web-scrapper/queue"
)

// Crawler manages the crawling process
type Crawler struct {
	cfg            *config.CrawlerConfig
	queue          queue.Queue
	storage        database.Storage
	httpClient     *http.Client
	metrics        *metrics.MetricsCollector
	robots         *RobotsCache
	rateLimiter    *HostRateLimiter
	circuitBreaker *CircuitBreaker
	proxyManager   *proxy.Manager
	stopChan       chan struct{} // Channel to signal workers to stop
	wg             sync.WaitGroup    // WaitGroup to wait for workers to finish
}

// NewCrawler creates a new Crawler instance
func NewCrawler(cfg *config.Config, q queue.Queue, s database.Storage, m *metrics.MetricsCollector, p *proxy.Manager) (*Crawler, error) {
	// Configure HTTP client with proxy and timeouts
	transport := p.GetTransport()
	httpClient := &http.Client{
		Timeout:   cfg.Crawler.RequestTimeout,
		Transport: transport,
	}

	// Create the robots.txt cache
	robotsCache := NewRobotsCache(cfg.Crawler.UserAgent, httpClient)

	// Create rate limiter (convert default delay to QPS)
	defaultQPS := 1.0 / cfg.Crawler.DefaultDelay.Seconds()
	rateLimiter := NewHostRateLimiter(defaultQPS, cfg.Crawler.MaxConcurrentHosts)

	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker(
		cfg.Crawler.CircuitBreakerRatio,
		cfg.Crawler.CircuitBreakerTime,
		3, // Success required to close
		20, // Rolling window size
		time.Hour, // Host error expiry
	)

	return &Crawler{
		cfg:            &cfg.Crawler,
		queue:          q,
		storage:        s,
		httpClient:     httpClient,
		metrics:        m,
		robots:         robotsCache,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		proxyManager:   p,
		stopChan:       make(chan struct{}),
	}, nil
}

// Start begins the crawling process by launching worker goroutines
func (c *Crawler) Start(ctx context.Context) {
	log.Printf("Starting %d crawler workers...", c.cfg.WorkerCount)
	c.wg.Add(c.cfg.WorkerCount)
	c.metrics.SetWorkersRunning(c.cfg.WorkerCount)
	
	for i := 0; i < c.cfg.WorkerCount; i++ {
		go c.worker(ctx, i)
	}
	log.Println("Crawler started.")
}

// Stop signals the crawler workers to stop gracefully
func (c *Crawler) Stop() {
	log.Println("Stopping crawler workers...")
	close(c.stopChan) // Signal workers
	c.wg.Wait()       // Wait for all workers to finish
	c.metrics.SetWorkersRunning(0)
	log.Println("Crawler stopped.")
}

// worker is the main loop for a single crawler worker
func (c *Crawler) worker(ctx context.Context, id int) {
	defer c.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-c.stopChan: // Check if stop signal received
			log.Printf("Worker %d stopping...", id)
			return
		case <-ctx.Done(): // Check if context cancelled (e.g., application shutdown)
			log.Printf("Worker %d stopping due to context cancellation...", id)
			return
		default:
			// Attempt to dequeue a URL with a shorter timeout
			dequeueCtx, cancel := context.WithTimeout(ctx, 1*time.Second) // Reduced timeout for dequeue
			urlToScrape, err := c.queue.Dequeue(dequeueCtx)
			cancel()

			if err != nil {
				// Only log actual errors, not timeouts
				if err != context.DeadlineExceeded && err != context.Canceled {
					log.Printf("Worker %d: Error dequeuing URL: %v", id, err)
				}
				time.Sleep(500 * time.Millisecond) // Pause briefly on dequeue error
				continue
			}

			if urlToScrape == "" {
				// Queue is empty, wait a bit before polling again to avoid CPU spinning
				time.Sleep(1 * time.Second)
				continue
			}

			log.Printf("Worker %d: Dequeued URL: %s", id, urlToScrape)
			
			// Record the processing start time for metrics
			startTime := time.Now()
			
			// Process URL with retry logic
			success := false
			var processErr error
			
			for retries := 0; retries <= c.cfg.MaxRetries; retries++ {
				if retries > 0 {
					log.Printf("Worker %d: Retry %d/%d for URL %s", id, retries, c.cfg.MaxRetries, urlToScrape)
					// Exponential backoff
					backoff := c.cfg.RetryDelay * time.Duration(1<<uint(retries-1))
					time.Sleep(backoff)
				}
				
				processErr = c.processURL(ctx, urlToScrape)
				if processErr == nil {
					success = true
					break
				}
				
				// Check for permanent errors (don't retry)
				if strings.Contains(processErr.Error(), "robots.txt disallowed") ||
				   strings.Contains(processErr.Error(), "invalid URL") {
					break
				}
			}

			// Record metrics
			c.metrics.RecordProcessingTime(time.Since(startTime))
			
			if !success {
				log.Printf("Worker %d: Failed to process URL %s after retries: %v", id, urlToScrape, processErr)
				c.metrics.IncrementScrapingErrors()
			}
		}
	}
}

// processURL handles the scraping of a single URL
func (c *Crawler) processURL(ctx context.Context, urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	host := parsedURL.Hostname()

	// Check cache for recent scrapes
	lastScrape, err := c.storage.GetLastScrapeTime(ctx, urlStr)
	if err == nil && time.Since(lastScrape) < c.cfg.CacheExpiration {
		log.Printf("URL %s was recently scraped (%v ago), skipping", urlStr, time.Since(lastScrape))
		return nil
	}

	// Check if circuit breaker is open for this host
	if !c.circuitBreaker.IsAllowed(host) {
		log.Printf("Circuit breaker is open for %s, skipping", host)
		return fmt.Errorf("circuit breaker open for host %s", host)
	}

	// Respect robots.txt 
	if c.cfg.RespectRobots {
		allowed, err := c.robots.IsAllowed(urlStr)
		if err != nil {
			log.Printf("Error checking robots.txt for %s: %v", urlStr, err)
			// Continue with caution
		} else if !allowed {
			log.Printf("URL %s is disallowed by robots.txt", urlStr)
			c.metrics.IncrementRobotsDisallowed()
			return fmt.Errorf("robots.txt disallowed URL %s", urlStr)
		}
	}

	// Apply rate limiting for the host
	limiterCtx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()
	
	if err := c.rateLimiter.Wait(limiterCtx, host); err != nil {
		return fmt.Errorf("rate limiting wait failed: %w", err)
	}

	// Create and execute the HTTP request
	log.Printf("Fetching %s...", urlStr)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)
	c.metrics.RecordScrapingDuration(requestDuration)

	if err != nil {
		c.circuitBreaker.RecordFailure(host)
		// If using proxy, record the failure
		if c.proxyManager != nil {
			proxyURL := req.URL.String() // This is not correct in all cases, but a simplification
			c.proxyManager.RecordFailure(proxyURL)
			c.metrics.IncrementProxyFailures()
		}
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Successfully fetched %s (%d) in %v", urlStr, resp.StatusCode, requestDuration)

	// Handle non-success status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.circuitBreaker.RecordFailure(host)
		return fmt.Errorf("received non-2xx status code: %d", resp.StatusCode)
	}

	// Read and process response body
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // Limit to 10MB
	if err != nil {
		c.circuitBreaker.RecordFailure(host)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Record response size metric
	c.metrics.RecordResponseSize(float64(len(bodyBytes)))

	// Calculate content hash
	hasher := sha256.New()
	hasher.Write(bodyBytes)
	contentHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Save scrape result to storage
	scrapedAt := time.Now()
	if err := c.storage.SaveScrapedData(ctx, urlStr, scrapedAt, contentHash); err != nil {
		log.Printf("Error saving scrape data for %s: %v", urlStr, err)
		// Not a fatal error, continue
	}

	// Record success in circuit breaker
	c.circuitBreaker.RecordSuccess(host)
	
	// If using proxy, record success
	if c.proxyManager != nil {
		proxyURL := req.URL.String() // This is not correct in all cases, but a simplification
		c.proxyManager.RecordSuccess(proxyURL)
	}

	// Increment successful scrapes counter
	c.metrics.IncrementScrapedPages()

	return nil
}

// EnqueueURL adds a URL to the queue for crawling
func (c *Crawler) EnqueueURL(ctx context.Context, urlStr string) error {
	if err := c.queue.Enqueue(ctx, urlStr); err != nil {
		return fmt.Errorf("failed to enqueue URL %s: %w", urlStr, err)
	}
	c.metrics.IncrementQueuedURLs()
	return nil
}
