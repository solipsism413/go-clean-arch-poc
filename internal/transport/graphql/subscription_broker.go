package graphql

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

type SubscriptionBroker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]chan event.Event
}

func NewSubscriptionBroker() *SubscriptionBroker {
	return &SubscriptionBroker{subscribers: make(map[uuid.UUID]chan event.Event)}
}

func (b *SubscriptionBroker) Subscribe(ctx context.Context) <-chan event.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New()
	ch := make(chan event.Event, 32)
	b.subscribers[id] = ch

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		if existing, ok := b.subscribers[id]; ok {
			close(existing)
			delete(b.subscribers, id)
		}
		b.mu.Unlock()
	}()

	return ch
}

func (b *SubscriptionBroker) Publish(evt event.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
		}
	}
}
