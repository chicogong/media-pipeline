package executor

import (
	"testing"
	"time"
)

func TestProgressParser_ParseLine(t *testing.T) {
	parser := NewProgressParser()

	// Test typical FFmpeg progress line
	line := "frame=  100 fps= 30 q=-1.0 size=    1024kB time=00:00:03.33 bitrate=2000.0kbits/s speed=1.0x"
	progress := parser.ParseLine(line)

	if progress == nil {
		t.Fatal("progress is nil")
	}

	if progress.Frame != 100 {
		t.Errorf("expected frame 100, got %d", progress.Frame)
	}

	if progress.FPS != 30 {
		t.Errorf("expected fps 30, got %.2f", progress.FPS)
	}

	expectedTime := 3*time.Second + 330*time.Millisecond
	if progress.Time != expectedTime {
		t.Errorf("expected time %v, got %v", expectedTime, progress.Time)
	}

	if progress.Speed != 1.0 {
		t.Errorf("expected speed 1.0, got %.2f", progress.Speed)
	}
}

func TestProgressParser_ParseLineVariations(t *testing.T) {
	parser := NewProgressParser()

	testCases := []struct {
		name         string
		line         string
		expectFrame  int
		expectFPS    float64
		expectSpeed  float64
	}{
		{
			name:        "normal progress",
			line:        "frame=  500 fps=60.0 q=28.0 size=    5120kB time=00:00:16.67 bitrate=2512.4kbits/s speed=2.0x",
			expectFrame: 500,
			expectFPS:   60.0,
			expectSpeed: 2.0,
		},
		{
			name:        "low fps",
			line:        "frame=   10 fps=5.5 q=-1.0 size=     128kB time=00:00:00.33 bitrate=3072.0kbits/s speed=0.18x",
			expectFrame: 10,
			expectFPS:   5.5,
			expectSpeed: 0.18,
		},
		{
			name:        "high frame count",
			line:        "frame=10000 fps=120 q=25.0 size=  102400kB time=00:05:33.33 bitrate=2048.0kbits/s speed=4.0x",
			expectFrame: 10000,
			expectFPS:   120,
			expectSpeed: 4.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			progress := parser.ParseLine(tc.line)
			if progress == nil {
				t.Fatal("progress is nil")
			}

			if progress.Frame != tc.expectFrame {
				t.Errorf("expected frame %d, got %d", tc.expectFrame, progress.Frame)
			}

			if progress.FPS != tc.expectFPS {
				t.Errorf("expected fps %.2f, got %.2f", tc.expectFPS, progress.FPS)
			}

			if progress.Speed != tc.expectSpeed {
				t.Errorf("expected speed %.2f, got %.2f", tc.expectSpeed, progress.Speed)
			}
		})
	}
}

func TestProgressParser_ParseLineInvalid(t *testing.T) {
	parser := NewProgressParser()

	testCases := []string{
		"",
		"random text",
		"ffmpeg version 4.4",
		"Input #0, mov,mp4,m4a,3gp,3g2,mj2, from 'input.mp4':",
	}

	for _, line := range testCases {
		progress := parser.ParseLine(line)
		if progress != nil {
			t.Errorf("expected nil for line '%s', got %+v", line, progress)
		}
	}
}

func TestProgressParser_ParseTime(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"00:00:00.00", 0},
		{"00:00:01.00", time.Second},
		{"00:00:03.33", 3*time.Second + 330*time.Millisecond},
		{"00:01:00.00", time.Minute},
		{"00:05:30.50", 5*time.Minute + 30*time.Second + 500*time.Millisecond},
		{"01:00:00.00", time.Hour},
	}

	for _, tc := range testCases {
		result := parseFFmpegTime(tc.input)
		if result != tc.expected {
			t.Errorf("parseFFmpegTime(%s): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

func TestProgressParser_ComputePercentage(t *testing.T) {
	parser := NewProgressParser()

	// Set total duration
	totalDuration := 60 * time.Second
	parser.SetTotalDuration(totalDuration)

	progress := &Progress{
		Time: 30 * time.Second,
	}

	percentage := parser.ComputePercentage(progress)
	if percentage != 50.0 {
		t.Errorf("expected 50%%, got %.2f%%", percentage)
	}

	// Test edge cases
	progress.Time = 0
	percentage = parser.ComputePercentage(progress)
	if percentage != 0.0 {
		t.Errorf("expected 0%%, got %.2f%%", percentage)
	}

	progress.Time = 60 * time.Second
	percentage = parser.ComputePercentage(progress)
	if percentage != 100.0 {
		t.Errorf("expected 100%%, got %.2f%%", percentage)
	}

	// Test without total duration set
	parser2 := NewProgressParser()
	percentage = parser2.ComputePercentage(progress)
	if percentage != 0.0 {
		t.Errorf("expected 0%% when total duration unknown, got %.2f%%", percentage)
	}
}
