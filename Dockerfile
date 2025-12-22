# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy source code
COPY . .

# Build the API server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-pipeline-api ./cmd/api

# Runtime stage
FROM alpine:latest

# Install runtime dependencies (FFmpeg)
RUN apk add --no-cache ffmpeg ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 media && \
    adduser -D -u 1000 -G media media

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/media-pipeline-api .

# Create directories for media processing
RUN mkdir -p /app/uploads /app/outputs /app/temp && \
    chown -R media:media /app

# Switch to non-root user
USER media

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the API server
ENTRYPOINT ["/app/media-pipeline-api"]
CMD ["-host", "0.0.0.0", "-port", "8080"]
