// Package s3 provides S3-compatible file storage using MinIO/AWS SDK.
package s3

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/output"
	appconfig "github.com/handiism/go-clean-arch-poc/pkg/config"
)

// Ensure FileStorage implements the output.FileStorage interface.
var _ output.FileStorage = (*FileStorage)(nil)

// FileStorage implements file storage using S3/MinIO.
type FileStorage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	logger    *slog.Logger
}

// NewFileStorage creates a new S3 file storage.
func NewFileStorage(ctx context.Context, cfg appconfig.S3Config, logger *slog.Logger) (*FileStorage, error) {
	// Create custom resolver for MinIO endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		if cfg.Endpoint != "" {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	presigner := s3.NewPresignClient(client)

	logger.Info("s3 client connected",
		"endpoint", cfg.Endpoint,
		"bucket", cfg.Bucket,
		"region", cfg.Region,
	)

	return &FileStorage{
		client:    client,
		presigner: presigner,
		bucket:    cfg.Bucket,
		logger:    logger,
	}, nil
}

// Upload uploads a file and returns its metadata.
func (fs *FileStorage) Upload(ctx context.Context, key string, reader io.Reader, options output.UploadOptions) (*output.FileMetadata, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}

	if options.ContentType != "" {
		input.ContentType = aws.String(options.ContentType)
	}

	if len(options.Metadata) > 0 {
		input.Metadata = options.Metadata
	}

	_, err := fs.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	// Get metadata
	metadata, err := fs.GetMetadata(ctx, key)
	if err != nil {
		return nil, err
	}

	fs.logger.Debug("file uploaded", "key", key)

	return metadata, nil
}

// Download downloads a file by key.
func (fs *FileStorage) Download(ctx context.Context, key string) (io.ReadCloser, *output.FileMetadata, error) {
	result, err := fs.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download object: %w", err)
	}

	metadata := &output.FileMetadata{
		Key:         key,
		ContentType: aws.ToString(result.ContentType),
		Size:        aws.ToInt64(result.ContentLength),
	}

	if result.ETag != nil {
		metadata.ETag = *result.ETag
	}

	if result.LastModified != nil {
		metadata.LastModified = *result.LastModified
	}

	return result.Body, metadata, nil
}

// Delete removes a file by key.
func (fs *FileStorage) Delete(ctx context.Context, key string) error {
	_, err := fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	fs.logger.Debug("file deleted", "key", key)

	return nil
}

// Exists checks if a file exists.
func (fs *FileStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// TODO: Check for specific "not found" error
		return false, nil
	}
	return true, nil
}

// GetMetadata retrieves file metadata without downloading.
func (fs *FileStorage) GetMetadata(ctx context.Context, key string) (*output.FileMetadata, error) {
	result, err := fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	metadata := &output.FileMetadata{
		Key:         key,
		ContentType: aws.ToString(result.ContentType),
		Size:        aws.ToInt64(result.ContentLength),
		Metadata:    result.Metadata,
	}

	if result.ETag != nil {
		metadata.ETag = *result.ETag
	}

	if result.LastModified != nil {
		metadata.LastModified = *result.LastModified
	}

	return metadata, nil
}

// GeneratePresignedURL generates a pre-signed URL for temporary download access.
func (fs *FileStorage) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	request, err := fs.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return request.URL, nil
}

// GenerateUploadURL generates a pre-signed URL for direct upload.
func (fs *FileStorage) GenerateUploadURL(ctx context.Context, key string, contentType string, expiration time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	request, err := fs.presigner.PresignPutObject(ctx, input, s3.WithPresignExpires(expiration))
	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return request.URL, nil
}

// List lists files with a given prefix.
func (fs *FileStorage) List(ctx context.Context, prefix string, maxKeys int) ([]*output.FileMetadata, error) {
	result, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(fs.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(maxKeys)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	files := make([]*output.FileMetadata, 0, len(result.Contents))
	for _, obj := range result.Contents {
		files = append(files, &output.FileMetadata{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
		})
	}

	return files, nil
}

// Copy copies a file from one key to another.
func (fs *FileStorage) Copy(ctx context.Context, sourceKey, destKey string) (*output.FileMetadata, error) {
	_, err := fs.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(fs.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", fs.bucket, sourceKey)),
		Key:        aws.String(destKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to copy object: %w", err)
	}

	return fs.GetMetadata(ctx, destKey)
}
