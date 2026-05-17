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

func TestTrimOperator_Category(t *testing.T) {
	op := &TrimOperator{}
	if got := op.Category(); got != operators.CategoryTimeline {
		t.Fatalf("Category() = %q, want %q", got, operators.CategoryTimeline)
	}
}

// TestTrimOperator_ComputeOutputMetadata_EndParam covers the "end" param branch
// (missed in the 47.4% partial coverage).
func TestTrimOperator_ComputeOutputMetadata_EndParam(t *testing.T) {
	op := &TrimOperator{}

	input := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 120 * time.Second,
		},
		VideoStreams: []schemas.VideoStream{{Width: 1280, Height: 720}},
	}

	// start=10s, end=40s → output duration should be 30s
	out, err := op.ComputeOutputMetadata(
		map[string]interface{}{"start": "10s", "end": "40s"},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("ComputeOutputMetadata(end) failed: %v", err)
	}
	if out.Format.Duration != 30*time.Second {
		t.Fatalf("duration mismatch: got=%v want=30s", out.Format.Duration)
	}
	// original input must not be mutated
	if input.Format.Duration != 120*time.Second {
		t.Fatalf("input mutated: got=%v want=120s", input.Format.Duration)
	}
}

// TestTrimOperator_ComputeOutputMetadata_NoTimingParams covers the case where
// neither "duration" nor "end" is supplied — output duration stays unchanged.
func TestTrimOperator_ComputeOutputMetadata_NoTimingParams(t *testing.T) {
	op := &TrimOperator{}

	input := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 90 * time.Second,
		},
	}

	out, err := op.ComputeOutputMetadata(
		map[string]interface{}{"start": "00:00:05"},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("ComputeOutputMetadata(no timing) failed: %v", err)
	}
	// Duration should be carried over unchanged
	if out.Format.Duration != 90*time.Second {
		t.Fatalf("duration mismatch: got=%v want=90s", out.Format.Duration)
	}
}

// TestTrimOperator_ComputeOutputMetadata_NoInputs checks the empty-input guard.
func TestTrimOperator_ComputeOutputMetadata_NoInputs(t *testing.T) {
	op := &TrimOperator{}
	_, err := op.ComputeOutputMetadata(map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected error for empty inputs, got nil")
	}
}

func TestTrimOperator_EstimateResources_Basic(t *testing.T) {
	op := &TrimOperator{}

	input := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 120 * time.Second,
			BitRate:  4_000_000,
		},
	}

	// With an explicit "duration" param — exercises that conversion branch.
	est, err := op.EstimateResources(
		map[string]interface{}{"start": "00:00:10", "duration": "00:00:30"},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("EstimateResources failed: %v", err)
	}
	if est == nil {
		t.Fatal("EstimateResources returned nil estimates")
	}
	if est.Duration <= 0 {
		t.Fatalf("expected positive Duration, got %v", est.Duration)
	}
	if est.MemoryMB != 100 {
		t.Fatalf("expected MemoryMB=100, got %d", est.MemoryMB)
	}
}

func TestTrimOperator_EstimateResources_DefaultBitrate(t *testing.T) {
	// BitRate == 0 exercises the default 5 Mbps fallback path.
	op := &TrimOperator{}

	input := &schemas.MediaInfo{
		Format: schemas.FormatInfo{
			Duration: 60 * time.Second,
			BitRate:  0,
		},
	}

	est, err := op.EstimateResources(
		map[string]interface{}{},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("EstimateResources failed: %v", err)
	}
	if est == nil {
		t.Fatal("EstimateResources returned nil estimates")
	}
	if est.MemoryMB != 100 {
		t.Fatalf("expected MemoryMB=100, got %d", est.MemoryMB)
	}
}

func TestTrimOperator_EstimateResources_NoInputs(t *testing.T) {
	op := &TrimOperator{}
	_, err := op.EstimateResources(map[string]interface{}{}, nil)
	if err == nil {
		t.Fatal("expected error for empty inputs, got nil")
	}
}
