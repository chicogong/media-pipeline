package validator

import (
	"testing"

	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/stretchr/testify/assert"
)

func TestValidator_Validate_ValidSpec(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video1", Params: map[string]interface{}{"start": "00:00:10"}, Output: "trimmed"},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "file:///tmp/output.mp4"},
		},
	}

	validator := New()
	err := validator.Validate(spec)
	assert.NoError(t, err)
}

func TestValidator_Validate_EmptyInputs(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs:     []schemas.Input{},
		Operations: []schemas.Operation{},
		Outputs:    []schemas.Output{},
	}

	validator := New()
	err := validator.Validate(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one input")
}

func TestValidator_Validate_EmptyOperations(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []schemas.Operation{},
		Outputs:    []schemas.Output{},
	}

	validator := New()
	err := validator.Validate(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one operation")
}

func TestValidator_Validate_InvalidScheme(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "ftp://example.com/video.mp4"}, // ftp not allowed
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "file:///tmp/output.mp4"},
		},
	}

	validator := New()
	err := validator.Validate(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme 'ftp' not allowed")
}

func TestValidator_Validate_SSRF_Protection(t *testing.T) {
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video1", Source: "http://127.0.0.1/internal.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "file:///tmp/output.mp4"},
		},
	}

	validator := New()
	err := validator.Validate(spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "localhost")
}
