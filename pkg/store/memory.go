package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// MemoryStore is an in-memory implementation of Store
// Thread-safe for concurrent access
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*Job),
	}
}

// CreateJob creates a new job
func (m *MemoryStore) CreateJob(ctx context.Context, job *Job) error {
	if job.JobID == "" {
		return ErrInvalidJobID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.JobID]; exists {
		return ErrJobExists
	}

	// Deep copy to avoid external modifications
	jobCopy := m.copyJob(job)
	m.jobs[job.JobID] = jobCopy

	return nil
}

// GetJob retrieves a job by ID
func (m *MemoryStore) GetJob(ctx context.Context, jobID string) (*Job, error) {
	if jobID == "" {
		return nil, ErrInvalidJobID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, ErrJobNotFound
	}

	// Return a copy to prevent external modifications
	return m.copyJob(job), nil
}

// UpdateJob updates an existing job
func (m *MemoryStore) UpdateJob(ctx context.Context, job *Job) error {
	if job.JobID == "" {
		return ErrInvalidJobID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[job.JobID]; !exists {
		return ErrJobNotFound
	}

	// Update timestamp
	job.Updated = time.Now()

	// Deep copy and store
	jobCopy := m.copyJob(job)
	m.jobs[job.JobID] = jobCopy

	return nil
}

// DeleteJob deletes a job by ID
func (m *MemoryStore) DeleteJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.jobs[jobID]; !exists {
		return ErrJobNotFound
	}

	delete(m.jobs, jobID)
	return nil
}

// ListJobs lists jobs with optional filtering
func (m *MemoryStore) ListJobs(ctx context.Context, filter *ListFilter) ([]*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all jobs
	var jobs []*Job
	for _, job := range m.jobs {
		if m.matchesFilter(job, filter) {
			jobs = append(jobs, m.copyJob(job))
		}
	}

	// Sort jobs
	m.sortJobs(jobs, filter)

	// Apply pagination
	return m.paginateJobs(jobs, filter), nil
}

// UpdateJobStatus updates job status and progress
func (m *MemoryStore) UpdateJobStatus(ctx context.Context, jobID string, status schemas.JobState, progress *schemas.Progress) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return ErrJobNotFound
	}

	// Update status
	job.Status = status
	job.Updated = time.Now()

	// Update progress
	if progress != nil {
		job.Progress = &schemas.Progress{
			OverallPercent:      progress.OverallPercent,
			CurrentStep:         progress.CurrentStep,
			StepProgress:        progress.StepProgress,
			EstimatedCompletion: progress.EstimatedCompletion,
		}
	}

	// Update timestamps based on status
	now := time.Now()
	if status == schemas.JobStateProcessing && job.StartedAt == nil {
		job.StartedAt = &now
	}
	if status == schemas.JobStateCompleted || status == schemas.JobStateFailed || status == schemas.JobStateCancelled {
		if job.CompletedAt == nil {
			job.CompletedAt = &now
		}
	}

	return nil
}

// UpdateJobError records an error for a job
func (m *MemoryStore) UpdateJobError(ctx context.Context, jobID string, err *schemas.ErrorInfo) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return ErrJobNotFound
	}

	// Copy error info
	if err != nil {
		job.Error = &schemas.ErrorInfo{
			Code:           err.Code,
			Message:        err.Message,
			Details:        err.Details,
			FFmpegStderr:   err.FFmpegStderr,
			FFmpegExitCode: err.FFmpegExitCode,
			StackTrace:     err.StackTrace,
			Retryable:      err.Retryable,
			RetryAfter:     err.RetryAfter,
		}
	}

	job.Updated = time.Now()

	return nil
}

// Close closes the store (no-op for memory store)
func (m *MemoryStore) Close() error {
	return nil
}

// Helper methods

func (m *MemoryStore) copyJob(job *Job) *Job {
	if job == nil {
		return nil
	}

	copy := &Job{
		JobID:       job.JobID,
		Created:     job.Created,
		Updated:     job.Updated,
		Status:      job.Status,
		RetryCount:  job.RetryCount,
		WorkerID:    job.WorkerID,
		Spec:        job.Spec,
		Plan:        job.Plan,
		OutputFiles: job.OutputFiles,
	}

	// Copy pointers
	if job.StartedAt != nil {
		t := *job.StartedAt
		copy.StartedAt = &t
	}
	if job.CompletedAt != nil {
		t := *job.CompletedAt
		copy.CompletedAt = &t
	}

	// Copy Progress
	if job.Progress != nil {
		copy.Progress = &schemas.Progress{
			OverallPercent:      job.Progress.OverallPercent,
			CurrentStep:         job.Progress.CurrentStep,
			StepProgress:        job.Progress.StepProgress,
			EstimatedCompletion: job.Progress.EstimatedCompletion,
		}
	}

	// Copy Error
	if job.Error != nil {
		copy.Error = &schemas.ErrorInfo{
			Code:           job.Error.Code,
			Message:        job.Error.Message,
			Details:        job.Error.Details,
			FFmpegStderr:   job.Error.FFmpegStderr,
			FFmpegExitCode: job.Error.FFmpegExitCode,
			StackTrace:     job.Error.StackTrace,
			Retryable:      job.Error.Retryable,
			RetryAfter:     job.Error.RetryAfter,
		}
	}

	return copy
}

func (m *MemoryStore) matchesFilter(job *Job, filter *ListFilter) bool {
	if filter == nil {
		return true
	}

	// Status filter
	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if job.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Time range filters
	if filter.CreatedAfter != nil && job.Created.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && job.Created.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

func (m *MemoryStore) sortJobs(jobs []*Job, filter *ListFilter) {
	if filter == nil || filter.SortBy == "" {
		// Default sort by created time descending
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Created.After(jobs[j].Created)
		})
		return
	}

	descending := filter.SortOrder == "desc"

	switch filter.SortBy {
	case "created":
		sort.Slice(jobs, func(i, j int) bool {
			if descending {
				return jobs[i].Created.After(jobs[j].Created)
			}
			return jobs[i].Created.Before(jobs[j].Created)
		})
	case "updated":
		sort.Slice(jobs, func(i, j int) bool {
			if descending {
				return jobs[i].Updated.After(jobs[j].Updated)
			}
			return jobs[i].Updated.Before(jobs[j].Updated)
		})
	case "status":
		sort.Slice(jobs, func(i, j int) bool {
			if descending {
				return jobs[i].Status > jobs[j].Status
			}
			return jobs[i].Status < jobs[j].Status
		})
	}
}

func (m *MemoryStore) paginateJobs(jobs []*Job, filter *ListFilter) []*Job {
	if filter == nil {
		return jobs
	}

	// Apply offset
	if filter.Offset > 0 {
		if filter.Offset >= len(jobs) {
			return []*Job{}
		}
		jobs = jobs[filter.Offset:]
	}

	// Apply limit
	if filter.Limit > 0 && filter.Limit < len(jobs) {
		jobs = jobs[:filter.Limit]
	}

	return jobs
}
