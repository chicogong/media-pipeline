package planner

import (
	"context"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestResourceEstimator_EstimateSimple(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a simple JobSpec: video -> trim -> scale -> output
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
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

	// Build DAG
	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	// Set input metadata
	inputNode := graph.GetNode("input_video")
	inputNode.Metadata = &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 60 * time.Second,
			Size:     1024 * 1024 * 100, // 100 MB
		},
		VideoStreams: []schemas.VideoStream{
			{Index: 0, Width: 1920, Height: 1080, FrameRate: 30.0},
		},
		AudioStreams: []schemas.AudioStream{
			{Index: 1, SampleRate: 48000, Channels: 2},
		},
	}

	// Propagate metadata first
	propagator := NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Estimate resources
	estimator := NewResourceEstimator(operators.GlobalRegistry())
	estimates, err := estimator.Estimate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Estimate failed: %v", err)
	}

	// Verify estimates exist
	if estimates == nil {
		t.Fatal("estimates is nil")
	}

	// Verify total estimates
	if estimates.TotalDuration <= 0 {
		t.Errorf("expected positive total duration, got %v", estimates.TotalDuration)
	}

	if estimates.PeakMemoryMB <= 0 {
		t.Errorf("expected positive peak memory, got %v", estimates.PeakMemoryMB)
	}

	if estimates.TotalDiskMB <= 0 {
		t.Errorf("expected positive total disk, got %v", estimates.TotalDiskMB)
	}

	// Verify per-node estimates
	if len(estimates.NodeEstimates) == 0 {
		t.Error("expected per-node estimates, got none")
	}

	// Verify trim node has estimates
	var trimEstimate *schemas.NodeEstimates
	for nodeID, est := range estimates.NodeEstimates {
		if nodeID == "op_0_trim" {
			trimEstimate = est
			break
		}
	}
	if trimEstimate == nil {
		t.Error("trim node estimate not found")
	}
	if trimEstimate != nil && trimEstimate.Duration <= 0 {
		t.Errorf("expected positive trim duration, got %v", trimEstimate.Duration)
	}
}

func TestResourceEstimator_MissingMetadata(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
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
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	// Don't set metadata or propagate - try to estimate directly
	estimator := NewResourceEstimator(operators.GlobalRegistry())
	_, err = estimator.Estimate(context.Background(), graph)

	// Should fail because nodes don't have metadata
	if err == nil {
		t.Error("expected error for missing metadata, got nil")
	}
}

func TestResourceEstimator_ParallelOperations(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
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

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	// Set metadata for both inputs
	baseMetadata := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 60 * time.Second,
			Size:     1024 * 1024 * 50, // 50 MB
		},
		VideoStreams: []schemas.VideoStream{
			{Index: 0, Width: 1920, Height: 1080, FrameRate: 30.0},
		},
	}

	input1 := graph.GetNode("input_video1")
	input1.Metadata = baseMetadata

	input2 := graph.GetNode("input_video2")
	input2.Metadata = baseMetadata

	// Propagate metadata
	propagator := NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Estimate resources
	estimator := NewResourceEstimator(operators.GlobalRegistry())
	estimates, err := estimator.Estimate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Estimate failed: %v", err)
	}

	// For parallel operations, total duration should be less than
	// the sum of individual durations (because they can run concurrently)
	// But in this simple implementation, we might just sum them
	if estimates.TotalDuration <= 0 {
		t.Errorf("expected positive total duration, got %v", estimates.TotalDuration)
	}

	// Both trim operations should have estimates
	trim1Found := false
	trim2Found := false
	for nodeID := range estimates.NodeEstimates {
		if nodeID == "op_0_trim" {
			trim1Found = true
		}
		if nodeID == "op_1_trim" {
			trim2Found = true
		}
	}
	if !trim1Found {
		t.Error("trim1 estimate not found")
	}
	if !trim2Found {
		t.Error("trim2 estimate not found")
	}
}

func TestResourceEstimator_InvalidOperator(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "nonexistent",
				Input:  "video",
				Output: "processed",
				Params: map[string]interface{}{},
			},
		},
		Outputs: []schemas.Output{
			{ID: "processed", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		t.Fatalf("BuildDAG failed: %v", err)
	}

	// Set input metadata
	inputNode := graph.GetNode("input_video")
	inputNode.Metadata = &schemas.MediaInfo{
		Format: schemas.FormatInfo{Duration: 60 * time.Second},
	}

	// Try to estimate with invalid operator
	estimator := NewResourceEstimator(operators.GlobalRegistry())
	_, err = estimator.Estimate(context.Background(), graph)

	// Should fail because operator doesn't exist
	if err == nil {
		t.Error("expected error for nonexistent operator, got nil")
	}
}
