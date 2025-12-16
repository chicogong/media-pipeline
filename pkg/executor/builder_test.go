package executor

import (
	"context"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestCommandBuilder_BuildSimple(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a simple JobSpec
	spec := &schemas.JobSpec{
		JobID: "test-job",
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4", Type: "video"},
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

	// Build command
	builder := NewCommandBuilder(operators.GlobalRegistry())
	cmd, err := builder.Build(context.Background(), plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify command exists
	if cmd == nil {
		t.Fatal("command is nil")
	}

	// Verify command has ffmpeg
	if len(cmd.Args) == 0 {
		t.Fatal("command has no arguments")
	}
	if cmd.Args[0] != "ffmpeg" {
		t.Errorf("expected first arg 'ffmpeg', got '%s'", cmd.Args[0])
	}

	// Verify input file is included
	hasInput := false
	for i, arg := range cmd.Args {
		if arg == "-i" && i+1 < len(cmd.Args) {
			if cmd.Args[i+1] == "/tmp/input.mp4" {
				hasInput = true
				break
			}
		}
	}
	if !hasInput {
		t.Error("command does not include input file")
	}

	// Verify output file is included
	if cmd.Args[len(cmd.Args)-1] != "/tmp/output.mp4" {
		t.Errorf("expected last arg '/tmp/output.mp4', got '%s'", cmd.Args[len(cmd.Args)-1])
	}
}

func TestCommandBuilder_BuildWithFilter(t *testing.T) {
	// Register operators
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
					"width":  1280,
					"height": 720,
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

	builder := NewCommandBuilder(operators.GlobalRegistry())
	cmd, err := builder.Build(context.Background(), plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify filter_complex is present
	hasFilter := false
	for i, arg := range cmd.Args {
		if arg == "-filter_complex" && i+1 < len(cmd.Args) {
			hasFilter = true
			// Filter should contain scale
			filterExpr := cmd.Args[i+1]
			if len(filterExpr) == 0 {
				t.Error("filter expression is empty")
			}
			break
		}
	}
	if !hasFilter {
		t.Error("command does not include -filter_complex")
	}
}

func TestCommandBuilder_MultipleInputs(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "/tmp/video1.mp4"},
			{ID: "video2", Source: "/tmp/video2.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video1", Output: "trimmed1",
				Params: map[string]interface{}{"start": "00:00:00", "duration": "00:00:30"}},
			{Op: "trim", Input: "video2", Output: "trimmed2",
				Params: map[string]interface{}{"start": "00:00:00", "duration": "00:00:30"}},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed1", Destination: "/tmp/output1.mp4"},
			{ID: "trimmed2", Destination: "/tmp/output2.mp4"},
		},
	}

	p := planner.NewPlanner()
	plan, err := p.Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	builder := NewCommandBuilder(operators.GlobalRegistry())
	cmd, err := builder.Build(context.Background(), plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Count input files
	inputCount := 0
	for i, arg := range cmd.Args {
		if arg == "-i" && i+1 < len(cmd.Args) {
			inputCount++
		}
	}
	if inputCount != 2 {
		t.Errorf("expected 2 inputs, got %d", inputCount)
	}
}

func TestCommandBuilder_InvalidOperator(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "nonexistent", Input: "video", Output: "processed"},
		},
		Outputs: []schemas.Output{
			{ID: "processed", Destination: "/tmp/output.mp4"},
		},
	}

	p := planner.NewPlanner()
	plan, err := p.Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	builder := NewCommandBuilder(operators.GlobalRegistry())
	_, err = builder.Build(context.Background(), plan)

	// Should fail because operator doesn't exist
	if err == nil {
		t.Error("expected error for nonexistent operator, got nil")
	}
}
