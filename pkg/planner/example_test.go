package planner_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Example demonstrates how to use the planner to build a DAG from a JobSpec
func Example() {
	// Create a JobSpec
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{
				ID:     "main_video",
				Source: "s3://bucket/input.mp4",
				Type:   "video",
			},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "main_video",
				Output: "trimmed",
				Params: map[string]interface{}{
					"start":    "00:00:10",
					"duration": "00:05:00",
				},
			},
			{
				Op:     "scale",
				Input:  "trimmed",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":     1280,
					"height":    720,
					"algorithm": "lanczos",
				},
			},
		},
		Outputs: []schemas.Output{
			{
				ID:          "scaled",
				Destination: "s3://bucket/output.mp4",
			},
		},
	}

	// Build DAG
	builder := planner.NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		fmt.Printf("Error building DAG: %v\n", err)
		return
	}

	fmt.Printf("Graph has %d nodes and %d edges\n", len(graph.Nodes), len(graph.Edges))

	// Check for cycles
	if err := graph.DetectCycles(); err != nil {
		fmt.Printf("Cycle detected: %v\n", err)
		return
	}
	fmt.Println("No cycles detected")

	// Compute topological order
	order, err := graph.TopologicalSort()
	if err != nil {
		fmt.Printf("Topological sort failed: %v\n", err)
		return
	}
	fmt.Printf("Topological order: %v\n", order)

	// Compute execution stages (for parallelization)
	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		fmt.Printf("Failed to compute stages: %v\n", err)
		return
	}

	fmt.Printf("Execution stages:\n")
	for i, stage := range stages {
		fmt.Printf("  Stage %d: %v\n", i, stage)
	}

	// Output:
	// Graph has 4 nodes and 3 edges
	// No cycles detected
	// Topological order: [input_main_video op_0_trim op_1_scale output_scaled]
	// Execution stages:
	//   Stage 0: [input_main_video]
	//   Stage 1: [op_0_trim]
	//   Stage 2: [op_1_scale]
	//   Stage 3: [output_scaled]
}

// Example_complexGraph demonstrates a more complex graph with multiple inputs
func Example_complexGraph() {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "s3://bucket/video1.mp4"},
			{ID: "video2", Source: "s3://bucket/video2.mp4"},
			{ID: "audio", Source: "s3://bucket/audio.mp3"},
		},
		Operations: []schemas.Operation{
			// Trim both videos
			{Op: "trim", Input: "video1", Output: "trimmed1",
				Params: map[string]interface{}{"start": "00:00:00", "duration": "00:05:00"}},
			{Op: "trim", Input: "video2", Output: "trimmed2",
				Params: map[string]interface{}{"start": "00:00:10", "duration": "00:05:00"}},

			// Concatenate videos
			{Op: "concat", Inputs: []string{"trimmed1", "trimmed2"}, Output: "concatenated"},

			// Mix audio
			{Op: "mix", Inputs: []string{"concatenated", "audio"}, Output: "final"},
		},
		Outputs: []schemas.Output{
			{ID: "final", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := planner.NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	stages, _ := graph.ComputeExecutionStages()
	fmt.Printf("Complex graph has %d nodes\n", len(graph.Nodes))
	fmt.Printf("Execution in %d stages\n", len(stages))
	fmt.Printf("Stage 1 (parallel): %v\n", stages[1])

	// Output:
	// Complex graph has 7 nodes
	// Execution in 5 stages
	// Stage 1 (parallel): [op_0_trim op_1_trim]
}

// Example_metadataPropagation demonstrates metadata propagation through the graph
func Example_metadataPropagation() {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a JobSpec: video -> trim -> scale -> output
	spec := &schemas.JobSpec{
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

	// Build DAG
	builder := planner.NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Set input metadata (this would normally come from MediaInfo)
	inputNode := graph.GetNode("input_video")
	inputNode.Metadata = &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Filename: "input.mp4",
			Duration: 60 * time.Second,
		},
		VideoStreams: []schemas.VideoStream{
			{
				Index:     0,
				Codec:     "h264",
				Width:     1920,
				Height:    1080,
				FrameRate: 30.0,
			},
		},
		AudioStreams: []schemas.AudioStream{
			{
				Index:      1,
				Codec:      "aac",
				SampleRate: 48000,
				Channels:   2,
			},
		},
	}

	// Propagate metadata through the graph
	propagator := planner.NewMetadataPropagator(operators.GlobalRegistry())
	err = propagator.Propagate(context.Background(), graph)
	if err != nil {
		fmt.Printf("Propagation failed: %v\n", err)
		return
	}

	// Check output metadata
	outputNode := graph.GetNode("output_scaled")
	metadata := outputNode.Metadata

	fmt.Printf("Input: %dx%d, %s\n",
		inputNode.Metadata.VideoStreams[0].Width,
		inputNode.Metadata.VideoStreams[0].Height,
		inputNode.Metadata.Format.Duration)

	trimNode := graph.GetNode("op_0_trim")
	fmt.Printf("After trim: duration=%s\n", trimNode.Metadata.Format.Duration)

	fmt.Printf("Output: %dx%d, %s\n",
		metadata.VideoStreams[0].Width,
		metadata.VideoStreams[0].Height,
		metadata.Format.Duration)

	// Output:
	// Input: 1920x1080, 1m0s
	// After trim: duration=30s
	// Output: 1280x720, 30s
}

// Example_planner demonstrates the complete integrated planner
func Example_planner() {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a JobSpec
	spec := &schemas.JobSpec{
		JobID: "example-job",
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

	// Create planner
	planner := planner.NewPlanner()

	// Validate operators exist
	if err := planner.ValidateOperators(spec); err != nil {
		fmt.Printf("Operator validation failed: %v\n", err)
		return
	}

	// Validate parameters
	if err := planner.ValidateParameters(spec); err != nil {
		fmt.Printf("Parameter validation failed: %v\n", err)
		return
	}

	// Generate plan (without metadata/estimation for now)
	plan, err := planner.Plan(context.Background(), spec, nil)
	if err != nil {
		fmt.Printf("Planning failed: %v\n", err)
		return
	}

	fmt.Printf("Job ID: %s\n", plan.JobID)
	fmt.Printf("Nodes: %d\n", len(plan.Nodes))
	fmt.Printf("Edges: %d\n", len(plan.Edges))
	fmt.Printf("Execution stages: %d\n", len(plan.ExecutionStages))
	fmt.Printf("Stage 0: %v\n", plan.ExecutionStages[0])
	fmt.Printf("Stage 1: %v\n", plan.ExecutionStages[1])

	// Output:
	// Job ID: example-job
	// Nodes: 4
	// Edges: 3
	// Execution stages: 4
	// Stage 0: [input_video]
	// Stage 1: [op_0_trim]
}
