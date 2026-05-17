package schemas

import (
	"testing"
)

// TestJobSpec_Validate_Valid verifies that a well-formed spec passes validation.
func TestJobSpec_Validate_Valid(t *testing.T) {
	js := &JobSpec{
		Inputs: []Input{
			{ID: "src", Source: "s3://bucket/input.mp4"},
		},
		Operations: []Operation{
			{Op: "transcode", Input: "src", Output: "transcoded"},
			{Op: "thumbnail", Inputs: []string{"transcoded"}, Output: "thumb"},
		},
		Outputs: []Output{
			{ID: "thumb", Destination: "s3://bucket/thumb.jpg"},
		},
	}

	if err := js.Validate(); err != nil {
		t.Fatalf("expected no error for valid spec, got: %v", err)
	}
}

// TestJobSpec_Validate_Errors verifies that each invalid configuration is rejected.
func TestJobSpec_Validate_Errors(t *testing.T) {
	// A helper that builds a minimal valid spec so each case can override one field.
	validBase := func() *JobSpec {
		return &JobSpec{
			Inputs: []Input{
				{ID: "src", Source: "s3://bucket/input.mp4"},
			},
			Operations: []Operation{
				{Op: "transcode", Input: "src", Output: "out1"},
			},
			Outputs: []Output{
				{ID: "out1", Destination: "s3://bucket/output.mp4"},
			},
		}
	}

	cases := []struct {
		name string
		spec *JobSpec
	}{
		{
			name: "empty input ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "", Source: "s3://bucket/input.mp4"},
				},
			},
		},
		{
			name: "empty input source",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: ""},
				},
			},
		},
		{
			name: "duplicate input ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: "s3://bucket/a.mp4"},
					{ID: "src", Source: "s3://bucket/b.mp4"},
				},
			},
		},
		{
			name: "empty operation Op",
			spec: func() *JobSpec {
				s := validBase()
				s.Operations[0].Op = ""
				return s
			}(),
		},
		{
			name: "operation Input references non-existent ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: "s3://bucket/input.mp4"},
				},
				Operations: []Operation{
					{Op: "transcode", Input: "nonexistent", Output: "out1"},
				},
			},
		},
		{
			name: "operation Inputs contains non-existent ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: "s3://bucket/input.mp4"},
				},
				Operations: []Operation{
					{Op: "merge", Inputs: []string{"src", "ghost"}, Output: "out1"},
				},
			},
		},
		{
			name: "operation Output duplicates an existing ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: "s3://bucket/input.mp4"},
				},
				Operations: []Operation{
					// Output collides with the input ID "src"
					{Op: "transcode", Input: "src", Output: "src"},
				},
			},
		},
		{
			name: "empty output ID",
			spec: func() *JobSpec {
				s := validBase()
				s.Outputs[0].ID = ""
				return s
			}(),
		},
		{
			name: "empty output destination",
			spec: func() *JobSpec {
				s := validBase()
				s.Outputs[0].Destination = ""
				return s
			}(),
		},
		{
			name: "output ID references non-existent ID",
			spec: &JobSpec{
				Inputs: []Input{
					{ID: "src", Source: "s3://bucket/input.mp4"},
				},
				Operations: []Operation{
					{Op: "transcode", Input: "src", Output: "out1"},
				},
				Outputs: []Output{
					{ID: "doesnotexist", Destination: "s3://bucket/output.mp4"},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.spec.Validate()
			if err == nil {
				t.Fatalf("expected an error for case %q, got nil", tc.name)
			}
			if err.Error() == "" {
				t.Fatalf("error message for case %q must not be empty", tc.name)
			}
		})
	}
}
