package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		wantBucket  string
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid S3 URI",
			uri:        "s3://my-bucket/path/to/file.mp4",
			wantBucket: "my-bucket",
			wantKey:    "path/to/file.mp4",
			wantErr:    false,
		},
		{
			name:       "S3 URI with single key",
			uri:        "s3://bucket/file.txt",
			wantBucket: "bucket",
			wantKey:    "file.txt",
			wantErr:    false,
		},
		{
			name:       "S3 URI with nested path",
			uri:        "s3://my-bucket/videos/2024/01/sample.mp4",
			wantBucket: "my-bucket",
			wantKey:    "videos/2024/01/sample.mp4",
			wantErr:    false,
		},
		{
			name:        "missing bucket",
			uri:         "s3:///path/to/file.mp4",
			wantErr:     true,
			errContains: "missing bucket name",
		},
		{
			name:        "missing key",
			uri:         "s3://my-bucket/",
			wantErr:     true,
			errContains: "missing object key",
		},
		{
			name:        "bucket only",
			uri:         "s3://my-bucket",
			wantErr:     true,
			errContains: "missing object key",
		},
		{
			name:        "wrong scheme",
			uri:         "https://bucket/file.txt",
			wantErr:     true,
			errContains: "S3 storage only supports s3://",
		},
		{
			name:        "empty URI",
			uri:         "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := parseS3URI(tt.uri)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantBucket, bucket)
				assert.Equal(t, tt.wantKey, key)
			}
		})
	}
}

func TestNewS3Storage(t *testing.T) {
	// This test checks if S3Storage can be created
	// It may fail if AWS credentials are not configured
	ctx := context.Background()

	storage, err := NewS3Storage(ctx)

	// We don't require AWS credentials in CI, so either outcome is acceptable
	if err != nil {
		t.Logf("NewS3Storage failed (expected if AWS credentials not configured): %v", err)
	} else {
		assert.NotNil(t, storage)
		assert.NotNil(t, storage.client)
	}
}

// TestS3StorageInterface verifies that S3Storage implements the Storage interface
func TestS3StorageInterface(t *testing.T) {
	ctx := context.Background()
	storage, err := NewS3Storage(ctx)

	// Skip if AWS credentials are not configured
	if err != nil {
		t.Skip("Skipping interface test: AWS credentials not configured")
	}

	// Verify it implements Storage interface
	var _ Storage = storage
}

// Note: Integration tests that actually interact with S3 should be in a separate
// file (e.g., s3_integration_test.go) and run with a build tag like:
// //go:build integration
// These tests would require:
// - AWS credentials configured
// - A test bucket set up
// - Proper cleanup after tests
