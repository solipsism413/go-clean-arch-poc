// Package memory_test contains tests for the in-memory event bus.
package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBus(t *testing.T) {
	t.Run("should create new event bus", func(t *testing.T) {
		bus := memory.NewEventBus()

		assert.NotNil(t, bus)
	})
}

func TestEventBus_Publish(t *testing.T) {
	ctx := context.Background()

	t.Run("should publish event without subscribers", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		evt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")

		err := bus.Publish(ctx, "user.events", &evt)
		assert.NoError(t, err)
	})

	t.Run("should publish event to subscriber", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "user.events"
		userID := uuid.New()
		evt := event.NewUserCreated(userID, "test@example.com", "Test User")

		// Subscribe first
		ch, err := bus.Subscribe(ctx, topic, "test-group")
		require.NoError(t, err)

		// Publish event
		err = bus.Publish(ctx, topic, &evt)
		require.NoError(t, err)

		// Receive event
		select {
		case receivedEvt := <-ch:
			assert.Equal(t, "user.created", receivedEvt.EventType())
			assert.Equal(t, userID, receivedEvt.AggregateID())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("should publish to multiple subscribers", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "user.events"
		userID := uuid.New()
		evt := event.NewUserCreated(userID, "test@example.com", "Test User")

		// Create multiple subscribers
		ch1, err := bus.Subscribe(ctx, topic, "group1")
		require.NoError(t, err)
		ch2, err := bus.Subscribe(ctx, topic, "group2")
		require.NoError(t, err)

		// Publish event
		err = bus.Publish(ctx, topic, &evt)
		require.NoError(t, err)

		// Both subscribers should receive the event
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			select {
			case receivedEvt := <-ch1:
				assert.Equal(t, userID, receivedEvt.AggregateID())
			case <-time.After(100 * time.Millisecond):
				t.Error("subscriber 1 timeout")
			}
		}()

		go func() {
			defer wg.Done()
			select {
			case receivedEvt := <-ch2:
				assert.Equal(t, userID, receivedEvt.AggregateID())
			case <-time.After(100 * time.Millisecond):
				t.Error("subscriber 2 timeout")
			}
		}()

		wg.Wait()
	})

	t.Run("should not block when channel is full", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "high-volume"

		ch, err := bus.Subscribe(ctx, topic, "test-group")
		require.NoError(t, err)

		// Publish more events than channel capacity (100)
		for i := 0; i < 150; i++ {
			evt := event.NewUserLoggedIn(uuid.New(), "127.0.0.1", "test-agent")
			err := bus.Publish(ctx, topic, &evt)
			assert.NoError(t, err)
		}

		// Drain some events to verify they were received
		eventCount := 0
		for {
			select {
			case <-ch:
				eventCount++
			default:
				goto done
			}
		}
	done:
		assert.LessOrEqual(t, eventCount, 100) // Should not exceed channel capacity
	})
}

func TestEventBus_PublishBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should publish batch of events", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "user.events"

		ch, err := bus.Subscribe(ctx, topic, "test-group")
		require.NoError(t, err)

		// Create batch of events
		events := []event.Event{
			func() event.Event { e := event.NewUserCreated(uuid.New(), "user1@example.com", "User 1"); return &e }(),
			func() event.Event { e := event.NewUserCreated(uuid.New(), "user2@example.com", "User 2"); return &e }(),
			func() event.Event { e := event.NewUserCreated(uuid.New(), "user3@example.com", "User 3"); return &e }(),
		}

		err = bus.PublishBatch(ctx, topic, events)
		require.NoError(t, err)

		// Receive all events
		receivedCount := 0
		timeout := time.After(200 * time.Millisecond)
		for receivedCount < len(events) {
			select {
			case <-ch:
				receivedCount++
			case <-timeout:
				t.Fatalf("timeout: received only %d of %d events", receivedCount, len(events))
			}
		}
		assert.Equal(t, len(events), receivedCount)
	})

	t.Run("should publish empty batch without error", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		err := bus.PublishBatch(ctx, "topic", []event.Event{})
		assert.NoError(t, err)
	})
}

