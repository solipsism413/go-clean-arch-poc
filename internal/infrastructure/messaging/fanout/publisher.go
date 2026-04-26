package fanout

import (
	"context"
	"errors"
	"fmt"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// Publisher mirrors domain events to multiple publishers.
type Publisher struct {
	primary   output.EventPublisher
	secondary output.EventPublisher
}

// NewPublisher creates a publisher that fans out to both publishers.
func NewPublisher(primary, secondary output.EventPublisher) *Publisher {
	return &Publisher{primary: primary, secondary: secondary}
}

// Publish publishes a single event to all configured publishers.
func (p *Publisher) Publish(ctx context.Context, topic string, evt event.Event) error {
	var errs []error
	if p.primary != nil {
		if err := p.primary.Publish(ctx, topic, evt); err != nil {
			errs = append(errs, fmt.Errorf("primary publish failed: %w", err))
		}
	}
	if p.secondary != nil {
		if err := p.secondary.Publish(ctx, topic, evt); err != nil {
			errs = append(errs, fmt.Errorf("secondary publish failed: %w", err))
		}
	}
	return errors.Join(errs...)
}

// PublishBatch publishes multiple events to all configured publishers.
func (p *Publisher) PublishBatch(ctx context.Context, topic string, events []event.Event) error {
	var errs []error
	if p.primary != nil {
		if err := p.primary.PublishBatch(ctx, topic, events); err != nil {
			errs = append(errs, fmt.Errorf("primary batch publish failed: %w", err))
		}
	}
	if p.secondary != nil {
		if err := p.secondary.PublishBatch(ctx, topic, events); err != nil {
			errs = append(errs, fmt.Errorf("secondary batch publish failed: %w", err))
		}
	}
	return errors.Join(errs...)
}

// Close closes both publishers.
func (p *Publisher) Close() error {
	var errs []error
	if p.primary != nil {
		if err := p.primary.Close(); err != nil {
			errs = append(errs, fmt.Errorf("primary close failed: %w", err))
		}
	}
	if p.secondary != nil {
		if err := p.secondary.Close(); err != nil {
			errs = append(errs, fmt.Errorf("secondary close failed: %w", err))
		}
	}
	return errors.Join(errs...)
}
