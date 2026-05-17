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

// --- HandleHealth ---

// TestHandleHealthMethodNotAllowed verifies that non-GET requests to /health return 405.
func TestHandleHealthMethodNotAllowed(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	server.HandleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// --- HandleGetJob ---

// TestHandleGetJobMissingID verifies that a path with no job ID returns 400.
func TestHandleGetJobMissingID(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	// Path equal to the prefix — no trailing ID segment.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/", nil)
	w := httptest.NewRecorder()
	server.HandleGetJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing job ID, got %d", w.Code)
	}
}

// TestHandleGetJobMethodNotAllowed verifies that non-GET requests return 405.
func TestHandleGetJobMethodNotAllowed(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/some-id", nil)
	w := httptest.NewRecorder()
	server.HandleGetJob(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// --- HandleDeleteJob ---

// TestHandleDeleteJobMissingID verifies that a DELETE with no job ID returns 400.
func TestHandleDeleteJobMissingID(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/", nil)
	w := httptest.NewRecorder()
	server.HandleDeleteJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing job ID, got %d", w.Code)
	}
}

// TestHandleDeleteJobNotFound verifies that deleting a non-existent job returns 404.
func TestHandleDeleteJobNotFound(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/jobs/does-not-exist", nil)
	w := httptest.NewRecorder()
	server.HandleDeleteJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent job, got %d", w.Code)
	}
}

// TestHandleDeleteJobMethodNotAllowed verifies that non-DELETE requests return 405.
func TestHandleDeleteJobMethodNotAllowed(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/some-id", nil)
	w := httptest.NewRecorder()
	server.HandleDeleteJob(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// --- HandleListJobs ---

// TestHandleListJobsMethodNotAllowed verifies that non-GET requests return 405.
func TestHandleListJobsMethodNotAllowed(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	server.HandleListJobs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// TestHandleListJobsWithLimitAndOffset exercises parseListFilter limit/offset parsing.
func TestHandleListJobsWithLimitAndOffset(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	// Seed five jobs.
	for i := 0; i < 5; i++ {
		job := &store.Job{
			JobID:   "paged-job-" + string(rune('0'+i)),
			Created: time.Now(),
			Updated: time.Now(),
			Status:  schemas.JobStatePending,
			Spec:    &schemas.JobSpec{},
		}
		if err := s.CreateJob(nil, job); err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?limit=2&offset=1", nil)
	w := httptest.NewRecorder()
	server.HandleListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp []*schemas.JobStatus
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	// With offset=1 and limit=2 we expect at most 2 results.
	if len(resp) > 2 {
		t.Errorf("Expected at most 2 jobs with limit=2, got %d", len(resp))
	}
}

// TestHandleListJobsInvalidLimit exercises parseListFilter with a non-numeric limit.
func TestHandleListJobsInvalidLimit(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	// Non-numeric limit: fmt.Sscanf will silently leave limit as 0 (no limit).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?limit=abc", nil)
	w := httptest.NewRecorder()
	server.HandleListJobs(w, req)

	// Should still return 200 (no hard error, filter just has limit=0).
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for invalid limit, got %d", w.Code)
	}
}

// TestHandleListJobsInvalidOffset exercises parseListFilter with a non-numeric offset.
func TestHandleListJobsInvalidOffset(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?offset=xyz", nil)
	w := httptest.NewRecorder()
	server.HandleListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for invalid offset, got %d", w.Code)
	}
}

// TestHandleListJobsMultipleStatuses verifies that multiple status query values are handled
// (the current implementation uses q.Get which only reads the first value, so we just
// confirm the endpoint doesn't error out).
func TestHandleListJobsMultipleStatuses(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs?status=pending&status=processing", nil)
	w := httptest.NewRecorder()
	server.HandleListJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for multi-status query, got %d", w.Code)
	}
}

// --- HandleCreateJob ---

// TestHandleCreateJobMissingSpec verifies that valid JSON with a nil spec returns 400.
func TestHandleCreateJobMissingSpec(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	body, _ := json.Marshal(map[string]interface{}{"spec": nil})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.HandleCreateJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing spec, got %d", w.Code)
	}
}

// TestHandleCreateJobInvalidSpec verifies that a spec that fails validation returns 400.
// An operation that references a non-existent input ID is invalid.
func TestHandleCreateJobInvalidSpec(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	// Spec has an operation referencing "no-such-input" which does not exist.
	reqBody := CreateJobRequest{
		Spec: &schemas.JobSpec{
			Inputs: []schemas.Input{
				{ID: "src", Source: "file://video.mp4"},
			},
			Operations: []schemas.Operation{
				{Op: "trim", Input: "no-such-input", Output: "trimmed"},
			},
			Outputs: []schemas.Output{
				{ID: "trimmed", Destination: "file://out.mp4"},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.HandleCreateJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid spec, got %d", w.Code)
	}
}

// TestHandleCreateJobMethodNotAllowed verifies that non-POST requests return 405.
func TestHandleCreateJobMethodNotAllowed(t *testing.T) {
	s := store.NewMemoryStore()
	defer s.Close()
	server := NewServer(s)
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	server.HandleCreateJob(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}
