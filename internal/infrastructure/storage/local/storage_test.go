// Package local_test contains tests for the local file storage.
package local_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	"github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*local.LocalStorage, string) {
	t.Helper()
	tempDir := t.TempDir()
	cfg := local.Config{
		BasePath: tempDir,
		BaseURL:  "http://localhost:8080/files",
	}

	storage, err := local.NewLocalStorage(cfg)
	require.NoError(t, err)

	return storage, tempDir
}

func TestNewLocalStorage(t *testing.T) {
	t.Run("should create local storage with existing directory", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := local.Config{
			BasePath: tempDir,
			BaseURL:  "http://localhost:8080/files",
		}

		storage, err := local.NewLocalStorage(cfg)

		require.NoError(t, err)
		assert.NotNil(t, storage)
	})

	t.Run("should create base directory if it does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		newDir := filepath.Join(tempDir, "new-storage-dir")
		cfg := local.Config{
			BasePath: newDir,
			BaseURL:  "http://localhost:8080/files",
		}

		storage, err := local.NewLocalStorage(cfg)

		require.NoError(t, err)
		assert.NotNil(t, storage)

		// Verify directory was created
		_, err = os.Stat(newDir)
		assert.NoError(t, err)
	})
}

func TestLocalStorage_Upload(t *testing.T) {
	ctx := context.Background()

	t.Run("should upload file successfully", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "test-file.txt"
		content := []byte("Hello, World!")
		reader := bytes.NewReader(content)

		metadata, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
		})

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
		assert.Equal(t, "text/plain", metadata.ContentType)

		// Verify file exists on disk
		fileContent, err := os.ReadFile(filepath.Join(tempDir, key))
		require.NoError(t, err)
		assert.Equal(t, content, fileContent)
	})

	t.Run("should create subdirectories for nested keys", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "subdir/nested/file.txt"
		content := []byte("Nested content")
		reader := bytes.NewReader(content)

		metadata, err := storage.Upload(ctx, key, reader, output.UploadOptions{})

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)

		// Verify file exists on disk
		fileContent, err := os.ReadFile(filepath.Join(tempDir, key))
		require.NoError(t, err)
		assert.Equal(t, content, fileContent)
	})

	t.Run("should store metadata in upload options", func(t *testing.T) {
		storage, _ := setupTestStorage(t)
		key := "metadata-file.txt"
		content := []byte("File with metadata")
		reader := bytes.NewReader(content)
		customMetadata := map[string]string{
			"author": "test-user",
			"type":   "document",
		}

		metadata, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
			Metadata:    customMetadata,
		})

		require.NoError(t, err)
		assert.Equal(t, customMetadata, metadata.Metadata)
	})
}

func TestLocalStorage_Download(t *testing.T) {
	ctx := context.Background()

	t.Run("should download existing file", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "download-test.txt"
		content := []byte("Download me!")

		// Create file directly
		err := os.WriteFile(filepath.Join(tempDir, key), content, 0644)
		require.NoError(t, err)

		reader, metadata, err := storage.Download(ctx, key)
		require.NoError(t, err)
		defer reader.Close()

		downloaded, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, downloaded)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		storage, _ := setupTestStorage(t)

		reader, metadata, err := storage.Download(ctx, "non-existent.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
		assert.Nil(t, reader)
		assert.Nil(t, metadata)
	})
}

func TestLocalStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete existing file", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "delete-test.txt"
		filePath := filepath.Join(tempDir, key)

		// Create file
		err := os.WriteFile(filePath, []byte("Delete me!"), 0644)
		require.NoError(t, err)

		err = storage.Delete(ctx, key)
		require.NoError(t, err)

		// Verify file no longer exists
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("should not error when deleting non-existent file", func(t *testing.T) {
		storage, _ := setupTestStorage(t)

		err := storage.Delete(ctx, "non-existent.txt")

		assert.NoError(t, err)
	})
}

