package prober

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestProbeLocalFile tests probing a local file
func TestProbeLocalFile(t *testing.T) {
	// Skip if ffprobe not available
	if !isFFprobeAvailable() {
		t.Skip("ffprobe not available")
	}

	// Create a test video file
	testFile := createTestVideoFile(t)
	defer os.Remove(testFile)

	p := NewProber()
	ctx := context.Background()

	info, err := p.Probe(ctx, testFile)
	if err != nil {
		t.Fatalf("Probe() failed: %v", err)
	}

	// Validate basic fields
	if info == nil {
		t.Fatal("Expected non-nil MediaInfo")
	}

	if info.Format.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	if len(info.VideoStreams) == 0 && len(info.AudioStreams) == 0 {
		t.Error("Expected at least one video or audio stream")
	}
}

// TestProbeNonExistentFile tests error handling for missing files
func TestProbeNonExistentFile(t *testing.T) {
	if !isFFprobeAvailable() {
		t.Skip("ffprobe not available")
	}

	p := NewProber()
	ctx := context.Background()

	_, err := p.Probe(ctx, "/nonexistent/file.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestProbeWithContext tests context cancellation
func TestProbeWithContext(t *testing.T) {
	if !isFFprobeAvailable() {
		t.Skip("ffprobe not available")
	}

	p := NewProber()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testFile := createTestVideoFile(t)
	defer os.Remove(testFile)

	_, err := p.Probe(ctx, testFile)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestProbeWithOptions tests prober with custom options
func TestProbeWithOptions(t *testing.T) {
	if !isFFprobeAvailable() {
		t.Skip("ffprobe not available")
	}

	p := NewProber(WithFFprobePath("/usr/local/bin/ffprobe"))
	if p == nil {
		t.Fatal("Expected non-nil Prober")
	}
}

// TestParseFFprobeOutput tests parsing ffprobe JSON output
func TestParseFFprobeOutput(t *testing.T) {
	// Sample ffprobe JSON output
	jsonOutput := `{
		"format": {
			"filename": "test.mp4",
			"format_name": "mov,mp4,m4a,3gp,3g2,mj2",
			"duration": "10.000000",
			"size": "1048576",
			"bit_rate": "838860"
		},
		"streams": [
			{
				"index": 0,
				"codec_type": "video",
				"codec_name": "h264",
				"width": 1920,
				"height": 1080,
				"r_frame_rate": "30/1",
				"bit_rate": "750000",
				"duration": "10.000000"
			},
			{
				"index": 1,
				"codec_type": "audio",
				"codec_name": "aac",
				"sample_rate": "48000",
				"channels": 2,
				"bit_rate": "128000",
				"duration": "10.000000"
			}
		]
	}`

	info, err := parseFFprobeOutput([]byte(jsonOutput))
	if err != nil {
		t.Fatalf("parseFFprobeOutput() failed: %v", err)
	}

	// Validate format info
	if info.Format.Filename != "test.mp4" {
		t.Errorf("Expected filename 'test.mp4', got '%s'", info.Format.Filename)
	}
	if info.Format.Duration != 10*time.Second {
		t.Errorf("Expected duration 10s, got %v", info.Format.Duration)
	}
	if info.Format.Size != 1048576 {
		t.Errorf("Expected size 1048576, got %d", info.Format.Size)
	}
	if info.Format.BitRate != 838860 {
		t.Errorf("Expected bitrate 838860, got %d", info.Format.BitRate)
	}

	// Validate video stream
	if len(info.VideoStreams) != 1 {
		t.Fatalf("Expected 1 video stream, got %d", len(info.VideoStreams))
	}
	video := info.VideoStreams[0]
	if video.Codec != "h264" {
		t.Errorf("Expected codec 'h264', got '%s'", video.Codec)
	}
	if video.Width != 1920 || video.Height != 1080 {
		t.Errorf("Expected resolution 1920x1080, got %dx%d", video.Width, video.Height)
	}
	if video.FrameRate != 30.0 {
		t.Errorf("Expected frame rate 30.0, got %f", video.FrameRate)
	}

	// Validate audio stream
	if len(info.AudioStreams) != 1 {
		t.Fatalf("Expected 1 audio stream, got %d", len(info.AudioStreams))
	}
	audio := info.AudioStreams[0]
	if audio.Codec != "aac" {
		t.Errorf("Expected codec 'aac', got '%s'", audio.Codec)
	}
	if audio.SampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", audio.SampleRate)
	}
	if audio.Channels != 2 {
		t.Errorf("Expected 2 channels, got %d", audio.Channels)
	}
}

// TestParseInvalidJSON tests error handling for invalid JSON
func TestParseInvalidJSON(t *testing.T) {
	_, err := parseFFprobeOutput([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Helper functions

func isFFprobeAvailable() bool {
	p := NewProber()
	return p.ffprobePath != ""
}

func createTestVideoFile(t *testing.T) string {
	// For testing, create a small test video using ffmpeg
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.mp4")

	// Try to create a minimal test video with ffmpeg
	// This is a 1-second black video with silent audio
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "lavfi",
		"-i", "color=black:s=320x240:r=1:d=1",
		"-f", "lavfi",
		"-i", "anullsrc=r=48000:cl=stereo:d=1",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-t", "1",
		"-y",
		testFile,
	)

	if err := cmd.Run(); err != nil {
		t.Skip("ffmpeg not available or failed to create test file")
	}

	return testFile
}
