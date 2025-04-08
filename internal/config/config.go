package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server" yaml:"server"`
	Upstream UpstreamConfig `json:"upstream" yaml:"upstream"`
	Cache    CacheConfig    `json:"cache" yaml:"cache"`
	RateLimit RateLimitConfig `json:"ratelimit" yaml:"ratelimit"`
	Logging  LoggingConfig  `json:"logging" yaml:"logging"`
	Metrics  MetricsConfig  `json:"metrics" yaml:"metrics"`
}

// ServerConfig holds server-specific settings
type ServerConfig struct {
	Address         string        `json:"address" yaml:"address"`
	Port            int           `json:"port" yaml:"port"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}

// UpstreamConfig holds upstream service settings
type UpstreamConfig struct {
	URL               string        `json:"url" yaml:"url"`
	Timeout           time.Duration `json:"timeout" yaml:"timeout"`
	MaxIdleConns      int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxConnsPerHost   int           `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	IdleConnTimeout   time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" yaml:"tls_handshake_timeout"`
	ForbiddenHeaders  []string      `json:"forbidden_headers" yaml:"forbidden_headers"`
}

// CacheConfig holds cache settings
type CacheConfig struct {
	Enabled         bool          `json:"enabled" yaml:"enabled"`
	MaxSize         int64         `json:"max_size" yaml:"max_size"`
	DefaultTTL      time.Duration `json:"default_ttl" yaml:"default_ttl"`
	RespectCacheControl bool       `json:"respect_cache_control" yaml:"respect_cache_control"`
	Type            string        `json:"type" yaml:"type"` // "memory" or "redis"
	Redis           RedisConfig   `json:"redis" yaml:"redis"`
}

// RedisConfig holds Redis-specific cache settings
type RedisConfig struct {
	Address  string `json:"address" yaml:"address"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled"`
	RequestsPerSecond int      `json:"requests_per_second" yaml:"requests_per_second"`
	Burst        int           `json:"burst" yaml:"burst"`
	ByIP         bool          `json:"by_ip" yaml:"by_ip"`
	ByAPIKey     bool          `json:"by_api_key" yaml:"by_api_key"`
	APIKeyHeader string        `json:"api_key_header" yaml:"api_key_header"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"` // "json" or "console"
	OutputPath string `json:"output_path" yaml:"output_path"`
}

// MetricsConfig holds metrics settings
type MetricsConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Path    string `json:"path" yaml:"path"`
	Port    int    `json:"port" yaml:"port"`
}

// Load loads configuration from a file or environment variables
func Load(filePath string) (*Config, error) {
	cfg := defaultConfig()

	if filePath != "" {
		if err := loadFromFile(filePath, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// defaultConfig returns default configuration values
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:         "0.0.0.0",
			Port:            8080,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		Upstream: UpstreamConfig{
			URL:                 "http://localhost:8081",
			Timeout:             30 * time.Second,
			MaxIdleConns:        100,
			MaxConnsPerHost:     100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ForbiddenHeaders:    []string{"Authorization", "Cookie", "Set-Cookie"},
		},
		Cache: CacheConfig{
			Enabled:             true,
			MaxSize:             100 * 1024 * 1024, // 100 MB
			DefaultTTL:          5 * time.Minute,
			RespectCacheControl: true,
			Type:                "memory",
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 100,
			Burst:             200,
			ByIP:              true,
			ByAPIKey:          false,
			APIKeyHeader:      "X-API-Key",
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Port:    9090,
		},
	}
}

// loadFromFile loads configuration from a YAML or JSON file
func loadFromFile(filePath string, cfg *Config) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	ext := filePath[len(filePath)-4:]
	switch ext {
	case "yaml", ".yml":
		return yaml.Unmarshal(data, cfg)
	case "json":
		return json.Unmarshal(data, cfg)
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return json.Unmarshal(data, cfg)
		}
		return nil
	}
}

// loadFromEnv overrides configuration with environment variables
func loadFromEnv(cfg *Config) error {
	if v := os.Getenv("PROXY_SERVER_ADDRESS"); v != "" {
		cfg.Server.Address = v
	}
	if v := os.Getenv("PROXY_SERVER_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("PROXY_UPSTREAM_URL"); v != "" {
		cfg.Upstream.URL = v
	}
	if v := os.Getenv("PROXY_CACHE_ENABLED"); v != "" {
		cfg.Cache.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("PROXY_RATELIMIT_ENABLED"); v != "" {
		cfg.RateLimit.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("PROXY_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Upstream.URL == "" {
		return fmt.Errorf("upstream URL is required")
	}
	if c.Cache.Enabled && c.Cache.MaxSize <= 0 {
		return fmt.Errorf("cache max size must be positive")
	}
	if c.RateLimit.Enabled && c.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("rate limit requests per second must be positive")
	}
	return nil
}

