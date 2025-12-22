package builtin

import (
	"strings"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestTrimOperator_ValidateParams(t *testing.T) {
	op := &TrimOperator{}

	err := op.ValidateParams(map[string]interface{}{
		"start":    "00:00:00",
		"duration": "00:00:10",
		"end":      "00:00:20",
	})
	if err == nil {
		t.Fatal("expected error when both duration and end are set, got nil")
	}
}

func TestTrimOperator_ComputeOutputMetadata(t *testing.T) {
	op := &TrimOperator{}

	input := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 60 * time.Second,
		},
		VideoStreams: []schemas.VideoStream{{Width: 1920, Height: 1080}},
	}

	out, err := op.ComputeOutputMetadata(
		map[string]interface{}{"start": "00:00:10", "duration": "00:00:30"},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("ComputeOutputMetadata failed: %v", err)
	}

	if input.Format.Duration != 60*time.Second {
		t.Fatalf("input mutated: got=%v want=%v", input.Format.Duration, 60*time.Second)
	}
	if out.Format.Duration != 30*time.Second {
		t.Fatalf("duration mismatch: got=%v want=%v", out.Format.Duration, 30*time.Second)
	}
	if out.VideoStreams[0].Width != 1920 || out.VideoStreams[0].Height != 1080 {
		t.Fatalf("expected trim to keep resolution, got=%dx%d", out.VideoStreams[0].Width, out.VideoStreams[0].Height)
	}
}

func TestTrimOperator_Compile_UsesSeparateVideoAndAudioInputs(t *testing.T) {
	op := &TrimOperator{}

	res, err := op.Compile(&operators.CompileContext{
		InputStreams: []operators.StreamRef{
			{Label: "[0:v]", StreamType: "video"},
			{Label: "[0:a]", StreamType: "audio"},
		},
		Params: map[string]interface{}{
			"start":    "00:00:10",
			"duration": "00:00:30",
		},
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if strings.Contains(res.FilterExpression, "[[") {
		t.Fatalf("unexpected double-bracket label in filter: %q", res.FilterExpression)
	}
	if !strings.Contains(res.FilterExpression, "[0:v]trim=") {
		t.Fatalf("expected video trim to reference [0:v], got: %q", res.FilterExpression)
	}
	if !strings.Contains(res.FilterExpression, "[0:a]atrim=") {
		t.Fatalf("expected audio atrim to reference [0:a], got: %q", res.FilterExpression)
	}
}

