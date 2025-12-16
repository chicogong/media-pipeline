package executor

import (
	"context"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestExecutor_BuildCommand(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		JobID: "test-job",
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video",
				Output: "trimmed",
				Params: map[string]interface{}{
					"start":    "00:00:10",
					"duration": "00:00:30",
				},
			},
			{
				Op:     "scale",
				Input:  "trimmed",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":  1280,
					"height": 720,
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "/tmp/output.mp4"},
		},
	}

	// Generate plan
	p := planner.NewPlanner()
	plan, err := p.Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Create executor
	executor := NewExecutor(operators.GlobalRegistry())

	// Build command
	cmd, err := executor.BuildCommand(context.Background(), plan)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	// Verify command
	if cmd == nil {
		t.Fatal("command is nil")
	}

	if len(cmd.Args) == 0 {
		t.Fatal("command has no arguments")
	}

	if cmd.Args[0] != "ffmpeg" {
		t.Errorf("expected ffmpeg, got %s", cmd.Args[0])
	}
}

func TestExecutor_ExecuteSimulation(t *testing.T) {
	// This test just verifies the executor can be created and
	// doesn't execute actual FFmpeg command
	// Real execution tests would require FFmpeg to be installed

	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "scale",
				Input:  "video",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":  640,
					"height": 360,
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "/tmp/output.mp4"},
		},
	}

	p := planner.NewPlanner()
	plan, err := p.Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	executor := NewExecutor(operators.GlobalRegistry())

	// Just verify we can build the command
	cmd, err := executor.BuildCommand(context.Background(), plan)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	// Verify the command structure
	t.Logf("FFmpeg command: %v", cmd.Args)

	// Verify it has expected components
	hasInput := false
	hasOutput := false
	for i, arg := range cmd.Args {
		if arg == "-i" && i+1 < len(cmd.Args) {
			hasInput = true
		}
		if arg == "/tmp/output.mp4" {
			hasOutput = true
		}
	}

	if !hasInput {
		t.Error("command missing input file")
	}
	if !hasOutput {
		t.Error("command missing output file")
	}
}

func TestExecutor_WithProgressCallback(t *testing.T) {
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video",
				Output: "trimmed",
				Params: map[string]interface{}{
					"start":    "00:00:00",
					"duration": "00:00:10",
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	p := planner.NewPlanner()
	plan, err := p.Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	executor := NewExecutor(operators.GlobalRegistry())

	// Verify we can create options with callbacks
	progressCalled := false
	opts := &ExecuteOptions{
		OnProgress: func(progress *Progress) {
			progressCalled = true
			t.Logf("Progress: frame=%d fps=%.2f time=%v speed=%.2fx",
				progress.Frame, progress.FPS, progress.Time, progress.Speed)
		},
		OnLog: func(line string) {
			t.Logf("FFmpeg: %s", line)
		},
	}

	// Just verify the options are accepted
	// Actual execution would require FFmpeg
	if opts.OnProgress == nil {
		t.Error("OnProgress callback is nil")
	}

	// We can't actually execute without FFmpeg, so just verify the structure
	cmd, err := executor.BuildCommand(context.Background(), plan)
	if err != nil {
		t.Fatalf("BuildCommand failed: %v", err)
	}

	if cmd == nil {
		t.Fatal("command is nil")
	}

	// Note: progressCalled would be true if we actually executed FFmpeg
	_ = progressCalled
}
