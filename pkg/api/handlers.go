// Package api provides HTTP handlers for the media pipeline API
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chicogong/media-pipeline/pkg/compiler/validator"
	"github.com/chicogong/media-pipeline/pkg/executor"
	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/planner"
	"github.com/chicogong/media-pipeline/pkg/prober"
	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/store"
)

// Server holds the API server dependencies
type Server struct {
	store     store.Store
	prober    *prober.Prober
	planner   *planner.Planner
	executor  *executor.Executor
	validator *validator.Validator
}

// NewServer creates a new API server
func NewServer(s store.Store) *Server {
	registry := operators.GlobalRegistry()
	return &Server{
		store:     s,
		prober:    prober.NewProber(),
		planner:   planner.NewPlanner(),
		executor:  executor.NewExecutor(registry),
		validator: &validator.Validator{},
	}
}

// CreateJobRequest represents the request body for creating a job
type CreateJobRequest struct {
	Spec *schemas.JobSpec `json:"spec"`
}

// CreateJobResponse represents the response for creating a job
type CreateJobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// HandleCreateJob handles POST /api/v1/jobs
func (s *Server) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Parse request body
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.Spec == nil {
		s.sendError(w, http.StatusBadRequest, "missing_spec", "Job specification is required")
		return
	}

	// Validate JobSpec
	if err := s.validator.Validate(req.Spec); err != nil {
		s.sendError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("Invalid job specification: %v", err))
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	// Create job in store
	job := &store.Job{
		JobID:   jobID,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  schemas.JobStatePending,
		Spec:    req.Spec,
	}

	ctx := r.Context()
	if err := s.store.CreateJob(ctx, job); err != nil {
		s.sendError(w, http.StatusInternalServerError, "store_error", fmt.Sprintf("Failed to create job: %v", err))
		return
	}

	// Start job processing in background
	go s.processJob(context.Background(), jobID)

	// Send response
	resp := CreateJobResponse{
		JobID:     jobID,
		Status:    string(schemas.JobStatePending),
		CreatedAt: job.Created,
	}

	s.sendJSON(w, http.StatusCreated, resp)
}

// HandleGetJob handles GET /api/v1/jobs/{id}
func (s *Server) HandleGetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Extract job ID from URL path
	jobID := extractJobID(r.URL.Path)
	if jobID == "" {
		s.sendError(w, http.StatusBadRequest, "invalid_job_id", "Job ID is required")
		return
	}

	// Get job from store
	ctx := r.Context()
	job, err := s.store.GetJob(ctx, jobID)
	if err == store.ErrJobNotFound {
		s.sendError(w, http.StatusNotFound, "job_not_found", fmt.Sprintf("Job %s not found", jobID))
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "store_error", fmt.Sprintf("Failed to get job: %v", err))
		return
	}

	// Convert to JobStatus and send response
	s.sendJSON(w, http.StatusOK, job.ToJobStatus())
}

// HandleListJobs handles GET /api/v1/jobs
func (s *Server) HandleListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Parse query parameters
	filter := s.parseListFilter(r)

	// List jobs from store
	ctx := r.Context()
	jobs, err := s.store.ListJobs(ctx, filter)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "store_error", fmt.Sprintf("Failed to list jobs: %v", err))
		return
	}

	// Convert to JobStatus array
	statuses := make([]*schemas.JobStatus, len(jobs))
	for i, job := range jobs {
		statuses[i] = job.ToJobStatus()
	}

	s.sendJSON(w, http.StatusOK, statuses)
}

// HandleDeleteJob handles DELETE /api/v1/jobs/{id}
func (s *Server) HandleDeleteJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	// Extract job ID
	jobID := extractJobID(r.URL.Path)
	if jobID == "" {
		s.sendError(w, http.StatusBadRequest, "invalid_job_id", "Job ID is required")
		return
	}

	ctx := r.Context()

	// Get job to check if it exists and can be cancelled
	job, err := s.store.GetJob(ctx, jobID)
	if err == store.ErrJobNotFound {
		s.sendError(w, http.StatusNotFound, "job_not_found", fmt.Sprintf("Job %s not found", jobID))
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "store_error", fmt.Sprintf("Failed to get job: %v", err))
		return
	}

	// Check if job is already terminal
	if job.IsTerminal() {
		s.sendError(w, http.StatusBadRequest, "job_terminal", "Job is already in terminal state")
		return
	}

	// Update job status to cancelled
	if err := s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateCancelled, nil); err != nil {
		s.sendError(w, http.StatusInternalServerError, "store_error", fmt.Sprintf("Failed to cancel job: %v", err))
		return
	}

	// Send success response
	w.WriteHeader(http.StatusNoContent)
}

