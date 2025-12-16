# API Interface Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md), [Schemas Design](./schemas-detailed-design.md)

---

## Overview

This document defines the complete HTTP API for media-pipeline, including:
- **RESTful Endpoints**: Job management, status queries, operator discovery
- **Request/Response Formats**: JSON schemas for all endpoints
- **Authentication**: API key and JWT support
- **Webhooks**: Event notifications for job state changes
- **Error Handling**: Structured error responses
- **Rate Limiting**: Request throttling and quota management

---

## API Architecture

### Design Principles

1. **RESTful**: Standard HTTP methods (GET, POST, DELETE)
2. **Idempotent**: Same request produces same result
3. **Versioned**: URL path includes version (e.g., `/v1/jobs`)
4. **Consistent**: Uniform response structure and error codes
5. **Discoverable**: Self-describing via OPTIONS and metadata endpoints

### Base URL

```
https://api.example.com/v1
```

### Content Type

All requests and responses use `application/json`.

---

## Authentication

### API Key Authentication

```http
POST /v1/jobs
Authorization: Bearer sk_live_abc123xyz...
Content-Type: application/json
```

API keys:
- `sk_live_*` - Production keys
- `sk_test_*` - Test mode keys (jobs not billed, limited resources)

### JWT Authentication (Optional)

For multi-user applications:

