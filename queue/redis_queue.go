package queue

import (
	"context"
	"time"

	"github.com/MunishMummadi/web-scrapper/config"
	"github.com/go-redis/redis/v8"
)

const (
	defaultQueueKey = "scraper:url_queue"
	defaultTimeout  = 1 * time.Second // Reduced timeout for blocking dequeue
)

// Queue defines the interface for a job queue
type Queue interface {
	Enqueue(ctx context.Context, url string) error
	Dequeue(ctx context.Context) (string, error)
	Close() error
}

// RedisQueue implements the Queue interface using Redis
type RedisQueue struct {
	client  *redis.Client
	queueKey string
}

// NewRedisQueue creates a new Redis-based queue
func NewRedisQueue(cfg config.RedisConfig) (Queue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping Redis to ensure connection is established
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisQueue{
		client:   client,
		queueKey: defaultQueueKey,
	}, nil
}

// Enqueue adds a URL to the end of the Redis list (queue)
func (q *RedisQueue) Enqueue(ctx context.Context, url string) error {
	return q.client.LPush(ctx, q.queueKey, url).Err()
}

// Dequeue retrieves and removes a URL from the front of the Redis list (queue)
// It uses a short timeout to avoid long-blocking operations that might cause context timeouts
func (q *RedisQueue) Dequeue(ctx context.Context) (string, error) {
	// First, check if the context is already expired/cancelled
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Create a local timeout that's shorter than the context timeout
	// This prevents long blocks on BRPop that can lead to context deadline errors
	localCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Use BRPOP with a short timeout
	result, err := q.client.BRPop(localCtx, defaultTimeout, q.queueKey).Result()
	
	// Handle specific errors
	if err != nil {
		// redis.Nil indicates timeout or empty queue - not an error condition
		if err == redis.Nil {
			// Sleep a small amount to prevent tight polling when queue is empty
			time.Sleep(100 * time.Millisecond)
			return "", nil // Return empty string, worker can retry
		}
		
		// Context cancellation or deadline exceeded - this is probably from our local context
		if err == context.Canceled || err == context.DeadlineExceeded {
			return "", nil // Not a real error, just empty queue
		}
		
		// For other Redis errors, return them
		return "", err
	}

	// BRPop returns a slice [key, value]
	if len(result) < 2 {
		// Should not happen with BRPop but handle defensively
		return "", nil
	}
	return result[1], nil // Return the URL
}

// Close closes the Redis client connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
