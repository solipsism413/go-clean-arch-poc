package output

import (
	"context"

	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// EventPublisher defines the output port for publishing domain events.
type EventPublisher interface {
	// Publish publishes a single event.
	Publish(ctx context.Context, topic string, event event.Event) error

	// PublishBatch publishes multiple events.
	PublishBatch(ctx context.Context, topic string, events []event.Event) error

	// Close closes the publisher connection.
	Close() error
}

// EventSubscriber defines the output port for subscribing to domain events.
type EventSubscriber interface {
	// Subscribe subscribes to a topic and returns a channel of events.
	Subscribe(ctx context.Context, topic string, groupID string) (<-chan event.Event, error)

	// Unsubscribe unsubscribes from a topic.
	Unsubscribe(ctx context.Context, topic string) error

	// Close closes the subscriber connection.
	Close() error
}

// EventHandler defines a handler for processing events.
type EventHandler func(ctx context.Context, event event.Event) error

// EventBus combines publisher and subscriber for in-process event handling.
type EventBus interface {
	EventPublisher
	EventSubscriber

	// RegisterHandler registers a handler for a specific event type.
	RegisterHandler(eventType string, handler EventHandler)
}

// Common event topics.
const (
	TopicTaskEvents = "task-events"
	TopicUserEvents = "user-events"
	TopicACLEvents  = "acl-events"
)
