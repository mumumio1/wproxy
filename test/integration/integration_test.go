package integration

import (
	"testing"
	"time"

	"github.com/mumumio1/wproxy/internal/cache"
	"github.com/mumumio1/wproxy/internal/ratelimit"
)

// TestCacheBasic tests basic cache operations
func TestCacheBasic(t *testing.T) {
	c := cache.NewMemoryCache(1024*1024, 5*time.Minute)

	// Test cache miss
	if _, ok := c.Get("test-key"); ok {
		t.Error("Expected cache miss")
	}

	// Test cache set and get
	entry := &cache.Entry{
		StatusCode: 200,
		Body:       []byte("test data"),
		ExpiresAt:  time.Now().Add(1 * time.Minute),
		Size:       9,
	}
	c.Set("test-key", entry)

	if retrieved, ok := c.Get("test-key"); !ok {
		t.Error("Expected cache hit")
	} else if string(retrieved.Body) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", retrieved.Body)
	}

	// Test cache size
	if c.Size() == 0 {
		t.Error("Expected non-zero cache size")
	}
}

// TestCacheExpiration tests cache entry expiration
func TestCacheExpiration(t *testing.T) {
	c := cache.NewMemoryCache(1024*1024, 5*time.Minute)

	entry := &cache.Entry{
		StatusCode: 200,
		Body:       []byte("test data"),
		ExpiresAt:  time.Now().Add(50 * time.Millisecond),
		Size:       9,
	}
	c.Set("test-key", entry)

	// Should exist immediately
	if _, ok := c.Get("test-key"); !ok {
		t.Error("Expected cache hit")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	if _, ok := c.Get("test-key"); ok {
		t.Error("Expected cache miss after expiration")
	}
}

// TestRateLimiting tests rate limiter functionality
func TestRateLimiting(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(10, 5)

	// First 5 requests should succeed (burst)
	for i := 0; i < 5; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Request %d should be allowed (burst)", i)
		}
	}

	// Next requests should be rate limited
	limited := false
	for i := 0; i < 10; i++ {
		if !limiter.Allow("test-key") {
			limited = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !limited {
		t.Error("Expected rate limiting to kick in")
	}
}

// TestRateLimitDifferentKeys tests that different keys are tracked separately
func TestRateLimitDifferentKeys(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(10, 3)

	// Exhaust key1
	for i := 0; i < 3; i++ {
		limiter.Allow("key1")
	}

	// key1 should be limited
	if limiter.Allow("key1") {
		t.Error("key1 should be rate limited")
	}

	// key2 should still work
	if !limiter.Allow("key2") {
		t.Error("key2 should not be rate limited")
	}
}
