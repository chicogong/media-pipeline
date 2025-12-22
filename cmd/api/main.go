// Package main provides the API server entry point
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chicogong/media-pipeline/pkg/api"
	"github.com/chicogong/media-pipeline/pkg/store"
)

var (
	port = flag.Int("port", 8080, "Server port")
	host = flag.String("host", "0.0.0.0", "Server host")
)

func main() {
	flag.Parse()

	// Create store
	log.Println("Initializing store...")
	s := store.NewMemoryStore()
	defer s.Close()

	// Create API server
	log.Println("Creating API server...")
	server := api.NewServer(s)
	defer server.Close()

	// Setup HTTP router
	mux := setupRoutes(server)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", *host, *port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func setupRoutes(server *api.Server) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", api.Chain(
		server.HandleHealth,
		api.LoggingMiddleware,
	))

	// API routes
	mux.HandleFunc("/api/v1/jobs", api.Chain(
		handleJobsRoute(server),
		api.RecoveryMiddleware,
		api.CORSMiddleware,
		api.LoggingMiddleware,
	))

	// Job detail route
	mux.HandleFunc("/api/v1/jobs/", api.Chain(
		handleJobDetailRoute(server),
		api.RecoveryMiddleware,
		api.CORSMiddleware,
		api.LoggingMiddleware,
	))

	return mux
}

// handleJobsRoute handles /api/v1/jobs (list and create)
func handleJobsRoute(server *api.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleListJobs(w, r)
		case http.MethodPost:
			server.HandleCreateJob(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

// handleJobDetailRoute handles /api/v1/jobs/{id} (get and delete)
func handleJobDetailRoute(server *api.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleGetJob(w, r)
		case http.MethodDelete:
			server.HandleDeleteJob(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}
