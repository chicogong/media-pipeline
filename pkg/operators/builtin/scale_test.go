package builtin

import (
	"strings"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestScaleOperator_ValidateParams(t *testing.T) {
	op := &ScaleOperator{}

	if err := op.ValidateParams(map[string]interface{}{}); err == nil {
		t.Fatal("expected error for missing required params, got nil")
	}

	if err := op.ValidateParams(map[string]interface{}{"width": -1, "height": -1}); err == nil {
		t.Fatal("expected error when both width and height are -1, got nil")
	}
}

func TestScaleOperator_ComputeOutputMetadata_DoesNotMutateInput(t *testing.T) {
	op := &ScaleOperator{}

	input := &schemas.MediaInfo{
		VideoStreams: []schemas.VideoStream{
			{Width: 1920, Height: 1080},
		},
		AudioStreams: []schemas.AudioStream{
			{Channels: 2},
		},
	}

	out, err := op.ComputeOutputMetadata(
		map[string]interface{}{"width": 1280, "height": 720},
		[]*schemas.MediaInfo{input},
	)
	if err != nil {
		t.Fatalf("ComputeOutputMetadata failed: %v", err)
	}

	if input.VideoStreams[0].Width != 1920 || input.VideoStreams[0].Height != 1080 {
		t.Fatalf("input mutated: got=%dx%d want=1920x1080", input.VideoStreams[0].Width, input.VideoStreams[0].Height)
	}
	if out.VideoStreams[0].Width != 1280 || out.VideoStreams[0].Height != 720 {
		t.Fatalf("output mismatch: got=%dx%d want=1280x720", out.VideoStreams[0].Width, out.VideoStreams[0].Height)
	}
	if len(out.AudioStreams) != 1 {
		t.Fatalf("expected audio streams to be preserved, got=%d", len(out.AudioStreams))
	}
}

func TestScaleOperator_Compile_UsesLabelsDirectly(t *testing.T) {
	op := &ScaleOperator{}

	res, err := op.Compile(&operators.CompileContext{
		InputStreams: []operators.StreamRef{
			{Label: "[0:v]", StreamType: "video"},
		},
		Params: map[string]interface{}{
			"width":     1280,
			"height":    720,
			"algorithm": "lanczos",
		},
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	if strings.Contains(res.FilterExpression, "[[") {
		t.Fatalf("unexpected double-bracket label in filter: %q", res.FilterExpression)
	}
	if !strings.Contains(res.FilterExpression, "[0:v]scale=") {
		t.Fatalf("expected filter to reference input label, got: %q", res.FilterExpression)
	}
}

