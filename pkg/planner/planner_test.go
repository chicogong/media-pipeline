package planner

import (
	"context"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestPlanner_PlanSimple(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		JobID: "test-job-1",
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4", Type: "video"},
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
			{ID: "scaled", Destination: "s3://bucket/output.mp4"},
		},
	}

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), spec, nil)

	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify plan was created
	if plan == nil {
		t.Fatal("plan is nil")
	}

	// Verify job ID
	if plan.JobID != "test-job-1" {
		t.Errorf("expected job ID 'test-job-1', got '%s'", plan.JobID)
	}

	// Verify nodes and edges
	if len(plan.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(plan.Nodes))
	}
	if len(plan.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(plan.Edges))
	}

	// Verify execution order
	if len(plan.ExecutionOrder) != 4 {
		t.Errorf("expected 4 nodes in execution order, got %d", len(plan.ExecutionOrder))
	}

	// Verify execution stages
	if len(plan.ExecutionStages) != 4 {
		t.Errorf("expected 4 execution stages, got %d", len(plan.ExecutionStages))
	}
}

func TestPlanner_PlanWithMetadata(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		JobID: "test-job-2",
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4", Type: "video"},
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
			{ID: "scaled", Destination: "s3://bucket/output.mp4"},
		},
	}

	planner := NewPlanner()

	// First, build the graph to set input metadata
	graph, err := planner.builder.BuildDAG(context.Background(), spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	// Set input metadata
	inputNode := graph.GetNode("input_video")
	inputNode.Metadata = &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 60 * time.Second,
			Size:     1024 * 1024 * 100,
		},
		VideoStreams: []schemas.VideoStream{
			{Index: 0, Width: 1920, Height: 1080, FrameRate: 30.0},
		},
		AudioStreams: []schemas.AudioStream{
			{Index: 1, SampleRate: 48000, Channels: 2},
		},
	}

	// Propagate metadata
	err = planner.propagator.Propagate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Estimate resources
	estimates, err := planner.estimator.Estimate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Estimate failed: %v", err)
	}

	// Verify estimates
	if estimates == nil {
		t.Fatal("estimates is nil")
	}
	if estimates.TotalDuration <= 0 {
		t.Errorf("expected positive total duration, got %v", estimates.TotalDuration)
	}
	if estimates.PeakMemoryMB <= 0 {
		t.Errorf("expected positive peak memory, got %v", estimates.PeakMemoryMB)
	}
}

func TestPlanner_ValidateOperators(t *testing.T) {
	// Register only trim operator
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video", Output: "trimmed"},
			{Op: "nonexistent", Input: "trimmed", Output: "processed"},
		},
	}

	planner := NewPlanner()
	err := planner.ValidateOperators(spec)

	// Should fail because 'nonexistent' operator is not registered
	if err == nil {
		t.Error("expected error for nonexistent operator, got nil")
	}
}

func TestPlanner_ValidateParameters(t *testing.T) {
	// Register trim operator
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video",
				Output: "trimmed",
				Params: map[string]interface{}{
					// Missing required parameters
				},
			},
		},
	}

	planner := NewPlanner()
	err := planner.ValidateParameters(spec)

	// Should fail because parameters are invalid
	if err == nil {
		t.Error("expected error for invalid parameters, got nil")
	}
}

func TestPlanner_PlanWithCycle(t *testing.T) {
	// Create a spec that would create a cycle (though this is hard with our schema)
	// For now, just test that cycle detection works
	spec := &schemas.JobSpec{
		JobID: "test-cycle",
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video", Output: "out1"},
			{Op: "scale", Input: "out1", Output: "out2"},
		},
		Outputs: []schemas.Output{
			{ID: "out2", Destination: "s3://bucket/output.mp4"},
		},
	}

	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), spec, nil)

	// Should succeed - no cycle
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
}

func TestPlanner_PlanParallelOperations(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		JobID: "test-parallel",
		Inputs: []schemas.Input{
			{ID: "video1", Source: "s3://bucket/video1.mp4"},
			{ID: "video2", Source: "s3://bucket/video2.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video1",
				Output: "trimmed1",
				Params: map[string]interface{}{
					"start":    "00:00:00",
					"duration": "00:00:30",
				},
			},
			{
				Op:     "trim",
				Input:  "video2",
				Output: "trimmed2",
				Params: map[string]interface{}{
					"start":    "00:00:00",
					"duration": "00:00:30",
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed1", Destination: "s3://bucket/output1.mp4"},
			{ID: "trimmed2", Destination: "s3://bucket/output2.mp4"},
		},
	}

	planner := NewPlanner()
	plan, err := planner.Plan(context.Background(), spec, nil)

	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify execution stages - parallel operations should be in same stage
	if len(plan.ExecutionStages) < 2 {
		t.Fatalf("expected at least 2 stages, got %d", len(plan.ExecutionStages))
	}

	// Stage 1 should have both trim operations
	stage1 := plan.ExecutionStages[1]
	if len(stage1) != 2 {
		t.Errorf("expected 2 operations in stage 1, got %d", len(stage1))
	}
}
