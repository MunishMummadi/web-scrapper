# Queue Component

This directory contains the job queue functionality for the web scraper.

## Overview

The queue component is responsible for:

1. Managing the queue of URLs to be scraped
2. Providing both Redis-based and in-memory queue implementations
3. Handling dequeuing with appropriate timeouts
4. Implementing backoff strategies when the queue is empty

## Key Features

- **Queue Interface**: Common interface for different queue implementations
- **Redis Queue**: Production-ready queue using Redis as a backend
- **Memory Queue**: Simple in-memory queue for testing or when Redis is unavailable
- **Timeout Management**: Optimized timeouts to prevent "context deadline exceeded" errors

## Implementation Details

As discovered in previous work, the queue implementation has been optimized with:
- Reduced Redis queue timeout from 5 to 1 second
- Better error handling for context timeouts
- Backoff when the queue is empty to prevent CPU spinning

## Usage

The queue is initialized in `main.go` with a fallback mechanism:

```go
// Example of queue initialization with fallback
var q queue.Queue
if useMemQueue {
    log.Println("Using in-memory queue...")
    q = queue.NewMemoryQueue()
} else {
    log.Println("Initializing Redis queue...")
    redisQueue, err := queue.NewRedisQueue(cfg.Redis)
    if err != nil {
        log.Printf("Failed to initialize Redis queue: %v", err)
        log.Println("Falling back to in-memory queue...")
        q = queue.NewMemoryQueue()
    } else {
        q = redisQueue
    }
}
defer q.Close()
```

The crawler uses the queue to get URLs for processing:

```go
// Example of dequeuing a URL
url, err := q.Dequeue(ctx)
if err != nil {
    // Handle error or empty queue
}
// Process the URL
```
