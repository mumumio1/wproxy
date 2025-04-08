package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mumumio1/wproxy/internal/cache"
	"github.com/mumumio1/wproxy/internal/config"
	"github.com/mumumio1/wproxy/internal/log"
	"github.com/mumumio1/wproxy/internal/metrics"
	"github.com/mumumio1/wproxy/internal/ratelimit"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("wproxy version %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := log.NewLogger(log.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		OutputPath: cfg.Logging.OutputPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting wproxy",
		log.String("version", version),
		log.String("build_time", buildTime),
	)

	// Initialize metrics
	var m *metrics.Metrics
	if cfg.Metrics.Enabled {
		m = metrics.NewMetrics()
		logger.Info("Metrics enabled",
			log.Int("port", cfg.Metrics.Port),
			log.String("path", cfg.Metrics.Path),
		)
	}

	// Initialize cache
	var c cache.Cache
	if cfg.Cache.Enabled {
		c = cache.NewMemoryCache(cfg.Cache.MaxSize, cfg.Cache.DefaultTTL)
		logger.Info("Cache enabled",
			log.Int64("max_size", cfg.Cache.MaxSize),
			log.Duration("default_ttl", cfg.Cache.DefaultTTL),
		)
	}

	// Initialize rate limiter
	var limiter ratelimit.Limiter
	var keyExtractor ratelimit.KeyExtractor
	if cfg.RateLimit.Enabled {
		limiter = ratelimit.NewTokenBucket(
			cfg.RateLimit.RequestsPerSecond,
			cfg.RateLimit.Burst,
		)

		if cfg.RateLimit.ByAPIKey {
			keyExtractor = ratelimit.APIKeyExtractor(cfg.RateLimit.APIKeyHeader)
		} else {
			keyExtractor = ratelimit.IPKeyExtractor
		}

		logger.Info("Rate limiting enabled",
			log.Int("requests_per_second", cfg.RateLimit.RequestsPerSecond),
			log.Int("burst", cfg.RateLimit.Burst),
			log.Bool("by_ip", cfg.RateLimit.ByIP),
			log.Bool("by_api_key", cfg.RateLimit.ByAPIKey),
		)
	}

	// Parse upstream URL
	upstreamURL, err := url.Parse(cfg.Upstream.URL)
	if err != nil {
		logger.Fatal("Invalid upstream URL", log.Error(err))
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = upstreamURL.Scheme
			req.URL.Host = upstreamURL.Host
			req.Host = upstreamURL.Host

			// Remove forbidden headers
			for _, header := range cfg.Upstream.ForbiddenHeaders {
				req.Header.Del(header)
			}
		},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          cfg.Upstream.MaxIdleConns,
			MaxIdleConnsPerHost:   cfg.Upstream.MaxConnsPerHost,
			IdleConnTimeout:       cfg.Upstream.IdleConnTimeout,
			TLSHandshakeTimeout:   cfg.Upstream.TLSHandshakeTimeout,
			ResponseHeaderTimeout: cfg.Upstream.Timeout,
		},
	}

	// Create proxy handler with middleware
	handler := createProxyHandler(proxy, cfg, logger, m, c, limiter, keyExtractor)

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start metrics server if enabled
	var metricsSrv *http.Server
	if cfg.Metrics.Enabled {
		metricsAddr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Metrics.Port)
		metricsSrv = &http.Server{
			Addr:    metricsAddr,
			Handler: m.Handler(),
		}

		go func() {
			logger.Info("Starting metrics server", log.String("address", metricsAddr))
			if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Metrics server error", log.Error(err))
			}
		}()
	}

	// Start main server
	go func() {
		logger.Info("Starting proxy server",
			log.String("address", serverAddr),
			log.String("upstream", cfg.Upstream.URL),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", log.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", log.Error(err))
	}

	if metricsSrv != nil {
		if err := metricsSrv.Shutdown(ctx); err != nil {
			logger.Error("Metrics server shutdown error", log.Error(err))
		}
	}

	logger.Info("Server stopped")
}

// createProxyHandler creates the main HTTP handler with all middleware
func createProxyHandler(
	proxy *httputil.ReverseProxy,
	cfg *config.Config,
	logger log.Logger,
	m *metrics.Metrics,
	c cache.Cache,
	limiter ratelimit.Limiter,
	keyExtractor ratelimit.KeyExtractor,
) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready"}`)
	})

	// Proxy handler
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleProxy(w, r, proxy, cfg, m, c)
	})

	mux.Handle("/", proxyHandler)

	// Apply middleware chain
	var handler http.Handler = mux

	// Request ID middleware
	handler = requestIDMiddleware(handler)

	// Logging middleware
	handler = loggingMiddleware(handler, logger)

	// Metrics middleware
	if m != nil {
		handler = metricsMiddleware(handler, m)
	}

	// Rate limiting middleware
	if limiter != nil {
		handler = rateLimitMiddleware(handler, limiter, keyExtractor, m, logger)
	}

	return handler
}

