package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		uri      string
		scheme   string
		path     string
		wantErr  bool
	}{
		{"https://example.com/video.mp4", "https", "example.com/video.mp4", false},
		{"s3://bucket/key/video.mp4", "s3", "bucket/key/video.mp4", false},
		{"file:///tmp/video.mp4", "file", "/tmp/video.mp4", false},
		{"gs://bucket/object", "gs", "bucket/object", false},
		{"invalid-uri", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			scheme, path, err := ParseURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.scheme, scheme)
				assert.Equal(t, tt.path, path)
			}
		})
	}
}

func TestIsAllowedScheme(t *testing.T) {
	tests := []struct {
		scheme  string
		allowed bool
	}{
		{"https", true},
		{"http", true},
		{"s3", true},
		{"gs", true},
		{"file", true},
		{"ftp", false},
		{"gopher", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			assert.Equal(t, tt.allowed, IsAllowedScheme(tt.scheme))
		})
	}
}
