package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	limiter := NewTokenBucket(10, 20) // 10 req/s, burst 20

	// Should allow burst requests
	for i := 0; i < 20; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("expected request %d to be allowed", i)
		}
	}

	// Next request should be denied
	if limiter.Allow("test-key") {
		t.Error("expected request to be denied after burst")
	}

	// Wait for token refill
	time.Sleep(150 * time.Millisecond)

	// Should allow another request after refill
	if !limiter.Allow("test-key") {
		t.Error("expected request to be allowed after refill")
	}
}

func TestTokenBucketMultipleKeys(t *testing.T) {
	limiter := NewTokenBucket(5, 5)

	// Exhaust tokens for key1
	for i := 0; i < 5; i++ {
		limiter.Allow("key1")
	}

	// key2 should still be allowed
	if !limiter.Allow("key2") {
		t.Error("expected key2 to be allowed")
	}

	// key1 should be denied
	if limiter.Allow("key1") {
		t.Error("expected key1 to be denied")
	}
}

func TestIPKeyExtractor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "192.168.1.1:1234",
			xff:        "203.0.113.1",
			want:       "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "192.168.1.1:1234",
			xff:        "203.0.113.1, 198.51.100.1",
			want:       "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:1234",
			xri:        "203.0.113.1",
			want:       "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.1:1234",
			want:       "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     http.Header{},
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			got := IPKeyExtractor(req)
			if got != tt.want {
				t.Errorf("IPKeyExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIKeyExtractor(t *testing.T) {
	extractor := APIKeyExtractor("X-API-Key")

	// Test with API key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "secret123")
	req.RemoteAddr = "192.168.1.1:1234"

	got := extractor(req)
	want := "apikey:secret123"
	if got != want {
		t.Errorf("APIKeyExtractor() = %v, want %v", got, want)
	}

	// Test fallback to IP when no API key
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	
	got2 := extractor(req2)
	// Should fall back to IP extraction
	if got2 == "" {
		t.Errorf("APIKeyExtractor() fallback returned empty string")
	}
}

func TestCompositeKeyExtractor(t *testing.T) {
	extractor := CompositeKeyExtractor(
		IPKeyExtractor,
		APIKeyExtractor("X-API-Key"),
	)

	req := &http.Request{
		Header:     http.Header{"X-API-Key": []string{"secret123"}},
		RemoteAddr: "192.168.1.1:1234",
	}

	got := extractor(req)
	// Should contain both IP and API key
	if got == "" {
		t.Error("CompositeKeyExtractor() returned empty string")
	}
}

func TestWait(t *testing.T) {
	limiter := NewTokenBucket(10, 1)

	// Exhaust tokens
	limiter.Allow("test-key")

	// Should need to wait
	wait := limiter.Wait("test-key")
	if wait <= 0 {
		t.Error("expected positive wait time")
	}
	if wait > 200*time.Millisecond {
		t.Errorf("wait time too long: %v", wait)
	}
}

func BenchmarkTokenBucketAllow(b *testing.B) {
	limiter := NewTokenBucket(1000, 2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow("test-key")
	}
}

func BenchmarkIPKeyExtractor(b *testing.B) {
	req := &http.Request{
		RemoteAddr: "192.168.1.1:1234",
		Header:     http.Header{},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IPKeyExtractor(req)
	}
}

