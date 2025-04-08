package cache

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func FuzzCacheKey(f *testing.F) {
	// Seed corpus
	f.Add("GET", "/api/test", "q=1", "application/json")
	f.Add("POST", "/api/data", "", "")
	f.Add("GET", "/", "foo=bar&baz=qux", "text/html")

	f.Fuzz(func(t *testing.T, method, path, query, accept string) {
		req := &http.Request{
			Method: method,
			URL:    &url.URL{Path: path, RawQuery: query},
			Header: http.Header{},
		}
		if accept != "" {
			req.Header.Set("Accept", accept)
		}

		// Should not panic
		key := CacheKey(req, []string{"Accept"})
		if key == "" {
			t.Error("CacheKey returned empty string")
		}
	})
}

func FuzzParseTTL(f *testing.F) {
	// Seed corpus
	f.Add("max-age=60")
	f.Add("public, max-age=3600, must-revalidate")
	f.Add("no-cache")
	f.Add("")

	f.Fuzz(func(t *testing.T, cacheControl string) {
		headers := http.Header{}
		if cacheControl != "" {
			headers.Set("Cache-Control", cacheControl)
		}

		// Should not panic
		ttl := ParseTTL(headers, 5*time.Minute)
		if ttl < 0 {
			t.Error("ParseTTL returned negative duration")
		}
	})
}

func FuzzGenerateETag(f *testing.F) {
	// Seed corpus
	f.Add([]byte("test data"))
	f.Add([]byte(""))
	f.Add([]byte("a very long string that should still generate a valid etag"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		etag := GenerateETag(data)
		if len(etag) < 2 {
			t.Error("ETag too short")
		}
		if etag[0] != '"' || etag[len(etag)-1] != '"' {
			t.Error("ETag should be quoted")
		}
	})
}

func FuzzIsCacheable(f *testing.F) {
	// Seed corpus
	f.Add("GET", 200, "max-age=60")
	f.Add("POST", 200, "")
	f.Add("GET", 500, "max-age=60")
	f.Add("GET", 200, "no-store")

	f.Fuzz(func(t *testing.T, method string, statusCode int, cacheControl string) {
		req := &http.Request{Method: method}
		headers := http.Header{}
		if cacheControl != "" {
			headers.Set("Cache-Control", cacheControl)
		}

		// Should not panic
		_ = IsCacheable(req, statusCode, headers)
	})
}

