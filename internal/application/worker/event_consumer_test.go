package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/worker"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/memory"
	"github.com/stretchr/testify/assert"
)

func TestEventConsumer_RegisterAndProcessEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	var called atomic.Bool
	consumer.RegisterHandler("user.created", func(ctx context.Context, evt event.Event) error {
		called.Store(true)
		return nil
	})

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Publish an event
	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Wait for handler to be called
	time.Sleep(100 * time.Millisecond)
	assert.True(t, called.Load(), "handler should have been called")

	consumer.Stop()
}

func TestEventConsumer_NoHandlerForEventType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	// Do not register any handler

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Publish an event with no handler
	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Should not panic; just wait briefly
	time.Sleep(50 * time.Millisecond)

	consumer.Stop()
}

func TestEventConsumer_HandlerError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	consumer.RegisterHandler("user.created", func(ctx context.Context, evt event.Event) error {
		return errors.New("handler error")
	})

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Wait for handler to process (error should be logged, not returned)
	time.Sleep(50 * time.Millisecond)

	consumer.Stop()
}

func TestEventConsumer_Stop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Stop should complete without panic or deadlock
	consumer.Stop()
}