// handleProxy handles the main proxy logic with caching
func handleProxy(
	w http.ResponseWriter,
	r *http.Request,
	proxy *httputil.ReverseProxy,
	cfg *config.Config,
	m *metrics.Metrics,
	c cache.Cache,
) {
	// Check cache if enabled
	if c != nil && cache.IsCacheable(r, 0, nil) {
		cacheKey := cache.CacheKey(r, nil)

		// Check If-None-Match (ETag)
		if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
			if entry, ok := c.Get(cacheKey); ok && entry.ETag == ifNoneMatch {
				if m != nil {
					m.RecordCacheHit(r.Method, r.URL.Path)
				}
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		// Try to get from cache
		if entry, ok := c.Get(cacheKey); ok {
			if m != nil {
				m.RecordCacheHit(r.Method, r.URL.Path)
			}

			// Write cached response
			for key, values := range entry.Headers {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.Header().Set("X-Cache", "HIT")
			if entry.ETag != "" {
				w.Header().Set("ETag", entry.ETag)
			}
			w.WriteHeader(entry.StatusCode)
			w.Write(entry.Body)
			return
		}

		if m != nil {
			m.RecordCacheMiss(r.Method, r.URL.Path)
		}
	}

	// Cache miss or caching disabled - proxy to upstream
	// Wrap response writer to capture response
	rec := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &[]byte{},
	}

	proxy.ServeHTTP(rec, r)

	// Cache response if applicable
	if c != nil && cache.IsCacheable(r, rec.statusCode, rec.Header()) {
		cacheKey := cache.CacheKey(r, nil)
		ttl := cache.ParseTTL(rec.Header(), cfg.Cache.DefaultTTL)
		etag := cache.GenerateETag(*rec.body)

		entry := &cache.Entry{
			StatusCode: rec.statusCode,
			Headers:    rec.Header().Clone(),
			Body:       *rec.body,
			ETag:       etag,
			ExpiresAt:  time.Now().Add(ttl),
			CreatedAt:  time.Now(),
			Size:       int64(len(*rec.body)),
		}

		c.Set(cacheKey, entry)

		// Set cache headers
		rec.Header().Set("X-Cache", "MISS")
		rec.Header().Set("ETag", etag)
	}
}

// responseRecorder wraps http.ResponseWriter to capture the response
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *[]byte
	written    bool
}

func (rec *responseRecorder) WriteHeader(code int) {
	if !rec.written {
		rec.statusCode = code
		rec.ResponseWriter.WriteHeader(code)
		rec.written = true
	}
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if !rec.written {
		rec.WriteHeader(http.StatusOK)
	}
	*rec.body = append(*rec.body, b...)
	return rec.ResponseWriter.Write(b)
}

// requestIDMiddleware adds a unique request ID to each request
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), log.RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		logger.Info("HTTP request",
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
			log.String("remote_addr", r.RemoteAddr),
			log.Int("status", ww.statusCode),
			log.Duration("duration", duration),
		)
	})
}

// metricsMiddleware records request metrics
func metricsMiddleware(next http.Handler, m *metrics.Metrics) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		m.IncActiveConnections()
		defer m.DecActiveConnections()

		ww := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Get request/response sizes
		requestSize := r.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		responseSize := ww.bytesWritten

		m.RecordRequest(
			r.Method,
			r.URL.Path,
			ww.statusCode,
			duration,
			requestSize,
			responseSize,
		)
	})
}

// rateLimitMiddleware applies rate limiting
func rateLimitMiddleware(
	next http.Handler,
	limiter ratelimit.Limiter,
	keyExtractor ratelimit.KeyExtractor,
	m *metrics.Metrics,
	logger log.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := keyExtractor(r)

		if !limiter.Allow(key) {
			if m != nil {
				m.RecordRateLimitDrop()
			}

			logger.Warn("Rate limit exceeded",
				log.String("key", key),
				log.String("path", r.URL.Path),
			)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", limiter.Wait(key).Seconds()))
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate limit exceeded"}`)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// wrappedWriter wraps http.ResponseWriter to capture status code and bytes written
type wrappedWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	written      bool
}

func (ww *wrappedWriter) WriteHeader(code int) {
	if !ww.written {
		ww.statusCode = code
		ww.ResponseWriter.WriteHeader(code)
		ww.written = true
	}
}

func (ww *wrappedWriter) Write(b []byte) (int, error) {
	if !ww.written {
		ww.WriteHeader(http.StatusOK)
	}
	n, err := ww.ResponseWriter.Write(b)
	ww.bytesWritten += int64(n)
	return n, err
}

func (ww *wrappedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := ww.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack not supported")
	}
	return h.Hijack()
}
