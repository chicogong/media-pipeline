package main

import (
	"testing"

	"github.com/chicogong/media-pipeline/pkg/operators"
)

// TestBuiltinOperatorsRegistered verifies that the operators the planner needs
// are present in the global registry within the cmd/api binary's import graph.
//
// Operators self-register via init() in pkg/operators/builtin, so that package
// must be imported (at least blank-imported) by main.go. Without it the API
// server starts fine but fails every job at the planning step with
// "operator '...' not found". This test guards that wiring.
func TestBuiltinOperatorsRegistered(t *testing.T) {
	for _, name := range []string{"trim", "scale"} {
		if _, err := operators.Get(name); err != nil {
			t.Errorf("operator %q not registered: %v "+
				"(is pkg/operators/builtin imported by main.go?)", name, err)
		}
	}
}
