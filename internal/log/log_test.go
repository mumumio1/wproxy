package log

import (
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "json logger",
			cfg: Config{
				Level:      "info",
				Format:     "json",
				OutputPath: "stdout",
			},
			wantErr: false,
		},
		{
			name: "console logger",
			cfg: Config{
				Level:      "debug",
				Format:     "console",
				OutputPath: "stdout",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("NewLogger() returned nil logger")
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	logger := NewNopLogger()

	// Should not panic
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 1))
	logger.Warn("warn message", Bool("flag", true))
	logger.Error("error message", Error(nil))

	// Test With
	child := logger.With(String("component", "test"))
	if child == nil {
		t.Error("With() returned nil")
	}

	// Test WithContext
	ctx := context.WithValue(context.Background(), "request_id", "test-123")
	ctxLogger := logger.WithContext(ctx)
	if ctxLogger == nil {
		t.Error("WithContext() returned nil")
	}
	ctxLogger.Info("test message")
}

func BenchmarkLogger(b *testing.B) {
	logger := NewNopLogger()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key", "value"),
			Int("count", i),
		)
	}
}

