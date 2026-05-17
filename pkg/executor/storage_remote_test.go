package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestStorageManager_DownloadInput_HTTP covers the remote-download branch of
// DownloadInput using an httptest server in place of a real HTTP source.
func TestStorageManager_DownloadInput_HTTP(t *testing.T) {
	const content = "remote video bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv.Close()

	sm := NewStorageManager()
	dir := t.TempDir()

	got, err := sm.DownloadInput(context.Background(), srv.URL+"/video.mp4", dir)
	if err != nil {
		t.Fatalf("DownloadInput returned unexpected error: %v", err)
	}

	if filepath.Dir(got) != dir {
		t.Errorf("downloaded file %q is not inside temp dir %q", got, dir)
	}

	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != content {
		t.Errorf("downloaded content = %q, want %q", string(data), content)
	}
}

// TestStorageManager_DownloadInput_HTTPError verifies a non-200 response
// surfaces as an error.
func TestStorageManager_DownloadInput_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	sm := NewStorageManager()
	if _, err := sm.DownloadInput(context.Background(), srv.URL+"/missing.mp4", t.TempDir()); err == nil {
		t.Error("expected error for HTTP 404 response, got nil")
	}
}

// TestStorageManager_UploadOutput_HTTPUnsupported covers the remote-upload
// branch of UploadOutput: HTTP storage is read-only, so the upload must fail.
func TestStorageManager_UploadOutput_HTTPUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	sm := NewStorageManager()
	dir := t.TempDir()
	src := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(src, []byte("encoded"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	if err := sm.UploadOutput(context.Background(), src, srv.URL+"/out.mp4"); err == nil {
		t.Error("expected error uploading to a read-only HTTP destination, got nil")
	}
}

// TestStorageManager_UploadOutput_MissingLocalFile covers the failure path when
// the local source file cannot be opened.
func TestStorageManager_UploadOutput_MissingLocalFile(t *testing.T) {
	sm := NewStorageManager()
	missing := filepath.Join(t.TempDir(), "does-not-exist.mp4")

	if err := sm.UploadOutput(context.Background(), missing, "https://example.com/x.mp4"); err == nil {
		t.Error("expected error for missing local source file, got nil")
	}
}
