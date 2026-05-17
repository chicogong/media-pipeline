package store

import (
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

func TestJob_IsTerminal(t *testing.T) {
	terminalStates := []schemas.JobState{
		schemas.JobStateCompleted,
		schemas.JobStateFailed,
		schemas.JobStateCancelled,
	}
	for _, s := range terminalStates {
		j := &Job{Status: s}
		if !j.IsTerminal() {
			t.Errorf("IsTerminal() = false for state %q, want true", s)
		}
	}

	nonTerminalStates := []schemas.JobState{
		schemas.JobStatePending,
		schemas.JobStateValidating,
		schemas.JobStateProcessing,
	}
	for _, s := range nonTerminalStates {
		j := &Job{Status: s}
		if j.IsTerminal() {
			t.Errorf("IsTerminal() = true for state %q, want false", s)
		}
	}
}

func TestJob_IsPending(t *testing.T) {
	j := &Job{Status: schemas.JobStatePending}
	if !j.IsPending() {
		t.Errorf("IsPending() = false for Pending state, want true")
	}

	j2 := &Job{Status: schemas.JobStateProcessing}
	if j2.IsPending() {
		t.Errorf("IsPending() = true for Processing state, want false")
	}
}

func TestJob_IsProcessing(t *testing.T) {
	processingStates := []schemas.JobState{
		schemas.JobStateValidating,
		schemas.JobStatePlanning,
		schemas.JobStateDownloadingInputs,
		schemas.JobStateProcessing,
		schemas.JobStateUploadingOutputs,
	}
	for _, s := range processingStates {
		j := &Job{Status: s}
		if !j.IsProcessing() {
			t.Errorf("IsProcessing() = false for state %q, want true", s)
		}
	}

	notProcessingStates := []schemas.JobState{
		schemas.JobStatePending,
		schemas.JobStateCompleted,
	}
	for _, s := range notProcessingStates {
		j := &Job{Status: s}
		if j.IsProcessing() {
			t.Errorf("IsProcessing() = true for state %q, want false", s)
		}
	}
}

func TestJob_ToJobStatus(t *testing.T) {
	created := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	progress := &schemas.Progress{}

	j := &Job{
		JobID:    "job-abc",
		Created:  created,
		Updated:  updated,
		Status:   schemas.JobStateProcessing,
		Progress: progress,
	}

	js := j.ToJobStatus()
	if js == nil {
		t.Fatal("ToJobStatus() returned nil")
	}
	if js.JobID != j.JobID {
		t.Errorf("JobID = %q, want %q", js.JobID, j.JobID)
	}
	if js.Status != j.Status {
		t.Errorf("Status = %q, want %q", js.Status, j.Status)
	}
	if !js.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", js.CreatedAt, created)
	}
	if !js.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %v, want %v", js.UpdatedAt, updated)
	}
	if js.Progress != progress {
		t.Errorf("Progress pointer mismatch: got %p, want %p", js.Progress, progress)
	}
}