```http
POST /v1/jobs
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

JWT claims:
```json
{
  "sub": "user_123",
  "org_id": "org_456",
  "scopes": ["jobs:create", "jobs:read"],
  "exp": 1735689600
}
```

---

## Core Endpoints

### 1. Create Job

Submit a new processing job.

**Endpoint**: `POST /v1/jobs`

**Request Body**:
```json
{
  "inputs": [
    {
      "id": "main_video",
      "source": "s3://bucket/video.mp4"
    }
  ],
  "operations": [
    {
      "op": "trim",
      "input": "main_video",
      "output": "trimmed",
      "params": {
        "start": "00:00:10",
        "duration": "00:05:00"
      }
    }
  ],
  "outputs": [
    {
      "id": "trimmed",
      "destination": "s3://bucket/output.mp4",
      "codec": {
        "video": {"codec": "libx264", "crf": 23},
        "audio": {"codec": "aac", "bitrate": "128k"}
      }
    }
  ],
  "webhook_url": "https://example.com/webhooks/job-complete",
  "priority": 5,
  "timeout": "30m"
}
```

**Response**: `201 Created`
```json
{
  "job_id": "job_abc123",
  "status": "pending",
  "created_at": "2025-12-15T10:00:00Z",
  "estimated_duration": "3m15s",
  "estimated_cost": {
    "amount": 0.15,
    "currency": "USD"
  }
}
```

**Error Response**: `400 Bad Request`
```json
{
  "error": {
    "code": "INVALID_JOB_SPEC",
    "message": "Validation failed",
    "details": {
      "operations[0].params.start": "invalid timecode format",
      "outputs[0].destination": "destination must use https:// or s3://"
    }
  }
}
```

---

### 2. Get Job Status

Query current job status and progress.

**Endpoint**: `GET /v1/jobs/{job_id}`

**Response**: `200 OK`
```json
{
  "job_id": "job_abc123",
  "status": "processing",
  "progress": {
    "overall_percent": 45.5,
    "current_step": "Processing video",
    "step_progress": {
      "ffmpeg_progress": {
        "frame": 1350,
        "fps": 30.2,
        "current_time": "00:00:45.000",
        "total_time": "00:05:00.000",
        "speed": "1.2x"
      }
    },
    "estimated_completion": "2025-12-15T10:03:45Z"
  },
  "created_at": "2025-12-15T10:00:00Z",
  "started_at": "2025-12-15T10:00:05Z",
  "updated_at": "2025-12-15T10:01:23Z"
}
```

**Error Response**: `404 Not Found`
```json
{
  "error": {
    "code": "JOB_NOT_FOUND",
    "message": "Job 'job_abc123' does not exist"
  }
}
```

---

### 3. List Jobs

List all jobs for the current user/organization.

**Endpoint**: `GET /v1/jobs`

**Query Parameters**:
- `status` (optional): Filter by status (pending, processing, completed, failed)
- `limit` (default: 20, max: 100): Number of results per page
- `cursor` (optional): Pagination cursor from previous response
- `created_after` (optional): ISO 8601 timestamp
- `created_before` (optional): ISO 8601 timestamp

**Example Request**:
```http
GET /v1/jobs?status=completed&limit=10&cursor=eyJjcmVhdGVkX2F0IjoxNzM1Njg5NjAwfQ==
```

**Response**: `200 OK`
```json
{
  "data": [
    {
      "job_id": "job_abc123",
      "status": "completed",
      "created_at": "2025-12-15T10:00:00Z",
      "completed_at": "2025-12-15T10:03:45Z",
      "output_files": [
        {
          "output_id": "trimmed",
          "destination": "s3://bucket/output.mp4",
          "file_size": 52428800,
          "duration": 300.0
        }
      ]
    },
    {
      "job_id": "job_def456",
      "status": "completed",
      "created_at": "2025-12-15T09:30:00Z",
      "completed_at": "2025-12-15T09:35:12Z",
      "output_files": [...]
    }
  ],
  "has_more": true,
  "next_cursor": "eyJjcmVhdGVkX2F0IjoxNzM1Njg3ODAwfQ=="
}
```

---

### 4. Cancel Job

Cancel a pending or running job.

**Endpoint**: `DELETE /v1/jobs/{job_id}`

**Response**: `200 OK`
```json
{
  "job_id": "job_abc123",
  "status": "cancelled",
  "cancelled_at": "2025-12-15T10:02:00Z"
}
```

**Error Response**: `409 Conflict`
```json
{
  "error": {
    "code": "CANNOT_CANCEL",
    "message": "Job has already completed"
  }
}
```

---

### 5. Get Job Logs

Retrieve execution logs for debugging.

**Endpoint**: `GET /v1/jobs/{job_id}/logs`

**Query Parameters**:
- `stage` (optional): Filter by stage (validation, planning, processing, etc.)
- `level` (optional): Filter by log level (debug, info, warn, error)

**Response**: `200 OK`
```json
{
  "job_id": "job_abc123",
  "logs": [
    {
      "timestamp": "2025-12-15T10:00:00Z",
      "level": "info",
      "stage": "validation",
      "message": "JobSpec validation passed"
    },
    {
      "timestamp": "2025-12-15T10:00:02Z",
      "level": "info",
      "stage": "planning",
      "message": "Built DAG with 5 nodes, 4 edges"
    },
    {
      "timestamp": "2025-12-15T10:00:05Z",
      "level": "info",
      "stage": "processing",
      "message": "Started FFmpeg command: ffmpeg -i ..."
    },
    {
      "timestamp": "2025-12-15T10:00:10Z",
      "level": "error",
      "stage": "processing",
      "message": "FFmpeg stderr: Invalid data found when processing input"
    }
  ]
}
```

---

### 6. Get Processing Plan

Retrieve the compiled processing plan (for debugging).

**Endpoint**: `GET /v1/jobs/{job_id}/plan`

**Response**: `200 OK`
```json
{
  "plan_id": "plan_abc123",
  "job_id": "job_abc123",
  "created_at": "2025-12-15T10:00:03Z",
  "nodes": [
    {
      "id": "input_main_video",
      "type": "input",
      "source_uri": "s3://bucket/video.mp4",
      "media_info": {
        "duration": 3600.5,
        "format": "mov,mp4",
        "video_streams": [...]
      }
    },
    {
      "id": "op_trim",
      "type": "operation",
      "operator": "trim",
      "params": {"start": "00:00:10", "duration": "00:05:00"}
    }
  ],
  "edges": [...],
  "estimates": {
    "total_cpu_time": "2m30s",
    "peak_memory": 524288000,
    "estimated_duration": "3m15s"
  },
  "commands": [
    {
      "id": "cmd_main",
      "command": "ffmpeg -i /tmp/...",
      "filtergraph": "[0:v]trim=start=10:duration=300[v]"
    }
  ]
}
```

---

## Operator Discovery

### 7. List Operators

List all available operators.

**Endpoint**: `GET /v1/operators`

**Query Parameters**:
- `category` (optional): Filter by category (audio, video, timeline, etc.)

**Response**: `200 OK`
```json
{
  "operators": [
    {
      "name": "trim",
      "category": "timeline",
      "description": "Trim video/audio to specified time range",
      "parameters": [
        {
          "name": "start",
          "type": "timecode",
          "required": false,
          "default": "00:00:00",
          "description": "Start time"
        },
        {
          "name": "duration",
          "type": "duration",
          "required": false,
          "description": "Duration"
        }
      ],
      "min_inputs": 1,
      "max_inputs": 1,
      "input_types": ["video+audio", "video", "audio"],
      "output_types": ["video+audio"]
    },
    {
      "name": "loudnorm",
      "category": "audio",
      "description": "EBU R128 loudness normalization",
      "parameters": [...],
      "requires_two_pass": true
    }
  ]
}
```

---

### 8. Get Operator Details

Get detailed information about a specific operator.

**Endpoint**: `GET /v1/operators/{operator_name}`

**Response**: `200 OK`
```json
{
  "name": "trim",
  "category": "timeline",
  "description": "Trim video/audio to specified time range",
  "parameters": [
    {
      "name": "start",
      "type": "timecode",
      "required": false,
      "default": "00:00:00",
      "description": "Start time",
      "validation": {
        "pattern": "^\\d{2}:\\d{2}:\\d{2}(\\.\\d{3})?$"
      },
      "examples": ["00:00:10", "00:00:10.500"]
    },
    {
      "name": "duration",
      "type": "duration",
      "required": false,
      "description": "Duration (if not specified, trim to end)",
      "examples": ["00:05:00", "5m", "300s"]
    },
    {
      "name": "end",
      "type": "timecode",
      "required": false,
      "description": "End time (alternative to duration)"
    }
  ],
  "min_inputs": 1,
  "max_inputs": 1,
  "input_types": ["video+audio", "video", "audio"],
  "output_types": ["video+audio"],
  "examples": [
    {
      "description": "Trim first 5 minutes",
      "params": {
        "start": "00:00:00",
        "duration": "00:05:00"
      }
    },
    {
      "description": "Extract middle section",
      "params": {
        "start": "00:01:30",
        "end": "00:03:45"
      }
    }
  ]
}
```

---

## Webhooks

### Event Notifications

Configure a webhook URL when creating a job to receive event notifications.

**Supported Events**:
- `job.pending` - Job created and queued
- `job.started` - Job execution started
- `job.progress` - Progress update (sent every 5 seconds during processing)
- `job.completed` - Job completed successfully
- `job.failed` - Job failed with error

### Webhook Payload

All webhook payloads follow this format:

```json
{
  "event": "job.completed",
  "timestamp": "2025-12-15T10:03:45Z",
  "data": {
    "job_id": "job_abc123",
    "status": "completed",
    "completed_at": "2025-12-15T10:03:45Z",
    "output_files": [
      {
        "output_id": "trimmed",
        "destination": "s3://bucket/output.mp4",
        "file_size": 52428800,
        "duration": 300.0,
        "md5": "098f6bcd4621d373cade4e832627b4f6"
      }
    ],
    "metrics": {
      "total_duration": "3m42s",
      "processing_time": "2m15s",
      "processing_speed": 1.33
    }
  }
}
```

### Webhook Signature

Each webhook includes an `X-Signature` header for verification:

```http
POST /webhooks/job-complete HTTP/1.1
Host: example.com
Content-Type: application/json
X-Signature: sha256=5d41402abc4b2a76b9719d911017c592
X-Webhook-Id: whk_abc123
X-Webhook-Timestamp: 1735689825

