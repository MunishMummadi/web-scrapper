package crawler

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HostRateLimiter manages rate limits for different hosts
type HostRateLimiter struct {
	limiters   map[string]*rate.Limiter
	mu         sync.RWMutex
	defaultQPS float64
	defaultRPS int
	cleanup    *time.Ticker
	ttl        time.Duration
	lastUsed   map[string]time.Time
}

// NewHostRateLimiter creates a new rate limiter for hosts
// defaultQPS is requests per second (e.g., 0.2 for one request per 5 seconds)
// defaultRPS is burst capacity (max requests allowed at once)
func NewHostRateLimiter(defaultQPS float64, defaultRPS int) *HostRateLimiter {
	h := &HostRateLimiter{
		limiters:   make(map[string]*rate.Limiter),
		defaultQPS: defaultQPS,
		defaultRPS: defaultRPS,
		ttl:        time.Hour, // Cleanup unused limiters after 1 hour
		lastUsed:   make(map[string]time.Time),
	}

	// Start a cleanup routine
	h.cleanup = time.NewTicker(10 * time.Minute)
	go h.cleanupRoutine()
	
	return h
}

// Wait blocks until the rate limit allows an event for the host or ctx is done
func (h *HostRateLimiter) Wait(ctx context.Context, host string) error {
	limiter := h.getLimiter(host)
	h.updateLastUsed(host)
	return limiter.Wait(ctx) // This blocks until rate limit allows or ctx cancelled
}

// Allow reports whether an event may happen for the host
// Does not block, but rather reports if rate limit would allow
func (h *HostRateLimiter) Allow(host string) bool {
	limiter := h.getLimiter(host)
	allowed := limiter.Allow()
	if allowed {
		h.updateLastUsed(host)
	}
	return allowed
}

// SetRate changes the rate limit for a specific host
func (h *HostRateLimiter) SetRate(host string, qps float64, rps int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.limiters[host] = rate.NewLimiter(rate.Limit(qps), rps)
	h.lastUsed[host] = time.Now()
}

// getLimiter gets or creates a rate limiter for a host
func (h *HostRateLimiter) getLimiter(host string) *rate.Limiter {
	h.mu.RLock()
	limiter, exists := h.limiters[host]
	h.mu.RUnlock()

	if !exists {
		h.mu.Lock()
		// Double-check (another goroutine might have created it)
		limiter, exists = h.limiters[host]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(h.defaultQPS), h.defaultRPS)
			h.limiters[host] = limiter
			h.lastUsed[host] = time.Now()
		}
		h.mu.Unlock()
	}
	
	return limiter
}

// updateLastUsed updates the last used timestamp for a host
func (h *HostRateLimiter) updateLastUsed(host string) {
	h.mu.Lock()
	h.lastUsed[host] = time.Now()
	h.mu.Unlock()
}

// cleanupRoutine periodically cleans up unused limiters
func (h *HostRateLimiter) cleanupRoutine() {
	for range h.cleanup.C {
		now := time.Now()
		h.mu.Lock()
		
		for host, lastUsed := range h.lastUsed {
			if now.Sub(lastUsed) > h.ttl {
				delete(h.limiters, host)
				delete(h.lastUsed, host)
			}
		}
		
		h.mu.Unlock()
	}
}

// Close stops the cleanup routine
func (h *HostRateLimiter) Close() {
	if h.cleanup != nil {
		h.cleanup.Stop()
	}
}
