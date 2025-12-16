package executor_test

import (
	"context"
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/executor"
	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/operators/builtin"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// ExampleExecutor demonstrates how to execute a processing plan
func ExampleExecutor() {
	// Register operators
	operators.Register(&builtin.TrimOperator{})
	operators.Register(&builtin.ScaleOperator{})

	// Create a job spec
	spec := &schemas.JobSpec{
		JobID: "example-job",
		Inputs: []schemas.Input{
			{ID: "video", Source: "/path/to/input.mp4", Type: "video"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video",
				Output: "trimmed",
				Params: map[string]interface{}{
					"start":    "00:00:10",
					"duration": "00:00:30",
				},
			},
			{
				Op:     "scale",
				Input:  "trimmed",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":  1280,
					"height": 720,
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "/path/to/output.mp4"},
		},
	}

	// Generate plan
	planner := planner.NewPlanner()
	plan, err := planner.Plan(context.Background(), spec, nil)
	if err != nil {
		fmt.Printf("Planning failed: %v\n", err)
		return
	}

	// Create executor
	exec := executor.NewExecutor(operators.GlobalRegistry())

	// Build command (without executing)
	cmd, err := exec.BuildCommand(context.Background(), plan)
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
		return
	}

	fmt.Printf("FFmpeg command built with %d arguments\n", len(cmd.Args))
	fmt.Printf("Command: %s\n", cmd.Args[0])

	// Output:
	// FFmpeg command built with 7 arguments
	// Command: ffmpeg
}

// ExampleExecutor_withProgress demonstrates progress tracking
func ExampleExecutor_withProgress() {
	// Register operators
	operators.Register(&builtin.ScaleOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/path/to/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "scale",
				Input:  "video",
				Output: "scaled",
				Params: map[string]interface{}{
					"width":  1920,
					"height": 1080,
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "/path/to/output.mp4"},
		},
	}

	// Generate plan
	p := planner.NewPlanner()
	plan, _ := p.Plan(context.Background(), spec, nil)

	// Create executor
	exec := executor.NewExecutor(operators.GlobalRegistry())

	// Execute with progress callback
	opts := &executor.ExecuteOptions{
		OnProgress: func(progress *executor.Progress) {
			percentage := float64(progress.Frame) / 1000.0 * 100.0
			fmt.Printf("Progress: %.1f%% (frame %d, %.1f fps, %.1fx speed)\n",
				percentage, progress.Frame, progress.FPS, progress.Speed)
		},
		OnLog: func(line string) {
			// Log FFmpeg output
			fmt.Printf("FFmpeg: %s\n", line)
		},
	}

	// Note: Actual execution would require FFmpeg to be installed
	// err := exec.Execute(context.Background(), plan, opts)

	_ = opts
	fmt.Println("Executor configured with callbacks")

	// Output:
	// Executor configured with callbacks
}

// ExampleCommandBuilder demonstrates building FFmpeg commands
func ExampleCommandBuilder() {
	operators.Register(&builtin.TrimOperator{})

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "video", Source: "/tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{
				Op:     "trim",
				Input:  "video",
				Output: "trimmed",
				Params: map[string]interface{}{
					"start":    "00:00:05",
					"duration": "00:00:15",
				},
			},
		},
		Outputs: []schemas.Output{
			{ID: "trimmed", Destination: "/tmp/output.mp4"},
		},
	}

	// Generate plan
	p := planner.NewPlanner()
	plan, _ := p.Plan(context.Background(), spec, nil)

	// Build command
	builder := executor.NewCommandBuilder(operators.GlobalRegistry())
	cmd, err := builder.Build(context.Background(), plan)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Command: %s\n", cmd.Args[0])
	fmt.Printf("Total arguments: %d\n", len(cmd.Args))

	// Output:
	// Command: ffmpeg
	// Total arguments: 7
}
