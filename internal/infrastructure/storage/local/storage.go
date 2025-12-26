// Package local provides local filesystem storage.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
)

// Ensure LocalStorage implements the output.FileStorage interface.
var _ output.FileStorage = (*LocalStorage)(nil)

// Config holds local storage configuration.
type Config struct {
	BasePath string
	BaseURL  string
}

// LocalStorage implements file storage using the local filesystem.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local file storage.
func NewLocalStorage(cfg Config) (*LocalStorage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(cfg.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		basePath: cfg.BasePath,
		baseURL:  cfg.BaseURL,
	}, nil
}

// Upload uploads a file and returns its metadata.
func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, options output.UploadOptions) (*output.FileMetadata, error) {
	fullPath := filepath.Join(s.basePath, key)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	written, err := io.Copy(file, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &output.FileMetadata{
		Key:          key,
		Size:         written,
		ContentType:  options.ContentType,
		LastModified: time.Now(),
		Metadata:     options.Metadata,
	}, nil
}

// Download downloads a file by key.
func (s *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, *output.FileMetadata, error) {
	fullPath := filepath.Join(s.basePath, key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	metadata := &output.FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}

	return file, metadata, nil
}

// Delete removes a file by key.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a file exists.
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := filepath.Join(s.basePath, key)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file: %w", err)
	}

	return true, nil
}

// GetMetadata retrieves file metadata without downloading.
func (s *LocalStorage) GetMetadata(ctx context.Context, key string) (*output.FileMetadata, error) {
	fullPath := filepath.Join(s.basePath, key)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &output.FileMetadata{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// GeneratePresignedURL generates a URL for the file.
// For local storage, this returns a direct URL.
func (s *LocalStorage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return fmt.Sprintf("%s/%s", s.baseURL, key), nil
}

// GenerateUploadURL generates a URL for direct upload.
// For local storage, this returns the same as GeneratePresignedURL.
func (s *LocalStorage) GenerateUploadURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	return s.GeneratePresignedURL(ctx, key, expiration)
}

// List lists files with a given prefix.
func (s *LocalStorage) List(ctx context.Context, prefix string, maxKeys int) ([]*output.FileMetadata, error) {
	searchPath := filepath.Join(s.basePath, prefix)
	dir := filepath.Dir(searchPath)

	var files []*output.FileMetadata
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if count >= maxKeys {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		// Get relative key
		key, err := filepath.Rel(s.basePath, path)
		if err != nil {
			return nil
		}

		files = append(files, &output.FileMetadata{
			Key:          key,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})
		count++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// Copy copies a file from one key to another.
func (s *LocalStorage) Copy(ctx context.Context, sourceKey, destKey string) (*output.FileMetadata, error) {
	srcPath := filepath.Join(s.basePath, sourceKey)
	dstPath := filepath.Join(s.basePath, destKey)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source
	src, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	// Create destination
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination: %w", err)
	}
	defer dst.Close()

	// Copy content
	written, err := io.Copy(dst, src)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	return &output.FileMetadata{
		Key:          destKey,
		Size:         written,
		LastModified: time.Now(),
	}, nil
}
