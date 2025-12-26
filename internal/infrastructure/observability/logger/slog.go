// Package logger provides structured logging using slog.
package logger

import (
	"context"
	"log/slog"
	"os"
)

// ContextKey is the type for context keys.
type ContextKey string

// Context keys for logging.
const (
	RequestIDKey ContextKey = "requestId"
	UserIDKey    ContextKey = "userId"
	TraceIDKey   ContextKey = "traceId"
	SpanIDKey    ContextKey = "spanId"
)

// Config holds logger configuration.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// New creates a new structured logger.
func New(cfg Config) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// WithContext returns a logger with context values added.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	attrs := make([]any, 0)

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		attrs = append(attrs, "requestId", requestID)
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		attrs = append(attrs, "userId", userID)
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		attrs = append(attrs, "traceId", traceID)
	}
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		attrs = append(attrs, "spanId", spanID)
	}

	if len(attrs) > 0 {
		return logger.With(attrs...)
	}
	return logger
}

// SetRequestID adds a request ID to the context.
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// SetUserID adds a user ID to the context.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// SetTraceID adds a trace ID to the context.
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
