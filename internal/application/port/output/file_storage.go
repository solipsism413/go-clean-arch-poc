package output

import (
	"context"
	"io"
	"time"
)

// FileMetadata contains metadata about a stored file.
type FileMetadata struct {
	Key          string
	Filename     string
	ContentType  string
	Size         int64
	LastModified time.Time
	ETag         string
	Metadata     map[string]string
}

// UploadOptions contains options for file upload.
type UploadOptions struct {
	ContentType string
	Metadata    map[string]string
	ACL         string // e.g., "private", "public-read"
}

// FileStorage defines the output port for file storage operations (S3).
type FileStorage interface {
	// Upload uploads a file and returns its key.
	Upload(ctx context.Context, key string, reader io.Reader, options UploadOptions) (*FileMetadata, error)

	// Download downloads a file by key.
	Download(ctx context.Context, key string) (io.ReadCloser, *FileMetadata, error)

	// Delete removes a file by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a file exists.
	Exists(ctx context.Context, key string) (bool, error)

	// GetMetadata retrieves file metadata without downloading.
	GetMetadata(ctx context.Context, key string) (*FileMetadata, error)

	// GeneratePresignedURL generates a pre-signed URL for temporary access.
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)

	// GenerateUploadURL generates a pre-signed URL for direct upload.
	GenerateUploadURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error)

	// List lists files with a given prefix.
	List(ctx context.Context, prefix string, maxKeys int) ([]*FileMetadata, error)

	// Copy copies a file from one key to another.
	Copy(ctx context.Context, sourceKey, destKey string) (*FileMetadata, error)
}

// FileKeyBuilder helps build consistent file storage keys.
type FileKeyBuilder struct {
	prefix string
}

// NewFileKeyBuilder creates a new file key builder.
func NewFileKeyBuilder(prefix string) *FileKeyBuilder {
	return &FileKeyBuilder{prefix: prefix}
}

// TaskAttachment returns a key for a task attachment.
func (b *FileKeyBuilder) TaskAttachment(taskID, filename string) string {
	return b.prefix + "/tasks/" + taskID + "/attachments/" + filename
}

// UserAvatar returns a key for a user avatar.
func (b *FileKeyBuilder) UserAvatar(userID, filename string) string {
	return b.prefix + "/users/" + userID + "/avatar/" + filename
}
