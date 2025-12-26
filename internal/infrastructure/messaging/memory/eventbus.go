// Package memory provides an in-memory event bus for testing and development.
package memory

import (
	"context"
	"sync"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// Ensure EventBus implements the interfaces.
var (
	_ output.EventPublisher  = (*EventBus)(nil)
	_ output.EventSubscriber = (*EventBus)(nil)
)

// EventBus implements in-memory pub/sub for events.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan event.Event
}

// NewEventBus creates a new in-memory event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan event.Event),
	}
}

// Publish publishes a single event.
func (b *EventBus) Publish(ctx context.Context, topic string, evt event.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if channels, ok := b.subscribers[topic]; ok {
		for _, ch := range channels {
			select {
			case ch <- evt:
			default:
				// Channel full, skip
			}
		}
	}

	return nil
}

// PublishBatch publishes multiple events.
func (b *EventBus) PublishBatch(ctx context.Context, topic string, events []event.Event) error {
	for _, evt := range events {
		if err := b.Publish(ctx, topic, evt); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe subscribes to a topic.
func (b *EventBus) Subscribe(ctx context.Context, topic string, groupID string) (<-chan event.Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan event.Event, 100)
	b.subscribers[topic] = append(b.subscribers[topic], ch)

	return ch, nil
}

// Unsubscribe unsubscribes from a topic.
func (b *EventBus) Unsubscribe(ctx context.Context, topic string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if channels, ok := b.subscribers[topic]; ok {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.subscribers, topic)
	}

	return nil
}

// Close closes the event bus.
func (b *EventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for topic, channels := range b.subscribers {
		for _, ch := range channels {
			close(ch)
		}
		delete(b.subscribers, topic)
	}

	return nil
}
