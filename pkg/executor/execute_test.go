package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestExecutor_executeCommand(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}

	e := NewExecutor(operators.GlobalRegistry())

	script := `echo 'frame=   10 fps= 5.0 time=00:00:01.00 speed=1.5x' 1>&2; echo 'stdout log line'`

	// OnProgress/OnLog are invoked concurrently from the stderr and stdout
	// streaming goroutines, so shared state must be guarded.
	var mu sync.Mutex
	var logLines []string
	progressCount := 0

	opts := &ExecuteOptions{
		OnProgress: func(p *Progress) {
			mu.Lock()
			progressCount++
			mu.Unlock()
		},
		OnLog: func(line string) {
			mu.Lock()
			logLines = append(logLines, line)
			mu.Unlock()
		},
	}

	cmd := &Command{
		Args: []string{"sh", "-c", script},
	}

	if err := e.executeCommand(context.Background(), cmd, opts); err != nil {
		t.Fatalf("executeCommand returned unexpected error: %v", err)
	}

	if progressCount == 0 {
		t.Error("expected OnProgress to be called at least once")
	}

	joined := strings.Join(logLines, "\n")
	if !strings.Contains(joined, "frame=") {
		t.Errorf("expected log to contain 'frame=', got: %s", joined)
	}
	if !strings.Contains(joined, "stdout log line") {
		t.Errorf("expected log to contain 'stdout log line', got: %s", joined)
	}
}

func TestExecutor_executeCommand_Failure(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}

	e := NewExecutor(operators.GlobalRegistry())

	cmd := &Command{
		Args: []string{"sh", "-c", "exit 3"},
	}

	err := e.executeCommand(context.Background(), cmd, &ExecuteOptions{})
	if err == nil {
		t.Error("expected non-nil error for non-zero exit, got nil")
	}
}

func TestExecutor_executeCommand_WorkDir(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}

	dir := t.TempDir()
	markerFile := filepath.Join(dir, "marker.txt")
	if err := os.WriteFile(markerFile, []byte("MARKER_OK"), 0644); err != nil {
		t.Fatalf("failed to write marker file: %v", err)
	}

	e := NewExecutor(operators.GlobalRegistry())

	// OnLog may be invoked from both streaming goroutines; guard the slice.
	var mu sync.Mutex
	var logLines []string
	opts := &ExecuteOptions{
		OnLog: func(line string) {
			mu.Lock()
			logLines = append(logLines, line)
			mu.Unlock()
		},
	}

	cmd := &Command{
		Args:    []string{"sh", "-c", "cat marker.txt"},
		WorkDir: dir,
	}

	if err := e.executeCommand(context.Background(), cmd, opts); err != nil {
		t.Fatalf("executeCommand returned unexpected error: %v", err)
	}

	joined := strings.Join(logLines, "\n")
	if !strings.Contains(joined, "MARKER_OK") {
		t.Errorf("expected log to contain 'MARKER_OK', got: %s", joined)
	}
}

func TestExecutor_Execute_EndToEnd(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.mp4")
	outputPath := filepath.Join(dir, "output.mp4")

	// Generate a short test video with ffmpeg
	genCmd := exec.Command(ffmpeg, "-y",
		"-f", "lavfi",
		"-i", "testsrc=duration=1:size=320x240:rate=10",
		"-pix_fmt", "yuv420p",
		inputPath,
	)
	if out, err := genCmd.CombinedOutput(); err != nil {
		t.Skipf("failed to generate test video: %v\n%s", err, string(out))
	}

	// Register scale operator
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "file://" + inputPath},
		},
		Operations: []schemas.Operation{
			{
				Op:     "scale",
				Input:  "video",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":  160,
					"height": 120,
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "file://" + outputPath},
		},
	}

	plan, err := planner.NewPlanner().Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	ex := NewExecutor(operators.GlobalRegistry())
	if err := ex.Execute(context.Background(), plan, &ExecuteOptions{
		OnProgress: func(*Progress) {},
		OnLog:      func(string) {},
	}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}