func TestLocalStorage_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("should return true for existing file", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "exists-test.txt"

		// Create file
		err := os.WriteFile(filepath.Join(tempDir, key), []byte("I exist!"), 0644)
		require.NoError(t, err)

		exists, err := storage.Exists(ctx, key)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false for non-existent file", func(t *testing.T) {
		storage, _ := setupTestStorage(t)

		exists, err := storage.Exists(ctx, "non-existent.txt")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLocalStorage_GetMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("should get metadata for existing file", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		key := "metadata-test.txt"
		content := []byte("Metadata test content")

		// Create file
		err := os.WriteFile(filepath.Join(tempDir, key), content, 0644)
		require.NoError(t, err)

		metadata, err := storage.GetMetadata(ctx, key)

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
		assert.WithinDuration(t, time.Now(), metadata.LastModified, 5*time.Second)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		storage, _ := setupTestStorage(t)

		metadata, err := storage.GetMetadata(ctx, "non-existent.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
		assert.Nil(t, metadata)
	})
}

func TestLocalStorage_GeneratePresignedURL(t *testing.T) {
	ctx := context.Background()

	t.Run("should generate direct URL", func(t *testing.T) {
		storage, _ := setupTestStorage(t)
		key := "presigned-test.txt"

		url, err := storage.GeneratePresignedURL(ctx, key, time.Hour)

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/files/presigned-test.txt", url)
	})
}

func TestLocalStorage_GenerateUploadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("should generate upload URL same as presigned URL", func(t *testing.T) {
		storage, _ := setupTestStorage(t)
		key := "upload-url-test.txt"

		url, err := storage.GenerateUploadURL(ctx, key, "text/plain", time.Hour)

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/files/upload-url-test.txt", url)
	})
}

func TestLocalStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should list files with prefix", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)

		// Create test directory structure
		subDir := filepath.Join(tempDir, "prefix")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		_ = os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content1"), 0644)
		_ = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644)

		files, err := storage.List(ctx, "prefix/", 100)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 2)
	})

	t.Run("should respect maxKeys limit", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)

		// Create test files
		for i := 0; i < 5; i++ {
			fileName := filepath.Join(tempDir, "list", "file"+string(rune('0'+i))+".txt")
			err := os.MkdirAll(filepath.Dir(fileName), 0755)
			require.NoError(t, err)
			err = os.WriteFile(fileName, []byte("content"), 0644)
			require.NoError(t, err)
		}

		files, err := storage.List(ctx, "list/", 2)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(files), 2)
	})
}

func TestLocalStorage_Copy(t *testing.T) {
	ctx := context.Background()

	t.Run("should copy file to new location", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		sourceKey := "source-file.txt"
		destKey := "dest-file.txt"
		content := []byte("Copy me!")

		// Create source file
		err := os.WriteFile(filepath.Join(tempDir, sourceKey), content, 0644)
		require.NoError(t, err)

		metadata, err := storage.Copy(ctx, sourceKey, destKey)

		require.NoError(t, err)
		assert.Equal(t, destKey, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)

		// Verify destination file exists with same content
		destContent, err := os.ReadFile(filepath.Join(tempDir, destKey))
		require.NoError(t, err)
		assert.Equal(t, content, destContent)

		// Verify source file still exists
		sourceContent, err := os.ReadFile(filepath.Join(tempDir, sourceKey))
		require.NoError(t, err)
		assert.Equal(t, content, sourceContent)
	})

	t.Run("should create destination directory if needed", func(t *testing.T) {
		storage, tempDir := setupTestStorage(t)
		sourceKey := "source.txt"
		destKey := "newdir/nested/dest.txt"
		content := []byte("Copy to nested!")

		// Create source file
		err := os.WriteFile(filepath.Join(tempDir, sourceKey), content, 0644)
		require.NoError(t, err)

		metadata, err := storage.Copy(ctx, sourceKey, destKey)

		require.NoError(t, err)
		assert.Equal(t, destKey, metadata.Key)

		// Verify destination file exists
		destContent, err := os.ReadFile(filepath.Join(tempDir, destKey))
		require.NoError(t, err)
		assert.Equal(t, content, destContent)
	})

	t.Run("should return error when source does not exist", func(t *testing.T) {
		storage, _ := setupTestStorage(t)

		metadata, err := storage.Copy(ctx, "non-existent.txt", "dest.txt")

		assert.Error(t, err)
		assert.Nil(t, metadata)
	})
}