func TestEventBus_Subscribe(t *testing.T) {
	ctx := context.Background()

	t.Run("should subscribe to topic", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		ch, err := bus.Subscribe(ctx, "test.topic", "test-group")

		require.NoError(t, err)
		assert.NotNil(t, ch)
	})

	t.Run("should allow multiple subscriptions to same topic", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		ch1, err := bus.Subscribe(ctx, "test.topic", "group1")
		require.NoError(t, err)

		ch2, err := bus.Subscribe(ctx, "test.topic", "group2")
		require.NoError(t, err)

		assert.NotNil(t, ch1)
		assert.NotNil(t, ch2)
		assert.NotEqual(t, ch1, ch2)
	})

	t.Run("should subscribe to different topics", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		ch1, err := bus.Subscribe(ctx, "topic1", "group")
		require.NoError(t, err)

		ch2, err := bus.Subscribe(ctx, "topic2", "group")
		require.NoError(t, err)

		// Publish to topic1 only
		evt := event.NewUserCreated(uuid.New(), "test@example.com", "Test")
		err = bus.Publish(ctx, "topic1", &evt)
		require.NoError(t, err)

		// Only ch1 should receive the event
		select {
		case <-ch1:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("ch1 should have received event")
		}

		select {
		case <-ch2:
			t.Fatal("ch2 should not have received event")
		case <-time.After(50 * time.Millisecond):
			// Expected
		}
	})
}

func TestEventBus_Unsubscribe(t *testing.T) {
	ctx := context.Background()

	t.Run("should unsubscribe from topic", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "test.topic"

		ch, err := bus.Subscribe(ctx, topic, "test-group")
		require.NoError(t, err)

		err = bus.Unsubscribe(ctx, topic)
		require.NoError(t, err)

		// Channel should be closed
		_, ok := <-ch
		assert.False(t, ok, "channel should be closed")
	})

	t.Run("should not error when unsubscribing from non-existent topic", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		err := bus.Unsubscribe(ctx, "non-existent")
		assert.NoError(t, err)
	})

	t.Run("should close all channels for topic", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "multi-subscriber"

		ch1, _ := bus.Subscribe(ctx, topic, "group1")
		ch2, _ := bus.Subscribe(ctx, topic, "group2")

		err := bus.Unsubscribe(ctx, topic)
		require.NoError(t, err)

		// Both channels should be closed
		_, ok1 := <-ch1
		_, ok2 := <-ch2
		assert.False(t, ok1)
		assert.False(t, ok2)
	})
}

func TestEventBus_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("should close all subscriptions", func(t *testing.T) {
		bus := memory.NewEventBus()

		ch1, _ := bus.Subscribe(ctx, "topic1", "group1")
		ch2, _ := bus.Subscribe(ctx, "topic2", "group2")

		err := bus.Close()
		require.NoError(t, err)

		// All channels should be closed
		_, ok1 := <-ch1
		_, ok2 := <-ch2
		assert.False(t, ok1)
		assert.False(t, ok2)
	})

	t.Run("should close without subscriptions", func(t *testing.T) {
		bus := memory.NewEventBus()

		err := bus.Close()
		assert.NoError(t, err)
	})
}

func TestEventBus_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("should handle concurrent publish and subscribe", func(t *testing.T) {
		bus := memory.NewEventBus()
		defer bus.Close()

		topic := "concurrent.events"
		eventCount := 100

		// Start subscriber
		ch, err := bus.Subscribe(ctx, topic, "test-group")
		require.NoError(t, err)

		// Publish events concurrently
		var wg sync.WaitGroup
		for i := 0; i < eventCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				evt := event.NewUserLoggedIn(uuid.New(), "127.0.0.1", "test-agent")
				err := bus.Publish(ctx, topic, &evt)
				assert.NoError(t, err)
			}()
		}

		// Drain received events in background
		receivedCount := 0
		done := make(chan struct{})
		go func() {
			defer close(done)
			timeout := time.After(1 * time.Second)
			for {
				select {
				case _, ok := <-ch:
					if !ok {
						return
					}
					receivedCount++
				case <-timeout:
					return
				}
			}
		}()

		wg.Wait()
		bus.Unsubscribe(ctx, topic)
		<-done

		// Should have received at least some events
		assert.Greater(t, receivedCount, 0)
	})
}
