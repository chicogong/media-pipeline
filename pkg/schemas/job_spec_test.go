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

func TestJobSpec_Validate_DuplicateInputIDs(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video1.mp4"},
			{ID: "video1", Source: "https://example.com/video2.mp4"}, // Duplicate ID
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate input ID: 'video1'")
}

func TestJobSpec_Validate_DuplicateOperationOutputIDs(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "processed"},
			{Op: "scale", Input: "processed", Output: "processed"}, // Duplicate output ID
		},
		Outputs: []Output{
			{ID: "processed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate output ID 'processed'")
}

func TestJobSpec_Validate_DuplicateOperationOutputMatchingInput(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "video1"}, // Output ID matches input ID
		},
		Outputs: []Output{
			{ID: "video1", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate output ID 'video1'")
}

func TestJobSpec_Validate_EmptyInputID(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "", Source: "https://example.com/video.mp4"}, // Empty ID
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input ID cannot be empty")
}

func TestJobSpec_Validate_EmptyInputSource(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: ""}, // Empty source
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input 'video1' source cannot be empty")
}

func TestJobSpec_Validate_EmptyOperatorName(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "", Input: "video1", Output: "trimmed"}, // Empty operator name
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operator name cannot be empty")
}

func TestJobSpec_Validate_EmptyOutputID(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "", Destination: "/tmp/output.mp4"}, // Empty output ID
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output")
	assert.Contains(t, err.Error(), "ID cannot be empty")
}

func TestJobSpec_Validate_EmptyOutputDestination(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "trimmed", Destination: ""}, // Empty destination
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output 'trimmed'")
	assert.Contains(t, err.Error(), "destination cannot be empty")
}

func TestJobSpec_Validate_MultiInputOperation(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video1.mp4"},
			{ID: "video2", Source: "https://example.com/video2.mp4"},
			{ID: "audio1", Source: "https://example.com/audio.mp3"},
		},
		Operations: []Operation{
			{Op: "concat", Inputs: []string{"video1", "video2"}, Output: "concatenated"},
			{Op: "merge", Inputs: []string{"concatenated", "audio1"}, Output: "final"},
		},
		Outputs: []Output{
			{ID: "final", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.NoError(t, err)
}

func TestJobSpec_Validate_MultiInputOperation_MissingInput(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video1.mp4"},
			{ID: "video2", Source: "https://example.com/video2.mp4"},
		},
		Operations: []Operation{
			{Op: "concat", Inputs: []string{"video1", "video3"}, Output: "concatenated"}, // video3 doesn't exist
		},
		Outputs: []Output{
			{ID: "concatenated", Destination: "/tmp/output.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input 'video3' not found")
}

func TestJobSpec_Validate_ChainedOperations(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "raw_video", Source: "https://example.com/raw.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "raw_video", Output: "trimmed", Params: map[string]interface{}{"start": "00:00:10", "duration": "00:00:30"}},
			{Op: "scale", Input: "trimmed", Output: "scaled", Params: map[string]interface{}{"width": 1280, "height": 720}},
			{Op: "encode", Input: "scaled", Output: "encoded", Params: map[string]interface{}{"codec": "h264"}},
		},
		Outputs: []Output{
			{ID: "encoded", Destination: "s3://bucket/final.mp4"},
		},
	}

	err := spec.Validate()
	assert.NoError(t, err)
}

func TestJobSpec_Validate_ChainedOperations_BrokenChain(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "raw_video", Source: "https://example.com/raw.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "raw_video", Output: "trimmed"},
			{Op: "scale", Input: "scaled_output", Output: "final"}, // Wrong input - chain broken
		},
		Outputs: []Output{
			{ID: "final", Destination: "s3://bucket/final.mp4"},
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input 'scaled_output' not found")
}

func TestJobSpec_Validate_OutputReferencesNonExistent(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "video1", Source: "https://example.com/video.mp4"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "video1", Output: "trimmed"},
		},
		Outputs: []Output{
			{ID: "nonexistent", Destination: "/tmp/output.mp4"}, // References non-existent ID
		},
	}

	err := spec.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output 'nonexistent'")
	assert.Contains(t, err.Error(), "non-existent")
}

func TestJobSpec_Validate_ComplexWorkflow(t *testing.T) {
	spec := &JobSpec{
		Inputs: []Input{
			{ID: "intro", Source: "s3://bucket/intro.mp4"},
			{ID: "main", Source: "s3://bucket/main.mp4"},
			{ID: "outro", Source: "s3://bucket/outro.mp4"},
			{ID: "bgm", Source: "s3://bucket/music.mp3"},
		},
		Operations: []Operation{
			{Op: "trim", Input: "main", Output: "main_trimmed", Params: map[string]interface{}{"start": "00:00:05", "duration": "00:10:00"}},
			{Op: "concat", Inputs: []string{"intro", "main_trimmed", "outro"}, Output: "video_concat"},
			{Op: "merge_audio", Inputs: []string{"video_concat", "bgm"}, Output: "with_audio"},
			{Op: "encode", Input: "with_audio", Output: "final_encoded", Params: map[string]interface{}{"codec": "h264", "bitrate": "5000k"}},
		},
		Outputs: []Output{
			{ID: "final_encoded", Destination: "s3://output/final.mp4"},
		},
	}

	err := spec.Validate()
	assert.NoError(t, err)
}
