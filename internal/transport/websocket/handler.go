// Package websocket provides WebSocket handler for real-time task updates.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// MessageType defines WebSocket message types.
type MessageType string

const (
	MessageTypeSubscribe         MessageType = "subscribe"
	MessageTypeUnsubscribe       MessageType = "unsubscribe"
	MessageTypeTaskCreated       MessageType = "task.created"
	MessageTypeTaskUpdated       MessageType = "task.updated"
	MessageTypeTaskDeleted       MessageType = "task.deleted"
	MessageTypeTaskAssigned      MessageType = "task.assigned"
	MessageTypeTaskUnassigned    MessageType = "task.unassigned"
	MessageTypeTaskCompleted     MessageType = "task.completed"
	MessageTypeTaskArchived      MessageType = "task.archived"
	MessageTypeTaskStatusChanged MessageType = "task.status_changed"
	MessageTypeTaskLabelAdded    MessageType = "task.label_added"
	MessageTypeTaskLabelRemoved  MessageType = "task.label_removed"
	MessageTypePing              MessageType = "ping"
	MessageTypePong              MessageType = "pong"
	MessageTypeError             MessageType = "error"
)

// Message represents a WebSocket message.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client represents a WebSocket client connection.
type Client struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Conn          *websocket.Conn
	Hub           *Hub
	Send          chan []byte
	Subscriptions map[string]bool
	mu            sync.RWMutex
}

// Hub maintains active clients and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the Hub's main loop.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client connected", "clientId", client.ID)
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			h.logger.Debug("client disconnected", "clientId", client.ID)
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastEvent broadcasts an event to all connected clients.
func (h *Hub) BroadcastEvent(eventType MessageType, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Type:    eventType,
		Payload: data,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- msgBytes
	return nil
}

// Handler handles WebSocket connections.
type Handler struct {
	hub         *Hub
	taskService input.TaskService
	authService input.AuthService
	upgrader    websocket.Upgrader
	logger      *slog.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, taskService input.TaskService, authService input.AuthService, logger *slog.Logger) *Handler {
	return &Handler{
		hub:         hub,
		taskService: taskService,
		authService: authService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", "error", err)
		return
	}

	// Create client
	client := &Client{
		ID:            uuid.New(),
		Conn:          conn,
		Hub:           h.hub,
		Send:          make(chan []byte, 256),
		Subscriptions: make(map[string]bool),
	}

	// Register client
	h.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump(h)
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump(h *Handler) {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("websocket error", "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Error("failed to unmarshal message", "error", err)
			continue
		}

		c.handleMessage(h, &msg)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch pending messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages.
func (c *Client) handleMessage(h *Handler, msg *Message) {
	switch msg.Type {
	case MessageTypeSubscribe:
		var payload struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		c.mu.Lock()
		c.Subscriptions[payload.Topic] = true
		c.mu.Unlock()

	case MessageTypeUnsubscribe:
		var payload struct {
			Topic string `json:"topic"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		c.mu.Lock()
		delete(c.Subscriptions, payload.Topic)
		c.mu.Unlock()

	case MessageTypePing:
		response := Message{Type: MessageTypePong}
		data, _ := json.Marshal(response)
		c.Send <- data
	}
}

// EventHandler handles domain events and broadcasts to WebSocket clients.
type EventHandler struct {
	hub    *Hub
	logger *slog.Logger
}

// NewEventHandler creates a new event handler.
func NewEventHandler(hub *Hub, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		hub:    hub,
		logger: logger,
	}
}

// HandleEvent handles a domain event.
func (h *EventHandler) HandleEvent(evt event.Event) {
	var msgType MessageType

	switch evt.EventType() {
	case "task.created":
		msgType = MessageTypeTaskCreated
	case "task.updated":
		msgType = MessageTypeTaskUpdated
	case "task.deleted":
		msgType = MessageTypeTaskDeleted
	case "task.assigned":
		msgType = MessageTypeTaskAssigned
	case "task.unassigned":
		msgType = MessageTypeTaskUnassigned
	case "task.completed":
		msgType = MessageTypeTaskCompleted
	case "task.archived":
		msgType = MessageTypeTaskArchived
	case "task.status_changed":
		msgType = MessageTypeTaskStatusChanged
	case "task.label_added":
		msgType = MessageTypeTaskLabelAdded
	case "task.label_removed":
		msgType = MessageTypeTaskLabelRemoved
	default:
		return
	}

	if err := h.hub.BroadcastEvent(msgType, evt); err != nil {
		h.logger.Error("failed to broadcast event", "eventType", evt.EventType(), "error", err)
	}
}
