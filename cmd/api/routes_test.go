package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chicogong/media-pipeline/pkg/api"
	"github.com/chicogong/media-pipeline/pkg/auth"
	"github.com/chicogong/media-pipeline/pkg/store"
)

// newTestServer builds an api.Server backed by an in-memory store, cleaned up
// automatically when the test finishes.
func newTestServer(t *testing.T) *api.Server {
	t.Helper()
	s := store.NewMemoryStore()
	t.Cleanup(func() { s.Close() })
	server := api.NewServer(s)
	t.Cleanup(func() { server.Close() })
	return server
}

// TestSetupRoutes_NoAuth exercises setupRoutes with no auth middleware and
// verifies routing and method dispatch through the returned mux.
func TestSetupRoutes_NoAuth(t *testing.T) {
	mux := setupRoutes(newTestServer(t), nil)

	cases := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodGet, "/api/v1/jobs", http.StatusOK},
		{http.MethodPut, "/api/v1/jobs", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/v1/jobs/does-not-exist", http.StatusNotFound},
		{http.MethodPatch, "/api/v1/jobs/abc", http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != c.wantStatus {
			t.Errorf("%s %s: got status %d, want %d", c.method, c.path, w.Code, c.wantStatus)
		}
	}
}

// TestSetupRoutes_WithAuth exercises the authenticated branch of setupRoutes.
// The middleware is in optional mode, so anonymous requests still pass.
func TestSetupRoutes_WithAuth(t *testing.T) {
	authMiddleware := auth.NewAuthMiddleware(nil, auth.NewAPIKeyManager(), true)
	mux := setupRoutes(newTestServer(t), authMiddleware)

	// /health is registered without auth.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /health: got status %d, want 200", w.Code)
	}

	// Authenticated route, optional mode: anonymous request is allowed through.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/v1/jobs (optional auth): got status %d, want 200", w.Code)
	}
}

// TestHandleJobsRoute_MethodDispatch verifies the /api/v1/jobs handler routes
// by HTTP method and rejects unsupported ones.
func TestHandleJobsRoute_MethodDispatch(t *testing.T) {
	handler := handleJobsRoute(newTestServer(t))

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	getW := httptest.NewRecorder()
	handler(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Errorf("GET: got status %d, want 200", getW.Code)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/jobs", nil)
	putW := httptest.NewRecorder()
	handler(putW, putReq)
	if putW.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT: got status %d, want 405", putW.Code)
	}
}

// TestHandleJobDetailRoute_MethodDispatch verifies the /api/v1/jobs/{id}
// handler rejects unsupported methods.
func TestHandleJobDetailRoute_MethodDispatch(t *testing.T) {
	handler := handleJobDetailRoute(newTestServer(t))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/jobs/abc", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH: got status %d, want 405", w.Code)
	}
}

// TestWrapAuthMiddleware verifies the auth middleware adapter invokes the
// wrapped handler (optional mode allows anonymous requests).
func TestWrapAuthMiddleware(t *testing.T) {
	authMiddleware := auth.NewAuthMiddleware(nil, auth.NewAPIKeyManager(), true)
	wrap := wrapAuthMiddleware(authMiddleware)

	called := false
	handler := wrap(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("wrapped handler was not invoked")
	}
	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", w.Code)
	}
}
