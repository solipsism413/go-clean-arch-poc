package task_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/application/usecase/task"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	"github.com/handiism/go-clean-arch-poc/internal/domain/event"
	"github.com/handiism/go-clean-arch-poc/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTaskUseCase_CreateTask_Validation(t *testing.T) {
	// Simple test to verify that the mock validator is called
	mockValidator := new(mocks.MockValidator)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// We don't need real repos for this validation check
	uc := task.NewTaskUseCase(nil, nil, nil, nil, nil, nil, nil, nil, mockValidator, logger)

	input := dto.CreateTaskInput{
		Title: "Test Task",
	}

	mockValidator.On("Validate", input).Return(assert.AnError)

	ctx := context.Background()
	_, err := uc.CreateTask(ctx, input)

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockValidator.AssertExpectations(t)
}

func TestTaskUseCase_UploadTaskAttachment_RequiresStorage(t *testing.T) {
	taskRepo := mocks.NewMockTaskRepository(t)
	attachmentRepo := mocks.NewMockTaskAttachmentRepository(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	taskID := uuid.New()
	taskRepo.EXPECT().ExistsByID(mock.Anything, taskID).Return(true, nil).Maybe()
	attachmentRepo.EXPECT().SaveAttachment(mock.Anything, mock.Anything).Return(nil).Maybe()

	uc := task.NewTaskUseCase(taskRepo, attachmentRepo, nil, nil, nil, nil, nil, nil, nil, logger)

	_, err := uc.UploadTaskAttachment(context.Background(), taskID, "report.pdf", "application/pdf", strings.NewReader("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file storage is unavailable")
}

func TestTaskUseCase_UploadTaskAttachment_UsesUniqueStorageKeys(t *testing.T) {
	taskRepo := mocks.NewMockTaskRepository(t)
	attachmentRepo := mocks.NewMockTaskAttachmentRepository(t)
	fileStorage := mocks.NewMockFileStorage(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	taskID := uuid.New()
	taskRepo.EXPECT().ExistsByID(mock.Anything, taskID).Return(true, nil).Twice()

	var savedKeys []string
	attachmentRepo.EXPECT().SaveAttachment(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, attachment *entity.TaskAttachment) error {
		savedKeys = append(savedKeys, attachment.S3Key)
		return nil
	}).Twice()

	fileStorage.EXPECT().Upload(mock.Anything, mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, key string, reader io.Reader, options output.UploadOptions) (*output.FileMetadata, error) {
		_, _ = io.ReadAll(reader)
		return &output.FileMetadata{Key: key, Size: 5}, nil
	}).Twice()

	uc := task.NewTaskUseCase(taskRepo, attachmentRepo, nil, nil, fileStorage, nil, nil, nil, nil, logger)

	_, err := uc.UploadTaskAttachment(context.Background(), taskID, "report.pdf", "application/pdf", strings.NewReader("one"))
	require.NoError(t, err)
	_, err = uc.UploadTaskAttachment(context.Background(), taskID, "report.pdf", "application/pdf", strings.NewReader("two"))
	require.NoError(t, err)

	require.Len(t, savedKeys, 2)
	assert.NotEqual(t, savedKeys[0], savedKeys[1])
	assert.Contains(t, savedKeys[0], "report.pdf")
	assert.Contains(t, savedKeys[1], "report.pdf")
}

func TestTaskUseCase_DeleteTask_PublishesCleanupRequestWhenBlobDeletionFails(t *testing.T) {
	taskRepo := mocks.NewMockTaskRepository(t)
	attachmentRepo := mocks.NewMockTaskAttachmentRepository(t)
	fileStorage := mocks.NewMockFileStorage(t)
	eventPublisher := mocks.NewMockEventPublisher(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	taskID := uuid.New()
	taskRepo.EXPECT().ExistsByID(mock.Anything, taskID).Return(true, nil).Once()
	attachmentRepo.EXPECT().FindAttachmentsByTaskID(mock.Anything, taskID).Return([]*entity.TaskAttachment{{ID: uuid.New(), TaskID: taskID, S3Key: "attachments/tasks/x/file"}}, nil).Once()
	taskRepo.EXPECT().Delete(mock.Anything, taskID).Return(nil).Once()
	fileStorage.EXPECT().Delete(mock.Anything, "attachments/tasks/x/file").Return(errors.New("s3 down")).Once()
	eventPublisher.EXPECT().Publish(mock.Anything, output.TopicTaskEvents, mock.MatchedBy(func(evt event.Event) bool {
		return evt.EventType() == "task.attachment_cleanup_requested"
	})).Return(nil).Once()
	eventPublisher.EXPECT().Publish(mock.Anything, output.TopicTaskEvents, mock.Anything).Return(nil).Once()

	uc := task.NewTaskUseCase(taskRepo, attachmentRepo, nil, nil, fileStorage, nil, eventPublisher, nil, nil, logger)

	err := uc.DeleteTask(context.Background(), taskID)
	require.NoError(t, err)
}

func TestTaskUseCase_DeleteTaskAttachment_RequiresCleanupChannelWhenBlobDeletionFails(t *testing.T) {
	attachmentRepo := mocks.NewMockTaskAttachmentRepository(t)
	fileStorage := mocks.NewMockFileStorage(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	taskID := uuid.New()
	attachmentID := uuid.New()
	attachmentRepo.EXPECT().FindAttachmentByID(mock.Anything, attachmentID).Return(&entity.TaskAttachment{ID: attachmentID, TaskID: taskID, S3Key: "attachments/tasks/x/file"}, nil).Once()
	fileStorage.EXPECT().Delete(mock.Anything, "attachments/tasks/x/file").Return(errors.New("s3 down")).Once()

	uc := task.NewTaskUseCase(nil, attachmentRepo, nil, nil, fileStorage, nil, nil, nil, nil, logger)

	err := uc.DeleteTaskAttachment(context.Background(), taskID, attachmentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attachment cleanup is unavailable")
}
