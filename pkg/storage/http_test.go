package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPStorage_Get(t *testing.T) {
	// Create a test HTTP server
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	storage := NewHTTPStorage()
	ctx := context.Background()

	// Test Get
	reader, err := storage.Get(ctx, server.URL+"/test.mp4")
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestHTTPStorage_Get_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	storage := NewHTTPStorage()
	ctx := context.Background()

	reader, err := storage.Get(ctx, server.URL+"/notfound.mp4")
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "404")
}

func TestHTTPStorage_Put_NotSupported(t *testing.T) {
	storage := NewHTTPStorage()
	ctx := context.Background()

	err := storage.Put(ctx, "https://example.com/file.mp4", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestHTTPStorage_Exists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			if r.URL.Path == "/exists.mp4" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	storage := NewHTTPStorage()
	ctx := context.Background()

	// Test existing file
	exists, err := storage.Exists(ctx, server.URL+"/exists.mp4")
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing file
	exists, err = storage.Exists(ctx, server.URL+"/notfound.mp4")
	require.NoError(t, err)
	assert.False(t, exists)
}
