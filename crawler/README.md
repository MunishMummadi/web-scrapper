# Crawler Component

This directory contains the core web crawling functionality of the scraper.

## Overview

The crawler component is responsible for:

1. Fetching web pages from URLs in the queue
2. Parsing HTML content
3. Extracting links and other data
4. Respecting robots.txt rules
5. Implementing rate limiting and circuit breaker patterns

## Key Features

- **Worker Pool**: Multiple concurrent workers process URLs from the queue
- **Circuit Breaker**: Prevents overwhelming websites by stopping requests when error rates are high
- **Rate Limiting**: Respects website constraints by limiting request rates
- **Proxy Rotation**: Uses different proxies to avoid IP bans
- **Graceful Timeouts**: Uses context timeouts for better error handling

## Implementation Details

As discovered in previous work, the crawler uses:
- 1-second timeouts instead of 5-second timeouts to prevent "context deadline exceeded" errors
- Backoff when the queue is empty to prevent CPU spinning
- Graceful handling of timeouts

## Usage

The crawler is initialized in `main.go` and receives dependencies like the queue, storage, and metrics collector:

```go
// Example of how the crawler is initialized
c, err := crawler.NewCrawler(cfg, q, sqliteStorage, metricsCollector, proxyManager)
if err != nil {
    log.Fatalf("Failed to initialize crawler: %v", err)
}

// Start crawler
c.Start(ctx)
```

To enqueue a URL for crawling:

```go
err := c.EnqueueURL(ctx, "https://example.com")
```
