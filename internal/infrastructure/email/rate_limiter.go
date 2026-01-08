package email

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter for email sending.
// This prevents abuse and ensures we stay within provider limits.
type RateLimiter struct {
	mu           sync.Mutex
	tokens       int
	maxTokens    int
	refillRate   time.Duration
	lastRefillAt time.Time
}

// NewRateLimiter creates a new rate limiter with the specified limits.
// maxTokens is the maximum number of emails allowed in the time window.
// refillRate is how often tokens are replenished.
//
// Example: NewRateLimiter(100, time.Hour) allows 100 emails per hour.
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		lastRefillAt: time.Now(),
	}
}

// Allow returns true if an email can be sent (token available).
// If a token is available, it is consumed and true is returned.
// If no tokens available, false is returned.
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// refill replenishes tokens based on time elapsed since last refill.
// This implements a leaky bucket algorithm.
// Must be called with mutex held.
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillAt)

	if elapsed >= rl.refillRate {
		// Refill all tokens
		rl.tokens = rl.maxTokens
		rl.lastRefillAt = now
	}
}

// Remaining returns the number of tokens currently available.
func (rl *RateLimiter) Remaining() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	return rl.tokens
}

// Reset resets the rate limiter to full capacity.
// This is primarily for testing.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.tokens = rl.maxTokens
	rl.lastRefillAt = time.Now()
}
