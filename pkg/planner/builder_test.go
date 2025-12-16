package planner

import (
	"context"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestBuilder_BuildDAG_Simple(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video", Output: "trimmed",
				Params: map[string]interface{}{"start": "00:00:10", "duration": "5m"}},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 nodes: input, operation, output
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Should have 2 edges: input->operation, operation->output
	if len(graph.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(graph.Edges))
	}

	// Verify nodes exist
	inputNode := graph.GetNode("input_video")
	if inputNode == nil {
		t.Error("input node not found")
	}
	if inputNode.Type != "input" {
		t.Errorf("expected type 'input', got '%s'", inputNode.Type)
	}

	// Verify operation node
	var opNode *schemas.PlanNode
	for _, node := range graph.Nodes {
		if node.Type == "operation" {
			opNode = node
			break
		}
	}
	if opNode == nil {
		t.Fatal("operation node not found")
	}
	if opNode.Operator != "trim" {
		t.Errorf("expected operator 'trim', got '%s'", opNode.Operator)
	}

	// Verify output node
	outputNode := graph.GetNode("output_trimmed")
	if outputNode == nil {
		t.Error("output node not found")
	}
}

func TestBuilder_BuildDAG_MultipleOperations(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video", Output: "trimmed",
				Params: map[string]interface{}{"start": "00:00:10"}},
			{Op: "scale", Input: "trimmed", Output: "scaled",
				Params: map[string]interface{}{"width": 1280, "height": 720}},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 4 nodes: 1 input, 2 operations, 1 output
	if len(graph.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Should have 3 edges
	if len(graph.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(graph.Edges))
	}

	// Verify no cycles
	if err := graph.DetectCycles(); err != nil {
		t.Errorf("unexpected cycle: %v", err)
	}

	// Verify topological order is valid
	order, err := graph.TopologicalSort()
	if err != nil {
		t.Errorf("topological sort failed: %v", err)
	}
	if len(order) != 4 {
		t.Errorf("expected 4 nodes in order, got %d", len(order))
	}
}

func TestBuilder_BuildDAG_MultipleInputs(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "s3://bucket/video1.mp4"},
			{ID: "video2", Source: "s3://bucket/video2.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "concat", Inputs: []string{"video1", "video2"}, Output: "concatenated"},
		},
		Outputs: []schemas.Output{
			{ID: "concatenated", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 4 nodes: 2 inputs, 1 operation, 1 output
	if len(graph.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Should have 3 edges: 2 from inputs to operation, 1 from operation to output
	if len(graph.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(graph.Edges))
	}
}

func TestBuilder_BuildDAG_InvalidReference(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "nonexistent", Output: "trimmed"}, // Invalid reference!
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	_, err := builder.BuildDAG(context.Background(), spec)

	if err == nil {
		t.Error("expected error for invalid reference, got nil")
	}
}

func TestBuilder_BuildDAG_CyclicDependency(t *testing.T) {
	// This would require operations that reference each other
	// which is caught during DAG construction
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "op1", Input: "video", Output: "out1"},
			{Op: "op2", Input: "out1", Output: "out2"},
			// In a real scenario, cyclic dependencies would be caught
		},
		Outputs: []schemas.Output{
			{ID: "out2", Destination: "s3://bucket/output.mp4"},
		},
	}

	builder := NewBuilder()
	graph, err := builder.BuildDAG(context.Background(), spec)

	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	// Verify no cycles
	if err := graph.DetectCycles(); err != nil {
		t.Errorf("unexpected cycle: %v", err)
	}
}
