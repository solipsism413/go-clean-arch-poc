// Package sse provides Server-Sent Events handler for real-time updates.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// EventType defines SSE event types.
type EventType string

const (
	EventTypeTaskCreated EventType = "task.created"
	EventTypeTaskUpdated EventType = "task.updated"
	EventTypeTaskDeleted EventType = "task.deleted"
	EventTypeHeartbeat   EventType = "heartbeat"
)

// Client represents an SSE client connection.
type Client struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Events  chan Event
	Done    chan struct{}
	Filters map[string]string
}

// Event represents an SSE event.
type Event struct {
	Type EventType
	ID   string
	Data any
}

// Broker manages SSE client connections and broadcasts events.
type Broker struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewBroker creates a new SSE broker.
func NewBroker(logger *slog.Logger) *Broker {
	return &Broker{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 256),
		logger:     logger,
	}
}

// Run starts the broker's main loop.
func (b *Broker) Run(ctx context.Context) {
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			for client := range b.clients {
				close(client.Events)
				delete(b.clients, client)
			}
			b.mu.Unlock()
			return

		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
			b.logger.Debug("sse client connected", "clientId", client.ID)

		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				close(client.Events)
				delete(b.clients, client)
			}
			b.mu.Unlock()
			b.logger.Debug("sse client disconnected", "clientId", client.ID)

		case event := <-b.broadcast:
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client.Events <- event:
				default:
					// Client buffer full, skip
				}
			}
			b.mu.RUnlock()

		case <-heartbeatTicker.C:
			// Send heartbeat to keep connections alive
			b.broadcast <- Event{
				Type: EventTypeHeartbeat,
				ID:   uuid.New().String(),
				Data: map[string]int64{"timestamp": time.Now().Unix()},
			}
		}
	}
}

// Publish publishes an event to all connected clients.
func (b *Broker) Publish(eventType EventType, data any) {
	b.broadcast <- Event{
		Type: eventType,
		ID:   uuid.New().String(),
		Data: data,
	}
}

// Handler handles SSE HTTP requests.
type Handler struct {
	broker *Broker
	logger *slog.Logger
}

// NewHandler creates a new SSE handler.
func NewHandler(broker *Broker, logger *slog.Logger) *Handler {
	return &Handler{
		broker: broker,
		logger: logger,
	}
}

// ServeHTTP handles SSE requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create client
	client := &Client{
		ID:      uuid.New(),
		Events:  make(chan Event, 100),
		Done:    make(chan struct{}),
		Filters: make(map[string]string),
	}

	// Register client
	h.broker.register <- client

	// Cleanup on disconnect
	defer func() {
		h.broker.unregister <- client
	}()

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"clientId\":\"%s\"}\n\n", client.ID)
	flusher.Flush()

	// Listen for events
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-client.Events:
			if !ok {
				return
			}

			// Serialize data
			data, err := json.Marshal(event.Data)
			if err != nil {
				h.logger.Error("failed to marshal event data", "error", err)
				continue
			}

			// Write SSE event
			fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data)
			flusher.Flush()
		}
	}
}

// EventHandler handles domain events and publishes to SSE clients.
type EventHandler struct {
	broker *Broker
	logger *slog.Logger
}

// NewEventHandler creates a new event handler.
func NewEventHandler(broker *Broker, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		broker: broker,
		logger: logger,
	}
}

// HandleEvent handles a domain event.
func (h *EventHandler) HandleEvent(evt event.Event) {
	var eventType EventType

	switch evt.EventType() {
	case "task.created":
		eventType = EventTypeTaskCreated
	case "task.updated":
		eventType = EventTypeTaskUpdated
	case "task.deleted":
		eventType = EventTypeTaskDeleted
	default:
		return
	}

	h.broker.Publish(eventType, evt)
}
