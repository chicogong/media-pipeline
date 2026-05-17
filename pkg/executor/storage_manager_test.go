package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestStorageManager_GetStorage(t *testing.T) {
	sm := NewStorageManager()

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"file scheme", "file:///some/path/file.mp4", false},
		{"http scheme", "http://example.com/file.mp4", false},
		{"https scheme", "https://example.com/file.mp4", false},
		{"ftp scheme unknown", "ftp://example.com/file.mp4", true},
		{"no scheme invalid", "not-a-uri", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stor, err := sm.getStorage(tc.uri)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for URI %q, got nil", tc.uri)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for URI %q: %v", tc.uri, err)
				}
				if stor == nil {
					t.Errorf("expected non-nil storage for URI %q", tc.uri)
				}
			}
		})
	}
}

func TestStorageManager_IsRemote(t *testing.T) {
	sm := NewStorageManager()

	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{"file URI is not remote", "file:///some/absolute/path.mp4", false},
		{"http URI is remote", "http://example.com/file.mp4", true},
		{"https URI is remote", "https://example.com/file.mp4", true},
		{"s3 URI is remote", "s3://bucket/key.mp4", true},
		{"bare path no scheme treated as not remote", "/no/scheme/x", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sm.isRemote(tc.uri)
			if got != tc.expected {
				t.Errorf("isRemote(%q) = %v, want %v", tc.uri, got, tc.expected)
			}
		})
	}
}

func TestStorageManager_DownloadInput_LocalFile(t *testing.T) {
	sm := NewStorageManager()
	ctx := context.Background()

	dir := t.TempDir()
	localFile := filepath.Join(dir, "input.mp4")
	if err := os.WriteFile(localFile, []byte("fake video data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	uri := "file://" + localFile
	got, err := sm.DownloadInput(ctx, uri, dir)
	if err != nil {
		t.Fatalf("DownloadInput returned unexpected error: %v", err)
	}
	// For file:// URIs the path is returned unchanged
	if got != localFile {
		t.Errorf("DownloadInput returned %q, want %q", got, localFile)
	}
}

func TestStorageManager_UploadOutput_LocalFile(t *testing.T) {
	sm := NewStorageManager()
	ctx := context.Background()

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "output.mp4")
	content := []byte("encoded video content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Destination inside a new subdir so we test MkdirAll
	dstFile := filepath.Join(dstDir, "subdir", "result.mp4")
	destURI := "file://" + dstFile

	if err := sm.UploadOutput(ctx, srcFile, destURI); err != nil {
		t.Fatalf("UploadOutput returned unexpected error: %v", err)
	}

	// Verify the file was copied correctly
	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("destination file content %q, want %q", string(got), string(content))
	}
}

func TestStorageManager_PrepareInputs(t *testing.T) {
	sm := NewStorageManager()
	ctx := context.Background()

	dir := t.TempDir()

	// Create a real local file to reference via file://
	localFile := filepath.Join(dir, "video.mp4")
	if err := os.WriteFile(localFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("success with file URI", func(t *testing.T) {
		plan := &schemas.ProcessingPlan{
			Nodes: []*schemas.PlanNode{
				{
					ID:        "input-video",
					Type:      "input",
					SourceURI: "file://" + localFile,
				},
				{
					ID:   "output-scaled",
					Type: "output",
				},
			},
		}

		inputMap, err := sm.PrepareInputs(ctx, plan, dir)
		if err != nil {
			t.Fatalf("PrepareInputs returned unexpected error: %v", err)
		}

		localPath, ok := inputMap["file://"+localFile]
		if !ok {
			t.Fatalf("inputMap missing entry for input URI")
		}
		if localPath != localFile {
			t.Errorf("localPath = %q, want %q", localPath, localFile)
		}
	})

	t.Run("error on unsupported scheme", func(t *testing.T) {
		plan := &schemas.ProcessingPlan{
			Nodes: []*schemas.PlanNode{
				{
					ID:        "input-bad",
					Type:      "input",
					SourceURI: "ftp://example.com/file.mp4",
				},
			},
		}

		_, err := sm.PrepareInputs(ctx, plan, dir)
		if err == nil {
			t.Error("expected error for unsupported scheme, got nil")
		}
	})
}

func TestStorageManager_UploadOutputs(t *testing.T) {
	sm := NewStorageManager()
	ctx := context.Background()

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	localFile := filepath.Join(srcDir, "encoded.mp4")
	content := []byte("result bytes")
	if err := os.WriteFile(localFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	destFile := filepath.Join(dstDir, "final.mp4")
	destURI := "file://" + destFile

	plan := &schemas.ProcessingPlan{
		Nodes: []*schemas.PlanNode{
			{
				ID:      "out1",
				Type:    "output",
				DestURI: destURI,
			},
		},
	}

	t.Run("success", func(t *testing.T) {
		outputFiles := map[string]string{
			"out1": localFile,
		}
		if err := sm.UploadOutputs(ctx, plan, outputFiles); err != nil {
			t.Fatalf("UploadOutputs returned unexpected error: %v", err)
		}
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("destination file does not exist: %v", err)
		}
	})

	t.Run("missing map entry returns error", func(t *testing.T) {
		outputFiles := map[string]string{} // "out1" missing
		if err := sm.UploadOutputs(ctx, plan, outputFiles); err == nil {
			t.Error("expected error for missing map entry, got nil")
		}
	})
}

func TestStorageManager_CleanupTempDir(t *testing.T) {
	sm := NewStorageManager()

	// Test rejection of dangerous paths
	rejectedPaths := []string{"", "/", ".", "/usr/local/lib"}
	for _, p := range rejectedPaths {
		t.Run("reject "+p, func(t *testing.T) {
			if err := sm.CleanupTempDir(p); err == nil {
				t.Errorf("expected error for path %q, got nil", p)
			}
		})
	}

	// Test success: create a directory whose path contains "tmp"
	t.Run("success removes tmpdir", func(t *testing.T) {
		base := t.TempDir()
		tmpDir := filepath.Join(base, "tmpdir")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		// Write a file inside to verify deep removal
		if err := os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("data"), 0644); err != nil {
			t.Fatalf("failed to create file in tmpdir: %v", err)
		}

		if err := sm.CleanupTempDir(tmpDir); err != nil {
			t.Errorf("CleanupTempDir returned unexpected error: %v", err)
		}
		if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
			t.Errorf("expected tmpdir to be removed, but Stat returned: %v", err)
		}
	})
}
