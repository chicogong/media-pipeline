package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/storage"
)

// StorageManager manages file downloads and uploads for different storage backends
type StorageManager struct {
	local *storage.LocalStorage
	http  *storage.HTTPStorage
	s3    *storage.S3Storage
}

// NewStorageManager creates a new storage manager
func NewStorageManager() *StorageManager {
	sm := &StorageManager{
		local: storage.NewLocalStorage(),
		http:  storage.NewHTTPStorage(),
	}

	// Try to initialize S3 (may fail if no AWS credentials)
	ctx := context.Background()
	s3Storage, err := storage.NewS3Storage(ctx)
	if err == nil {
		sm.s3 = s3Storage
	}

	return sm
}

// getStorage returns the appropriate storage backend for a URI
func (sm *StorageManager) getStorage(uri string) (storage.Storage, error) {
	scheme, _, err := storage.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	switch scheme {
	case "file":
		return sm.local, nil
	case "http", "https":
		return sm.http, nil
	case "s3":
		if sm.s3 == nil {
			return nil, fmt.Errorf("S3 storage not initialized (AWS credentials may be missing)")
		}
		return sm.s3, nil
	default:
		return nil, fmt.Errorf("unsupported URI scheme: %s", scheme)
	}
}

// isRemote checks if a URI points to a remote resource
func (sm *StorageManager) isRemote(uri string) bool {
	scheme, _, err := storage.ParseURI(uri)
	if err != nil {
		return false
	}
	return scheme != "file"
}

// DownloadInput downloads a remote file to a local temporary location
// Returns the local path if successful
func (sm *StorageManager) DownloadInput(ctx context.Context, uri, tempDir string) (string, error) {
	// If it's a local file, return the path as-is
	if !sm.isRemote(uri) {
		scheme, path, err := storage.ParseURI(uri)
		if err != nil {
			return "", err
		}
		if scheme == "file" {
			return path, nil
		}
	}

	// Get appropriate storage backend
	stor, err := sm.getStorage(uri)
	if err != nil {
		return "", err
	}

	// Create temp file
	fileName := filepath.Base(uri)
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "input"
	}
	tempPath := filepath.Join(tempDir, fileName)

	// Download file
	reader, err := stor.Get(ctx, uri)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", uri, err)
	}
	defer reader.Close()

	// Create temp file
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Copy data
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tempPath, nil
}

// UploadOutput uploads a local file to a remote destination
func (sm *StorageManager) UploadOutput(ctx context.Context, localPath, destURI string) error {
	// If destination is local, just copy/move the file
	if !sm.isRemote(destURI) {
		scheme, destPath, err := storage.ParseURI(destURI)
		if err != nil {
			return err
		}
		if scheme == "file" {
			// Create destination directory if needed
			dir := filepath.Dir(destPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			// Copy file
			return sm.copyFile(localPath, destPath)
		}
	}

	// Get appropriate storage backend
	stor, err := sm.getStorage(destURI)
	if err != nil {
		return err
	}

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Upload file
	err = stor.Put(ctx, destURI, file)
	if err != nil {
		return fmt.Errorf("failed to upload to %s: %w", destURI, err)
	}

	return nil
}

// copyFile copies a file from src to dst
func (sm *StorageManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// PrepareInputs downloads all remote inputs and returns a map of original URI -> local path
func (sm *StorageManager) PrepareInputs(ctx context.Context, plan *schemas.ProcessingPlan, tempDir string) (map[string]string, error) {
	inputMap := make(map[string]string)

	for _, node := range plan.Nodes {
		if node.Type == "input" {
			originalURI := node.SourceURI
			localPath, err := sm.DownloadInput(ctx, originalURI, tempDir)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare input %s: %w", originalURI, err)
			}
			inputMap[originalURI] = localPath
		}
	}

	return inputMap, nil
}

// UploadOutputs uploads all outputs to their destination URIs
func (sm *StorageManager) UploadOutputs(ctx context.Context, plan *schemas.ProcessingPlan, outputFiles map[string]string) error {
	for _, node := range plan.Nodes {
		if node.Type == "output" {
			localPath, ok := outputFiles[node.ID]
			if !ok {
				return fmt.Errorf("output file not found for node %s", node.ID)
			}

			err := sm.UploadOutput(ctx, localPath, node.DestURI)
			if err != nil {
				return fmt.Errorf("failed to upload output %s: %w", node.ID, err)
			}
		}
	}

	return nil
}

// CleanupTempDir removes temporary directory and all its contents
func (sm *StorageManager) CleanupTempDir(tempDir string) error {
	if tempDir == "" || tempDir == "/" || tempDir == "." {
		return fmt.Errorf("invalid temp directory: %s", tempDir)
	}

	// Only cleanup if it's in a temp location
	if !strings.Contains(tempDir, "tmp") && !strings.Contains(tempDir, "temp") {
		return fmt.Errorf("refusing to cleanup non-temp directory: %s", tempDir)
	}

	return os.RemoveAll(tempDir)
}
