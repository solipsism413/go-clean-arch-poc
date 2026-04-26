package background

import (
	"context"
	"encoding/json"
	"expvar"
	"log/slog"
	"sync"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/worker"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

const alertThreshold = 1

type workflowState struct {
	Successes     int64     `json:"successes"`
	Failures      int64     `json:"failures"`
	FailureStreak int64     `json:"failureStreak"`
	TotalDuration int64     `json:"totalDurationMs"`
	LastError     string    `json:"lastError,omitempty"`
	LastFailureAt time.Time `json:"lastFailureAt,omitempty"`
	AlertActive   bool      `json:"alertActive"`
}

// Monitor tracks background workflow execution metrics and active alerts.
type Monitor struct {
	logger *slog.Logger
	mu     sync.Mutex
	state  map[string]*workflowState
	stats  *expvar.Map
	alerts *expvar.Map
}

// NewMonitor creates a monitor and exports its state through expvar.
func NewMonitor(logger *slog.Logger) *Monitor {
	monitor := &Monitor{
		logger: logger,
		state:  make(map[string]*workflowState),
		stats:  expvar.NewMap("background_workflow_metrics"),
		alerts: expvar.NewMap("background_workflow_alerts"),
	}
	monitor.stats.Set("exported_at_unix", expvar.Func(func() any {
		return time.Now().Unix()
	}))
	return monitor
}

// WrapHandler instruments a worker handler with duration, success, failure, and alert tracking.
func (m *Monitor) WrapHandler(name string, handler worker.EventHandler) worker.EventHandler {
	return func(ctx context.Context, evt event.Event) error {
		startedAt := time.Now()
		err := handler(ctx, evt)
		m.record(name, time.Since(startedAt), err)
		return err
	}
}

func (m *Monitor) record(name string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := m.state[name]
	if state == nil {
		state = &workflowState{}
		m.state[name] = state
	}

	state.TotalDuration += duration.Milliseconds()
	m.stats.Add(name+".runs", 1)
	m.stats.Add(name+".duration_ms", duration.Milliseconds())

	if err == nil {
		state.Successes++
		state.FailureStreak = 0
		state.AlertActive = false
		state.LastError = ""
		m.stats.Add(name+".successes", 1)
		m.alerts.Set(name, expvar.Func(func() any {
			return map[string]any{"active": false}
		}))
		return
	}

	state.Failures++
	state.FailureStreak++
	state.LastError = err.Error()
	state.LastFailureAt = time.Now().UTC()
	state.AlertActive = state.FailureStreak >= alertThreshold
	if state.AlertActive {
		m.logger.Warn("background workflow alert raised",
			"workflow", name,
			"failureStreak", state.FailureStreak,
			"error", err,
		)
	}
	m.stats.Add(name+".failures", 1)
	m.alerts.Set(name, expvar.Func(func() any {
		return map[string]any{
			"active":        state.AlertActive,
			"failureStreak": state.FailureStreak,
			"lastError":     state.LastError,
			"lastFailureAt": state.LastFailureAt,
		}
	}))
}

// Snapshot returns the current workflow metrics and alert state.
func (m *Monitor) Snapshot() map[string]workflowState {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]workflowState, len(m.state))
	for name, state := range m.state {
		result[name] = *state
	}
	return result
}

// MarshalJSON allows the monitor snapshot to be serialized in tests.
func (m *Monitor) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Snapshot())
}
