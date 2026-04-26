package worker

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// NewTaskAttachmentCleanupHandler returns a background handler with structured
// logs for attachment cleanup retry attempts and failures.
func NewTaskAttachmentCleanupHandler(fileStorage output.FileStorage, logger *slog.Logger) EventHandler {
	return func(ctx context.Context, evt event.Event) error {
		cleanupEvent, err := taskAttachmentCleanupEventFromEvent(evt)
		if err != nil {
			return err
		}

		if isNilDependency(fileStorage) {
			err := fmt.Errorf("file storage unavailable for cleanup retry")
			logger.Error("background handler: attachment cleanup retry unavailable",
				"taskID", cleanupEvent.AggregateID(),
				"attachmentId", cleanupEvent.AttachmentID,
				"objectKey", cleanupEvent.ObjectKey,
				"error", err,
			)
			return err
		}

		logger.Info("background handler: retrying attachment cleanup",
			"taskID", cleanupEvent.AggregateID(),
			"attachmentId", cleanupEvent.AttachmentID,
			"objectKey", cleanupEvent.ObjectKey,
		)

		if err := fileStorage.Delete(ctx, cleanupEvent.ObjectKey); err != nil {
			logger.Error("background handler: attachment cleanup retry failed",
				"taskID", cleanupEvent.AggregateID(),
				"attachmentId", cleanupEvent.AttachmentID,
				"objectKey", cleanupEvent.ObjectKey,
				"error", err,
			)
			return err
		}

		logger.Info("background handler: attachment cleanup completed",
			"taskID", cleanupEvent.AggregateID(),
			"attachmentId", cleanupEvent.AttachmentID,
			"objectKey", cleanupEvent.ObjectKey,
		)
		return nil
	}
}

func isNilDependency(dep any) bool {
	if dep == nil {
		return true
	}
	v := reflect.ValueOf(dep)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func taskAttachmentCleanupEventFromEvent(evt event.Event) (event.TaskAttachmentCleanupRequested, error) {
	switch cleanupEvent := evt.(type) {
	case event.TaskAttachmentCleanupRequested:
		return cleanupEvent, nil
	case *event.TaskAttachmentCleanupRequested:
		if cleanupEvent == nil {
			return event.TaskAttachmentCleanupRequested{}, fmt.Errorf("unexpected nil event %T", evt)
		}
		return *cleanupEvent, nil
	default:
		return event.TaskAttachmentCleanupRequested{}, fmt.Errorf("unexpected event type %T", evt)
	}
}
