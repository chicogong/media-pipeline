package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHTTPStorage_Delete covers the Delete method which always returns an error
// because HTTP storage is read-only.
func TestHTTPStorage_Delete(t *testing.T) {
	stor := NewHTTPStorage()
	ctx := context.Background()

	err := stor.Delete(ctx, "https://example.com/file.mp4")
	assert.Error(t, err, "Delete should return an error for HTTP storage (read-only)")
	assert.Contains(t, err.Error(), "read-only")
}
