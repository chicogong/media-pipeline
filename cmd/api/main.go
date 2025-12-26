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
	"github.com/chicogong/media-pipeline/pkg/auth"
	"github.com/chicogong/media-pipeline/pkg/store"
)

var (
	port      = flag.Int("port", 8080, "Server port")
	host      = flag.String("host", "0.0.0.0", "Server host")
	jwtSecret = flag.String("jwt-secret", getEnv("JWT_SECRET", ""), "JWT secret key")
	authMode  = flag.String("auth-mode", getEnv("AUTH_MODE", "optional"), "Authentication mode: required or optional")
)

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	flag.Parse()

	// Validate JWT secret if auth is required
	if *authMode == "required" && *jwtSecret == "" {
		log.Fatal("JWT_SECRET is required when AUTH_MODE=required")
	}

	// Create authentication managers
	var jwtManager *auth.JWTManager
	var apiKeyManager *auth.APIKeyManager

	if *jwtSecret != "" {
		log.Println("Initializing JWT authentication...")
		jwtManager = auth.NewJWTManager(*jwtSecret, 24*time.Hour)
	}

	log.Println("Initializing API Key authentication...")
	apiKeyManager = auth.NewAPIKeyManager()

	// Create auth middleware
	authRequired := (*authMode == "required")
	var authMiddleware *auth.AuthMiddleware

	if jwtManager != nil || apiKeyManager != nil {
		authMiddleware = auth.NewAuthMiddleware(jwtManager, apiKeyManager, !authRequired)
		if authRequired {
			log.Println("Authentication: REQUIRED")
		} else {
			log.Println("Authentication: OPTIONAL")
		}
	}

	// Create store
	log.Println("Initializing store...")
	s := store.NewMemoryStore()
	defer s.Close()

	// Create API server
	log.Println("Creating API server...")
	server := api.NewServer(s)
	defer server.Close()

	// Setup HTTP router
	mux := setupRoutes(server, authMiddleware)

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

func setupRoutes(server *api.Server, authMiddleware *auth.AuthMiddleware) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("/health", api.Chain(
		server.HandleHealth,
		api.LoggingMiddleware,
	))

	// API routes with authentication
	if authMiddleware != nil {
		// Authenticated job routes
		mux.HandleFunc("/api/v1/jobs", api.Chain(
			handleJobsRoute(server),
			wrapAuthMiddleware(authMiddleware),
			api.RecoveryMiddleware,
			api.CORSMiddleware,
			api.LoggingMiddleware,
		))

		// Authenticated job detail route
		mux.HandleFunc("/api/v1/jobs/", api.Chain(
			handleJobDetailRoute(server),
			wrapAuthMiddleware(authMiddleware),
			api.RecoveryMiddleware,
			api.CORSMiddleware,
			api.LoggingMiddleware,
		))
	} else {
		// No authentication
		mux.HandleFunc("/api/v1/jobs", api.Chain(
			handleJobsRoute(server),
			api.RecoveryMiddleware,
			api.CORSMiddleware,
			api.LoggingMiddleware,
		))

		mux.HandleFunc("/api/v1/jobs/", api.Chain(
			handleJobDetailRoute(server),
			api.RecoveryMiddleware,
			api.CORSMiddleware,
			api.LoggingMiddleware,
		))
	}

	return mux
}

// wrapAuthMiddleware adapts auth.AuthMiddleware to work with api.Chain
func wrapAuthMiddleware(authMiddleware *auth.AuthMiddleware) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authMiddleware.Handler(next).ServeHTTP(w, r)
		}
	}
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
