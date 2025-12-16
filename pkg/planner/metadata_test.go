package planner

import (
	"context"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestMetadataPropagator_PropagateSimple(t *testing.T) {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a simple JobSpec: video -> trim -> scale -> output
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{
				ID:     "video",
				Source: "s3://bucket/input.mp4",
				Type:   "video",
			},
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
	if inputNode == nil {
		t.Fatal("input node not found")
	}
	inputNode.Metadata = &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Filename:  "input.mp4",
			Format:    "mp4",
			Duration:  60 * time.Second,
			Size:      1024 * 1024 * 100, // 100 MB
			BitRate:   5000000,            // 5 Mbps
			StartTime: 0,
		},
		VideoStreams: []schemas.VideoStream{
			{
				Index:     0,
				Codec:     "h264",
				Width:     1920,
				Height:    1080,
				FrameRate: 30.0,
				BitRate:   4000000,
				Duration:  60 * time.Second,
			},
		},
		AudioStreams: []schemas.AudioStream{
			{
				Index:      1,
				Codec:      "aac",
				SampleRate: 48000,
				Channels:   2,
				BitRate:    128000,
				Duration:   60 * time.Second,
			},
		},
	}

	// Propagate metadata
	propagator := NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)
	if err != nil {
		t.Fatalf("Propagate failed: %v", err)
	}

	// Verify trim operation metadata
	trimNode := graph.GetNode("op_0_trim")
	if trimNode == nil {
		t.Fatal("trim node not found")
	}
	if trimNode.Metadata == nil {
		t.Fatal("trim node metadata is nil")
	}
	// Duration should be 30 seconds after trim
	if trimNode.Metadata.Format.Duration != 30*time.Second {
		t.Errorf("expected duration 30s, got %v", trimNode.Metadata.Format.Duration)
	}
	// Resolution should remain unchanged
	if len(trimNode.Metadata.VideoStreams) != 1 {
		t.Fatalf("expected 1 video stream, got %d", len(trimNode.Metadata.VideoStreams))
	}
	if trimNode.Metadata.VideoStreams[0].Width != 1920 {
		t.Errorf("expected width 1920, got %d", trimNode.Metadata.VideoStreams[0].Width)
	}

	// Verify scale operation metadata
	scaleNode := graph.GetNode("op_1_scale")
	if scaleNode == nil {
		t.Fatal("scale node not found")
	}
	if scaleNode.Metadata == nil {
		t.Fatal("scale node metadata is nil")
	}
	// Resolution should be 1280x720 after scale
	if len(scaleNode.Metadata.VideoStreams) != 1 {
		t.Fatalf("expected 1 video stream, got %d", len(scaleNode.Metadata.VideoStreams))
	}
	if scaleNode.Metadata.VideoStreams[0].Width != 1280 {
		t.Errorf("expected width 1280, got %d", scaleNode.Metadata.VideoStreams[0].Width)
	}
	if scaleNode.Metadata.VideoStreams[0].Height != 720 {
		t.Errorf("expected height 720, got %d", scaleNode.Metadata.VideoStreams[0].Height)
	}
	// Duration should remain 30 seconds
	if scaleNode.Metadata.Format.Duration != 30*time.Second {
		t.Errorf("expected duration 30s, got %v", scaleNode.Metadata.Format.Duration)
	}

	// Verify output node has metadata
	outputNode := graph.GetNode("output_scaled")
	if outputNode == nil {
		t.Fatal("output node not found")
	}
	if outputNode.Metadata == nil {
		t.Fatal("output node metadata is nil")
	}
	if outputNode.Metadata.VideoStreams[0].Width != 1280 {
		t.Errorf("expected output width 1280, got %d", outputNode.Metadata.VideoStreams[0].Width)
	}
}

func TestMetadataPropagator_MissingInputMetadata(t *testing.T) {
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

	// Don't set input metadata
	propagator := NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)

	// Should fail because input has no metadata
	if err == nil {
		t.Error("expected error for missing input metadata, got nil")
	}
}

func TestMetadataPropagator_MultipleInputs(t *testing.T) {
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
		},
		VideoStreams: []schemas.VideoStream{
			{
				Index:  0,
				Codec:  "h264",
				Width:  1920,
				Height: 1080,
			},
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

	// Both operation nodes should have metadata
	trim1 := graph.GetNode("op_0_trim")
	if trim1.Metadata == nil {
		t.Error("trim1 metadata is nil")
	}

	trim2 := graph.GetNode("op_1_trim")
	if trim2.Metadata == nil {
		t.Error("trim2 metadata is nil")
	}
}

func TestMetadataPropagator_InvalidOperator(t *testing.T) {
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
		VideoStreams: []schemas.VideoStream{
			{Index: 0, Width: 1920, Height: 1080},
		},
	}

	// Propagate should fail because operator doesn't exist
	propagator := NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)

	if err == nil {
		t.Error("expected error for nonexistent operator, got nil")
	}
}
