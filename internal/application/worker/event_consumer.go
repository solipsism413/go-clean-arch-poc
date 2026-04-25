// Package worker provides background workers for processing domain events.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// EventHandler is a function that processes a domain event.
type EventHandler func(ctx context.Context, evt event.Event) error

// EventConsumer consumes domain events from a subscriber and routes them to
// registered handlers.
type EventConsumer struct {
	handlers map[string]EventHandler
	logger   *slog.Logger
	mu       sync.RWMutex
	stopCh   chan struct{}
	stopMu   sync.Mutex
	stopped  bool
	wg       sync.WaitGroup
}

// NewEventConsumer creates a new event consumer.
func NewEventConsumer(logger *slog.Logger) *EventConsumer {
	return &EventConsumer{
		handlers: make(map[string]EventHandler),
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a specific event type.
func (c *EventConsumer) RegisterHandler(eventType string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

// Start begins consuming events from the given topics. It spawns a goroutine
// per topic and returns immediately.
func (c *EventConsumer) Start(ctx context.Context, subscriber output.EventSubscriber, topics []string) error {
	for _, topic := range topics {
		eventCh, err := subscriber.Subscribe(ctx, topic, "event-consumer")
		if err != nil {
			// Stop any already-started consumers before returning
			c.signalStopAndWait()
			c.resetStopState()
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
		}

		c.wg.Add(1)
		go func(t string, ch <-chan event.Event) {
			defer c.wg.Done()
			c.logger.Info("event consumer started", "topic", t)
			c.consume(ctx, t, ch)
			c.logger.Info("event consumer stopped", "topic", t)
		}(topic, eventCh)
	}

	return nil
}

// Stop signals the consumer to stop and waits for all goroutines to finish.
// It is safe to call multiple times.
func (c *EventConsumer) Stop() {
	c.signalStopAndWait()
}

func (c *EventConsumer) signalStopAndWait() {
	c.stopMu.Lock()
	if !c.stopped {
		close(c.stopCh)
		c.stopped = true
	}
	c.stopMu.Unlock()
	c.wg.Wait()
}

func (c *EventConsumer) resetStopState() {
	c.stopMu.Lock()
	c.stopCh = make(chan struct{})
	c.stopped = false
	c.stopMu.Unlock()
}

func (c *EventConsumer) consume(ctx context.Context, topic string, eventCh <-chan event.Event) {
	for {
		select {
		case evt, ok := <-eventCh:
			if !ok {
				return
			}
			c.handleEvent(ctx, evt)
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		}
	}
}

func (c *EventConsumer) handleEvent(ctx context.Context, evt event.Event) {
	c.mu.RLock()
	handler, ok := c.handlers[evt.EventType()]
	c.mu.RUnlock()

	if !ok {
		c.logger.Debug("no handler registered for event type",
			"eventType", evt.EventType(),
			"aggregateID", evt.AggregateID(),
		)
		return
	}

	c.logger.Info("processing event",
		"eventType", evt.EventType(),
		"aggregateID", evt.AggregateID(),
		"occurredAt", evt.OccurredAt(),
	)

	if err := handler(ctx, evt); err != nil {
		c.logger.Error("event handler failed",
			"eventType", evt.EventType(),
			"aggregateID", evt.AggregateID(),
			"error", err,
		)
	}
}
