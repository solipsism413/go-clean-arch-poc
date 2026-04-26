package worker_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/worker"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/messaging/memory"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventConsumer_RegisterAndProcessEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	var called atomic.Bool
	consumer.RegisterHandler("user.created", func(ctx context.Context, evt event.Event) error {
		called.Store(true)
		return nil
	})

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Publish an event
	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Wait for handler to be called
	time.Sleep(100 * time.Millisecond)
	assert.True(t, called.Load(), "handler should have been called")

	consumer.Stop()
}

func TestEventConsumer_NoHandlerForEventType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	// Do not register any handler

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Publish an event with no handler
	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Should not panic; just wait briefly
	time.Sleep(50 * time.Millisecond)

	consumer.Stop()
}

func TestEventConsumer_HandlerError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	consumer.RegisterHandler("user.created", func(ctx context.Context, evt event.Event) error {
		return errors.New("handler error")
	})

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	testEvt := event.NewUserCreated(uuid.New(), "test@example.com", "Test User")
	err = bus.Publish(ctx, "test-topic", testEvt)
	assert.NoError(t, err)

	// Wait for handler to process (error should be logged, not returned)
	time.Sleep(50 * time.Millisecond)

	consumer.Stop()
}

func TestEventConsumer_Stop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := context.Background()

	bus := memory.NewEventBus()
	defer bus.Close()

	consumer := worker.NewEventConsumer(logger)

	err := consumer.Start(ctx, bus, []string{"test-topic"})
	assert.NoError(t, err)

	// Stop should complete without panic or deadlock
	consumer.Stop()
}

func TestNewTaskAttachmentCleanupHandler_LogsAndDeletesObject(t *testing.T) {
	storage := mocks.NewMockFileStorage(t)
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	taskID := uuid.New()
	attachmentID := uuid.New()
	evt := event.NewTaskAttachmentCleanupRequested(taskID, attachmentID, "attachments/tasks/x/file")

	storage.EXPECT().Delete(context.Background(), evt.ObjectKey).Return(nil).Once()

	handler := worker.NewTaskAttachmentCleanupHandler(storage, logger)
	err := handler(context.Background(), evt)
	require.NoError(t, err)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "retrying attachment cleanup")
	assert.Contains(t, logOutput, "attachment cleanup completed")
	assert.Contains(t, logOutput, evt.ObjectKey)
}

func TestNewTaskAttachmentCleanupHandler_ReturnsContextualErrorWhenStorageUnavailable(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	evt := event.NewTaskAttachmentCleanupRequested(uuid.New(), uuid.New(), "attachments/tasks/x/file")
	handler := worker.NewTaskAttachmentCleanupHandler(nil, logger)
	err := handler(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file storage unavailable for cleanup retry")
	assert.Contains(t, logBuf.String(), "attachment cleanup retry unavailable")
}

func TestNewTaskAttachmentCleanupHandler_AlsoAcceptsPointerEvents(t *testing.T) {
	storage := mocks.NewMockFileStorage(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	evt := event.NewTaskAttachmentCleanupRequested(uuid.New(), uuid.New(), "attachments/tasks/x/file")
	storage.EXPECT().Delete(context.Background(), evt.ObjectKey).Return(nil).Once()

	handler := worker.NewTaskAttachmentCleanupHandler(storage, logger)
	err := handler(context.Background(), &evt)
	require.NoError(t, err)
}

func TestNewTaskAttachmentCleanupHandler_TreatsTypedNilStorageAsUnavailable(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	evt := event.NewTaskAttachmentCleanupRequested(uuid.New(), uuid.New(), "attachments/tasks/x/file")
	var storage *typedNilFileStorage

	handler := worker.NewTaskAttachmentCleanupHandler(storage, logger)
	err := handler(context.Background(), evt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file storage unavailable for cleanup retry")
	assert.Contains(t, logBuf.String(), "attachment cleanup retry unavailable")
}

type typedNilFileStorage struct{}

func (fs *typedNilFileStorage) Upload(context.Context, string, io.Reader, output.UploadOptions) (*output.FileMetadata, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) Download(context.Context, string) (io.ReadCloser, *output.FileMetadata, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) Delete(context.Context, string) error {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) Exists(context.Context, string) (bool, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) GetMetadata(context.Context, string) (*output.FileMetadata, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) GeneratePresignedURL(context.Context, string, time.Duration) (string, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) GenerateUploadURL(context.Context, string, string, time.Duration) (string, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) List(context.Context, string, int) ([]*output.FileMetadata, error) {
	panic("unexpected call")
}

func (fs *typedNilFileStorage) Copy(context.Context, string, string) (*output.FileMetadata, error) {
	panic("unexpected call")
}
