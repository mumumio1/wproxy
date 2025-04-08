package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	registry           *prometheus.Registry
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	requestSize        *prometheus.HistogramVec
	responseSize       *prometheus.HistogramVec
	cacheHits          *prometheus.CounterVec
	cacheMisses        *prometheus.CounterVec
	rateLimitDropped   prometheus.Counter
	activeConnections  prometheus.Gauge
}

var (
	defaultBuckets = []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
)

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	
	m := &Metrics{
		registry: reg,
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: defaultBuckets,
			},
			[]string{"method", "path", "status"},
		),
		requestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"method", "path"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"method", "path"},
		),
		rateLimitDropped: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "rate_limit_dropped_total",
				Help: "Total number of requests dropped by rate limiter",
			},
		),
		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active connections",
			},
		),
	}

	// Register metrics with custom registry (for tests)
	reg.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestSize,
		m.responseSize,
		m.cacheHits,
		m.cacheMisses,
		m.rateLimitDropped,
		m.activeConnections,
	)

	return m
}

// RecordRequest records request metrics
func (m *Metrics) RecordRequest(method, path string, status int, duration time.Duration, requestSize, responseSize int64) {
	statusStr := strconv.Itoa(status)
	m.requestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.requestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())
	m.requestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.responseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(method, path string) {
	m.cacheHits.WithLabelValues(method, path).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(method, path string) {
	m.cacheMisses.WithLabelValues(method, path).Inc()
}

// RecordRateLimitDrop records a rate limit drop
func (m *Metrics) RecordRateLimitDrop() {
	m.rateLimitDropped.Inc()
}

// IncActiveConnections increments active connections
func (m *Metrics) IncActiveConnections() {
	m.activeConnections.Inc()
}

// DecActiveConnections decrements active connections
func (m *Metrics) DecActiveConnections() {
	m.activeConnections.Dec()
}

// Handler returns the Prometheus HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

