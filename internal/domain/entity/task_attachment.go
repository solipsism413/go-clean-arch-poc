package entity

import (
	"time"

	"github.com/google/uuid"
)

// TaskAttachment represents a file attached to a task.
type TaskAttachment struct {
	// ID is the unique identifier for the attachment.
	ID uuid.UUID

	// TaskID is the ID of the task this attachment belongs to.
	TaskID uuid.UUID

	// Filename is the original name of the uploaded file.
	Filename string

	// S3Key is the storage key in S3/MinIO.
	S3Key string

	// ContentType is the MIME type of the file.
	ContentType string

	// SizeBytes is the file size in bytes.
	SizeBytes int64

	// UploadedBy is the ID of the user who uploaded the file.
	UploadedBy uuid.UUID

	// CreatedAt is the timestamp when the attachment was created.
	CreatedAt time.Time
}

// NewTaskAttachment creates a new TaskAttachment.
func NewTaskAttachment(taskID uuid.UUID, filename, s3Key, contentType string, sizeBytes int64, uploadedBy uuid.UUID) *TaskAttachment {
	return &TaskAttachment{
		ID:          uuid.New(),
		TaskID:      taskID,
		Filename:    filename,
		S3Key:       s3Key,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		UploadedBy:  uploadedBy,
		CreatedAt:   time.Now().UTC(),
	}
}
