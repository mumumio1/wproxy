package log

import (
	"context"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
}

// Field represents a log field
type Field = zapcore.Field

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// RequestIDKey is the context key for request IDs
const RequestIDKey ContextKey = "request_id"

// String creates a string field
func String(key, val string) Field {
	return zap.String(key, val)
}

// Int creates an int field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Duration creates a duration field
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Error creates an error field
func Error(err error) Field {
	return zap.Error(err)
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Any creates a field with any value
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// zapLogger wraps zap.Logger to implement our Logger interface
type zapLogger struct {
	logger *zap.Logger
	ctx    context.Context
}

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // "json" or "console"
	OutputPath string
}

// NewLogger creates a new logger instance
func NewLogger(cfg Config) (Logger, error) {
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writer io.Writer = os.Stdout
	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(writer),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &zapLogger{
		logger: logger,
	}, nil
}

// Debug logs a debug message
func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, l.addContextFields(fields)...)
}

// Info logs an info message
func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, l.addContextFields(fields)...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, l.addContextFields(fields)...)
}

// Error logs an error message
func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, l.addContextFields(fields)...)
}

// Fatal logs a fatal message and exits
func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, l.addContextFields(fields)...)
}

// With creates a child logger with additional fields
func (l *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{
		logger: l.logger.With(fields...),
		ctx:    l.ctx,
	}
}

// WithContext creates a logger with context
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	return &zapLogger{
		logger: l.logger,
		ctx:    ctx,
	}
}

// addContextFields adds request ID from context if present
func (l *zapLogger) addContextFields(fields []Field) []Field {
	if l.ctx == nil {
		return fields
	}

	if requestID := l.ctx.Value(RequestIDKey); requestID != nil {
		fields = append(fields, String("request_id", requestID.(string)))
	}

	return fields
}

// NewNopLogger creates a no-op logger for testing
func NewNopLogger() Logger {
	return &zapLogger{
		logger: zap.NewNop(),
	}
}

