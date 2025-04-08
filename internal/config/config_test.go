package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if !cfg.Cache.Enabled {
		t.Error("expected cache to be enabled by default")
	}
	if !cfg.RateLimit.Enabled {
		t.Error("expected rate limit to be enabled by default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PROXY_SERVER_PORT", "9000")
	os.Setenv("PROXY_UPSTREAM_URL", "http://example.com")
	defer os.Unsetenv("PROXY_SERVER_PORT")
	defer os.Unsetenv("PROXY_UPSTREAM_URL")

	cfg := defaultConfig()
	if err := loadFromEnv(cfg); err != nil {
		t.Fatalf("failed to load from env: %v", err)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Upstream.URL != "http://example.com" {
		t.Errorf("expected upstream URL http://example.com, got %s", cfg.Upstream.URL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     defaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: &Config{
				Server: ServerConfig{Port: 70000},
			},
			wantErr: true,
		},
		{
			name: "missing upstream URL",
			cfg: &Config{
				Server:   ServerConfig{Port: 8080},
				Upstream: UpstreamConfig{URL: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	yamlContent := `
server:
  port: 9090
upstream:
  url: http://test.example.com
cache:
  enabled: false
`
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg := defaultConfig()
	if err := loadFromFile(tmpfile.Name(), cfg); err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Upstream.URL != "http://test.example.com" {
		t.Errorf("expected upstream URL http://test.example.com, got %s", cfg.Upstream.URL)
	}
	if cfg.Cache.Enabled {
		t.Error("expected cache to be disabled")
	}
}

