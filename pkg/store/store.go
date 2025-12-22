// Package store provides job state persistence
package store

import (
	"context"
	"errors"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

var (
	// ErrJobNotFound is returned when a job does not exist
	ErrJobNotFound = errors.New("job not found")

	// ErrJobExists is returned when attempting to create a job that already exists
	ErrJobExists = errors.New("job already exists")

	// ErrInvalidJobID is returned for invalid job IDs
	ErrInvalidJobID = errors.New("invalid job ID")
)

// Store is the interface for job state persistence
type Store interface {
	// CreateJob creates a new job with initial state
	CreateJob(ctx context.Context, job *Job) error

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// UpdateJob updates an existing job
	UpdateJob(ctx context.Context, job *Job) error

	// DeleteJob deletes a job by ID
	DeleteJob(ctx context.Context, jobID string) error

	// ListJobs lists jobs with optional filtering
	ListJobs(ctx context.Context, filter *ListFilter) ([]*Job, error)

	// UpdateJobStatus updates job status and progress
	UpdateJobStatus(ctx context.Context, jobID string, status schemas.JobState, progress *schemas.Progress) error

	// UpdateJobError records an error for a job
	UpdateJobError(ctx context.Context, jobID string, err *schemas.ErrorInfo) error

	// Close closes the store and releases resources
	Close() error
}

// Job represents a complete job record in the store
type Job struct {
	// Core identifiers
	JobID   string    `json:"job_id"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`

	// Job specification
	Spec *schemas.JobSpec `json:"spec"`

	// Processing plan
	Plan *schemas.ProcessingPlan `json:"plan,omitempty"`

	// Current status
	Status    schemas.JobState    `json:"status"`
	Progress  *schemas.Progress   `json:"progress,omitempty"`
	Error     *schemas.ErrorInfo  `json:"error,omitempty"`
	StartedAt *time.Time          `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`

	// Outputs
	OutputFiles []schemas.OutputFile `json:"output_files,omitempty"`

	// Metadata
	RetryCount int `json:"retry_count"`
	WorkerID   string `json:"worker_id,omitempty"`
}

// ListFilter defines filtering criteria for listing jobs
type ListFilter struct {
	// Status filters
	Status []schemas.JobState `json:"status,omitempty"`

	// Time range filters
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Max results (0 = no limit)
	Offset int `json:"offset,omitempty"` // Skip N results

	// Sorting
	SortBy    string `json:"sort_by,omitempty"`    // Field to sort by
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"
}

// ToJobStatus converts a Job to schemas.JobStatus
func (j *Job) ToJobStatus() *schemas.JobStatus {
	return &schemas.JobStatus{
		JobID:       j.JobID,
		Status:      j.Status,
		Progress:    j.Progress,
		Error:       j.Error,
		CreatedAt:   j.Created,
		UpdatedAt:   j.Updated,
		StartedAt:   j.StartedAt,
		CompletedAt: j.CompletedAt,
		OutputFiles: j.OutputFiles,
	}
}

// IsTerminal returns true if the job is in a terminal state
func (j *Job) IsTerminal() bool {
	return j.Status == schemas.JobStateCompleted ||
		j.Status == schemas.JobStateFailed ||
		j.Status == schemas.JobStateCancelled
}

// IsPending returns true if the job is pending execution
func (j *Job) IsPending() bool {
	return j.Status == schemas.JobStatePending
}

// IsProcessing returns true if the job is currently being processed
func (j *Job) IsProcessing() bool {
	return j.Status == schemas.JobStateValidating ||
		j.Status == schemas.JobStatePlanning ||
		j.Status == schemas.JobStateDownloadingInputs ||
		j.Status == schemas.JobStateProcessing ||
		j.Status == schemas.JobStateUploadingOutputs
}
