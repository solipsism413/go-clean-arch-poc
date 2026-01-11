// Package tracer_test contains tests for the OpenTelemetry tracer.
package tracer_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/observability/tracer"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should create tracer with tracing disabled", func(t *testing.T) {
		cfg := config.OTELConfig{
			Enabled:     false,
			ServiceName: "test-service",
		}

		tr, err := tracer.New(ctx, cfg, logger)

		require.NoError(t, err)
		assert.NotNil(t, tr)
	})

	t.Run("should return tracer even when disabled", func(t *testing.T) {
		cfg := config.OTELConfig{
			Enabled:     false,
			ServiceName: "test-service",
		}

		tr, err := tracer.New(ctx, cfg, logger)
		require.NoError(t, err)

		// Should be able to start spans without error
		spanCtx, span := tr.Start(ctx, "test-span")
		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
	})
}

func TestTracer_Start(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should start a new span", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation")

		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("should create child span from parent context", func(t *testing.T) {
		parentCtx, parentSpan := tr.Start(ctx, "parent-operation")
		defer parentSpan.End()

		childCtx, childSpan := tr.Start(parentCtx, "child-operation")
		defer childSpan.End()

		assert.NotNil(t, childCtx)
		assert.NotNil(t, childSpan)
	})
}

func TestTracer_StartWithAttributes(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should start span with attributes", func(t *testing.T) {
		spanCtx, span := tr.StartWithAttributes(ctx, "test-operation-with-attrs",
			attribute.String("key", "value"),
			attribute.Int("count", 42),
		)

		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
	})
}

func TestTracer_Shutdown(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	t.Run("should shutdown tracer without error when disabled", func(t *testing.T) {
		cfg := config.OTELConfig{
			Enabled:     false,
			ServiceName: "test-service",
		}

		tr, err := tracer.New(ctx, cfg, logger)
		require.NoError(t, err)

		err = tr.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestSpanFromContext(t *testing.T) {
	t.Run("should return span from context", func(t *testing.T) {
		ctx := context.Background()
		logger := getTestLogger()

		cfg := config.OTELConfig{
			Enabled:     false,
			ServiceName: "test-service",
		}

		tr, err := tracer.New(ctx, cfg, logger)
		require.NoError(t, err)

		spanCtx, span := tr.Start(ctx, "test-operation")
		defer span.End()

		retrievedSpan := tracer.SpanFromContext(spanCtx)
		assert.NotNil(t, retrievedSpan)
	})

	t.Run("should return noop span when no span in context", func(t *testing.T) {
		ctx := context.Background()

		span := tracer.SpanFromContext(ctx)

		assert.NotNil(t, span)
		// The span should be a no-op span
		assert.False(t, span.SpanContext().IsValid())
	})
}

func TestAddEvent(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should add event to span without error", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation")
		defer span.End()

		// This should not panic
		tracer.AddEvent(spanCtx, "test-event",
			attribute.String("event-key", "event-value"),
		)
	})

	t.Run("should handle adding event to context without span", func(t *testing.T) {
		// This should not panic even without a valid span
		tracer.AddEvent(ctx, "test-event")
	})
}

func TestSetAttribute(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	spanCtx, span := tr.Start(ctx, "test-operation")
	defer span.End()

	t.Run("should set string attribute", func(t *testing.T) {
		tracer.SetAttribute(spanCtx, "string-key", "string-value")
	})

	t.Run("should set int attribute", func(t *testing.T) {
		tracer.SetAttribute(spanCtx, "int-key", 42)
	})

	t.Run("should set int64 attribute", func(t *testing.T) {
		tracer.SetAttribute(spanCtx, "int64-key", int64(9223372036854775807))
	})

	t.Run("should set float64 attribute", func(t *testing.T) {
		tracer.SetAttribute(spanCtx, "float64-key", 3.14159)
	})

	t.Run("should set bool attribute", func(t *testing.T) {
		tracer.SetAttribute(spanCtx, "bool-key", true)
	})

	t.Run("should handle unsupported type gracefully", func(t *testing.T) {
		// This should not panic
		tracer.SetAttribute(spanCtx, "unsupported-key", []string{"array", "value"})
	})
}

func TestRecordError(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should record error on span", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation")
		defer span.End()

		testErr := errors.New("test error")
		tracer.RecordError(spanCtx, testErr)
	})

	t.Run("should handle recording error on context without span", func(t *testing.T) {
		testErr := errors.New("test error")
		// This should not panic
		tracer.RecordError(ctx, testErr)
	})
}

func TestGetTraceID(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should return empty string when no valid span", func(t *testing.T) {
		traceID := tracer.GetTraceID(ctx)
		assert.Equal(t, "", traceID)
	})

	t.Run("should return trace ID from valid span context", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation")
		defer span.End()

		// When tracing is disabled, the span context is not valid
		// so this returns empty string
		traceID := tracer.GetTraceID(spanCtx)
		// With tracing disabled, the trace ID will be empty
		assert.Equal(t, "", traceID)
	})
}

func TestGetSpanID(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should return empty string when no valid span", func(t *testing.T) {
		spanID := tracer.GetSpanID(ctx)
		assert.Equal(t, "", spanID)
	})

	t.Run("should return span ID from valid span context", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation")
		defer span.End()

		// When tracing is disabled, the span context is not valid
		// so this returns empty string
		spanID := tracer.GetSpanID(spanCtx)
		// With tracing disabled, the span ID will be empty
		assert.Equal(t, "", spanID)
	})
}

func TestSpanOptions(t *testing.T) {
	ctx := context.Background()
	logger := getTestLogger()

	cfg := config.OTELConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	tr, err := tracer.New(ctx, cfg, logger)
	require.NoError(t, err)

	t.Run("should accept span start options", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation",
			trace.WithSpanKind(trace.SpanKindServer),
		)

		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
	})

	t.Run("should accept multiple span options", func(t *testing.T) {
		spanCtx, span := tr.Start(ctx, "test-operation",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attribute.String("key", "value")),
		)

		assert.NotNil(t, spanCtx)
		assert.NotNil(t, span)
		span.End()
	})
}
