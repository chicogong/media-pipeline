package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/storage"
)

// Executor executes processing plans using FFmpeg
type Executor struct {
	builder        *CommandBuilder
	parser         *ProgressParser
	storageManager *StorageManager
}

// NewExecutor creates a new executor
func NewExecutor(registry *operators.Registry) *Executor {
	return &Executor{
		builder:        NewCommandBuilder(registry),
		parser:         NewProgressParser(),
		storageManager: NewStorageManager(),
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

	// Create temporary directory for downloaded inputs and intermediate outputs
	tempDir, err := os.MkdirTemp("", "media-pipeline-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		// Cleanup temp directory (ignore errors)
		e.storageManager.CleanupTempDir(tempDir)
	}()

	// Download remote inputs to local temp directory
	inputMap, err := e.storageManager.PrepareInputs(ctx, plan, tempDir)
	if err != nil {
		return fmt.Errorf("failed to prepare inputs: %w", err)
	}

	// Prepare outputs: generate local temp paths and store original destinations
	outputFiles := make(map[string]string) // node.ID -> local temp path
	origDestURIs := make(map[string]string) // node.ID -> original destination URI

	for _, node := range plan.Nodes {
		if node.Type == "output" {
			// Generate local temp path for this output
			// Use node ID as base name, preserve extension from original URI if possible
			origURI := node.DestURI
			origDestURIs[node.ID] = origURI

			// Try to extract filename from URI
			var filename string
			if origURI != "" {
				// Parse URI to get path component
				_, path, err := storage.ParseURI(origURI)
				if err == nil && path != "" {
					filename = filepath.Base(path)
				}
			}
			if filename == "" || filename == "." || filename == "/" {
				filename = fmt.Sprintf("output-%s", node.ID)
			}

			localPath := filepath.Join(tempDir, filename)
			outputFiles[node.ID] = localPath
		}
	}

	// Replace remote URIs with local paths in a copy of the plan
	// We need to modify the plan so CommandBuilder uses local paths
	// Create a shallow copy of nodes to avoid modifying the original plan
	nodesCopy := make([]*schemas.PlanNode, len(plan.Nodes))
	for i, node := range plan.Nodes {
		nodeCopy := *node // shallow copy
		if nodeCopy.Type == "input" {
			// Replace remote URI with local path
			if localPath, ok := inputMap[nodeCopy.SourceURI]; ok {
				nodeCopy.SourceURI = localPath
			}
		} else if nodeCopy.Type == "output" {
			// Replace destination URI with local temp path
			if localPath, ok := outputFiles[nodeCopy.ID]; ok {
				nodeCopy.DestURI = localPath
			}
		}
		nodesCopy[i] = &nodeCopy
	}
	// Create a plan copy with modified nodes
	planCopy := &schemas.ProcessingPlan{
		PlanID:          plan.PlanID,
		JobID:           plan.JobID,
		CreatedAt:       plan.CreatedAt,
		Nodes:           nodesCopy,
		Edges:           plan.Edges,
		ExecutionOrder:  plan.ExecutionOrder,
		ExecutionStages: plan.ExecutionStages,
		ResourceEstimate: plan.ResourceEstimate,
		FFmpegVersion:   plan.FFmpegVersion,
		Commands:        plan.Commands,
	}

	// Build FFmpeg command using the modified plan
	cmd, err := e.builder.Build(ctx, planCopy)
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Set total duration for progress tracking
	if plan.ResourceEstimate != nil {
		e.parser.SetTotalDuration(plan.ResourceEstimate.TotalDuration)
	}

	// Execute command
	if err := e.executeCommand(ctx, cmd, opts); err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Upload outputs to remote destinations
	for nodeID, localPath := range outputFiles {
		destURI := origDestURIs[nodeID]
		if destURI == "" {
			// No destination specified, output was written locally only
			continue
		}
		if err := e.storageManager.UploadOutput(ctx, localPath, destURI); err != nil {
			return fmt.Errorf("failed to upload output %s: %w", nodeID, err)
		}
	}

	return nil
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
