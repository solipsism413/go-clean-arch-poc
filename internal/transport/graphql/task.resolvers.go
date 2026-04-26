package graphql

import (
	"context"

	"github.com/google/uuid"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
)

// Attachments is the resolver for the attachments field.
func (r *taskResolver) Attachments(ctx context.Context, obj *Task) ([]*TaskAttachment, error) {
	if obj == nil {
		return []*TaskAttachment{}, nil
	}
	attachments, err := r.taskService.ListTaskAttachments(ctx, obj.ID)
	if err != nil {
		if domainerror.IsNotFoundError(err) {
			return []*TaskAttachment{}, nil
		}
		return nil, err
	}
	return taskAttachmentListToGraphQL(attachments), nil
}

type taskResolver struct{ *Resolver }

func (r *subscriptionResolver) streamTasks(ctx context.Context, match func(event.Event) bool) (<-chan *Task, error) {
	subscription := r.subscriptions.Subscribe(ctx)
	out := make(chan *Task, 1)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-subscription:
				if !ok {
					return
				}
				if !match(evt) {
					continue
				}
				task, err := r.taskService.GetTask(ctx, evt.AggregateID())
				if err != nil {
					if domainerror.IsNotFoundError(err) {
						continue
					}
					continue
				}
				mapped, err := r.enrichAndMapTask(ctx, task)
				if err != nil {
					continue
				}
				select {
				case out <- mapped:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func matchesAssignee(evt event.Event, assigneeID uuid.UUID) bool {
	if assigned, ok := evt.(*event.TaskAssigned); ok {
		return assigned.AssigneeID == assigneeID
	}
	if assigned, ok := evt.(event.TaskAssigned); ok {
		return assigned.AssigneeID == assigneeID
	}
	return false
}
