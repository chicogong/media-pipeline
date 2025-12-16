package planner

import (
	"context"
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Planner creates processing plans from job specifications
type Planner struct {
	builder    *Builder
	propagator *MetadataPropagator
	estimator  *ResourceEstimator
	registry   *operators.Registry
}

// NewPlanner creates a new planner with default configuration
func NewPlanner() *Planner {
	registry := operators.GlobalRegistry()
	return &Planner{
		builder:    NewBuilder(),
		propagator: NewMetadataPropagator(registry),
		estimator:  NewResourceEstimator(registry),
		registry:   registry,
	}
}

// NewPlannerWithRegistry creates a new planner with a custom operator registry
func NewPlannerWithRegistry(registry *operators.Registry) *Planner {
	return &Planner{
		builder:    NewBuilder(),
		propagator: NewMetadataPropagator(registry),
		estimator:  NewResourceEstimator(registry),
		registry:   registry,
	}
}

// PlanOptions contains options for plan generation
type PlanOptions struct {
	// SkipMetadataValidation skips metadata propagation (for testing)
	SkipMetadataValidation bool

	// SkipResourceEstimation skips resource estimation (for testing)
	SkipResourceEstimation bool
}

// Plan generates a complete processing plan from a JobSpec
func (p *Planner) Plan(ctx context.Context, spec *schemas.JobSpec, opts *PlanOptions) (*schemas.ProcessingPlan, error) {
	if opts == nil {
		opts = &PlanOptions{}
	}

	// Step 1: Build DAG
	graph, err := p.builder.BuildDAG(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to build DAG: %w", err)
	}

	// Step 2: Validate graph structure
	if err := graph.DetectCycles(); err != nil {
		return nil, fmt.Errorf("graph validation failed: %w", err)
	}

	// Step 3: Get execution order
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to compute execution order: %w", err)
	}

	// Step 4: Get execution stages for parallelization
	stages, err := graph.ComputeExecutionStages()
	if err != nil {
		return nil, fmt.Errorf("failed to compute execution stages: %w", err)
	}

	// Step 5: Propagate metadata (if inputs have metadata)
	if !opts.SkipMetadataValidation {
		// Check if any input nodes have metadata
		inputNodes := graph.GetInputNodes()
		hasMetadata := false
		for _, node := range inputNodes {
			if node.Metadata != nil {
				hasMetadata = true
				break
			}
		}

		if hasMetadata {
			if err := p.propagator.Propagate(ctx, graph); err != nil {
				return nil, fmt.Errorf("metadata propagation failed: %w", err)
			}
		}
	}

	// Step 6: Estimate resources (if metadata exists)
	var estimates *schemas.ResourceEstimates
	if !opts.SkipResourceEstimation {
		// Check if metadata was propagated
		hasMetadata := false
		for _, node := range graph.Nodes {
			if node.Type == "operation" && node.Metadata != nil {
				hasMetadata = true
				break
			}
		}

		if hasMetadata {
			estimates, err = p.estimator.Estimate(ctx, graph)
			if err != nil {
				return nil, fmt.Errorf("resource estimation failed: %w", err)
			}
		}
	}

	// Step 7: Build processing plan
	plan := &schemas.ProcessingPlan{
		JobID:            spec.JobID,
		Nodes:            graph.Nodes,
		Edges:            graph.Edges,
		ExecutionOrder:   order,
		ExecutionStages:  stages,
		ResourceEstimate: estimates,
	}

	return plan, nil
}

// ValidateOperators validates that all operators in the spec are registered
func (p *Planner) ValidateOperators(spec *schemas.JobSpec) error {
	for i, op := range spec.Operations {
		_, err := p.registry.Get(op.Op)
		if err != nil {
			return fmt.Errorf("operation %d: operator '%s' not found", i, op.Op)
		}
	}
	return nil
}

// ValidateParameters validates parameters for all operations in the spec
func (p *Planner) ValidateParameters(spec *schemas.JobSpec) error {
	for i, op := range spec.Operations {
		operator, err := p.registry.Get(op.Op)
		if err != nil {
			return fmt.Errorf("operation %d: operator '%s' not found", i, op.Op)
		}

		if err := operator.ValidateParams(op.Params); err != nil {
			return fmt.Errorf("operation %d (%s): %w", i, op.Op, err)
		}
	}
	return nil
}
