package cache

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 5*time.Minute)

	entry := &Entry{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"test": "data"}`),
		ETag:       `"abc123"`,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		CreatedAt:  time.Now(),
		Size:       17,
	}

	// Test Set and Get
	cache.Set("test-key", entry)
	retrieved, ok := cache.Get("test-key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if retrieved.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", retrieved.StatusCode)
	}

	// Test Size and Len
	if cache.Len() != 1 {
		t.Errorf("expected len 1, got %d", cache.Len())
	}
	if cache.Size() != 17 {
		t.Errorf("expected size 17, got %d", cache.Size())
	}

	// Test Delete
	cache.Delete("test-key")
	_, ok = cache.Get("test-key")
	if ok {
		t.Error("expected cache miss after delete")
	}

	// Test Clear
	cache.Set("key1", entry)
	cache.Set("key2", entry)
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("expected len 0 after clear, got %d", cache.Len())
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewMemoryCache(1024*1024, 5*time.Minute)

	entry := &Entry{
		StatusCode: 200,
		Body:       []byte("test"),
		ExpiresAt:  time.Now().Add(10 * time.Millisecond),
		CreatedAt:  time.Now(),
		Size:       4,
	}

	cache.Set("test-key", entry)
	time.Sleep(20 * time.Millisecond)

	_, ok := cache.Get("test-key")
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestCacheLRU(t *testing.T) {
	cache := NewMemoryCache(50, 5*time.Minute) // Small cache for testing

	// Add entries until eviction occurs
	for i := 0; i < 10; i++ {
		entry := &Entry{
			Body:      []byte("test data"),
			ExpiresAt: time.Now().Add(5 * time.Minute),
			CreatedAt: time.Now(),
			Size:      9,
		}
		cache.Set(string(rune('a'+i)), entry)
	}

	// Cache should have evicted older entries
	if cache.Size() > 50 {
		t.Errorf("cache size %d exceeds max size 50", cache.Size())
	}
}

func TestCacheKey(t *testing.T) {
	req1 := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/api/test"},
		Header: http.Header{"Accept": []string{"application/json"}},
	}
	req2 := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/api/test"},
		Header: http.Header{"Accept": []string{"application/json"}},
	}
	req3 := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/api/other"},
		Header: http.Header{"Accept": []string{"application/json"}},
	}

	key1 := CacheKey(req1, []string{"Accept"})
	key2 := CacheKey(req2, []string{"Accept"})
	key3 := CacheKey(req3, []string{"Accept"})

	if key1 != key2 {
		t.Error("expected same cache key for identical requests")
	}
	if key1 == key3 {
		t.Error("expected different cache key for different paths")
	}
}

func TestIsCacheable(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		statusCode int
		headers    http.Header
		want       bool
	}{
		{
			name:       "GET request with 200",
			method:     "GET",
			statusCode: 200,
			headers:    http.Header{},
			want:       true,
		},
		{
			name:       "POST request",
			method:     "POST",
			statusCode: 200,
			headers:    http.Header{},
			want:       false,
		},
		{
			name:       "GET with no-store",
			method:     "GET",
			statusCode: 200,
			headers:    http.Header{"Cache-Control": []string{"no-store"}},
			want:       false,
		},
		{
			name:       "GET with 500",
			method:     "GET",
			statusCode: 500,
			headers:    http.Header{},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Method: tt.method}
			got := IsCacheable(req, tt.statusCode, tt.headers)
			if got != tt.want {
				t.Errorf("IsCacheable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTTL(t *testing.T) {
	defaultTTL := 5 * time.Minute

	tests := []struct {
		name    string
		headers http.Header
		want    time.Duration
	}{
		{
			name:    "no cache headers",
			headers: http.Header{},
			want:    defaultTTL,
		},
		{
			name:    "max-age 60",
			headers: http.Header{"Cache-Control": []string{"max-age=60"}},
			want:    60 * time.Second,
		},
		{
			name:    "max-age with other directives",
			headers: http.Header{"Cache-Control": []string{"public, max-age=120"}},
			want:    120 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseTTL(tt.headers, defaultTTL)
			if got != tt.want {
				t.Errorf("ParseTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateETag(t *testing.T) {
	body1 := []byte("test data")
	body2 := []byte("test data")
	body3 := []byte("different data")

	etag1 := GenerateETag(body1)
	etag2 := GenerateETag(body2)
	etag3 := GenerateETag(body3)

	if etag1 != etag2 {
		t.Error("expected same ETag for identical data")
	}
	if etag1 == etag3 {
		t.Error("expected different ETag for different data")
	}
	if etag1[0] != '"' || etag1[len(etag1)-1] != '"' {
		t.Error("ETag should be quoted")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewMemoryCache(10*1024*1024, 5*time.Minute)
	entry := &Entry{
		Body:      []byte("test data"),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
		Size:      9,
	}
	cache.Set("test-key", entry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("test-key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewMemoryCache(10*1024*1024, 5*time.Minute)
	entry := &Entry{
		Body:      []byte("test data"),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
		Size:      9,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("test-key", entry)
	}
}

