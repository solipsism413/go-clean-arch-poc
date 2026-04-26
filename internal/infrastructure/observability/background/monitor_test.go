package background

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/stretchr/testify/require"
)

func TestMonitorWrapHandlerTracksSuccessAndFailure(t *testing.T) {
	monitor := NewMonitor(slog.New(slog.NewTextHandler(io.Discard, nil)))
	evt := event.NewTaskDeleted(uuid.New(), uuid.New())

	success := monitor.WrapHandler("task.created", func(ctx context.Context, evt event.Event) error {
		return nil
	})
	failure := monitor.WrapHandler("task.created", func(ctx context.Context, evt event.Event) error {
		return context.DeadlineExceeded
	})

	require.NoError(t, success(context.Background(), evt))
	require.ErrorIs(t, failure(context.Background(), evt), context.DeadlineExceeded)

	snapshot := monitor.Snapshot()
	require.Equal(t, int64(1), snapshot["task.created"].Successes)
	require.Equal(t, int64(1), snapshot["task.created"].Failures)
	require.True(t, snapshot["task.created"].AlertActive)
	require.WithinDuration(t, time.Now(), snapshot["task.created"].LastFailureAt, time.Second)
}
