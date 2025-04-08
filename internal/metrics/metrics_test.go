package metrics

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
}

func TestRecordRequest(t *testing.T) {
	m := NewMetrics()
	m.RecordRequest("GET", "/api/test", 200, 10*time.Millisecond, 1024, 2048)
	// No panic means success
}

func TestRecordCache(t *testing.T) {
	m := NewMetrics()
	m.RecordCacheHit("GET", "/api/test")
	m.RecordCacheMiss("GET", "/api/test")
	// No panic means success
}

func TestRecordRateLimitDrop(t *testing.T) {
	m := NewMetrics()
	m.RecordRateLimitDrop()
	// No panic means success
}

func TestActiveConnections(t *testing.T) {
	m := NewMetrics()
	m.IncActiveConnections()
	m.DecActiveConnections()
	// No panic means success
}

func BenchmarkRecordRequest(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordRequest("GET", "/api/test", 200, time.Millisecond, 1024, 2048)
	}
}

