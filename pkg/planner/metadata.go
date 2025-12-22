package planner

import (
	"context"
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// MetadataPropagator propagates media metadata through the graph
type MetadataPropagator struct {
	registry *operators.Registry
}

func cloneMediaInfo(mi *schemas.MediaInfo) *schemas.MediaInfo {
	if mi == nil {
		return nil
	}

	clone := *mi
	clone.VideoStreams = append([]schemas.VideoStream(nil), mi.VideoStreams...)
	clone.AudioStreams = append([]schemas.AudioStream(nil), mi.AudioStreams...)
	return &clone
}

// NewMetadataPropagator creates a new metadata propagator
func NewMetadataPropagator(registry *operators.Registry) *MetadataPropagator {
	return &MetadataPropagator{
		registry: registry,
	}
}

// Propagate propagates metadata through the graph in topological order
func (mp *MetadataPropagator) Propagate(ctx context.Context, graph *Graph) error {
	// Get topological order
	order, err := graph.TopologicalSort()
	if err != nil {
		return fmt.Errorf("failed to get topological order: %w", err)
	}

	// Process nodes in topological order
	for _, nodeID := range order {
		node := graph.GetNode(nodeID)
		if node == nil {
			return fmt.Errorf("node %s not found", nodeID)
		}

		switch node.Type {
		case "input":
			// Input nodes must have metadata set externally
			if node.Metadata == nil {
				return fmt.Errorf("input node %s has no metadata", nodeID)
			}

		case "operation":
			// Get the operator
			op, err := mp.registry.Get(node.Operator)
			if err != nil {
				return fmt.Errorf("node %s: operator %s not found: %w", nodeID, node.Operator, err)
			}

			// Collect input metadata from predecessors
			inputMetadata, err := mp.collectInputMetadata(graph, node)
			if err != nil {
				return fmt.Errorf("node %s: failed to collect input metadata: %w", nodeID, err)
			}

			// Compute output metadata
			outputMetadata, err := op.ComputeOutputMetadata(node.Params, inputMetadata)
			if err != nil {
				return fmt.Errorf("node %s: failed to compute output metadata: %w", nodeID, err)
			}

			// Store metadata in node
			node.Metadata = outputMetadata

			case "output":
				// Output nodes inherit metadata from their predecessor
				predecessors := graph.GetPredecessors(nodeID)
				if len(predecessors) == 0 {
					return fmt.Errorf("output node %s has no predecessors", nodeID)
			}
			if len(predecessors) > 1 {
				return fmt.Errorf("output node %s has multiple predecessors", nodeID)
			}

			predNode := predecessors[0]
				if predNode.Metadata == nil {
					return fmt.Errorf("output node %s: predecessor %s has no metadata", nodeID, predNode.ID)
				}

				// Copy metadata to output node
				node.Metadata = cloneMediaInfo(predNode.Metadata)

			default:
				return fmt.Errorf("unknown node type: %s", node.Type)
			}
		}

	return nil
}

// collectInputMetadata collects metadata from all predecessor nodes
func (mp *MetadataPropagator) collectInputMetadata(graph *Graph, node *schemas.PlanNode) ([]*schemas.MediaInfo, error) {
	predecessors := graph.GetPredecessors(node.ID)
	if len(predecessors) == 0 {
		return nil, fmt.Errorf("operation node %s has no inputs", node.ID)
	}

	inputs := make([]*schemas.MediaInfo, 0, len(predecessors))
	for _, pred := range predecessors {
		if pred.Metadata == nil {
			return nil, fmt.Errorf("predecessor %s has no metadata", pred.ID)
		}
		inputs = append(inputs, cloneMediaInfo(pred.Metadata))
	}

	return inputs, nil
}
