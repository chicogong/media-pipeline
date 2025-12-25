package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Storage implements Storage for Amazon S3
type S3Storage struct {
	client *s3.Client
}

// NewS3Storage creates a new S3 storage backend
// Uses AWS SDK default credentials chain (env vars, config files, IAM roles)
func NewS3Storage(ctx context.Context) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &S3Storage{
		client: s3.NewFromConfig(cfg),
	}, nil
}

// NewS3StorageWithClient creates a new S3 storage with a custom client
// Useful for testing and custom configurations
func NewS3StorageWithClient(client *s3.Client) *S3Storage {
	return &S3Storage{
		client: client,
	}
}

// parseS3URI parses s3://bucket/key/path into bucket and key
func parseS3URI(uri string) (bucket, key string, err error) {
	scheme, path, err := ParseURI(uri)
	if err != nil {
		return "", "", err
	}

	if scheme != "s3" {
		return "", "", fmt.Errorf("S3 storage only supports s3:// URIs, got %s://", scheme)
	}

	// path is "bucket/key/path"
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid S3 URI: missing bucket name")
	}

	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}

	if key == "" {
		return "", "", fmt.Errorf("invalid S3 URI: missing object key")
	}

	return bucket, key, nil
}

// Get downloads an object from S3 and returns a reader
func (s *S3Storage) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return nil, err
	}

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}

	return result.Body, nil
}

// Put uploads data to S3
func (s *S3Storage) Put(ctx context.Context, uri string, data io.Reader) error {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to put S3 object: %w", err)
	}

	return nil
}

// Delete removes an object from S3
func (s *S3Storage) Delete(ctx context.Context, uri string) error {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 object: %w", err)
	}

	return nil
}

// Exists checks if an object exists in S3
func (s *S3Storage) Exists(ctx context.Context, uri string) (bool, error) {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return false, err
	}

	_, err = s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			// NotFound can be either NoSuchKey or NotFound (404)
			if apiErr.ErrorCode() == "NotFound" {
				return false, nil
			}
			// Check for 404 status code
			if httpResp, ok := apiErr.(interface{ HTTPStatusCode() int }); ok {
				if httpResp.HTTPStatusCode() == http.StatusNotFound {
					return false, nil
				}
			}
		}

		// Check for specific S3 error types
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}

		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return false, nil
		}

		// Other error
		return false, fmt.Errorf("failed to check S3 object existence: %w", err)
	}

	return true, nil
}
