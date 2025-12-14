package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// AllowedSchemes is the whitelist of allowed URI schemes
var AllowedSchemes = []string{"https", "http", "s3", "gs", "azure", "file"}

// Storage is the interface for all storage backends
type Storage interface {
	// Get downloads a file from the given URI and returns a reader
	Get(ctx context.Context, uri string) (io.ReadCloser, error)

	// Put uploads data to the given URI
	Put(ctx context.Context, uri string, data io.Reader) error

	// Delete removes a file at the given URI
	Delete(ctx context.Context, uri string) error

	// Exists checks if a file exists at the given URI
	Exists(ctx context.Context, uri string) (bool, error)
}

// ParseURI parses a URI and returns scheme and path
func ParseURI(uri string) (scheme string, path string, err error) {
	if uri == "" {
		return "", "", fmt.Errorf("URI cannot be empty")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "", "", fmt.Errorf("invalid URI: %w", err)
	}

	if parsed.Scheme == "" {
		return "", "", fmt.Errorf("URI must have a scheme (e.g., https://, s3://)")
	}

	// For file:// URIs, use the full path
	if parsed.Scheme == "file" {
		return parsed.Scheme, parsed.Path, nil
	}

	// For other URIs (s3://, https://, etc.), combine host and path
	path = parsed.Host
	if parsed.Path != "" {
		path = path + parsed.Path
	}

	return parsed.Scheme, path, nil
}

// IsAllowedScheme checks if a URI scheme is in the whitelist
func IsAllowedScheme(scheme string) bool {
	for _, allowed := range AllowedSchemes {
		if scheme == allowed {
			return true
		}
	}
	return false
}
