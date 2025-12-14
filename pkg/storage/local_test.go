package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_GetPut(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello world"

	storage := NewLocalStorage()
	ctx := context.Background()

	// Test Put
	uri := "file://" + testFile
	err := storage.Put(ctx, uri, strings.NewReader(testContent))
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, testFile)

	// Test Get
	reader, err := storage.Get(ctx, uri)
	require.NoError(t, err)
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestLocalStorage_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(existingFile, []byte("test"), 0644)

	storage := NewLocalStorage()
	ctx := context.Background()

	// Test existing file
	exists, err := storage.Exists(ctx, "file://"+existingFile)
	require.NoError(t, err)
	assert.True(t, exists)

	// Test non-existing file
	exists, err = storage.Exists(ctx, "file://"+filepath.Join(tmpDir, "nonexistent.txt"))
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "delete-me.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	storage := NewLocalStorage()
	ctx := context.Background()

	// Delete the file
	err := storage.Delete(ctx, "file://"+testFile)
	require.NoError(t, err)

	// Verify file was deleted
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}
