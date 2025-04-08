package cache

import (
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Entry represents a cached HTTP response
type Entry struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	ETag       string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	Size       int64
}

// Cache is the interface for cache implementations
type Cache interface {
	Get(key string) (*Entry, bool)
	Set(key string, entry *Entry)
	Delete(key string)
	Clear()
	Size() int64
	Len() int
}

// memoryCache implements an LRU cache with TTL
type memoryCache struct {
	mu       sync.RWMutex
	maxSize  int64
	size     int64
	items    map[string]*list.Element
	lru      *list.List
	defaultTTL time.Duration
}

type cacheItem struct {
	key   string
	entry *Entry
}

// NewMemoryCache creates a new in-memory LRU cache
func NewMemoryCache(maxSize int64, defaultTTL time.Duration) Cache {
	return &memoryCache{
		maxSize:    maxSize,
		items:      make(map[string]*list.Element),
		lru:        list.New(),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves an entry from the cache
func (c *memoryCache) Get(key string) (*Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	item := elem.Value.(*cacheItem)
	
	// Check if entry has expired
	if time.Now().After(item.entry.ExpiresAt) {
		c.deleteElement(elem)
		return nil, false
	}

	// Move to front (most recently used)
	c.lru.MoveToFront(elem)
	return item.entry, true
}

// Set adds an entry to the cache
func (c *memoryCache) Set(key string, entry *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*cacheItem)
		c.size -= item.entry.Size
		item.entry = entry
		c.size += entry.Size
		c.lru.MoveToFront(elem)
	} else {
		// Add new entry
		item := &cacheItem{key: key, entry: entry}
		elem := c.lru.PushFront(item)
		c.items[key] = elem
		c.size += entry.Size
	}

	// Evict if over size limit
	for c.size > c.maxSize && c.lru.Len() > 0 {
		elem := c.lru.Back()
		if elem != nil {
			c.deleteElement(elem)
		}
	}
}

// Delete removes an entry from the cache
func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.deleteElement(elem)
	}
}

// Clear removes all entries from the cache
func (c *memoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
	c.size = 0
}

// Size returns the total size of cached data in bytes
func (c *memoryCache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Len returns the number of entries in the cache
func (c *memoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// deleteElement removes an element from the cache (must be called with lock held)
func (c *memoryCache) deleteElement(elem *list.Element) {
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
	c.lru.Remove(elem)
	c.size -= item.entry.Size
}

// CacheKey generates a cache key for a request
func CacheKey(r *http.Request, varyHeaders []string) string {
	// Start with method and URL
	parts := []string{r.Method, r.URL.Path}
	
	// Add normalized query parameters (sorted)
	if r.URL.RawQuery != "" {
		parts = append(parts, r.URL.RawQuery)
	}

	// Add varying headers if specified
	for _, header := range varyHeaders {
		if val := r.Header.Get(header); val != "" {
			parts = append(parts, header+":"+val)
		}
	}

	// Generate MD5 hash
	h := md5.New()
	h.Write([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h.Sum(nil))
}

// IsCacheable determines if a request/response is cacheable
func IsCacheable(r *http.Request, statusCode int, headers http.Header) bool {
	// Only cache GET and HEAD requests
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	// Don't cache error responses (except 404)
	if statusCode >= 500 || (statusCode >= 400 && statusCode != 404) {
		return false
	}

	// Check Cache-Control header
	cacheControl := headers.Get("Cache-Control")
	if cacheControl != "" {
		directives := strings.Split(cacheControl, ",")
		for _, directive := range directives {
			directive = strings.TrimSpace(strings.ToLower(directive))
			if directive == "no-store" || directive == "no-cache" || directive == "private" {
				return false
			}
		}
	}

	return true
}

// ParseTTL extracts TTL from Cache-Control header
func ParseTTL(headers http.Header, defaultTTL time.Duration) time.Duration {
	cacheControl := headers.Get("Cache-Control")
	if cacheControl == "" {
		return defaultTTL
	}

	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(strings.ToLower(directive), "max-age=") {
			parts := strings.SplitN(directive, "=", 2)
			if len(parts) == 2 {
				if seconds, err := strconv.Atoi(parts[1]); err == nil {
					return time.Duration(seconds) * time.Second
				}
			}
		}
	}

	// Check Expires header
	if expires := headers.Get("Expires"); expires != "" {
		if t, err := http.ParseTime(expires); err == nil {
			ttl := time.Until(t)
			if ttl > 0 {
				return ttl
			}
		}
	}

	return defaultTTL
}

// GenerateETag generates an ETag for response body
func GenerateETag(body []byte) string {
	h := md5.New()
	h.Write(body)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(h.Sum(nil)))
}

