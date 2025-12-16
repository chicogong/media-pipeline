package planner

import (
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Graph represents a directed acyclic graph (DAG) of processing nodes
type Graph struct {
	Nodes []*schemas.PlanNode
	Edges []*schemas.PlanEdge

	// Internal indexes for fast lookup
	nodeIndex map[string]*schemas.PlanNode
	outgoing  map[string][]*schemas.PlanEdge
	incoming  map[string][]*schemas.PlanEdge
}

// NewGraph creates a new empty graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:     []*schemas.PlanNode{},
		Edges:     []*schemas.PlanEdge{},
		nodeIndex: make(map[string]*schemas.PlanNode),
		outgoing:  make(map[string][]*schemas.PlanEdge),
		incoming:  make(map[string][]*schemas.PlanEdge),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *schemas.PlanNode) {
	g.Nodes = append(g.Nodes, node)
	g.nodeIndex[node.ID] = node
}

// AddEdge adds an edge to the graph
func (g *Graph) AddEdge(edge *schemas.PlanEdge) {
	g.Edges = append(g.Edges, edge)

	// Update outgoing edges
	g.outgoing[edge.From] = append(g.outgoing[edge.From], edge)

	// Update incoming edges
	g.incoming[edge.To] = append(g.incoming[edge.To], edge)
}

// GetNode retrieves a node by ID
func (g *Graph) GetNode(id string) *schemas.PlanNode {
	return g.nodeIndex[id]
}

// GetOutgoingEdges returns all edges from a node
func (g *Graph) GetOutgoingEdges(nodeID string) []*schemas.PlanEdge {
	return g.outgoing[nodeID]
}

// GetIncomingEdges returns all edges to a node
func (g *Graph) GetIncomingEdges(nodeID string) []*schemas.PlanEdge {
	return g.incoming[nodeID]
}

// GetPredecessors returns all predecessor nodes
func (g *Graph) GetPredecessors(nodeID string) []*schemas.PlanNode {
	incoming := g.GetIncomingEdges(nodeID)
	predecessors := make([]*schemas.PlanNode, 0, len(incoming))

	for _, edge := range incoming {
		if node := g.GetNode(edge.From); node != nil {
			predecessors = append(predecessors, node)
		}
	}

	return predecessors
}

// GetSuccessors returns all successor nodes
func (g *Graph) GetSuccessors(nodeID string) []*schemas.PlanNode {
	outgoing := g.GetOutgoingEdges(nodeID)
	successors := make([]*schemas.PlanNode, 0, len(outgoing))

	for _, edge := range outgoing {
		if node := g.GetNode(edge.To); node != nil {
			successors = append(successors, node)
		}
	}

	return successors
}

// DetectCycles checks if the graph contains any cycles using DFS
func (g *Graph) DetectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			if err := g.dfsCheckCycle(node.ID, visited, recStack); err != nil {
				return err
			}
		}
	}

	return nil
}

// dfsCheckCycle performs DFS to detect cycles
func (g *Graph) dfsCheckCycle(nodeID string, visited, recStack map[string]bool) error {
	visited[nodeID] = true
	recStack[nodeID] = true

	// Visit all successors
	for _, edge := range g.GetOutgoingEdges(nodeID) {
		successor := edge.To

		if !visited[successor] {
			// Recurse
			if err := g.dfsCheckCycle(successor, visited, recStack); err != nil {
				return err
			}
		} else if recStack[successor] {
			// Back edge found - cycle detected
			return fmt.Errorf("cycle detected: %s -> %s", nodeID, successor)
		}
	}

	recStack[nodeID] = false
	return nil
}

// GetInputNodes returns all input nodes
func (g *Graph) GetInputNodes() []*schemas.PlanNode {
	inputs := []*schemas.PlanNode{}
	for _, node := range g.Nodes {
		if node.Type == "input" {
			inputs = append(inputs, node)
		}
	}
	return inputs
}

// GetOutputNodes returns all output nodes
func (g *Graph) GetOutputNodes() []*schemas.PlanNode {
	outputs := []*schemas.PlanNode{}
	for _, node := range g.Nodes {
		if node.Type == "output" {
			outputs = append(outputs, node)
		}
	}
	return outputs
}
