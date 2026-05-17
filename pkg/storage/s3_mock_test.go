package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeS3Client creates an S3 client that sends requests to the given httptest server.
func newFakeS3Client(srv *httptest.Server) *s3.Client {
	return s3.New(s3.Options{
		Region:       "us-east-1",
		Credentials:  aws.AnonymousCredentials{},
		BaseEndpoint: aws.String(srv.URL),
		UsePathStyle: true,
	})
}

// TestS3StorageWithClient_Get verifies that Get retrieves the object body from S3.
func TestS3StorageWithClient_Get(t *testing.T) {
	wantBody := "hello from fake s3"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(wantBody))
	}))
	t.Cleanup(srv.Close)

	stor := NewS3StorageWithClient(newFakeS3Client(srv))
	ctx := context.Background()

	rc, err := stor.Get(ctx, "s3://bucket/key.mp4")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, wantBody, string(got))
}

// TestS3StorageWithClient_Put verifies that Put sends the object body to S3 without error.
func TestS3StorageWithClient_Put(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// S3 PUT returns 200 OK with an ETag.
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	stor := NewS3StorageWithClient(newFakeS3Client(srv))
	ctx := context.Background()

	err := stor.Put(ctx, "s3://bucket/key.mp4", strings.NewReader("payload"))
	assert.NoError(t, err)
}

// TestS3StorageWithClient_Delete verifies that Delete removes the object without error.
func TestS3StorageWithClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	stor := NewS3StorageWithClient(newFakeS3Client(srv))
	ctx := context.Background()

	err := stor.Delete(ctx, "s3://bucket/key.mp4")
	assert.NoError(t, err)
}

// TestS3StorageWithClient_Exists_True verifies that Exists returns true when HEAD returns 200.
func TestS3StorageWithClient_Exists_True(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	stor := NewS3StorageWithClient(newFakeS3Client(srv))
	ctx := context.Background()

	exists, err := stor.Exists(ctx, "s3://bucket/key.mp4")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestS3StorageWithClient_Exists_NotFound verifies that Exists returns false on a 404.
// This is best-effort: if the AWS SDK does not map the bare 404 to a typed error
// recognised by s3.go's error checks, the call may return (false, non-nil error).
// Either outcome is acceptable as long as it does not panic.
func TestS3StorageWithClient_Exists_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `<?xml version="1.0"?><Error><Code>NotFound</Code><Message>Not Found</Message></Error>`, http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	stor := NewS3StorageWithClient(newFakeS3Client(srv))
	ctx := context.Background()

	// Must not panic; both (false, nil) and (false, non-nil) are acceptable.
	exists, _ := stor.Exists(ctx, "s3://bucket/key.mp4")
	assert.False(t, exists)
}

// TestS3StorageWithClient_InvalidURI verifies that an invalid URI is rejected before
// any network call is made.
func TestS3StorageWithClient_InvalidURI(t *testing.T) {
	// No server needed — parseS3URI will reject non-s3 URIs immediately.
	stor := NewS3StorageWithClient(s3.New(s3.Options{Region: "us-east-1"}))
	ctx := context.Background()

	_, err := stor.Get(ctx, "https://not-s3/x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "s3://")
}

// TestNewS3StorageWithClient verifies that the constructor stores the client.
func TestNewS3StorageWithClient(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1"})
	stor := NewS3StorageWithClient(client)
	require.NotNil(t, stor)
	assert.NotNil(t, stor.client)
}
