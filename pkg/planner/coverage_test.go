package planner

import (
	"context"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// ---------------------------------------------------------------------------
// graph.go – GetOutputNodes
// ---------------------------------------------------------------------------

func TestGraph_GetOutputNodes_HasOutputs(t *testing.T) {
	graph := NewGraph()

	graph.AddNode(&schemas.PlanNode{ID: "in1", Type: "input"})
	graph.AddNode(&schemas.PlanNode{ID: "op1", Type: "operation"})
	graph.AddNode(&schemas.PlanNode{ID: "out1", Type: "output"})
	graph.AddNode(&schemas.PlanNode{ID: "out2", Type: "output"})

	outputs := graph.GetOutputNodes()
	if len(outputs) != 2 {
		t.Fatalf("expected 2 output nodes, got %d", len(outputs))
	}

	ids := make(map[string]bool)
	for _, n := range outputs {
		ids[n.ID] = true
	}
	if !ids["out1"] || !ids["out2"] {
		t.Errorf("expected out1 and out2 in output nodes, got %v", ids)
	}

	// Non-output nodes must not appear.
	for _, n := range outputs {
		if n.Type != "output" {
			t.Errorf("unexpected node type %q in GetOutputNodes result", n.Type)
		}
	}
}

func TestGraph_GetOutputNodes_Empty(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&schemas.PlanNode{ID: "in1", Type: "input"})
	graph.AddNode(&schemas.PlanNode{ID: "op1", Type: "operation"})

	outputs := graph.GetOutputNodes()
	if len(outputs) != 0 {
		t.Fatalf("expected 0 output nodes, got %d", len(outputs))
	}
}

func TestGraph_GetOutputNodes_EmptyGraph(t *testing.T) {
	graph := NewGraph()
	outputs := graph.GetOutputNodes()
	if len(outputs) != 0 {
		t.Fatalf("expected 0 output nodes on empty graph, got %d", len(outputs))
	}
}

// ---------------------------------------------------------------------------
// planner.go – NewPlannerWithRegistry
// ---------------------------------------------------------------------------

func TestNewPlannerWithRegistry_NonNil(t *testing.T) {
	// GlobalRegistry() always returns a valid, non-nil registry.
	reg := operators.GlobalRegistry()
	p := NewPlannerWithRegistry(reg)
	if p == nil {
		t.Fatal("NewPlannerWithRegistry returned nil")
	}
}

func TestNewPlannerWithRegistry_CanPlan(t *testing.T) {
	// Register operators into the global registry (safe: re-registration is a no-op).
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	reg := operators.GlobalRegistry()
	p := NewPlannerWithRegistry(reg)
	if p == nil {
		t.Fatal("NewPlannerWithRegistry returned nil")
	}

	spec := &schemas.JobSpec{
		JobID: "registry-test-job",
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4", Type: "video"},
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
			{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
		},
	}

	plan, err := p.Plan(context.Background(), spec, &PlanOptions{
		SkipMetadataValidation: true,
		SkipResourceEstimation: true,
	})
	if err != nil {
		t.Fatalf("Plan with custom registry failed: %v", err)
	}
	if plan == nil {
		t.Fatal("plan is nil")
	}
	if plan.JobID != "registry-test-job" {
		t.Errorf("expected job ID 'registry-test-job', got %q", plan.JobID)
	}
}

// ---------------------------------------------------------------------------
// planner.go – Plan error/edge branches
// ---------------------------------------------------------------------------

// Plan should fail when BuildDAG fails (unresolvable input reference).
func TestPlan_BuildDAGError(t *testing.T) {
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		JobID: "bad-ref",
		Inputs: []schemas.Input{
			{ID: "video", Source: "s3://bucket/input.mp4"},
		},
		Operations: []schemas.Operation{
			// "does_not_exist" is not a declared input → BuildDAG returns an error.
			{Op: "trim", Input: "does_not_exist", Output: "trimmed"},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
		},
	}

	p := NewPlanner()
	_, err := p.Plan(context.Background(), spec, &PlanOptions{
		SkipMetadataValidation: true,
		SkipResourceEstimation: true,
	})
	if err == nil {
		t.Fatal("expected error for bad input reference, got nil")
	}
}

// TopologicalSort on a cyclic graph should return an error (covers the
// "failed to compute execution order" branch in Plan and the TopologicalSort
// error path in sort.go).
func TestGraph_TopologicalSort_CycleError(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	// Manually introduce a cycle, bypassing BuildDAG's own cycle guard.
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "A"})

	_, err := graph.TopologicalSort()
	if err == nil {
		t.Fatal("expected TopologicalSort to return an error for a cyclic graph")
	}
}

// ComputeExecutionStages on a cyclic graph should return an error.
func TestGraph_ComputeExecutionStages_CycleError(t *testing.T) {
	graph := NewGraph()
	graph.AddNode(&schemas.PlanNode{ID: "X"})
	graph.AddNode(&schemas.PlanNode{ID: "Y"})
	graph.AddEdge(&schemas.PlanEdge{From: "X", To: "Y"})
	graph.AddEdge(&schemas.PlanEdge{From: "Y", To: "X"})

	_, err := graph.ComputeExecutionStages()
	if err == nil {
		t.Fatal("expected ComputeExecutionStages to return an error for a cyclic graph")
	}
}

// ValidateOperators: operator not registered → error returned.
func TestPlannerWithRegistry_ValidateOperators_Missing(t *testing.T) {
	// Use the global registry; "definitely_not_registered_op" will never exist.
	reg := operators.GlobalRegistry()
	p := NewPlannerWithRegistry(reg)

	spec := &schemas.JobSpec{
		Operations: []schemas.Operation{
			{Op: "definitely_not_registered_op", Input: "x", Output: "y"},
		},
	}

	err := p.ValidateOperators(spec)
	if err == nil {
		t.Fatal("expected ValidateOperators to fail for unregistered operator")
	}
}

// ValidateParameters: operator not found → error on the operator-lookup branch.
func TestPlannerWithRegistry_ValidateParameters_OperatorMissing(t *testing.T) {
	reg := operators.GlobalRegistry()
	p := NewPlannerWithRegistry(reg)

	spec := &schemas.JobSpec{
		Operations: []schemas.Operation{
			{Op: "another_unregistered_op_xyz", Input: "x", Output: "y"},
		},
	}

	err := p.ValidateParameters(spec)
	if err == nil {
		t.Fatal("expected error when operator not found in ValidateParameters")
	}
}
