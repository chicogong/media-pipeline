# Media Pipeline Deployment Guide

Complete guide for deploying the Media Pipeline service in production.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Docker Deployment](#docker-deployment)
- [Configuration](#configuration)
- [Production Setup](#production-setup)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Software

- **Docker**: Version 20.10+ ([Install Docker](https://docs.docker.com/get-docker/))
- **Docker Compose**: Version 2.0+ (included with Docker Desktop)
- **Git**: For cloning the repository

### System Requirements

**Minimum**:
- CPU: 2 cores
- RAM: 4 GB
- Storage: 20 GB (for OS, Docker images, and temporary files)

**Recommended**:
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 100+ GB SSD (for media processing)

### Port Requirements

Ensure these ports are available:
- `8080`: API server (can be changed)
- `5432`: PostgreSQL (optional, for production database)
- `6379`: Redis (optional, for caching/queuing)

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/chicogong/media-pipeline.git
cd media-pipeline
```

### 2. Create Data Directories

```bash
mkdir -p data/uploads data/outputs data/temp
```

### 3. Start Services

```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f api
```

### 4. Verify Deployment

```bash
# Health check
curl http://localhost:8080/health

# Expected output:
# {"status":"healthy","time":"2024-01-15T10:30:00Z"}
```

### 5. Create a Test Job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "spec": {
      "inputs": [
        {"id": "input1", "source": "test.mp4"}
      ],
      "operations": [
        {
          "op": "trim",
          "input": "input1",
          "output": "trimmed",
          "params": {
            "start": "00:00:10",
            "duration": "00:00:30"
          }
        }
      ],
      "outputs": [
        {"id": "trimmed", "destination": "output.mp4"}
      ]
    }
  }'

# Response:
# {
#   "job_id": "job_1705315800000000000",
#   "status": "pending",
#   "created_at": "2024-01-15T10:30:00Z"
# }
```

## Docker Deployment

### Architecture

```
┌─────────────────────────────────────────────────┐
│                Docker Network                    │
│                                                  │
│  ┌────────────┐    ┌────────────┐              │
│  │   Redis    │    │ PostgreSQL │              │
│  │  (Cache)   │    │ (Database) │              │
│  └─────┬──────┘    └─────┬──────┘              │
│        │                  │                      │
│        └────────┬─────────┘                      │
│                 │                                │
│         ┌───────▼────────┐                      │
│         │   API Server   │                      │
│         │   (Go + FFmpeg)│                      │
│         └───────┬────────┘                      │
│                 │                                │
└─────────────────┼────────────────────────────────┘
                  │
          ┌───────▼────────┐
          │   Port 8080    │
          │  (HTTP API)    │
          └────────────────┘
```

### Service Details

#### API Server
- **Image**: Custom build from `Dockerfile`
- **Base**: Alpine Linux + Go 1.21 + FFmpeg
- **Port**: 8080
- **Health Check**: `GET /health` every 30s
- **Volumes**:
  - `./data/uploads`: Input media files
  - `./data/outputs`: Processed output files
  - `./data/temp`: Temporary processing files

#### Redis (Optional)
- **Image**: `redis:7-alpine`
- **Purpose**: Future caching and job queuing
- **Port**: 6379
- **Persistence**: Append-only file (AOF)

#### PostgreSQL (Optional)
- **Image**: `postgres:15-alpine`
- **Purpose**: Future production database
- **Port**: 5432
- **Database**: `media_pipeline`

### Build and Deploy

#### Build Custom Image

```bash
# Build API server image
docker build -t media-pipeline-api:latest .

# View image details
docker images media-pipeline-api
```

#### Deploy with Docker Compose

```bash
# Start all services
docker-compose up -d

# Start specific services
docker-compose up -d api redis

# Scale API instances (future load balancing)
docker-compose up -d --scale api=3
```

#### Stop Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes data)
docker-compose down -v
```

### Docker Commands

```bash
# View running containers
docker-compose ps

# View logs
docker-compose logs -f api
docker-compose logs -f redis
docker-compose logs -f postgres

# Execute commands in container
docker-compose exec api sh
docker-compose exec postgres psql -U media media_pipeline

# Restart services
docker-compose restart api

# Update and restart
docker-compose pull
docker-compose up -d --build
```

## Configuration

### Environment Variables

Create a `.env` file for custom configuration:

```bash
# API Configuration
API_HOST=0.0.0.0
API_PORT=8080

# Redis Configuration
REDIS_URL=redis://redis:6379
REDIS_PASSWORD=

# Database Configuration
DATABASE_URL=postgresql://media:media123@postgres:5432/media_pipeline?sslmode=disable

# Storage Configuration
UPLOAD_DIR=/app/uploads
OUTPUT_DIR=/app/outputs
TEMP_DIR=/app/temp
MAX_UPLOAD_SIZE=1073741824  # 1GB

# FFmpeg Configuration
FFMPEG_PATH=/usr/bin/ffmpeg
FFPROBE_PATH=/usr/bin/ffprobe
FFMPEG_THREADS=0  # 0 = auto

# Job Configuration
MAX_CONCURRENT_JOBS=5
JOB_TIMEOUT=3600  # 1 hour
CLEANUP_INTERVAL=3600  # 1 hour

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

Load environment file:

```bash
# With docker-compose
docker-compose --env-file .env up -d

# Or add to docker-compose.yml
services:
  api:
    env_file:
      - .env
```

### Volume Configuration

#### Custom Data Paths

Edit `docker-compose.yml`:

```yaml
services:
  api:
    volumes:
      - /mnt/storage/uploads:/app/uploads
      - /mnt/storage/outputs:/app/outputs
      - /mnt/ssd/temp:/app/temp
```

#### Shared Network Storage

For multiple API instances:

```yaml
volumes:
  - nfs-uploads:/app/uploads
  - nfs-outputs:/app/outputs

volumes:
  nfs-uploads:
    driver: local
    driver_opts:
      type: nfs
      o: addr=nfs.example.com,rw
      device: ":/exports/uploads"
```

### Network Configuration

#### Custom Network

```yaml
networks:
  media-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.25.0.0/16
```

#### External Network

```yaml
networks:
  media-network:
    external: true
```

## Production Setup

### Security Hardening

#### 1. Use Secrets for Sensitive Data

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  postgres:
    environment:
      - POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password
    secrets:
      - postgres_password

secrets:
  postgres_password:
    file: ./secrets/postgres_password.txt
```

#### 2. Enable TLS

Use a reverse proxy (Nginx/Traefik):

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    depends_on:
      - api
```

#### 3. Restrict Network Access

```yaml
services:
  api:
    networks:
      - frontend
      - backend

  postgres:
    networks:
      - backend  # Not accessible from outside

networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
```

### High Availability

#### Load Balancer Setup

```yaml
services:
  api:
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
```

#### Health Checks

Already configured in `docker-compose.yml`:

```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 10s
```

### Backup Strategy

#### Database Backup

```bash
# Create backup directory
mkdir -p backups

# Backup PostgreSQL
docker-compose exec -T postgres pg_dump -U media media_pipeline > backups/db_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
cat backups/db_20240115_103000.sql | docker-compose exec -T postgres psql -U media media_pipeline
```

#### Media Files Backup

```bash
# Backup uploads and outputs
tar -czf backups/media_$(date +%Y%m%d_%H%M%S).tar.gz data/uploads data/outputs

# Restore
tar -xzf backups/media_20240115_103000.tar.gz
```

### Resource Limits

Add to `docker-compose.yml`:

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

## Monitoring

### Health Checks

```bash
# API health
curl http://localhost:8080/health

# Redis health
docker-compose exec redis redis-cli ping

# PostgreSQL health
docker-compose exec postgres pg_isready -U media
```

### Logs

```bash
# View real-time logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f api

# View last 100 lines
docker-compose logs --tail=100 api

# Export logs
docker-compose logs api > api_logs.txt
```

### Metrics

For production monitoring, integrate with:

- **Prometheus**: Metrics collection
- **Grafana**: Visualization
- **Loki**: Log aggregation

Example Prometheus configuration:

```yaml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

```bash
# Find process using port
lsof -i :8080

# Change port in docker-compose.yml
ports:
  - "8081:8080"
```

#### 2. Container Fails to Start

```bash
# Check container logs
docker-compose logs api

# Inspect container
docker-compose ps
docker inspect media-pipeline-api
```

#### 3. FFmpeg Not Found

```bash
# Verify FFmpeg in container
docker-compose exec api ffmpeg -version
docker-compose exec api ffprobe -version

# Rebuild image if needed
docker-compose build --no-cache api
```

#### 4. Permission Denied on Volumes

```bash
# Fix ownership
sudo chown -R 1000:1000 data/

# Or run container as root (not recommended)
# user: "0:0"
```

#### 5. Out of Disk Space

```bash
# Check disk usage
df -h

# Clean up Docker
docker system prune -a --volumes

# Remove old images
docker images | grep "<none>" | awk '{print $3}' | xargs docker rmi
```

### Debug Mode

Enable verbose logging:

```yaml
services:
  api:
    environment:
      - LOG_LEVEL=debug
    command: ["-host", "0.0.0.0", "-port", "8080", "-v"]
```

### Performance Tuning

#### Increase File Upload Limits

Nginx reverse proxy configuration:

```nginx
client_max_body_size 10G;
proxy_read_timeout 600s;
proxy_send_timeout 600s;
```

#### FFmpeg Performance

```yaml
environment:
  - FFMPEG_THREADS=4  # Limit CPU usage
  - FFMPEG_PRESET=fast  # Balance speed/quality
```

## Next Steps

After deployment:

1. **Configure Monitoring**: Set up Prometheus and Grafana
2. **Setup Backups**: Automate database and media backups
3. **Enable HTTPS**: Configure SSL certificates
4. **Load Testing**: Verify performance under load
5. **Documentation**: Create runbooks for operations team

## Support

For issues and questions:
- GitHub Issues: https://github.com/chicogong/media-pipeline/issues
- Documentation: See `README.md` and `ARCHITECTURE.md`
