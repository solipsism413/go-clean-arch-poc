// Package s3_test contains integration tests for the S3 file storage.
package s3_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	s3storage "github.com/handiism/go-clean-arch-poc/internal/infrastructure/storage/s3"
	appconfig "github.com/handiism/go-clean-arch-poc/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testAccessKey  = "minioadmin"
	testSecretKey  = "minioadmin"
	testBucketName = "test-bucket"
	testRegion     = "us-east-1"
)

// MinIOContainer holds the MinIO container for testing.
type MinIOContainer struct {
	Container testcontainers.Container
	Endpoint  string
}

// SetupMinIO creates a MinIO testcontainer.
func SetupMinIO(t *testing.T) *MinIOContainer {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     testAccessKey,
			"MINIO_ROOT_PASSWORD": testSecretKey,
		},
		Cmd: []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").
			WithPort("9000/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start minio container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	endpoint := "http://" + host + ":" + port.Port()

	// Create the test bucket using AWS SDK
	createTestBucket(t, ctx, endpoint)

	return &MinIOContainer{
		Container: container,
		Endpoint:  endpoint,
	}
}

// Cleanup stops the MinIO container.
func (m *MinIOContainer) Cleanup(t *testing.T) {
	t.Helper()
	if m.Container != nil {
		if err := m.Container.Terminate(context.Background()); err != nil {
			t.Logf("failed to terminate minio container: %v", err)
		}
	}
}

// createTestBucket creates a bucket in MinIO for testing.
func createTestBucket(t *testing.T, ctx context.Context, endpoint string) {
	t.Helper()

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			testAccessKey,
			testSecretKey,
			"",
		)),
	)
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(testBucketName),
	})
	if err != nil {
		t.Fatalf("failed to create test bucket: %v", err)
	}
}

func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewFileStorage(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()

	t.Run("should create file storage successfully", func(t *testing.T) {
		cfg := appconfig.S3Config{
			Endpoint:        minioContainer.Endpoint,
			Region:          testRegion,
			Bucket:          testBucketName,
			AccessKeyID:     testAccessKey,
			SecretAccessKey: testSecretKey,
			UsePathStyle:    true,
		}

		storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())

		require.NoError(t, err)
		assert.NotNil(t, storage)
	})
}

func TestFileStorage_Upload(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should upload file successfully", func(t *testing.T) {
		key := "test-upload.txt"
		content := []byte("Hello, S3!")
		reader := bytes.NewReader(content)

		metadata, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
		})

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
	})

	t.Run("should upload file with custom metadata", func(t *testing.T) {
		key := "test-upload-metadata.txt"
		content := []byte("File with metadata")
		reader := bytes.NewReader(content)
		customMetadata := map[string]string{
			"author": "test-user",
		}

		metadata, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
			Metadata:    customMetadata,
		})

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)
	})
}

func TestFileStorage_Download(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should download uploaded file", func(t *testing.T) {
		key := "test-download.txt"
		content := []byte("Download me!")
		reader := bytes.NewReader(content)

		_, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		downloadReader, metadata, err := storage.Download(ctx, key)
		require.NoError(t, err)
		defer downloadReader.Close()

		downloaded, err := io.ReadAll(downloadReader)
		require.NoError(t, err)
		assert.Equal(t, content, downloaded)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		downloadReader, metadata, err := storage.Download(ctx, "non-existent.txt")

		assert.Error(t, err)
		assert.Nil(t, downloadReader)
		assert.Nil(t, metadata)
	})
}

func TestFileStorage_Delete(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should delete uploaded file", func(t *testing.T) {
		key := "test-delete.txt"
		content := []byte("Delete me!")
		reader := bytes.NewReader(content)

		_, err := storage.Upload(ctx, key, reader, output.UploadOptions{})
		require.NoError(t, err)

		err = storage.Delete(ctx, key)
		require.NoError(t, err)

		// Verify file no longer exists
		exists, _ := storage.Exists(ctx, key)
		assert.False(t, exists)
	})
}

