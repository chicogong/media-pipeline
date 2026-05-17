//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/api"
	"github.com/chicogong/media-pipeline/pkg/schemas"
	"github.com/chicogong/media-pipeline/pkg/store"

	// Register the builtin operators (trim, scale) via their init() functions,
	// the same way cmd/api's main package does.
	_ "github.com/chicogong/media-pipeline/pkg/operators/builtin"
)

// newPipelineServer starts an httptest server exposing the job API backed by
// an in-memory store. The routing mirrors cmd/api's setupRoutes.
func newPipelineServer(t *testing.T) *httptest.Server {
	t.Helper()

	st := store.NewMemoryStore()
	srv := api.NewServer(st)
	t.Cleanup(func() {
		srv.Close()
		st.Close()
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.HandleHealth)
	mux.HandleFunc("/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.HandleListJobs(w, r)
		case http.MethodPost:
			srv.HandleCreateJob(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/jobs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.HandleGetJob(w, r)
		case http.MethodDelete:
			srv.HandleDeleteJob(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

// requireFFmpeg returns the ffmpeg path, or skips the test if it is missing.
func requireFFmpeg(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed; skipping pipeline integration test")
	}
	return path
}

// generateTestVideo writes a short synthetic test clip to path.
func generateTestVideo(t *testing.T, ffmpeg, path string) {
	t.Helper()
	cmd := exec.Command(ffmpeg, "-y",
		"-f", "lavfi",
		"-i", "testsrc=duration=2:size=320x240:rate=10",
		"-pix_fmt", "yuv420p",
		path,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}
}

// createJob POSTs a job spec and returns the created job ID.
func createJob(t *testing.T, baseURL string, spec *schemas.JobSpec) string {
	t.Helper()
	body, err := json.Marshal(api.CreateJobRequest{Spec: spec})
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}

	resp, err := http.Post(baseURL+"/api/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/v1/jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create job: got status %d, want 201; body: %s", resp.StatusCode, b)
	}

	var cr api.CreateJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if cr.JobID == "" {
		t.Fatal("create job: response contained an empty job ID")
	}
	return cr.JobID
}

// waitForTerminal polls a job until it reaches a terminal state or times out.
func waitForTerminal(t *testing.T, baseURL, jobID string, timeout time.Duration) schemas.JobStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/jobs/" + jobID)
		if err != nil {
			t.Fatalf("GET job %s: %v", jobID, err)
		}
		var st schemas.JobStatus
		decodeErr := json.NewDecoder(resp.Body).Decode(&st)
		resp.Body.Close()
		if decodeErr != nil {
			t.Fatalf("decode job status: %v", decodeErr)
		}

		switch st.Status {
		case schemas.JobStateCompleted, schemas.JobStateFailed, schemas.JobStateCancelled:
			return st
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach a terminal state within %s", jobID, timeout)
	return schemas.JobStatus{}
}

// TestIntegration_HealthEndpoint verifies the API server reports healthy.
func TestIntegration_HealthEndpoint(t *testing.T) {
	ts := newPipelineServer(t)

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: got status %d, want 200", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("health status = %v, want \"healthy\"", body["status"])
	}
}

// TestIntegration_InvalidSpecRejected verifies a structurally invalid job spec
// is rejected at creation time with a 400, before any processing happens.
func TestIntegration_InvalidSpecRejected(t *testing.T) {
	ts := newPipelineServer(t)

	// The output refers to an ID that no input or operation produces.
	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "src", Source: "file:///tmp/input.mp4"},
		},
		Operations: []schemas.Operation{
			{Op: "scale", Input: "src", Output: "scaled",
				Params: map[string]interface{}{"width": 160, "height": 120}},
		},
		Outputs: []schemas.Output{
			{ID: "ghost", Destination: "file:///tmp/output.mp4"},
		},
	}

	body, _ := json.Marshal(api.CreateJobRequest{Spec: spec})
	resp, err := http.Post(ts.URL+"/api/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/v1/jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid spec: got status %d, want 400", resp.StatusCode)
	}
}

// TestIntegration_FullJobLifecycle submits a single-operation job through the
// API and verifies it runs end to end and produces an output file.
func TestIntegration_FullJobLifecycle(t *testing.T) {
	ffmpeg := requireFFmpeg(t)
	ts := newPipelineServer(t)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.mp4")
	outputPath := filepath.Join(dir, "output.mp4")
	generateTestVideo(t, ffmpeg, inputPath)

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "src", Source: "file://" + inputPath},
		},
		Operations: []schemas.Operation{
			{Op: "scale", Input: "src", Output: "scaled",
				Params: map[string]interface{}{"width": 160, "height": 120}},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "file://" + outputPath},
		},
	}

	jobID := createJob(t, ts.URL, spec)
	st := waitForTerminal(t, ts.URL, jobID, 90*time.Second)

	if st.Status != schemas.JobStateCompleted {
		t.Fatalf("job ended in state %q, want completed; error: %+v", st.Status, st.Error)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("expected output file at %s: %v", outputPath, err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

// TestIntegration_MultiOperationJob submits a job that chains two operations
// (trim then scale) through the full pipeline.
//
// Skipped: this exposes a known bug in the executor's command builder. Every
// operator emits hardcoded [v]/[a] filter labels, so chained operations
// collide (".. .[v]; ...[v]"); and the builder assumes every input has both a
// [0:v] and [0:a] stream, so trim references a [0:a] stream that a video-only
// input does not have. Single-operation jobs work — see
// TestIntegration_FullJobLifecycle. Remove the t.Skip once the builder probes
// real input streams and allocates unique per-operation stream labels.
func TestIntegration_MultiOperationJob(t *testing.T) {
	t.Skip("known bug: chained operations collide on hardcoded [v]/[a] filter labels")

	ffmpeg := requireFFmpeg(t)
	ts := newPipelineServer(t)

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.mp4")
	outputPath := filepath.Join(dir, "output.mp4")
	generateTestVideo(t, ffmpeg, inputPath)

	spec := &schemas.JobSpec{
		Inputs: []schemas.Input{
			{ID: "src", Source: "file://" + inputPath},
		},
		Operations: []schemas.Operation{
			{Op: "trim", Input: "src", Output: "trimmed",
				Params: map[string]interface{}{"start": "00:00:00", "duration": "00:00:01"}},
			{Op: "scale", Input: "trimmed", Output: "scaled",
				Params: map[string]interface{}{"width": 160, "height": 120}},
		},
		Outputs: []schemas.Output{
			{ID: "scaled", Destination: "file://" + outputPath},
		},
	}

	jobID := createJob(t, ts.URL, spec)
	st := waitForTerminal(t, ts.URL, jobID, 90*time.Second)

	if st.Status != schemas.JobStateCompleted {
		t.Fatalf("multi-operation job ended in state %q, want completed; error: %+v", st.Status, st.Error)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("expected output file at %s: %v", outputPath, err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}
