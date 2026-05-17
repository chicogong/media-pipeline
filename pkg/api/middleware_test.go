package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLoggingMiddleware verifies that the wrapped handler runs and its status
// code is captured by the responseWriter wrapper.
func TestLoggingMiddleware(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}

	wrapped := LoggingMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
}

// TestCORSMiddleware verifies that CORS headers are set and the next handler
// runs for a normal (non-preflight) request.
func TestCORSMiddleware(t *testing.T) {
	handlerRan := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerRan = true
		w.WriteHeader(http.StatusOK)
	}

	wrapped := CORSMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if !handlerRan {
		t.Error("expected next handler to run, but it did not")
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin to be \"*\", got %q", origin)
	}

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("expected Access-Control-Allow-Methods to be non-empty")
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers == "" {
		t.Error("expected Access-Control-Allow-Headers to be non-empty")
	}
}

// TestCORSMiddleware_Preflight verifies that an OPTIONS request receives a 204
// and does NOT invoke the next handler.
func TestCORSMiddleware_Preflight(t *testing.T) {
	handlerRan := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerRan = true
	}

	wrapped := CORSMiddleware(handler)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if handlerRan {
		t.Error("expected next handler NOT to run for OPTIONS preflight, but it did")
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d for preflight, got %d", http.StatusNoContent, w.Code)
	}
}

// TestRecoveryMiddleware verifies that a panicking handler is caught, does not
// propagate the panic, and produces a 500 JSON response.
func TestRecoveryMiddleware(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// Must not panic.
	wrapped(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d after panic, got %d", http.StatusInternalServerError, w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type application/json after panic, got %q", ct)
	}
}

// TestRecoveryMiddleware_NoPanic verifies that a normal handler still executes
// correctly when no panic occurs.
func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	wrapped(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestChain verifies that Chain applies middlewares so the first argument is
// outermost: execution order should be first → second → handler.
func TestChain(t *testing.T) {
	var order []string

	makeMiddleware := func(name string) func(http.HandlerFunc) http.HandlerFunc {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next(w, r)
			}
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}

	chained := Chain(handler, makeMiddleware("first"), makeMiddleware("second"))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chained(w, req)

	expected := []string{"first", "second", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected execution order %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("at index %d expected %q, got %q", i, v, order[i])
		}
	}
}

// TestChain_NoMiddlewares verifies that Chain with no middlewares still calls
// the handler directly.
func TestChain_NoMiddlewares(t *testing.T) {
	handlerRan := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerRan = true
		w.WriteHeader(http.StatusOK)
	}

	chained := Chain(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	chained(w, req)

	if !handlerRan {
		t.Error("expected handler to run when no middlewares are given")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestResponseWriter_WriteHeader verifies that the responseWriter struct
// records the status code and forwards it to the underlying ResponseWriter.
func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected responseWriter.statusCode %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected underlying recorder Code %d, got %d", http.StatusNotFound, rec.Code)
	}
}
