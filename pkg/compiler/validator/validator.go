package validator

import (
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/storage"
)

// Validator validates JobSpec
type Validator struct{}

// New creates a new Validator
func New() *Validator {
	return &Validator{}
}

// Validate checks if a JobSpec is valid
func (v *Validator) Validate(spec *schemas.JobSpec) error {
	// Check for at least one input
	if len(spec.Inputs) == 0 {
		return fmt.Errorf("JobSpec must have at least one input")
	}

	// Check for at least one operation
	if len(spec.Operations) == 0 {
		return fmt.Errorf("JobSpec must have at least one operation")
	}

	// Validate input URIs
	for i, input := range spec.Inputs {
		scheme, _, err := storage.ParseURI(input.Source)
		if err != nil {
			return fmt.Errorf("input %d (%s): invalid URI: %w", i, input.ID, err)
		}

		if !storage.IsAllowedScheme(scheme) {
			return fmt.Errorf("input %d (%s): scheme '%s' not allowed", i, input.ID, scheme)
		}

		// For HTTP/HTTPS URIs, perform SSRF checks
		if scheme == "http" || scheme == "https" {
			if err := ValidateHTTPURI(input.Source); err != nil {
				return fmt.Errorf("input %d (%s): security check failed: %w", i, input.ID, err)
			}
		}
	}

	// Validate output URIs
	for i, output := range spec.Outputs {
		scheme, _, err := storage.ParseURI(output.Destination)
		if err != nil {
			return fmt.Errorf("output %d (%s): invalid URI: %w", i, output.ID, err)
		}

		if !storage.IsAllowedScheme(scheme) {
			return fmt.Errorf("output %d (%s): scheme '%s' not allowed", i, output.ID, scheme)
		}
	}

	// Use JobSpec's built-in validation for dependency checking
	if err := spec.Validate(); err != nil {
		return err
	}

	return nil
}