{...}
```

**Signature Verification** (HMAC-SHA256):
```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### Webhook Retry Policy

- **Timeout**: 10 seconds
- **Retries**: Up to 3 attempts with exponential backoff (1s, 4s, 16s)
- **Success**: HTTP 2xx response
- **Failure**: HTTP 4xx/5xx or timeout

---

## Error Handling

### Error Response Format

All error responses follow this structure:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {
      "field1": "specific error for field1",
      "field2": "specific error for field2"
    },
    "request_id": "req_abc123"
  }
}
```

### HTTP Status Codes

| Code | Meaning | Use Case |
|------|---------|----------|
| 200 | OK | Successful GET, DELETE |
| 201 | Created | Successful POST (job created) |
| 400 | Bad Request | Invalid JobSpec, validation errors |
| 401 | Unauthorized | Missing or invalid API key |
| 403 | Forbidden | API key doesn't have permission |
| 404 | Not Found | Job ID doesn't exist |
| 409 | Conflict | Cannot cancel completed job |
| 422 | Unprocessable Entity | Valid JSON but semantic errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unexpected server error |
| 503 | Service Unavailable | System overloaded |

### Error Codes

**Client Errors (4xx)**:
- `INVALID_JOB_SPEC` - JobSpec validation failed
- `INVALID_OPERATION` - Unknown operator or invalid parameters
- `INVALID_INPUT` - Input source unreachable or invalid
- `RESOURCE_LIMIT_EXCEEDED` - Job exceeds account limits
- `UNSUPPORTED_FORMAT` - Media format not supported
- `SSRF_BLOCKED` - Input URL blocked by SSRF protection

**Server Errors (5xx)**:
- `INTERNAL_ERROR` - Unexpected server error
- `FFMPEG_FAILED` - FFmpeg processing failed
- `TIMEOUT` - Job exceeded timeout limit
- `INSUFFICIENT_RESOURCES` - Not enough disk/memory

### Rate Limiting Headers

All responses include rate limit headers:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1735689600
```

