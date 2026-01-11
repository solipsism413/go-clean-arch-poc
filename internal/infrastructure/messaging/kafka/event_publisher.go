// Package kafka provides Kafka event publishing using Redpanda/Kafka.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/pkg/config"
)

// Ensure EventPublisher implements the output.EventPublisher interface.
var _ output.EventPublisher = (*EventPublisher)(nil)

// EventPublisher implements event publishing using Kafka.
type EventPublisher struct {
	producer sarama.SyncProducer
	logger   *slog.Logger
}

// NewEventPublisher creates a new Kafka event publisher.
func NewEventPublisher(ctx context.Context, cfg config.KafkaConfig, logger *slog.Logger) (*EventPublisher, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 5
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger.Info("kafka producer connected", "brokers", cfg.Brokers)

	return &EventPublisher{
		producer: producer,
		logger:   logger,
	}, nil
}

// Publish publishes a single event.
func (p *EventPublisher) Publish(ctx context.Context, topic string, evt event.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Key:       sarama.StringEncoder(evt.AggregateID().String()),
		Value:     sarama.ByteEncoder(data),
		Timestamp: time.Now(),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event_type"), Value: []byte(evt.EventType())},
			{Key: []byte("occurred_at"), Value: []byte(evt.OccurredAt().Format(time.RFC3339))},
		},
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.logger.Debug("event published",
		"topic", topic,
		"eventType", evt.EventType(),
		"partition", partition,
		"offset", offset,
	)

	return nil
}

// PublishBatch publishes multiple events.
func (p *EventPublisher) PublishBatch(ctx context.Context, topic string, events []event.Event) error {
	messages := make([]*sarama.ProducerMessage, 0, len(events))

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		msg := &sarama.ProducerMessage{
			Topic:     topic,
			Key:       sarama.StringEncoder(evt.AggregateID().String()),
			Value:     sarama.ByteEncoder(data),
			Timestamp: time.Now(),
			Headers: []sarama.RecordHeader{
				{Key: []byte("event_type"), Value: []byte(evt.EventType())},
				{Key: []byte("occurred_at"), Value: []byte(evt.OccurredAt().Format(time.RFC3339))},
			},
		}
		messages = append(messages, msg)
	}

	if err := p.producer.SendMessages(messages); err != nil {
		return fmt.Errorf("failed to send batch messages: %w", err)
	}

	p.logger.Debug("batch events published",
		"topic", topic,
		"count", len(events),
	)

	return nil
}

// Close closes the producer connection.
func (p *EventPublisher) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}
	p.logger.Info("kafka producer closed")
	return nil
}

// Ensure EventSubscriber implements the output.EventSubscriber interface.
var _ output.EventSubscriber = (*EventSubscriber)(nil)

// EventSubscriber implements event subscribing using Kafka consumer groups.
type EventSubscriber struct {
	consumerGroup sarama.ConsumerGroup
	logger        *slog.Logger
	handlers      map[string][]output.EventHandler
}

// NewEventSubscriber creates a new Kafka event subscriber.
func NewEventSubscriber(ctx context.Context, cfg config.KafkaConfig, logger *slog.Logger) (*EventSubscriber, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	if cfg.AutoOffsetReset == "earliest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	logger.Info("kafka consumer group connected",
		"brokers", cfg.Brokers,
		"group", cfg.ConsumerGroup,
	)

	return &EventSubscriber{
		consumerGroup: consumerGroup,
		logger:        logger,
		handlers:      make(map[string][]output.EventHandler),
	}, nil
}

// Subscribe subscribes to a topic and returns a channel of events.
func (s *EventSubscriber) Subscribe(ctx context.Context, topic string, groupID string) (<-chan event.Event, error) {
	eventChan := make(chan event.Event, 100)

	go func() {
		defer close(eventChan)

		handler := &consumerGroupHandler{
			eventChan: eventChan,
			logger:    s.logger,
		}

		for {
			if err := s.consumerGroup.Consume(ctx, []string{topic}, handler); err != nil {
				// Check if the consumer group was closed intentionally
				if err == sarama.ErrClosedConsumerGroup {
					return
				}
				s.logger.Error("consumer group error", "error", err)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	return eventChan, nil
}

// Unsubscribe unsubscribes from a topic.
func (s *EventSubscriber) Unsubscribe(ctx context.Context, topic string) error {
	// Consumer group handles this automatically
	return nil
}

// Close closes the subscriber connection.
func (s *EventSubscriber) Close() error {
	if err := s.consumerGroup.Close(); err != nil {
		return fmt.Errorf("failed to close consumer group: %w", err)
	}
	s.logger.Info("kafka consumer group closed")
	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	eventChan chan<- event.Event
	logger    *slog.Logger
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		// Parse event type from headers
		var eventType string
		for _, header := range msg.Headers {
			if string(header.Key) == "event_type" {
				eventType = string(header.Value)
				break
			}
		}

		// Deserialize event based on type
		evt, err := deserializeEvent(eventType, msg.Value)
		if err != nil {
			h.logger.Error("failed to deserialize event",
				"eventType", eventType,
				"error", err,
			)
			session.MarkMessage(msg, "")
			continue
		}

		// Send to channel
		select {
		case h.eventChan <- evt:
			session.MarkMessage(msg, "")
		case <-session.Context().Done():
			return nil
		}
	}
	return nil
}

// deserializeEvent deserializes an event based on its type.
func deserializeEvent(eventType string, data []byte) (event.Event, error) {
	// This is a simplified implementation
	// In production, use a registry pattern for event types
	var evt event.Event
	var err error

	switch eventType {
	case "task.created":
		var e event.TaskCreated
		err = json.Unmarshal(data, &e)
		evt = &e
	case "task.updated":
		var e event.TaskUpdated
		err = json.Unmarshal(data, &e)
		evt = &e
	case "task.deleted":
		var e event.TaskDeleted
		err = json.Unmarshal(data, &e)
		evt = &e
	case "user.created":
		var e event.UserCreated
		err = json.Unmarshal(data, &e)
		evt = &e
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return evt, nil
}
