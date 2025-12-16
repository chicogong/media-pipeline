package planner

import (
	"testing"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestGraph_AddNode(t *testing.T) {
	graph := NewGraph()

	node := &schemas.PlanNode{
		ID:   "node1",
		Type: "input",
	}

	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(graph.Nodes))
	}

	retrieved := graph.GetNode("node1")
	if retrieved == nil {
		t.Error("node not found")
	}
	if retrieved.ID != "node1" {
		t.Errorf("expected node1, got %s", retrieved.ID)
	}
}

func TestGraph_AddEdge(t *testing.T) {
	graph := NewGraph()

	node1 := &schemas.PlanNode{ID: "node1", Type: "input"}
	node2 := &schemas.PlanNode{ID: "node2", Type: "operation"}

	graph.AddNode(node1)
	graph.AddNode(node2)

	edge := &schemas.PlanEdge{
		From:       "node1",
		To:         "node2",
		StreamType: "video",
	}

	graph.AddEdge(edge)

	if len(graph.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(graph.Edges))
	}

	outgoing := graph.GetOutgoingEdges("node1")
	if len(outgoing) != 1 {
		t.Errorf("expected 1 outgoing edge, got %d", len(outgoing))
	}

	incoming := graph.GetIncomingEdges("node2")
	if len(incoming) != 1 {
		t.Errorf("expected 1 incoming edge, got %d", len(incoming))
	}
}

func TestGraph_DetectCycles_NoCycle(t *testing.T) {
	graph := NewGraph()

	// Linear graph: A -> B -> C
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})

	err := graph.DetectCycles()
	if err != nil {
		t.Errorf("expected no cycle, got error: %v", err)
	}
}

func TestGraph_DetectCycles_SimpleCycle(t *testing.T) {
	graph := NewGraph()

	// Cycle: A -> B -> A
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "A"})

	err := graph.DetectCycles()
	if err == nil {
		t.Error("expected cycle error, got nil")
	}
}

func TestGraph_DetectCycles_ComplexCycle(t *testing.T) {
	graph := NewGraph()

	// Complex cycle: A -> B -> C -> D -> B
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})
	graph.AddNode(&schemas.PlanNode{ID: "D"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})
	graph.AddEdge(&schemas.PlanEdge{From: "C", To: "D"})
	graph.AddEdge(&schemas.PlanEdge{From: "D", To: "B"}) // Cycle!

	err := graph.DetectCycles()
	if err == nil {
		t.Error("expected cycle error, got nil")
	}
}

func TestGraph_DetectCycles_SelfLoop(t *testing.T) {
	graph := NewGraph()

	// Self-loop: A -> A
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "A"})

	err := graph.DetectCycles()
	if err == nil {
		t.Error("expected cycle error for self-loop, got nil")
	}
}

func TestGraph_GetPredecessors(t *testing.T) {
	graph := NewGraph()

	// A -> C, B -> C
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "C"})
	graph.AddEdge(&schemas.PlanEdge{From: "B", To: "C"})

	predecessors := graph.GetPredecessors("C")
	if len(predecessors) != 2 {
		t.Errorf("expected 2 predecessors, got %d", len(predecessors))
	}

	// Check both A and B are predecessors
	foundA, foundB := false, false
	for _, pred := range predecessors {
		if pred.ID == "A" {
			foundA = true
		}
		if pred.ID == "B" {
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Error("expected both A and B as predecessors")
	}
}

func TestGraph_GetSuccessors(t *testing.T) {
	graph := NewGraph()

	// A -> B, A -> C
	graph.AddNode(&schemas.PlanNode{ID: "A"})
	graph.AddNode(&schemas.PlanNode{ID: "B"})
	graph.AddNode(&schemas.PlanNode{ID: "C"})

	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "B"})
	graph.AddEdge(&schemas.PlanEdge{From: "A", To: "C"})

	successors := graph.GetSuccessors("A")
	if len(successors) != 2 {
		t.Errorf("expected 2 successors, got %d", len(successors))
	}
}
