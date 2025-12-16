package planner

import (
	"testing"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestTopologicalSort_LinearGraph(t *testing.T) {
	graph := NewGraph()

	// Linear: A -> B -> C
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})

	order, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}

	// Verify order: A must come before B, B before C
	posA, posB, posC := -1, -1, -1
	for i, id := range order {
		switch id {
		case "A":
			posA = i
		case "B":
			posB = i
		case "C":
			posC = i
		}
	}

	if posA >= posB || posB >= posC {
		t.Errorf("invalid order: %v (expected A < B < C)", order)
	}
}

func TestTopologicalSort_DiamondGraph(t *testing.T) {
	graph := NewGraph()

	// Diamond: A -> B, A -> C, B -> D, C -> D
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})
	graph.AddNode(&schemas.PlanNode{ID: "D"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "C"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "D"})
	graph.AddEdge(&schemas.PlanEdge{From: "C", To: "D"})

	order, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(order))
	}

	// Verify constraints
	posA, posB, posC, posD := -1, -1, -1, -1
	for i, id := range order {
		switch id {
		case "A":
			posA = i
		case "B":
			posB = i
		case "C":
			posC = i
		case "D":
			posD = i
		}
	}

	// A must come before B and C
	if posA >= posB || posA >= posC {
		t.Errorf("A must come before B and C, got order: %v", order)
	}

	// B and C must come before D
	if posB >= posD || posC >= posD {
		t.Errorf("B and C must come before D, got order: %v", order)
	}
}

func TestTopologicalSort_WithCycle(t *testing.T) {
	graph := NewGraph()

	// Cycle: A -> B -> C -> A
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})
	graph.AddEdge(&schemas.PlanEdge{From: "C", To: "A"}) // Cycle!

	_, err := graph.TopologicalSort()
	if err == nil {
		t.Error("expected error for cyclic graph, got nil")
	}
}

func TestComputeExecutionStages_Linear(t *testing.T) {
	graph := NewGraph()

	// Linear: A -> B -> C
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})

	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(stages))
	}

	// Stage 0: A
	if len(stages[0]) != 1 || stages[0][0] != "A" {
		t.Errorf("expected stage 0 to be [A], got %v", stages[0])
	}

	// Stage 1: B
	if len(stages[1]) != 1 || stages[1][0] != "B" {
		t.Errorf("expected stage 1 to be [B], got %v", stages[1])
	}

	// Stage 2: C
	if len(stages[2]) != 1 || stages[2][0] != "C" {
		t.Errorf("expected stage 2 to be [C], got %v", stages[2])
	}
}

func TestComputeExecutionStages_Parallel(t *testing.T) {
	graph := NewGraph()

	// Parallel: A -> B, A -> C (B and C can run in parallel)
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "C"})

	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}

	// Stage 0: A
	if len(stages[0]) != 1 || stages[0][0] != "A" {
		t.Errorf("expected stage 0 to be [A], got %v", stages[0])
	}

	// Stage 1: B and C (parallel)
	if len(stages[1]) != 2 {
		t.Errorf("expected stage 1 to have 2 nodes, got %d", len(stages[1]))
	}

	// Verify both B and C are in stage 1
	foundB, foundC := false, false
	for _, id := range stages[1] {
		if id == "B" {
			foundB = true
		}
		if id == "C" {
			foundC = true
		}
	}

	if !foundB || !foundC {
		t.Errorf("expected both B and C in stage 1, got %v", stages[1])
	}
}

func TestComputeExecutionStages_Diamond(t *testing.T) {
	graph := NewGraph()

	// Diamond: A -> B, A -> C, B -> D, C -> D
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})
	graph.AddNode(&schemas.PlanNode{ID: "D"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "C"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "D"})
	graph.AddEdge(&schemas.PlanEdge{From: "C", To: "D"})

	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(stages))
	}

	// Stage 0: A
	// Stage 1: B, C (parallel)
	// Stage 2: D

	if len(stages[0]) != 1 || stages[0][0] != "A" {
		t.Errorf("expected stage 0 to be [A], got %v", stages[0])
	}

	if len(stages[1]) != 2 {
		t.Errorf("expected stage 1 to have 2 nodes, got %d", len(stages[1]))
	}

	if len(stages[2]) != 1 || stages[2][0] != "D" {
		t.Errorf("expected stage 2 to be [D], got %v", stages[2])
	}
}
