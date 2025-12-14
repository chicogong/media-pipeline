package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HTTPStorage implements Storage for HTTP/HTTPS downloads
type HTTPStorage struct {
	client *http.Client
}

// NewHTTPStorage creates a new HTTP storage backend
func NewHTTPStorage() *HTTPStorage {
	return &HTTPStorage{
		client: &http.Client{},
	}
}

// Get downloads a file over HTTP/HTTPS
func (hs *HTTPStorage) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	scheme, _, err := ParseURI(uri)
	if err != nil {
		return nil, err
	}

	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("HTTP storage only supports http:// and https:// URIs, got %s://", scheme)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := hs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// Put is not supported for HTTP storage (read-only)
func (hs *HTTPStorage) Put(ctx context.Context, uri string, data io.Reader) error {
	return fmt.Errorf("Put operation not supported for HTTP storage (read-only)")
}

// Delete is not supported for HTTP storage (read-only)
func (hs *HTTPStorage) Delete(ctx context.Context, uri string) error {
	return fmt.Errorf("HTTP storage does not support Delete operations (read-only)")
}

// Exists checks if a file exists by sending a HEAD request
func (hs *HTTPStorage) Exists(ctx context.Context, uri string) (bool, error) {
	scheme, _, err := ParseURI(uri)
	if err != nil {
		return false, err
	}

	if scheme != "http" && scheme != "https" {
		return false, fmt.Errorf("HTTP storage only supports http:// and https:// URIs, got %s://", scheme)
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", uri, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := hs.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
