package schemas

import (
	"fmt"
)

// Input represents an input source for the job
type Input struct {
	ID     string `json:"id"`
	Source string `json:"source"` // URI: s3://, https://, file://
}

// Operation represents a single processing operation
type Operation struct {
	Op     string                 `json:"op"`     // Operator name (trim, concat, etc.)
	Input  string                 `json:"input"`  // Input ID (single input ops)
	Inputs []string               `json:"inputs"` // Input IDs (multi-input ops like concat)
	Params map[string]interface{} `json:"params"` // Operation-specific parameters
	Output string                 `json:"output"` // Output ID for this operation
}

// Output represents an output destination
type Output struct {
	ID          string `json:"id"`
	Destination string `json:"destination"` // URI: s3://, file://
}

// JobSpec is the user-submitted job definition
type JobSpec struct {
	Inputs     []Input     `json:"inputs"`
	Operations []Operation `json:"operations"`
	Outputs    []Output    `json:"outputs"`
	Debug      bool        `json:"debug,omitempty"` // Enable debug mode
}

// Validate checks if the JobSpec is valid
func (js *JobSpec) Validate() error {
	// Build a map of available inputs (initially just the inputs array)
	availableInputs := make(map[string]bool)
	for _, input := range js.Inputs {
		if input.ID == "" {
			return fmt.Errorf("input ID cannot be empty")
		}
		if input.Source == "" {
			return fmt.Errorf("input '%s' source cannot be empty", input.ID)
		}
		// Check for duplicate input IDs
		if availableInputs[input.ID] {
			return fmt.Errorf("duplicate input ID: '%s'", input.ID)
		}
		availableInputs[input.ID] = true
	}

	// Validate operations and track outputs as new available inputs
	for i, op := range js.Operations {
		if op.Op == "" {
			return fmt.Errorf("operation %d: operator name cannot be empty", i)
		}

		// Check single input reference
		if op.Input != "" {
			if !availableInputs[op.Input] {
				return fmt.Errorf("operation %d (%s): input '%s' not found", i, op.Op, op.Input)
			}
		}

		// Check multi-input references
		for _, inputID := range op.Inputs {
			if !availableInputs[inputID] {
				return fmt.Errorf("operation %d (%s): input '%s' not found", i, op.Op, inputID)
			}
		}

		// Add output as available input for subsequent operations
		if op.Output != "" {
			// Check for duplicate operation output IDs
			if availableInputs[op.Output] {
				return fmt.Errorf("operation %d (%s): duplicate output ID '%s'", i, op.Op, op.Output)
			}
			availableInputs[op.Output] = true
		}
	}

	// Validate outputs
	for i, output := range js.Outputs {
		if output.ID == "" {
			return fmt.Errorf("output %d: ID cannot be empty", i)
		}
		if output.Destination == "" {
			return fmt.Errorf("output '%s': destination cannot be empty", output.ID)
		}
		// Check that output ID refers to something that was produced
		if !availableInputs[output.ID] {
			return fmt.Errorf("output '%s': refers to non-existent input/operation output", output.ID)
		}
	}

	return nil
}