When rate limit is exceeded:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Try again in 60 seconds.",
    "retry_after": 60
  }
}
```

---

## Rate Limiting

### Limits

**Per API Key**:
- 1000 requests per minute
- 10,000 requests per hour
- 100 concurrent jobs

**Per Endpoint**:
- `POST /v1/jobs`: 100 requests per minute
- `GET /v1/jobs/{job_id}`: 1000 requests per minute
- Other endpoints: 500 requests per minute

### Implementation

Use Token Bucket algorithm:
```go
type RateLimiter struct {
    limit     int           // Requests per window
    window    time.Duration // Time window
    tokens    map[string]int
    lastReset map[string]time.Time
    mu        sync.Mutex
}

func (rl *RateLimiter) Allow(apiKey string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    lastReset := rl.lastReset[apiKey]

    // Reset if window expired
    if now.Sub(lastReset) >= rl.window {
        rl.tokens[apiKey] = rl.limit
        rl.lastReset[apiKey] = now
    }

    // Check and consume token
    if rl.tokens[apiKey] > 0 {
        rl.tokens[apiKey]--
        return true
    }

    return false
}
```

---

## Pagination

### Cursor-Based Pagination

For list endpoints, use cursor-based pagination:

**Request**:
```http
GET /v1/jobs?limit=20&cursor=eyJjcmVhdGVkX2F0IjoxNzM1Njg5NjAwfQ==
```

**Response**:
```json
{
  "data": [...],
  "has_more": true,
  "next_cursor": "eyJjcmVhdGVkX2F0IjoxNzM1Njg3ODAwfQ=="
}
```

**Cursor Format**: Base64-encoded JSON
```json
{"created_at": 1735687800}
```

---

## CORS Configuration

Support CORS for browser-based applications:

```http
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Max-Age: 86400
```

---

## OpenAPI Specification

### Metadata Endpoint

**Endpoint**: `GET /v1/openapi.json`

**Response**: Full OpenAPI 3.0 specification

```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "Media Pipeline API",
    "version": "1.0.0",
    "description": "Declarative media processing pipeline API"
  },
  "servers": [
    {
      "url": "https://api.example.com/v1"
    }
  ],
  "paths": {
    "/jobs": {
      "post": {
        "summary": "Create a new processing job",
        "requestBody": {...},
        "responses": {...}
      }
    }
  },
  "components": {
    "schemas": {
      "JobSpec": {...},
      "JobStatus": {...}
    },
    "securitySchemes": {
      "ApiKeyAuth": {
        "type": "http",
        "scheme": "bearer"
      }
    }
  }
}
```

---

## Health Check

### System Health

**Endpoint**: `GET /health`

**Response**: `200 OK`
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "components": {
    "api": "healthy",
    "database": "healthy",
    "queue": "healthy",
    "storage": "healthy"
  },
  "uptime": 3600
}
```

**Response**: `503 Service Unavailable` (if unhealthy)
```json
{
  "status": "unhealthy",
  "components": {
    "api": "healthy",
    "database": "unhealthy",
    "queue": "degraded"
  }
}
```

---

## SDK Examples

### JavaScript/TypeScript

```typescript
import { MediaPipeline } from '@media-pipeline/sdk';

const client = new MediaPipeline({
  apiKey: 'sk_live_abc123...',
});

// Create job
const job = await client.jobs.create({
  inputs: [
    { id: 'video', source: 's3://bucket/input.mp4' }
  ],
  operations: [
    {
      op: 'trim',
      input: 'video',
      output: 'trimmed',
      params: { start: '00:00:10', duration: '00:05:00' }
    }
  ],
  outputs: [
    { id: 'trimmed', destination: 's3://bucket/output.mp4' }
  ]
});

console.log(`Job created: ${job.job_id}`);

// Poll for completion
while (true) {
  const status = await client.jobs.get(job.job_id);

  if (status.status === 'completed') {
    console.log('Job completed:', status.output_files);
    break;
  } else if (status.status === 'failed') {
    console.error('Job failed:', status.error);
    break;
  }

  await new Promise(r => setTimeout(r, 5000)); // Wait 5s
}
```

### Python

```python
from media_pipeline import MediaPipeline

client = MediaPipeline(api_key='sk_live_abc123...')

# Create job
job = client.jobs.create(
    inputs=[
        {'id': 'video', 'source': 's3://bucket/input.mp4'}
    ],
    operations=[
        {
            'op': 'trim',
            'input': 'video',
            'output': 'trimmed',
            'params': {'start': '00:00:10', 'duration': '00:05:00'}
        }
    ],
    outputs=[
        {'id': 'trimmed', 'destination': 's3://bucket/output.mp4'}
    ]
)

print(f"Job created: {job.job_id}")

# Wait for completion
result = client.jobs.wait(job.job_id, timeout=600)
if result.status == 'completed':
    print(f"Output files: {result.output_files}")
```

