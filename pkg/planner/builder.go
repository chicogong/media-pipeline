package planner

import (
	"context"
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Builder builds a processing plan from a JobSpec
type Builder struct {
	outputMap map[string]string // Maps output ID to node ID
}

// NewBuilder creates a new plan builder
func NewBuilder() *Builder {
	return &Builder{
		outputMap: make(map[string]string),
	}
}

// BuildDAG builds a directed acyclic graph from a JobSpec
func (b *Builder) BuildDAG(ctx context.Context, spec *schemas.JobSpec) (*Graph, error) {
	graph := NewGraph()
	b.outputMap = make(map[string]string) // Reset

	// Step 1: Create input nodes
	for _, input := range spec.Inputs {
		node := &schemas.PlanNode{
			ID:        "input_" + input.ID,
			Type:      "input",
			InputID:   input.ID,
			SourceURI: input.Source,
		}
		graph.AddNode(node)

		// Map input ID to node ID for reference resolution
		b.outputMap[input.ID] = node.ID
	}

	// Step 2: Create operation nodes and edges
	for i, op := range spec.Operations {
		nodeID := fmt.Sprintf("op_%d_%s", i, op.Op)
		node := &schemas.PlanNode{
			ID:       nodeID,
			Type:     "operation",
			Operator: op.Op,
			Params:   op.Params,
		}
		graph.AddNode(node)

		// Map output ID to node ID
		b.outputMap[op.Output] = nodeID

		// Create edges from inputs
		if op.Input != "" {
			// Single input
			sourceID, err := b.resolveReference(op.Input)
			if err != nil {
				return nil, fmt.Errorf("operation %d (%s): %w", i, op.Op, err)
			}

			edge := &schemas.PlanEdge{
				From:       sourceID,
				To:         nodeID,
				StreamType: "both", // Default to both video and audio
			}
			graph.AddEdge(edge)
		}

		if len(op.Inputs) > 0 {
			// Multiple inputs
			for _, inputRef := range op.Inputs {
				sourceID, err := b.resolveReference(inputRef)
				if err != nil {
					return nil, fmt.Errorf("operation %d (%s): %w", i, op.Op, err)
				}

				edge := &schemas.PlanEdge{
					From:       sourceID,
					To:         nodeID,
					StreamType: "both",
				}
				graph.AddEdge(edge)
			}
		}
	}

	// Step 3: Create output nodes and edges
	for _, output := range spec.Outputs {
		node := &schemas.PlanNode{
			ID:       "output_" + output.ID,
			Type:     "output",
			OutputID: output.ID,
			DestURI:  output.Destination,
		}
		graph.AddNode(node)

		// Link to operation that produces this output
		sourceID, err := b.resolveReference(output.ID)
		if err != nil {
			return nil, fmt.Errorf("output '%s': %w", output.ID, err)
		}

		edge := &schemas.PlanEdge{
			From:       sourceID,
			To:         node.ID,
			StreamType: "both",
		}
		graph.AddEdge(edge)
	}

	// Step 4: Validate the graph
	if err := graph.DetectCycles(); err != nil {
		return nil, fmt.Errorf("cyclic dependency detected: %w", err)
	}

	return graph, nil
}

// resolveReference resolves an input/output reference to a node ID
func (b *Builder) resolveReference(ref string) (string, error) {
	nodeID, ok := b.outputMap[ref]
	if !ok {
		return "", fmt.Errorf("reference '%s' not found", ref)
	}
	return nodeID, nil
}
