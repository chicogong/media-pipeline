package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// TestCommandBuilder_ChainedOperationsUseUniqueLabels guards against the filter
// label-collision bug: before the fix every operator emitted a hardcoded "[v]"
// output label, so chaining two operations produced "...[v];[v]...[v]" which
// FFmpeg rejects. Each operation must now get a unique label prefix.
func TestCommandBuilder_ChainedOperationsUseUniqueLabels(t *testing.T) {
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "video", Output: "trimmed",
				Params: map[string]interface{}{"start": "00:00:00", "duration": "00:00:05"}},
			{Op: "scale", Input: "trimmed", Output: "scaled",
				Params: map[string]interface{}{"width": 640, "height": 360}},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "/tmp/output.mp4"},
		},
	}

	plan, err := planner.NewPlanner().Plan(context.Background(), spec, nil)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	cmd, err := NewCommandBuilder(operators.GlobalRegistry()).Build(context.Background(), plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	var graph string
	for i, a := range cmd.Args {
		if a == "-filter_complex" && i+1 < len(cmd.Args) {
			graph = cmd.Args[i+1]
		}
	}
	if graph == "" {
		t.Fatal("command has no -filter_complex filtergraph")
	}

	// trim is operation f0, scale is operation f1: each must use its own label.
	if !strings.Contains(graph, "[vf0]") || !strings.Contains(graph, "[vf1]") {
		t.Errorf("expected unique per-operation labels [vf0] and [vf1] in filtergraph: %q", graph)
	}
	// The ambiguous bare "[v]" label must no longer appear.
	if strings.Contains(graph, "[v]") {
		t.Errorf("filtergraph still uses the collision-prone bare [v] label: %q", graph)
	}
}

func TestToMapArg(t *testing.T) {
	cases := map[string]string{
		"[0:v]":  "0:v", // raw input stream reference -> no brackets
		"[0:a]":  "0:a",
		"[12:a]": "12:a",
		"[vf0]":  "[vf0]", // filtergraph output label -> kept as-is
		"[af1]":  "[af1]",
		"[v]":    "[v]",
	}
	for in, want := range cases {
		if got := toMapArg(in); got != want {
			t.Errorf("toMapArg(%q) = %q, want %q", in, got, want)
		}
	}
}