### Go

```go
package main

import (
    "context"
    "fmt"
    "time"

    pipeline "github.com/example/media-pipeline-go"
)

func main() {
    client := pipeline.NewClient("sk_live_abc123...")

    // Create job
    job, err := client.Jobs.Create(context.Background(), &pipeline.JobSpec{
        Inputs: []pipeline.Input{
            {ID: "video", Source: "s3://bucket/input.mp4"},
        },
        Operations: []pipeline.Operation{
            {
                Op:     "trim",
                Input:  "video",
                Output: "trimmed",
                Params: map[string]interface{}{
                    "start":    "00:00:10",
                    "duration": "00:05:00",
                },
            },
        },
        Outputs: []pipeline.Output{
            {ID: "trimmed", Destination: "s3://bucket/output.mp4"},
        },
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Job created: %s\n", job.JobID)

    // Poll for completion
    for {
        status, err := client.Jobs.Get(context.Background(), job.JobID)
        if err != nil {
            panic(err)
        }

        if status.Status == "completed" {
            fmt.Println("Job completed:", status.OutputFiles)
            break
        } else if status.Status == "failed" {
            fmt.Println("Job failed:", status.Error)
            break
        }

        time.Sleep(5 * time.Second)
    }
}
```

---

## Implementation Notes

### 1. API Gateway

Use API gateway for:
- Authentication/authorization
- Rate limiting
- Request logging
- Metrics collection
- SSL termination

### 2. Request ID Tracing

Add `X-Request-ID` header to all requests/responses for tracing:

```http
X-Request-ID: req_abc123
```

Include in logs and error responses.

### 3. Idempotency

Support idempotent job creation with `Idempotency-Key` header:

```http
POST /v1/jobs
Idempotency-Key: unique-key-123
```

Store mapping of `(api_key, idempotency_key) → job_id` for 24 hours.

### 4. Compression

Support gzip compression for responses:

```http
Accept-Encoding: gzip
```

```http
Content-Encoding: gzip
```

### 5. Partial Responses

Support field filtering to reduce bandwidth:

```http
GET /v1/jobs/job_abc123?fields=job_id,status,progress.overall_percent
```

---

## Security Considerations

### 1. Input Validation

- Validate all input URIs against SSRF attacks
- Blocklist: localhost, 127.0.0.1, 169.254.0.0/16, 10.0.0.0/8, etc.
- Allowlist protocols: https://, s3://, gs://

### 2. Output Validation

- Verify destination permissions before job creation
- Prevent writing to arbitrary locations

### 3. API Key Security

- Use bcrypt to hash API keys in database
- Prefix keys with type: `sk_live_`, `sk_test_`
- Support key rotation
- Log all API key usage

### 4. Webhook Security

- Require HTTPS for webhook URLs
- Sign all webhook payloads
- Implement retry with exponential backoff
- Timeout after 10 seconds

---

## Versioning Strategy

### URL Versioning

Version included in URL path: `/v1/jobs`, `/v2/jobs`

### Deprecation Policy

1. Announce deprecation 6 months in advance
2. Add `Deprecation` header to responses:
   ```http
   Deprecation: true
   Sunset: Tue, 15 Jun 2026 00:00:00 GMT
   ```
3. Maintain old version for 12 months after deprecation
4. Remove old version after sunset date

### Breaking Changes

Require new version:
- Removing fields from responses
- Changing field types
- Removing endpoints
- Changing error codes

Non-breaking changes (no version bump):
- Adding optional fields
- Adding new endpoints
- Adding new error codes
- Adding new operators

---

## Summary

The API provides a complete interface for:

1. **Job Management**: Create, query, cancel jobs
2. **Operator Discovery**: List available operators and their parameters
3. **Real-time Updates**: Webhooks for job state changes
4. **Error Handling**: Structured errors with detailed information
5. **Rate Limiting**: Fair usage with token bucket algorithm
6. **Authentication**: API key and JWT support
7. **Documentation**: OpenAPI specification and SDK examples

Key features:
- RESTful design with consistent patterns
- Comprehensive error handling
- Webhook support for async notifications
- Rate limiting and quotas
- Versioned API with clear deprecation policy
- Security best practices (SSRF protection, webhook signing)

---

**Status**: Ready for implementation
