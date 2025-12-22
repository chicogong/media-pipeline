package store_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/store"
)

// Example_basic demonstrates basic store operations
func Example_basic() {
	// Create a new memory store
	s := store.NewMemoryStore()
	defer s.Close()

	ctx := context.Background()

	// Create a new job
	job := &store.Job{
		JobID:   "example-job-1",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStatePending,
		Spec: &schemas.JobSpec{
			Inputs: []schemas.Input{
				{ID: "input1", Source: "s3://bucket/input.mp4"},
			},
			Operations: []schemas.Operation{
				{Op: "trim", Input: "input1", Output: "trimmed", Params: map[string]interface{}{"start": "00:00:10", "duration": "00:05:00"}},
			},
			Outputs: []schemas.Output{
				{ID: "trimmed", Destination: "s3://bucket/output.mp4"},
			},
		},
	}

	if err := s.CreateJob(ctx, job); err != nil {
		log.Fatal(err)
	}

	// Retrieve the job
	retrieved, err := s.GetJob(ctx, job.JobID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Job ID: %s\n", retrieved.JobID)
	fmt.Printf("Status: %s\n", retrieved.Status)
	fmt.Printf("Inputs: %d\n", len(retrieved.Spec.Inputs))
}

// Example_updateStatus demonstrates updating job status
func Example_updateStatus() {
	s := store.NewMemoryStore()
	defer s.Close()

	ctx := context.Background()

	// Create a job
	job := &store.Job{
		JobID:   "status-update-job",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStatePending,
	}

	if err := s.CreateJob(ctx, job); err != nil {
		log.Fatal(err)
	}

	// Update status to processing with progress
	progress := &schemas.Progress{
		OverallPercent: 50.0,
		CurrentStep:    "processing",
	}

	if err := s.UpdateJobStatus(ctx, job.JobID, schemas.JobStateProcessing, progress); err != nil {
		log.Fatal(err)
	}

	// Retrieve updated job
	updated, err := s.GetJob(ctx, job.JobID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Status: %s\n", updated.Status)
	fmt.Printf("Progress: %.1f%%\n", updated.Progress.OverallPercent)
}

// Example_errorHandling demonstrates error recording
func Example_errorHandling() {
	s := store.NewMemoryStore()
	defer s.Close()

	ctx := context.Background()

	// Create a job
	job := &store.Job{
		JobID:   "error-job",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStateProcessing,
	}

	if err := s.CreateJob(ctx, job); err != nil {
		log.Fatal(err)
	}

	// Record an error
	errorInfo := &schemas.ErrorInfo{
		Code:      "FFMPEG_ERROR",
		Message:   "FFmpeg execution failed",
		Retryable: true,
	}

	if err := s.UpdateJobError(ctx, job.JobID, errorInfo); err != nil {
		log.Fatal(err)
	}

	// Retrieve job with error
	failed, err := s.GetJob(ctx, job.JobID)
	if err != nil {
		log.Fatal(err)
	}

	if failed.Error != nil {
		fmt.Printf("Error Code: %s\n", failed.Error.Code)
		fmt.Printf("Error Message: %s\n", failed.Error.Message)
		fmt.Printf("Retryable: %v\n", failed.Error.Retryable)
	}
}

// Example_listJobs demonstrates listing and filtering jobs
func Example_listJobs() {
	s := store.NewMemoryStore()
	defer s.Close()

	ctx := context.Background()

	// Create multiple jobs
	statuses := []schemas.JobState{
		schemas.JobStatePending,
		schemas.JobStateProcessing,
		schemas.JobStateCompleted,
		schemas.JobStateFailed,
	}

	for i, status := range statuses {
		job := &store.Job{
			JobID:   fmt.Sprintf("job-%d", i+1),
			Created: time.Now(),
			Updated: time.Now(),
			Status:  status,
		}
		if err := s.CreateJob(ctx, job); err != nil {
			log.Fatal(err)
		}
	}

	// List all jobs
	allJobs, err := s.ListJobs(ctx, &store.ListFilter{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total jobs: %d\n", len(allJobs))

	// Filter for pending and processing jobs
	activeFilter := &store.ListFilter{
		Status: []schemas.JobState{
			schemas.JobStatePending,
			schemas.JobStateProcessing,
		},
	}
	activeJobs, err := s.ListJobs(ctx, activeFilter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Active jobs: %d\n", len(activeJobs))

	// List with pagination
	paginatedFilter := &store.ListFilter{
		Limit:  2,
		Offset: 0,
	}
	firstPage, err := s.ListJobs(ctx, paginatedFilter)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("First page: %d jobs\n", len(firstPage))
}

// Example_concurrentAccess demonstrates thread-safe operations
func Example_concurrentAccess() {
	s := store.NewMemoryStore()
	defer s.Close()

	ctx := context.Background()

	// Create a job
	job := &store.Job{
		JobID:   "concurrent-job",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStatePending,
	}

	if err := s.CreateJob(ctx, job); err != nil {
		log.Fatal(err)
	}

	// Simulate concurrent updates (in real app, from different goroutines)
	done := make(chan bool, 2)

	// Goroutine 1: Update status
	go func() {
		if err := s.UpdateJobStatus(ctx, job.JobID, schemas.JobStateProcessing, nil); err != nil {
			log.Printf("Update 1 failed: %v", err)
		}
		done <- true
	}()

	// Goroutine 2: Update progress
	go func() {
		progress := &schemas.Progress{
			OverallPercent: 25.0,
			CurrentStep:    "downloading",
		}
		if err := s.UpdateJobStatus(ctx, job.JobID, schemas.JobStateProcessing, progress); err != nil {
			log.Printf("Update 2 failed: %v", err)
		}
		done <- true
	}()

	// Wait for both updates
	<-done
	<-done

	// Retrieve final state
	final, err := s.GetJob(ctx, job.JobID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final status: %s\n", final.Status)
	if final.Progress != nil {
		fmt.Printf("Progress: %.1f%%\n", final.Progress.OverallPercent)
	}
}
