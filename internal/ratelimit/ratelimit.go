package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limiter is the interface for rate limiting
type Limiter interface {
	Allow(key string) bool
	Wait(key string) time.Duration
}

// tokenBucket implements a token bucket rate limiter
type tokenBucket struct {
	mu            sync.RWMutex
	rate          float64 // tokens per second
	burst         int     // maximum tokens
	buckets       map[string]*bucket
	cleanupTicker *time.Ticker
	done          chan struct{}
}

type bucket struct {
	tokens       float64
	lastRefill   time.Time
	mu           sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(requestsPerSecond int, burst int) Limiter {
	tb := &tokenBucket{
		rate:          float64(requestsPerSecond),
		burst:         burst,
		buckets:       make(map[string]*bucket),
		cleanupTicker: time.NewTicker(1 * time.Minute),
		done:          make(chan struct{}),
	}

	// Start cleanup goroutine
	go tb.cleanup()

	return tb
}

// Allow checks if a request should be allowed
func (tb *tokenBucket) Allow(key string) bool {
	tb.mu.Lock()
	b, exists := tb.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     float64(tb.burst),
			lastRefill: time.Now(),
		}
		tb.buckets[key] = b
	}
	tb.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	
	// Refill tokens based on elapsed time
	b.tokens = min(float64(tb.burst), b.tokens+elapsed*tb.rate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// Wait returns how long to wait before the next token is available
func (tb *tokenBucket) Wait(key string) time.Duration {
	tb.mu.RLock()
	b, exists := tb.buckets[key]
	tb.mu.RUnlock()

	if !exists {
		return 0
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tokens >= 1 {
		return 0
	}

	tokensNeeded := 1 - b.tokens
	waitTime := time.Duration(tokensNeeded/tb.rate*1000) * time.Millisecond
	return waitTime
}

// cleanup removes stale buckets
func (tb *tokenBucket) cleanup() {
	for {
		select {
		case <-tb.cleanupTicker.C:
			tb.mu.Lock()
			now := time.Now()
			for key, b := range tb.buckets {
				b.mu.Lock()
				if now.Sub(b.lastRefill) > 5*time.Minute {
					delete(tb.buckets, key)
				}
				b.mu.Unlock()
			}
			tb.mu.Unlock()
		case <-tb.done:
			tb.cleanupTicker.Stop()
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (tb *tokenBucket) Stop() {
	close(tb.done)
}

// KeyExtractor extracts a rate limit key from a request
type KeyExtractor func(*http.Request) string

// IPKeyExtractor extracts the client IP address
func IPKeyExtractor(r *http.Request) string {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				return xff[:idx]
			}
		}
		return xff
	}

	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// APIKeyExtractor extracts an API key from a header
func APIKeyExtractor(headerName string) KeyExtractor {
	return func(r *http.Request) string {
		key := r.Header.Get(headerName)
		if key == "" {
			// Fall back to IP if no API key
			return IPKeyExtractor(r)
		}
		return "apikey:" + key
	}
}

// CompositeKeyExtractor combines multiple extractors
func CompositeKeyExtractor(extractors ...KeyExtractor) KeyExtractor {
	return func(r *http.Request) string {
		keys := make([]string, 0, len(extractors))
		for _, extractor := range extractors {
			keys = append(keys, extractor(r))
		}
		result := ""
		for i, key := range keys {
			if i > 0 {
				result += ":"
			}
			result += key
		}
		return result
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

