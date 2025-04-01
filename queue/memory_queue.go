package queue

import (
	"context"
	"sync"
)

// MemoryQueue implements the Queue interface using in-memory storage
// This is primarily for testing purposes or when Redis is not available
type MemoryQueue struct {
	queue []string
	mu    sync.Mutex
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() Queue {
	return &MemoryQueue{
		queue: make([]string, 0),
	}
}

// Enqueue adds a URL to the queue
func (q *MemoryQueue) Enqueue(ctx context.Context, url string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.queue = append(q.queue, url)
	return nil
}

// Dequeue retrieves and removes a URL from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if len(q.queue) == 0 {
		return "", nil // Return empty string for empty queue
	}
	
	url := q.queue[0]
	q.queue = q.queue[1:]
	return url, nil
}

// Close is a no-op for memory queue
func (q *MemoryQueue) Close() error {
	return nil
}
