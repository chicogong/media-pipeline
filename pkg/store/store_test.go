package store

import (
	"context"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// testStore runs a suite of tests against any Store implementation
func testStore(t *testing.T, newStore func() Store) {
	t.Helper()

	t.Run("CreateJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "test-job-1",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
			Spec: &schemas.JobSpec{
				Inputs: []schemas.Input{{ID: "input1", Source: "test.mp4"}},
			},
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		// Verify job was created
		retrieved, err := s.GetJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("GetJob() failed: %v", err)
		}

		if retrieved.JobID != job.JobID {
			t.Errorf("Expected JobID %s, got %s", job.JobID, retrieved.JobID)
		}
		if retrieved.Status != schemas.JobStatePending {
			t.Errorf("Expected status pending, got %s", retrieved.Status)
		}
	})

	t.Run("CreateDuplicateJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "duplicate-job",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("First CreateJob() failed: %v", err)
		}

		// Try to create same job again
		err = s.CreateJob(ctx, job)
		if err != ErrJobExists {
			t.Errorf("Expected ErrJobExists, got %v", err)
		}
	})

	t.Run("GetJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "get-job-test",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		retrieved, err := s.GetJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("GetJob() failed: %v", err)
		}

		if retrieved.JobID != job.JobID {
			t.Errorf("Job ID mismatch")
		}
	})

	t.Run("GetNonExistentJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		_, err := s.GetJob(ctx, "nonexistent")
		if err != ErrJobNotFound {
			t.Errorf("Expected ErrJobNotFound, got %v", err)
		}
	})

	t.Run("UpdateJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "update-job-test",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		// Update job status
		job.Status = schemas.JobStateProcessing
		job.Updated = time.Now()
		err = s.UpdateJob(ctx, job)
		if err != nil {
			t.Fatalf("UpdateJob() failed: %v", err)
		}

		// Verify update
		retrieved, err := s.GetJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("GetJob() failed: %v", err)
		}

		if retrieved.Status != schemas.JobStateProcessing {
			t.Errorf("Expected status processing, got %s", retrieved.Status)
		}
	})

	t.Run("UpdateJobStatus", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "status-update-test",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		// Update status with progress
		progress := &schemas.Progress{
			OverallPercent: 50.0,
			CurrentStep:    "processing",
		}

		err = s.UpdateJobStatus(ctx, job.JobID, schemas.JobStateProcessing, progress)
		if err != nil {
			t.Fatalf("UpdateJobStatus() failed: %v", err)
		}

		// Verify update
		retrieved, err := s.GetJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("GetJob() failed: %v", err)
		}

		if retrieved.Status != schemas.JobStateProcessing {
			t.Errorf("Expected status processing, got %s", retrieved.Status)
		}
		if retrieved.Progress == nil {
			t.Fatal("Expected progress to be set")
		}
		if retrieved.Progress.OverallPercent != 50.0 {
			t.Errorf("Expected progress 50%%, got %.1f%%", retrieved.Progress.OverallPercent)
		}
	})

	t.Run("UpdateJobError", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "error-update-test",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		// Update with error
		errorInfo := &schemas.ErrorInfo{
			Code:      "FFMPEG_ERROR",
			Message:   "FFmpeg execution failed",
			Retryable: true,
		}

		err = s.UpdateJobError(ctx, job.JobID, errorInfo)
		if err != nil {
			t.Fatalf("UpdateJobError() failed: %v", err)
		}

		// Verify error was recorded
		retrieved, err := s.GetJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("GetJob() failed: %v", err)
		}

		if retrieved.Error == nil {
			t.Fatal("Expected error to be set")
		}
		if retrieved.Error.Code != "FFMPEG_ERROR" {
			t.Errorf("Expected error code FFMPEG_ERROR, got %s", retrieved.Error.Code)
		}
	})

	t.Run("DeleteJob", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()
		job := &Job{
			JobID:   "delete-job-test",
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
		}

		err := s.CreateJob(ctx, job)
		if err != nil {
			t.Fatalf("CreateJob() failed: %v", err)
		}

		// Delete job
		err = s.DeleteJob(ctx, job.JobID)
		if err != nil {
			t.Fatalf("DeleteJob() failed: %v", err)
		}

		// Verify job is deleted
		_, err = s.GetJob(ctx, job.JobID)
		if err != ErrJobNotFound {
			t.Errorf("Expected ErrJobNotFound after delete, got %v", err)
		}
	})

	t.Run("ListJobs", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()

		// Create multiple jobs
		jobs := []*Job{
			{JobID: "list-1", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStatePending},
			{JobID: "list-2", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStateProcessing},
			{JobID: "list-3", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStateCompleted},
		}

		for _, job := range jobs {
			if err := s.CreateJob(ctx, job); err != nil {
				t.Fatalf("CreateJob() failed: %v", err)
			}
		}

		// List all jobs
		filter := &ListFilter{}
		listed, err := s.ListJobs(ctx, filter)
		if err != nil {
			t.Fatalf("ListJobs() failed: %v", err)
		}

		if len(listed) != 3 {
			t.Errorf("Expected 3 jobs, got %d", len(listed))
		}
	})

	t.Run("ListJobsWithFilter", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()

		// Create jobs with different statuses
		jobs := []*Job{
			{JobID: "filter-1", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStatePending},
			{JobID: "filter-2", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStatePending},
			{JobID: "filter-3", Created: time.Now(), Updated: time.Now(), Status: schemas.JobStateCompleted},
		}

		for _, job := range jobs {
			if err := s.CreateJob(ctx, job); err != nil {
				t.Fatalf("CreateJob() failed: %v", err)
			}
		}

		// Filter for pending jobs only
		filter := &ListFilter{
			Status: []schemas.JobState{schemas.JobStatePending},
		}
		listed, err := s.ListJobs(ctx, filter)
		if err != nil {
			t.Fatalf("ListJobs() failed: %v", err)
		}

		if len(listed) != 2 {
			t.Errorf("Expected 2 pending jobs, got %d", len(listed))
		}

		for _, job := range listed {
			if job.Status != schemas.JobStatePending {
				t.Errorf("Expected pending job, got status %s", job.Status)
			}
		}
	})

	t.Run("ListJobsWithLimit", func(t *testing.T) {
		s := newStore()
		defer s.Close()

		ctx := context.Background()

		// Create multiple jobs
		for i := 0; i < 5; i++ {
			job := &Job{
				JobID:   "limit-" + string(rune(i+'0')),
				Created: time.Now(),
				Updated: time.Now(),
				Status:  schemas.JobStatePending,
			}
			if err := s.CreateJob(ctx, job); err != nil {
				t.Fatalf("CreateJob() failed: %v", err)
			}
		}

		// List with limit
		filter := &ListFilter{Limit: 3}
		listed, err := s.ListJobs(ctx, filter)
		if err != nil {
			t.Fatalf("ListJobs() failed: %v", err)
		}

		if len(listed) != 3 {
			t.Errorf("Expected 3 jobs (limit), got %d", len(listed))
		}
	})
}

// TestMemoryStore runs all tests against the memory store
func TestMemoryStore(t *testing.T) {
	testStore(t, func() Store {
		return NewMemoryStore()
	})
}
