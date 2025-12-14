package schemas

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobSpec_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"inputs": [
			{"id": "video1", "source": "s3://bucket/video.mp4"}
		],
		"operations": [
			{
				"op": "trim",
				"input": "video1",
				"params": {"start": "00:00:10", "duration": "00:00:30"},
				"output": "trimmed"
			}
		],
		"outputs": [
			{"id": "result.mp4", "destination": "s3://bucket/output.mp4"}
		]
	}`

	var spec JobSpec
	err := json.Unmarshal([]byte(jsonData), &spec)

	require.NoError(t, err)
	assert.Len(t, spec.Inputs, 1)
	assert.Equal(t, "video1", spec.Inputs[0].ID)
	assert.Equal(t, "s3://bucket/video.mp4", spec.Inputs[0].Source)
	assert.Len(t, spec.Operations, 1)
	assert.Equal(t, "trim", spec.Operations[0].Op)
	assert.Equal(t, "video1", spec.Operations[0].Input)
	assert.Len(t, spec.Outputs, 1)
}

func TestJobSpec_Validate_ValidSpec(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.NoError(t, err)
}

func TestJobSpec_Validate_MissingInput(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video2", Output: "trimmed"}, // video2 doesn't exist
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input 'video2' not found")
}
