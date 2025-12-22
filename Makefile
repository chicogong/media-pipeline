.PHONY: help build test clean run docker-build docker-up docker-down docker-logs install lint fmt coverage

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=media-pipeline-api
DOCKER_IMAGE=media-pipeline-api
DOCKER_TAG=latest
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')

## help: Show this help message
help:
	@echo "Media Pipeline - Available Commands:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo ""

## install: Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies installed"

## build: Build the API server binary
build:
	@echo "Building API server..."
	go build -o bin/$(BINARY_NAME) ./cmd/api
	@echo "✓ Build complete: bin/$(BINARY_NAME)"

## run: Run the API server locally
run: build
	@echo "Starting API server..."
	./bin/$(BINARY_NAME) -host 0.0.0.0 -port 8080

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race ./...
	@echo "✓ All tests passed"

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✓ Linting complete"; \
	else \
		echo "⚠ golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean
	@echo "✓ Clean complete"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-up: Start all Docker services
docker-up:
	@echo "Starting Docker services..."
	mkdir -p data/uploads data/outputs data/temp
	docker-compose up -d
	@echo "✓ Services started"
	@echo ""
	@echo "API server: http://localhost:8080"
	@echo "Health check: curl http://localhost:8080/health"
	@echo ""
	@echo "View logs: make docker-logs"

## docker-down: Stop all Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down
	@echo "✓ Services stopped"

## docker-down-volumes: Stop services and remove volumes
docker-down-volumes:
	@echo "Stopping services and removing volumes..."
	docker-compose down -v
	@echo "✓ Services stopped and volumes removed"

## docker-logs: View Docker logs
docker-logs:
	docker-compose logs -f

## docker-logs-api: View API server logs
docker-logs-api:
	docker-compose logs -f api

## docker-restart: Restart Docker services
docker-restart: docker-down docker-up

## docker-rebuild: Rebuild and restart Docker services
docker-rebuild:
	@echo "Rebuilding and restarting services..."
	docker-compose down
	docker-compose build --no-cache
	docker-compose up -d
	@echo "✓ Services rebuilt and restarted"

## docker-shell: Open shell in API container
docker-shell:
	docker-compose exec api sh

## docker-ps: Show Docker container status
docker-ps:
	docker-compose ps

## health: Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health | jq '.' || echo "❌ Service not responding"

## setup: Initial setup (install deps, create dirs, copy config)
setup: install
	@echo "Creating data directories..."
	mkdir -p data/uploads data/outputs data/temp
	@echo "✓ Directories created"
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "✓ .env created - please customize it"; \
	else \
		echo "⚠ .env already exists"; \
	fi
	@echo ""
	@echo "✓ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Customize .env if needed"
	@echo "  2. Run 'make docker-up' to start services"
	@echo "  3. Run 'make health' to verify"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies updated"

## generate: Run code generation
generate:
	@echo "Running code generation..."
	go generate ./...
	@echo "✓ Code generation complete"

## verify: Run all verification checks (fmt, lint, test)
verify: fmt lint test
	@echo "✓ All verification checks passed"
