package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// CommandBuilder builds FFmpeg commands from processing plans
type CommandBuilder struct {
	registry *operators.Registry
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder(registry *operators.Registry) *CommandBuilder {
	return &CommandBuilder{
		registry: registry,
	}
}

// Command represents an FFmpeg command to execute
type Command struct {
	Args    []string
	WorkDir string
}

// Build generates an FFmpeg command from a processing plan
func (cb *CommandBuilder) Build(ctx context.Context, plan *schemas.ProcessingPlan) (*Command, error) {
	// Collect input files
	inputs := cb.collectInputs(plan)
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input files found in plan")
	}

	// Build filter expressions for each operation
	filterExprs := []string{}
	streamLabels := make(map[string][]string) // node ID -> output labels

	// Initialize input stream labels
	for i, input := range inputs {
		// Input streams from FFmpeg are [0:v], [0:a], [1:v], [1:a], etc.
		streamLabels[input.nodeID] = []string{
			fmt.Sprintf("[%d:v]", i),
			fmt.Sprintf("[%d:a]", i),
		}
	}

	// Process nodes in execution order
	for _, nodeID := range plan.ExecutionOrder {
		node := cb.getNode(plan, nodeID)
		if node == nil {
			continue
		}

		// Skip input and output nodes
		if node.Type != "operation" {
			continue
		}

		// Get operator
		op, err := cb.registry.Get(node.Operator)
		if err != nil {
			return nil, fmt.Errorf("node %s: operator %s not found: %w", nodeID, node.Operator, err)
		}

		// Build compile context
		compileCtx := cb.buildCompileContext(plan, node, streamLabels)

		// Compile operator
		result, err := op.Compile(compileCtx)
		if err != nil {
			return nil, fmt.Errorf("node %s: compile failed: %w", nodeID, err)
		}

		// Add filter expression
		if result.FilterExpression != "" {
			filterExprs = append(filterExprs, result.FilterExpression)
		}

		// Store output labels for this node
		if len(result.OutputLabels) > 0 {
			streamLabels[nodeID] = result.OutputLabels
		}
	}

	// Build FFmpeg command
	args := []string{"ffmpeg"}

	// Add inputs
	for _, input := range inputs {
		args = append(args, "-i", input.source)
	}

	// Add filter_complex if we have filters
	if len(filterExprs) > 0 {
		filterGraph := strings.Join(filterExprs, ";")
		args = append(args, "-filter_complex", filterGraph)
	}

	// Add outputs
	outputs := cb.collectOutputs(plan)
	for i, output := range outputs {
		// Map output streams
		if labels, ok := streamLabels[output.sourceNodeID]; ok && len(labels) > 0 {
			// Use the output labels from the last operation
			for _, label := range labels {
				args = append(args, "-map", label)
			}
		}

		// Output file
		args = append(args, output.destination)

		// For multiple outputs, we need to use split filter or multiple maps
		// For now, we handle single output case
		if i > 0 {
			// TODO: Handle multiple outputs properly
		}
	}

	return &Command{
		Args: args,
	}, nil
}

// inputFile represents an input file in the plan
type inputFile struct {
	nodeID string
	source string
}

// outputFile represents an output file in the plan
type outputFile struct {
	nodeID       string
	sourceNodeID string // Node that produces this output
	destination  string
}

// collectInputs finds all input nodes in the plan
func (cb *CommandBuilder) collectInputs(plan *schemas.ProcessingPlan) []inputFile {
	inputs := []inputFile{}
	for _, node := range plan.Nodes {
		if node.Type == "input" {
			inputs = append(inputs, inputFile{
				nodeID: node.ID,
				source: node.SourceURI,
			})
		}
	}
	return inputs
}

// collectOutputs finds all output nodes in the plan
func (cb *CommandBuilder) collectOutputs(plan *schemas.ProcessingPlan) []outputFile {
	outputs := []outputFile{}
	for _, node := range plan.Nodes {
		if node.Type == "output" {
			// Find the node that produces this output
			var sourceNodeID string
			for _, edge := range plan.Edges {
				if edge.To == node.ID {
					sourceNodeID = edge.From
					break
				}
			}

			outputs = append(outputs, outputFile{
				nodeID:       node.ID,
				sourceNodeID: sourceNodeID,
				destination:  node.DestURI,
			})
		}
	}
	return outputs
}

// getNode finds a node by ID
func (cb *CommandBuilder) getNode(plan *schemas.ProcessingPlan, nodeID string) *schemas.PlanNode {
	for _, node := range plan.Nodes {
		if node.ID == nodeID {
			return node
		}
	}
	return nil
}

// buildCompileContext creates a compile context for an operator
func (cb *CommandBuilder) buildCompileContext(plan *schemas.ProcessingPlan, node *schemas.PlanNode, streamLabels map[string][]string) *operators.CompileContext {
	// Find input streams
	inputStreams := []operators.StreamRef{}
	for _, edge := range plan.Edges {
		if edge.To == node.ID {
			// Get labels for the source node
			if labels, ok := streamLabels[edge.From]; ok {
				for i, label := range labels {
					inputStreams = append(inputStreams, operators.StreamRef{
						SourceID:    edge.From,
						StreamIndex: i,
						StreamType:  cb.inferStreamType(label),
						Label:       label,
					})
				}
			}
		}
	}

	// Build metadata
	inputMetadata := []*schemas.MediaInfo{}
	for _, edge := range plan.Edges {
		if edge.To == node.ID {
			sourceNode := cb.getNode(plan, edge.From)
			if sourceNode != nil && sourceNode.Metadata != nil {
				inputMetadata = append(inputMetadata, sourceNode.Metadata)
			}
		}
	}

	return &operators.CompileContext{
		InputStreams:  inputStreams,
		Params:        node.Params,
		InputMetadata: inputMetadata,
	}
}

// inferStreamType infers stream type from label
func (cb *CommandBuilder) inferStreamType(label string) string {
	if strings.Contains(label, ":v") || strings.Contains(label, "[v") {
		return "video"
	}
	if strings.Contains(label, ":a") || strings.Contains(label, "[a") {
		return "audio"
	}
	return "both"
}
