package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/store"
)

func TestHandleHealth(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", resp["status"])
	}
}

func TestHandleCreateJob(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create request body
	reqBody := CreateJobRequest{
		Spec: &schemas.JobSpec{
			Inputs: []schemas.Input{
				{ID: "input1", Source: "file://test.mp4"},
			},
			Operations: []schemas.Operation{
				{Op: "trim", Input: "input1", Output: "trimmed"},
			},
			Outputs: []schemas.Output{
				{ID: "trimmed", Destination: "file://output.mp4"},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.HandleCreateJob(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp CreateJobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.JobID == "" {
		t.Error("Expected non-empty JobID")
	}
	if resp.Status != string(schemas.JobStatePending) {
		t.Errorf("Expected status pending, got %s", resp.Status)
	}

	// Wait a bit for background processing to start
	time.Sleep(100 * time.Millisecond)

	// Verify job was created in store
	job, err := s.GetJob(req.Context(), resp.JobID)
	if err != nil {
		t.Fatalf("Failed to get job from store: %v", err)
	}
	if job.JobID != resp.JobID {
		t.Errorf("Job ID mismatch")
	}
}

func TestHandleCreateJobInvalidRequest(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.HandleCreateJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleGetJob(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create a test job
	job := &store.Job{
		JobID:   "test-job-123",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStatePending,
		Spec:    &schemas.JobSpec{},
	}

	if err := s.CreateJob(nil, job); err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Get job via API
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/test-job-123", nil)
	w := httptest.NewRecorder()

	server.HandleGetJob(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp schemas.JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.JobID != job.JobID {
		t.Errorf("Expected JobID %s, got %s", job.JobID, resp.JobID)
	}
	if resp.Status != schemas.JobStatePending {
		t.Errorf("Expected status pending, got %s", resp.Status)
	}
}

func TestHandleGetJobNotFound(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/nonexistent", nil)
	w := httptest.NewRecorder()

	server.HandleGetJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleListJobs(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create test jobs
	for i := 0; i < 3; i++ {
		job := &store.Job{
			JobID:   "list-job-" + string(rune(i+'0')),
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
			Spec:    &schemas.JobSpec{},
		}
		if err := s.CreateJob(nil, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	// List jobs via API
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()

	server.HandleListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp []*schemas.JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(resp))
	}
}

func TestHandleListJobsWithFilter(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create jobs with different statuses
	statuses := []schemas.JobState{
		schemas.JobStatePending,
		schemas.JobStateProcessing,
		schemas.JobStateCompleted,
	}

	for i, status := range statuses {
		job := &store.Job{
			JobID:   "filter-job-" + string(rune(i+'0')),
			Created: time.Now(),
			Updated: time.Now(),
			Status:  status,
			Spec:    &schemas.JobSpec{},
		}
		if err := s.CreateJob(nil, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	// Filter for pending jobs only
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?status=pending", nil)
	w := httptest.NewRecorder()

	server.HandleListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp []*schemas.JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp) != 1 {
		t.Errorf("Expected 1 pending job, got %d", len(resp))
	}
	if len(resp) > 0 && resp[0].Status != schemas.JobStatePending {
		t.Errorf("Expected pending status, got %s", resp[0].Status)
	}
}

func TestHandleDeleteJob(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create a test job
	job := &store.Job{
		JobID:   "delete-job-123",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStateProcessing,
		Spec:    &schemas.JobSpec{},
	}

	if err := s.CreateJob(nil, job); err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Delete job via API
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/delete-job-123", nil)
	w := httptest.NewRecorder()

	server.HandleDeleteJob(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify job status was updated to cancelled
	updated, err := s.GetJob(nil, job.JobID)
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}
	if updated.Status != schemas.JobStateCancelled {
		t.Errorf("Expected status cancelled, got %s", updated.Status)
	}
}

func TestHandleDeleteJobTerminal(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()

	server := NewServer(s)
	defer server.Close()

	// Create a completed job
	job := &store.Job{
		JobID:   "terminal-job",
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStateCompleted,
		Spec:    &schemas.JobSpec{},
	}

	if err := s.CreateJob(nil, job); err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	// Try to delete
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/terminal-job", nil)
	w := httptest.NewRecorder()

	server.HandleDeleteJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
