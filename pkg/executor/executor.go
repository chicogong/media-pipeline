package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Executor executes processing plans using FFmpeg
type Executor struct {
	builder *CommandBuilder
	parser  *ProgressParser
}

// NewExecutor creates a new executor
func NewExecutor(registry *operators.Registry) *Executor {
	return &Executor{
		builder: NewCommandBuilder(registry),
		parser:  NewProgressParser(),
	}
}

// ExecuteOptions contains options for execution
type ExecuteOptions struct {
	// WorkDir is the working directory for execution
	WorkDir string

	// OnProgress is called for progress updates
	OnProgress func(*Progress)

	// OnLog is called for FFmpeg log output
	OnLog func(string)
}

// Execute executes a processing plan
func (e *Executor) Execute(ctx context.Context, plan *schemas.ProcessingPlan, opts *ExecuteOptions) error {
	if opts == nil {
		opts = &ExecuteOptions{}
	}

	// Build FFmpeg command
	cmd, err := e.builder.Build(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Set total duration for progress tracking
	if plan.ResourceEstimate != nil {
		e.parser.SetTotalDuration(plan.ResourceEstimate.TotalDuration)
	}

	// Execute command
	return e.executeCommand(ctx, cmd, opts)
}

// executeCommand executes an FFmpeg command
func (e *Executor) executeCommand(ctx context.Context, cmd *Command, opts *ExecuteOptions) error {
	// Create exec.Cmd
	execCmd := exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)

	if cmd.WorkDir != "" {
		execCmd.Dir = cmd.WorkDir
	} else if opts.WorkDir != "" {
		execCmd.Dir = opts.WorkDir
	}

	// FFmpeg writes progress to stderr
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Also capture stdout
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream stderr for progress
	stderrDone := make(chan error, 1)
	go func() {
		stderrDone <- e.streamStderr(stderr, opts)
	}()

	// Stream stdout for logs
	stdoutDone := make(chan error, 1)
	go func() {
		stdoutDone <- e.streamStdout(stdout, opts)
	}()

	// Wait for command to complete
	cmdErr := execCmd.Wait()

	// Wait for streaming to finish
	<-stderrDone
	<-stdoutDone

	if cmdErr != nil {
		return fmt.Errorf("ffmpeg execution failed: %w", cmdErr)
	}

	return nil
}

// streamStderr reads and processes stderr output
func (e *Executor) streamStderr(reader io.Reader, opts *ExecuteOptions) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse progress
		progress := e.parser.ParseLine(line)
		if progress != nil && opts.OnProgress != nil {
			opts.OnProgress(progress)
		}

		// Also send to log handler
		if opts.OnLog != nil {
			opts.OnLog(line)
		}
	}

	return scanner.Err()
}

// streamStdout reads and processes stdout output
func (e *Executor) streamStdout(reader io.Reader, opts *ExecuteOptions) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if opts.OnLog != nil {
			opts.OnLog(line)
		}
	}

	return scanner.Err()
}

// BuildCommand builds an FFmpeg command from a plan without executing it
func (e *Executor) BuildCommand(ctx context.Context, plan *schemas.ProcessingPlan) (*Command, error) {
	return e.builder.Build(ctx, plan)
}

// ExecutionResult contains the result of executing a plan
type ExecutionResult struct {
	Duration   time.Duration
	Success    bool
	Error      error
	FinalFrame int
}