// HandleHealth handles GET /health
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	health := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now(),
	}

	s.sendJSON(w, http.StatusOK, health)
}

// processJob processes a job in the background
func (s *Server) processJob(ctx context.Context, jobID string) {
	// Get job from store
	job, err := s.store.GetJob(ctx, jobID)
	if err != nil {
		return
	}

	// Update status to validating
	s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateValidating, &schemas.Progress{
		OverallPercent: 10,
		CurrentStep:    "validating",
	})

	// TODO: Validate JobSpec

	// Update status to planning
	s.store.UpdateJobStatus(ctx, jobID, schemas.JobStatePlanning, &schemas.Progress{
		OverallPercent: 20,
		CurrentStep:    "planning",
	})

	// Create processing plan
	plan, err := s.planner.Plan(ctx, job.Spec, nil)
	if err != nil {
		s.store.UpdateJobError(ctx, jobID, &schemas.ErrorInfo{
			Code:      "PLANNING_ERROR",
			Message:   fmt.Sprintf("Failed to create plan: %v", err),
			Retryable: false,
		})
		s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateFailed, nil)
		return
	}

	// Save plan
	job.Plan = plan
	s.store.UpdateJob(ctx, job)

	// Update status to processing
	s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateProcessing, &schemas.Progress{
		OverallPercent: 50,
		CurrentStep:    "processing",
	})

	// Execute plan
	execOpts := &executor.ExecuteOptions{
		OnProgress: func(progress *executor.Progress) {
			// Update progress in store (simple progress based on frame count)
			percent := 50.0 + (float64(progress.Frame) / 1000.0) // Simplified progress
			if percent > 90 {
				percent = 90 // Cap at 90% until completion
			}
			s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateProcessing, &schemas.Progress{
				OverallPercent: percent,
				CurrentStep:    "processing",
			})
		},
	}

	if err := s.executor.Execute(ctx, plan, execOpts); err != nil {
		s.store.UpdateJobError(ctx, jobID, &schemas.ErrorInfo{
			Code:      "EXECUTION_ERROR",
			Message:   fmt.Sprintf("Failed to execute: %v", err),
			Retryable: true,
		})
		s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateFailed, nil)
		return
	}

	// Update status to completed
	s.store.UpdateJobStatus(ctx, jobID, schemas.JobStateCompleted, &schemas.Progress{
		OverallPercent: 100,
		CurrentStep:    "completed",
	})
}

// Helper methods

func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) sendError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Error:   code,
		Message: message,
		Code:    status,
	}
	s.sendJSON(w, status, resp)
}

func (s *Server) parseListFilter(r *http.Request) *store.ListFilter {
	q := r.URL.Query()
	filter := &store.ListFilter{}

	// Parse status filter
	if statusStr := q.Get("status"); statusStr != "" {
		filter.Status = []schemas.JobState{schemas.JobState(statusStr)}
	}

	// Parse limit and offset
	if limitStr := q.Get("limit"); limitStr != "" {
		var limit int
		fmt.Sscanf(limitStr, "%d", &limit)
		filter.Limit = limit
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		var offset int
		fmt.Sscanf(offsetStr, "%d", &offset)
		filter.Offset = offset
	}

	return filter
}

// extractJobID extracts job ID from URL path like "/api/v1/jobs/{id}"
func extractJobID(path string) string {
	// Simple extraction: assume path is /api/v1/jobs/{id}
	const prefix = "/api/v1/jobs/"
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}

// Close closes the server and releases resources
func (s *Server) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// Ensure io is not reported as unused
var _ = io.EOF
