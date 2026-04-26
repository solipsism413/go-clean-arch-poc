// Package socketio provides Socket.IO handler for real-time collaboration.
package socketio

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// Room represents a Socket.IO room (e.g., task:123, user:456)
type Room string

const (
	// RoomTasks is the global tasks room for all task updates
	RoomTasks Room = "tasks"
	// RoomPrefix for task-specific rooms
	RoomTaskPrefix = "task:"
	// RoomPrefix for user-specific rooms
	RoomUserPrefix = "user:"
)

// EventName defines Socket.IO event names.
type EventName string

const (
	EventTaskCreated       EventName = "task:created"
	EventTaskUpdated       EventName = "task:updated"
	EventTaskDeleted       EventName = "task:deleted"
	EventTaskAssigned      EventName = "task:assigned"
	EventTaskUnassigned    EventName = "task:unassigned"
	EventTaskCompleted     EventName = "task:completed"
	EventTaskArchived      EventName = "task:archived"
	EventTaskStatusChanged EventName = "task:status_changed"
	EventTaskLabelAdded    EventName = "task:label_added"
	EventTaskLabelRemoved  EventName = "task:label_removed"
	EventJoinRoom          EventName = "join"
	EventLeaveRoom         EventName = "leave"
	EventError             EventName = "error"
)

// Handler manages Socket.IO connections and events.
type Handler struct {
	server      *socketio.Server
	taskService input.TaskService
	logger      *slog.Logger
	clients     map[string]socketio.Conn
	mu          sync.RWMutex
}

// NewHandler creates a new Socket.IO handler.
func NewHandler(taskService input.TaskService, logger *slog.Logger) (*Handler, error) {
	server := socketio.NewServer(nil)

	h := &Handler{
		server:      server,
		taskService: taskService,
		logger:      logger,
		clients:     make(map[string]socketio.Conn),
	}

	// Set up event handlers
	h.setupEventHandlers()

	return h, nil
}

// setupEventHandlers configures Socket.IO event handlers.
func (h *Handler) setupEventHandlers() {
	h.server.OnConnect("/", func(s socketio.Conn) error {
		clientID := uuid.New().String()
		s.SetContext(clientID)

		h.mu.Lock()
		h.clients[clientID] = s
		h.mu.Unlock()

		// Join global tasks room by default
		s.Join(string(RoomTasks))

		h.logger.Debug("socket.io client connected", "clientId", clientID)
		return nil
	})

	h.server.OnEvent("/", string(EventJoinRoom), func(s socketio.Conn, room string) {
		s.Join(room)
		h.logger.Debug("client joined room", "clientId", s.Context(), "room", room)
	})

	h.server.OnEvent("/", string(EventLeaveRoom), func(s socketio.Conn, room string) {
		s.Leave(room)
		h.logger.Debug("client left room", "clientId", s.Context(), "room", room)
	})

	h.server.OnError("/", func(s socketio.Conn, e error) {
		h.logger.Error("socket.io error", "error", e)
	})

	h.server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		clientID, ok := s.Context().(string)
		if ok {
			h.mu.Lock()
			delete(h.clients, clientID)
			h.mu.Unlock()
		}
		h.logger.Debug("socket.io client disconnected", "clientId", clientID, "reason", reason)
	})
}

// Start starts the Socket.IO server.
func (h *Handler) Start(ctx context.Context) error {
	go func() {
		if err := h.server.Serve(); err != nil {
			h.logger.Error("socket.io server error", "error", err)
		}
	}()

	// Stop server when context is cancelled
	go func() {
		<-ctx.Done()
		h.server.Close()
	}()

	return nil
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// BroadcastToRoom broadcasts an event to a specific room.
func (h *Handler) BroadcastToRoom(room Room, eventName EventName, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	h.server.BroadcastToRoom("/", string(room), string(eventName), string(payload))
	return nil
}

// BroadcastToAll broadcasts an event to all connected clients.
func (h *Handler) BroadcastToAll(eventName EventName, data any) error {
	return h.BroadcastToRoom(RoomTasks, eventName, data)
}

// EventHandler handles domain events and broadcasts to Socket.IO clients.
type EventHandler struct {
	handler *Handler
	logger  *slog.Logger
}

// NewEventHandler creates a new event handler.
func NewEventHandler(handler *Handler, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		handler: handler,
		logger:  logger,
	}
}

// HandleEvent handles a domain event.
func (h *EventHandler) HandleEvent(evt event.Event) {
	var eventName EventName

	switch evt.EventType() {
	case "task.created":
		eventName = EventTaskCreated
	case "task.updated":
		eventName = EventTaskUpdated
	case "task.deleted":
		eventName = EventTaskDeleted
	case "task.assigned":
		eventName = EventTaskAssigned
	case "task.unassigned":
		eventName = EventTaskUnassigned
	case "task.completed":
		eventName = EventTaskCompleted
	case "task.archived":
		eventName = EventTaskArchived
	case "task.status_changed":
		eventName = EventTaskStatusChanged
	case "task.label_added":
		eventName = EventTaskLabelAdded
	case "task.label_removed":
		eventName = EventTaskLabelRemoved
	default:
		return
	}

	if err := h.handler.BroadcastToAll(eventName, evt); err != nil {
		h.logger.Error("failed to broadcast event", "eventType", evt.EventType(), "error", err)
	}
}
