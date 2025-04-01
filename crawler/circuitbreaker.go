package crawler

import (
	"sync"
	"time"
)

const (
	// Status constants for circuit breaker
	circuitClosed   = "closed"   // Normal operation, requests flow through
	circuitOpen     = "open"     // Failing too much, block requests
	circuitHalfOpen = "halfOpen" // Testing if system is healthy again
)

// CircuitBreaker implements the circuit breaker pattern for hosts
type CircuitBreaker struct {
	hosts                map[string]*hostCircuit
	mu                   sync.RWMutex
	failureThreshold     float64 // Percentage of failures that trips the circuit (0.0-1.0)
	resetTimeout         time.Duration // How long to wait before trying half-open state
	succRequiredToClose  int      // Number of consecutive successes needed to close circuit
	rollingWindowSize    int      // Size of the rolling window for calculating error rates
	hostErrorExpiry      time.Duration // Time before a host error is expired from tracking
}

// hostCircuit tracks the state for a specific host
type hostCircuit struct {
	state            string
	openedAt         time.Time
	attemptedResetAt time.Time
	halfOpenSuccess  int
	failures         []time.Time // Timestamps of recent failures
	successes        []time.Time // Timestamps of recent successes
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(
	failureThreshold float64,
	resetTimeout time.Duration,
	succRequiredToClose int,
	rollingWindowSize int,
	hostErrorExpiry time.Duration,
) *CircuitBreaker {
	return &CircuitBreaker{
		hosts:               make(map[string]*hostCircuit),
		failureThreshold:    failureThreshold,
		resetTimeout:        resetTimeout,
		succRequiredToClose: succRequiredToClose,
		rollingWindowSize:   rollingWindowSize,
		hostErrorExpiry:     hostErrorExpiry,
	}
}

// IsAllowed checks if requests are allowed for the host
func (cb *CircuitBreaker) IsAllowed(host string) bool {
	cb.mu.RLock()
	circuit, exists := cb.hosts[host]
	cb.mu.RUnlock()

	if !exists {
		// No circuit for this host yet, initialize one
		cb.mu.Lock()
		// Double-check, it might have been created by another goroutine
		circuit, exists = cb.hosts[host]
		if !exists {
			circuit = &hostCircuit{
				state:     circuitClosed,
				failures:  make([]time.Time, 0, cb.rollingWindowSize),
				successes: make([]time.Time, 0, cb.rollingWindowSize),
			}
			cb.hosts[host] = circuit
		}
		cb.mu.Unlock()
		return true
	}

	// Check circuit state
	now := time.Now()
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch circuit.state {
	case circuitClosed:
		return true
	case circuitOpen:
		// Check if circuit has been open long enough to try reset
		if now.Sub(circuit.openedAt) > cb.resetTimeout {
			circuit.state = circuitHalfOpen
			circuit.attemptedResetAt = now
			circuit.halfOpenSuccess = 0
			return true // Allow one request for testing
		}
		return false
	case circuitHalfOpen:
		// In half-open state, only allow one request at a time to test the service
		return circuit.halfOpenSuccess < cb.succRequiredToClose
	default:
		return true
	}
}

// RecordSuccess records a successful request to the host
func (cb *CircuitBreaker) RecordSuccess(host string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	circuit, exists := cb.hosts[host]
	if !exists {
		// Should not happen if IsAllowed() was called first, but handle it
		circuit = &hostCircuit{
			state:     circuitClosed,
			failures:  make([]time.Time, 0, cb.rollingWindowSize),
			successes: make([]time.Time, 0, cb.rollingWindowSize),
		}
		cb.hosts[host] = circuit
	}
	
	now := time.Time{}
	
	switch circuit.state {
	case circuitClosed:
		// Add to success window and clean up old entries
		circuit.successes = append(circuit.successes, now)
		if len(circuit.successes) > cb.rollingWindowSize {
			circuit.successes = circuit.successes[1:]
		}
		// Clean up old failures
		cb.cleanExpiredEvents(circuit.failures)
	case circuitHalfOpen:
		// In half-open, track consecutive successes
		circuit.halfOpenSuccess++
		if circuit.halfOpenSuccess >= cb.succRequiredToClose {
			// Enough successes, close the circuit
			circuit.state = circuitClosed
			circuit.failures = make([]time.Time, 0, cb.rollingWindowSize)
			circuit.successes = append(circuit.successes, now)
			if len(circuit.successes) > cb.rollingWindowSize {
				circuit.successes = circuit.successes[1:]
			}
		}
	}
}

// RecordFailure records a failed request to the host
func (cb *CircuitBreaker) RecordFailure(host string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	circuit, exists := cb.hosts[host]
	if !exists {
		// Should not happen if IsAllowed() was called first, but handle it
		circuit = &hostCircuit{
			state:     circuitClosed,
			failures:  make([]time.Time, 0, cb.rollingWindowSize),
			successes: make([]time.Time, 0, cb.rollingWindowSize),
		}
		cb.hosts[host] = circuit
	}
	
	now := time.Now()
	
	switch circuit.state {
	case circuitClosed:
		// Add to failure window and clean up old entries
		circuit.failures = append(circuit.failures, now)
		if len(circuit.failures) > cb.rollingWindowSize {
			circuit.failures = circuit.failures[1:]
		}
		
		// Calculate failure rate
		cb.cleanExpiredEvents(circuit.failures)
		cb.cleanExpiredEvents(circuit.successes)
		total := len(circuit.failures) + len(circuit.successes)
		
		if total > 0 {
			failureRate := float64(len(circuit.failures)) / float64(total)
			if failureRate >= cb.failureThreshold && len(circuit.failures) >= 3 {
				// Trip the circuit
				circuit.state = circuitOpen
				circuit.openedAt = now
			}
		}
	case circuitHalfOpen:
		// In half-open, any failure trips the circuit again
		circuit.state = circuitOpen
		circuit.openedAt = now
	}
}

// cleanExpiredEvents removes events older than the expiry window
func (cb *CircuitBreaker) cleanExpiredEvents(events []time.Time) []time.Time {
	now := time.Now()
	cutoff := now.Add(-cb.hostErrorExpiry)
	
	i := 0
	for i < len(events) && events[i].Before(cutoff) {
		i++
	}
	
	if i > 0 {
		return events[i:]
	}
	return events
}

// GetState returns the current state of the circuit for a host
func (cb *CircuitBreaker) GetState(host string) string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	circuit, exists := cb.hosts[host]
	if !exists {
		return circuitClosed
	}
	return circuit.state
}

// Reset resets the circuit for a host to closed state
func (cb *CircuitBreaker) Reset(host string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	circuit, exists := cb.hosts[host]
	if exists {
		circuit.state = circuitClosed
		circuit.failures = make([]time.Time, 0, cb.rollingWindowSize)
		circuit.successes = make([]time.Time, 0, cb.rollingWindowSize)
	}
}
