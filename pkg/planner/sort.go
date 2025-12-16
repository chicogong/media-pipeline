package planner

import "fmt"

// TopologicalSort performs topological sort using Kahn's algorithm
// Returns a list of node IDs in topological order
func (g *Graph) TopologicalSort() ([]string, error) {
	// Count incoming edges for each node
	inDegree := make(map[string]int)
	for _, node := range g.Nodes {
		inDegree[node.ID] = len(g.GetIncomingEdges(node.ID))
	}

	// Queue of nodes with no incoming edges
	queue := []string{}
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	// Process nodes
	result := []string{}
	for len(queue) > 0 {
		// Dequeue
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		// Reduce in-degree of successors
		for _, edge := range g.GetOutgoingEdges(nodeID) {
			successor := edge.To
			inDegree[successor]--

			if inDegree[successor] == 0 {
				queue = append(queue, successor)
			}
		}
	}

	// Check if all nodes were processed
	if len(result) != len(g.Nodes) {
		return nil, fmt.Errorf("graph contains cycle (processed %d/%d nodes)", len(result), len(g.Nodes))
	}

	return result, nil
}

// ComputeExecutionStages groups nodes into stages for parallel execution
// Nodes in the same stage have no dependencies on each other
func (g *Graph) ComputeExecutionStages() ([][]string, error) {
	// Count incoming edges for each node
	inDegree := make(map[string]int)
	for _, node := range g.Nodes {
		inDegree[node.ID] = len(g.GetIncomingEdges(node.ID))
	}

	stages := [][]string{}
	processed := make(map[string]bool)

	for len(processed) < len(g.Nodes) {
		stage := []string{}

		// Find all nodes with in-degree 0 (no unprocessed dependencies)
		for _, node := range g.Nodes {
			if !processed[node.ID] && inDegree[node.ID] == 0 {
				stage = append(stage, node.ID)
			}
		}

		if len(stage) == 0 {
			return nil, fmt.Errorf("cannot compute stages (possible cycle)")
		}

		stages = append(stages, stage)

		// Mark as processed and update in-degrees
		for _, nodeID := range stage {
			processed[nodeID] = true

			// Reduce in-degree of successors
			for _, edge := range g.GetOutgoingEdges(nodeID) {
				inDegree[edge.To]--
			}
		}
	}

	return stages, nil
}