func TestFileStorage_Exists(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should return true for existing file", func(t *testing.T) {
		key := "test-exists.txt"
		content := []byte("I exist!")
		reader := bytes.NewReader(content)

		_, err := storage.Upload(ctx, key, reader, output.UploadOptions{})
		require.NoError(t, err)

		exists, err := storage.Exists(ctx, key)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("should return false for non-existent file", func(t *testing.T) {
		exists, err := storage.Exists(ctx, "non-existent-file.txt")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFileStorage_GetMetadata(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should get metadata for existing file", func(t *testing.T) {
		key := "test-metadata.txt"
		content := []byte("Metadata test content")
		reader := bytes.NewReader(content)

		_, err := storage.Upload(ctx, key, reader, output.UploadOptions{
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		metadata, err := storage.GetMetadata(ctx, key)

		require.NoError(t, err)
		assert.Equal(t, key, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)
		assert.Equal(t, "text/plain", metadata.ContentType)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		metadata, err := storage.GetMetadata(ctx, "non-existent.txt")

		assert.Error(t, err)
		assert.Nil(t, metadata)
	})
}

func TestFileStorage_GeneratePresignedURL(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should generate presigned URL", func(t *testing.T) {
		key := "presigned-test.txt"

		url, err := storage.GeneratePresignedURL(ctx, key, time.Hour)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, key)
		assert.Contains(t, url, "X-Amz-Signature")
	})
}

func TestFileStorage_GenerateUploadURL(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should generate upload URL", func(t *testing.T) {
		key := "upload-url-test.txt"

		url, err := storage.GenerateUploadURL(ctx, key, "text/plain", time.Hour)

		require.NoError(t, err)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, key)
		assert.Contains(t, url, "X-Amz-Signature")
	})
}

func TestFileStorage_List(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should list files with prefix", func(t *testing.T) {
		// Upload test files
		for i := 0; i < 3; i++ {
			key := "list-test/file" + string(rune('0'+i)) + ".txt"
			reader := bytes.NewReader([]byte("content"))
			_, err := storage.Upload(ctx, key, reader, output.UploadOptions{})
			require.NoError(t, err)
		}

		files, err := storage.List(ctx, "list-test/", 100)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 3)
	})

	t.Run("should respect maxKeys limit", func(t *testing.T) {
		// Upload test files
		for i := 0; i < 5; i++ {
			key := "list-limit/file" + string(rune('0'+i)) + ".txt"
			reader := bytes.NewReader([]byte("content"))
			_, err := storage.Upload(ctx, key, reader, output.UploadOptions{})
			require.NoError(t, err)
		}

		files, err := storage.List(ctx, "list-limit/", 2)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(files), 2)
	})
}

func TestFileStorage_Copy(t *testing.T) {
	minioContainer := SetupMinIO(t)
	defer minioContainer.Cleanup(t)

	ctx := context.Background()
	cfg := appconfig.S3Config{
		Endpoint:        minioContainer.Endpoint,
		Region:          testRegion,
		Bucket:          testBucketName,
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		UsePathStyle:    true,
	}
	storage, err := s3storage.NewFileStorage(ctx, cfg, getTestLogger())
	require.NoError(t, err)

	t.Run("should copy file to new location", func(t *testing.T) {
		sourceKey := "copy-source.txt"
		destKey := "copy-dest.txt"
		content := []byte("Copy me!")
		reader := bytes.NewReader(content)

		_, err := storage.Upload(ctx, sourceKey, reader, output.UploadOptions{})
		require.NoError(t, err)

		metadata, err := storage.Copy(ctx, sourceKey, destKey)

		require.NoError(t, err)
		assert.Equal(t, destKey, metadata.Key)
		assert.Equal(t, int64(len(content)), metadata.Size)

		// Verify destination file exists
		exists, _ := storage.Exists(ctx, destKey)
		assert.True(t, exists)

		// Verify source file still exists
		exists, _ = storage.Exists(ctx, sourceKey)
		assert.True(t, exists)
	})
}
