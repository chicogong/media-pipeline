package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements Storage for local filesystem
type LocalStorage struct{}

// NewLocalStorage creates a new local storage backend
func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

// Get reads a local file
func (ls *LocalStorage) Get(ctx context.Context, uri string) (io.ReadCloser, error) {
	scheme, path, err := ParseURI(uri)
	if err != nil {
		return nil, err
	}

	if scheme != "file" {
		return nil, fmt.Errorf("local storage only supports file:// URIs, got %s://", scheme)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Put writes data to a local file
func (ls *LocalStorage) Put(ctx context.Context, uri string, data io.Reader) error {
	scheme, path, err := ParseURI(uri)
	if err != nil {
		return err
	}

	if scheme != "file" {
		return fmt.Errorf("local storage only supports file:// URIs, got %s://", scheme)
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Delete removes a local file
func (ls *LocalStorage) Delete(ctx context.Context, uri string) error {
	scheme, path, err := ParseURI(uri)
	if err != nil {
		return err
	}

	if scheme != "file" {
		return fmt.Errorf("local storage only supports file:// URIs, got %s://", scheme)
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if a local file exists
func (ls *LocalStorage) Exists(ctx context.Context, uri string) (bool, error) {
	scheme, path, err := ParseURI(uri)
	if err != nil {
		return false, err
	}

	if scheme != "file" {
		return false, fmt.Errorf("local storage only supports file:// URIs, got %s://", scheme)
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
