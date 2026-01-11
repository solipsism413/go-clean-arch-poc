// Package logger_test contains tests for the structured logger.
package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("should create logger with debug level", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "debug",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should create logger with info level", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "info",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should create logger with warn level", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "warn",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should create logger with error level", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "error",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should default to info level for unknown level", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "unknown",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should create JSON handler when format is json", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "info",
			Format: "json",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})

	t.Run("should create text handler when format is not json", func(t *testing.T) {
		cfg := logger.Config{
			Level:  "info",
			Format: "text",
		}

		log := logger.New(cfg)

		assert.NotNil(t, log)
	})
}

func TestWithContext(t *testing.T) {
	t.Run("should add request ID to logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.WithValue(context.Background(), logger.RequestIDKey, "test-request-id")

		enrichedLogger := logger.WithContext(ctx, log)
		enrichedLogger.Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "test-request-id", logEntry["requestId"])
	})

	t.Run("should add user ID to logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.WithValue(context.Background(), logger.UserIDKey, "test-user-id")

		enrichedLogger := logger.WithContext(ctx, log)
		enrichedLogger.Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "test-user-id", logEntry["userId"])
	})

	t.Run("should add trace ID to logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.WithValue(context.Background(), logger.TraceIDKey, "test-trace-id")

		enrichedLogger := logger.WithContext(ctx, log)
		enrichedLogger.Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "test-trace-id", logEntry["traceId"])
	})

	t.Run("should add span ID to logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.WithValue(context.Background(), logger.SpanIDKey, "test-span-id")

		enrichedLogger := logger.WithContext(ctx, log)
		enrichedLogger.Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "test-span-id", logEntry["spanId"])
	})

	t.Run("should add all context values to logger", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.Background()
		ctx = context.WithValue(ctx, logger.RequestIDKey, "req-123")
		ctx = context.WithValue(ctx, logger.UserIDKey, "user-456")
		ctx = context.WithValue(ctx, logger.TraceIDKey, "trace-789")
		ctx = context.WithValue(ctx, logger.SpanIDKey, "span-abc")

		enrichedLogger := logger.WithContext(ctx, log)
		enrichedLogger.Info("test message")

		var logEntry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "req-123", logEntry["requestId"])
		assert.Equal(t, "user-456", logEntry["userId"])
		assert.Equal(t, "trace-789", logEntry["traceId"])
		assert.Equal(t, "span-abc", logEntry["spanId"])
	})

	t.Run("should return original logger when no context values", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewJSONHandler(&buf, nil)
		log := slog.New(handler)

		ctx := context.Background()

		enrichedLogger := logger.WithContext(ctx, log)

		// Should return the same logger reference
		assert.Equal(t, log, enrichedLogger)
	})
}

func TestSetRequestID(t *testing.T) {
	t.Run("should set request ID in context", func(t *testing.T) {
		ctx := context.Background()

		newCtx := logger.SetRequestID(ctx, "test-request-id")

		value, ok := newCtx.Value(logger.RequestIDKey).(string)
		assert.True(t, ok)
		assert.Equal(t, "test-request-id", value)
	})
}

func TestSetUserID(t *testing.T) {
	t.Run("should set user ID in context", func(t *testing.T) {
		ctx := context.Background()

		newCtx := logger.SetUserID(ctx, "test-user-id")

		value, ok := newCtx.Value(logger.UserIDKey).(string)
		assert.True(t, ok)
		assert.Equal(t, "test-user-id", value)
	})
}

func TestSetTraceID(t *testing.T) {
	t.Run("should set trace ID in context", func(t *testing.T) {
		ctx := context.Background()

		newCtx := logger.SetTraceID(ctx, "test-trace-id")

		value, ok := newCtx.Value(logger.TraceIDKey).(string)
		assert.True(t, ok)
		assert.Equal(t, "test-trace-id", value)
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("should get request ID from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), logger.RequestIDKey, "test-request-id")

		result := logger.GetRequestID(ctx)

		assert.Equal(t, "test-request-id", result)
	})

	t.Run("should return empty string when request ID not in context", func(t *testing.T) {
		ctx := context.Background()

		result := logger.GetRequestID(ctx)

		assert.Equal(t, "", result)
	})

	t.Run("should return empty string when context value is wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), logger.RequestIDKey, 123)

		result := logger.GetRequestID(ctx)

		assert.Equal(t, "", result)
	})
}
