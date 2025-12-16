package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// ResourceEstimator estimates resource requirements for a processing plan
type ResourceEstimator struct {
	registry *operators.Registry
}

// NewResourceEstimator creates a new resource estimator
func NewResourceEstimator(registry *operators.Registry) *ResourceEstimator {
	return &ResourceEstimator{
		registry: registry,
	}
}

// Estimate computes resource estimates for all nodes in the graph
func (re *ResourceEstimator) Estimate(ctx context.Context, graph *Graph) (*schemas.ResourceEstimates, error) {
	// Get execution stages for parallel estimation
	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		return nil, fmt.Errorf("failed to compute execution stages: %w", err)
	}

	nodeEstimates := make(map[string]*schemas.NodeEstimates)
	var totalDuration time.Duration
	var peakMemoryMB int64
	var totalDiskMB int64

	// Process each stage
	for _, stage := range stages {
		var stageMaxDuration time.Duration
		var stageMemoryMB int64

		// Process nodes in the stage
		for _, nodeID := range stage {
			node := graph.GetNode(nodeID)
			if node == nil {
				return nil, fmt.Errorf("node %s not found", nodeID)
			}

			// Only estimate for operation nodes
			if node.Type != "operation" {
				continue
			}

			// Validate node has metadata
			if node.Metadata == nil {
				return nil, fmt.Errorf("node %s has no metadata (run metadata propagation first)", nodeID)
			}

			// Get the operator
			op, err := re.registry.Get(node.Operator)
			if err != nil {
				return nil, fmt.Errorf("node %s: operator %s not found: %w", nodeID, node.Operator, err)
			}

			// Collect input metadata
			inputMetadata, err := re.collectInputMetadata(graph, node)
			if err != nil {
				return nil, fmt.Errorf("node %s: failed to collect input metadata: %w", nodeID, err)
			}

			// Estimate resources for this node
			estimate, err := op.EstimateResources(node.Params, inputMetadata)
			if err != nil {
				return nil, fmt.Errorf("node %s: failed to estimate resources: %w", nodeID, err)
			}

			// Store node estimate
			nodeEstimates[nodeID] = estimate

			// Update stage estimates (for parallel operations)
			if estimate.Duration > stageMaxDuration {
				stageMaxDuration = estimate.Duration
			}
			stageMemoryMB += estimate.MemoryMB

			// Update total disk usage
			totalDiskMB += estimate.DiskMB
		}

		// Add stage duration to total (stages run sequentially)
		totalDuration += stageMaxDuration

		// Update peak memory (max across all stages)
		if stageMemoryMB > peakMemoryMB {
			peakMemoryMB = stageMemoryMB
		}
	}

	return &schemas.ResourceEstimates{
		NodeEstimates: nodeEstimates,
		TotalDuration: totalDuration,
		PeakMemoryMB:  peakMemoryMB,
		TotalDiskMB:   totalDiskMB,
	}, nil
}

// collectInputMetadata collects metadata from all predecessor nodes
func (re *ResourceEstimator) collectInputMetadata(graph *Graph, node *schemas.PlanNode) ([]*schemas.MediaInfo, error) {
	predecessors := graph.GetPredecessors(node.ID)
	if len(predecessors) == 0 {
		return nil, fmt.Errorf("operation node %s has no inputs", node.ID)
	}

	inputs := make([]*schemas.MediaInfo, 0, len(predecessors))
	for _, pred := range predecessors {
		if pred.Metadata == nil {
			return nil, fmt.Errorf("predecessor %s has no metadata", pred.ID)
		}
		inputs = append(inputs, pred.Metadata)
	}

	return inputs, nil
}
