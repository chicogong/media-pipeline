package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLocalStorage_Get_WrongScheme exercises the scheme-rejection branch in Get.
func TestLocalStorage_Get_WrongScheme(t *testing.T) {
	stor := NewLocalStorage()
	ctx := context.Background()

	reader, err := stor.Get(ctx, "http://x/y")
	assert.Nil(t, reader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file://")
}

// TestLocalStorage_Put_WrongScheme exercises the scheme-rejection branch in Put.
func TestLocalStorage_Put_WrongScheme(t *testing.T) {
	stor := NewLocalStorage()
	ctx := context.Background()

	err := stor.Put(ctx, "http://x/y", strings.NewReader("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file://")
}

// TestLocalStorage_Delete_WrongScheme exercises the scheme-rejection branch in Delete.
func TestLocalStorage_Delete_WrongScheme(t *testing.T) {
	stor := NewLocalStorage()
	ctx := context.Background()

	err := stor.Delete(ctx, "http://x/y")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file://")
}

// TestLocalStorage_Get_NonExistentFile checks that Get returns an error when the
// file path is valid (file:// scheme) but the file does not exist on disk.
func TestLocalStorage_Get_NonExistentFile(t *testing.T) {
	stor := NewLocalStorage()
	ctx := context.Background()

	// Use a path that is guaranteed not to exist.
	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist.txt")
	reader, err := stor.Get(ctx, "file://"+nonExistentPath)
	assert.Nil(t, reader)
	assert.Error(t, err)
}
