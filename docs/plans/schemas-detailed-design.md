# Data Schemas Detailed Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md)

---

## Overview

This document provides the complete field definitions for all core data structures in media-pipeline:
- **JobSpec**: User-submitted job specification
- **ProcessingPlan**: Intermediate representation (IR) after compilation
- **JobStatus**: Real-time job status and progress
- **JobLog**: Complete execution record for debugging and reproducibility

---

## 1. JobSpec

User-submitted declarative job specification.

### Go Struct Definition

```go
package schemas

import "time"

type JobSpec struct {
    // Metadata
    JobID       string            `json:"job_id,omitempty"`        // Auto-generated if not provided
    CreatedAt   time.Time         `json:"created_at,omitempty"`    // Server timestamp
    UserID      string            `json:"user_id,omitempty"`       // For multi-tenant scenarios
    Tags        map[string]string `json:"tags,omitempty"`          // User-defined tags (e.g., project_id, customer_id)

    // Configuration
    Debug       bool              `json:"debug,omitempty"`         // Keep intermediate files, verbose logging
    Priority    int               `json:"priority,omitempty"`      // Job priority (0-10, default 5)
    Timeout     *Duration         `json:"timeout,omitempty"`       // Max execution time (default: 30min)

    // Core Specification
    Inputs      []Input           `json:"inputs"`                  // Input sources
    Operations  []Operation       `json:"operations"`              // Processing operations
    Outputs     []Output          `json:"outputs"`                 // Output destinations

    // Resource Limits (optional, overrides defaults)
    Limits      *ResourceLimits   `json:"limits,omitempty"`
}

type Input struct {
    ID          string            `json:"id"`                      // Unique identifier (e.g., "video1")
    Source      string            `json:"source"`                  // URI: file://, http://, https://, s3://, gs://
    Type        string            `json:"type,omitempty"`          // Optional: "video", "audio", "image" (auto-detect if empty)
    Format      string            `json:"format,omitempty"`        // Optional: force format (e.g., "mp4", "mov")
    StartOffset *Duration         `json:"start_offset,omitempty"`  // Skip first N seconds of input
    Duration    *Duration         `json:"duration,omitempty"`      // Only use first N seconds
    Metadata    map[string]string `json:"metadata,omitempty"`      // User-defined metadata
}

type Operation struct {
    Op          string                 `json:"op"`                  // Operator name (e.g., "trim", "concat", "loudnorm")
    Input       string                 `json:"input,omitempty"`     // Single input ID (for single-input operators)
    Inputs      []string               `json:"inputs,omitempty"`    // Multiple input IDs (for multi-input operators like concat, mix)
    Output      string                 `json:"output"`              // Output ID (used as input for downstream operations)
    Params      map[string]interface{} `json:"params,omitempty"`    // Operator-specific parameters
}

type Output struct {
    ID          string            `json:"id"`                      // References an operation output
    Destination string            `json:"destination"`             // URI: file://, s3://, gs://
    Format      string            `json:"format,omitempty"`        // Output format (default: infer from destination extension)
    Codec       *CodecParams      `json:"codec,omitempty"`         // Codec settings
    Metadata    map[string]string `json:"metadata,omitempty"`      // Metadata to embed in output file
}

type CodecParams struct {
    Video       *VideoCodec       `json:"video,omitempty"`
    Audio       *AudioCodec       `json:"audio,omitempty"`
}

type VideoCodec struct {
    Codec       string            `json:"codec,omitempty"`         // e.g., "libx264", "libx265", "vp9"
    Bitrate     string            `json:"bitrate,omitempty"`       // e.g., "5M", "2000k"
    CRF         *int              `json:"crf,omitempty"`           // Constant Rate Factor (0-51 for x264)
    Preset      string            `json:"preset,omitempty"`        // e.g., "fast", "medium", "slow"
    Profile     string            `json:"profile,omitempty"`       // e.g., "main", "high"
    PixelFormat string            `json:"pixel_format,omitempty"`  // e.g., "yuv420p"
}

type AudioCodec struct {
    Codec       string            `json:"codec,omitempty"`         // e.g., "aac", "libopus", "libmp3lame"
    Bitrate     string            `json:"bitrate,omitempty"`       // e.g., "128k", "192k"
    SampleRate  int               `json:"sample_rate,omitempty"`   // e.g., 44100, 48000
    Channels    int               `json:"channels,omitempty"`      // e.g., 1 (mono), 2 (stereo), 6 (5.1)
}

type ResourceLimits struct {
    MaxDuration   *Duration `json:"max_duration,omitempty"`      // e.g., "2h"
    MaxResolution string    `json:"max_resolution,omitempty"`    // e.g., "3840x2160"
    MaxOutputSize int64     `json:"max_output_size,omitempty"`   // Bytes (e.g., 5GB)
    MaxMemory     int64     `json:"max_memory,omitempty"`        // Bytes (for FFmpeg process)
}

// Duration is a wrapper for time.Duration with JSON marshaling support
type Duration struct {
    time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
    return []byte(`"` + d.String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
    // Parse formats: "1h30m", "00:05:30", "5.5s"
    // Implementation handles multiple formats
    return nil
}
```

### Example JobSpec

```json
{
  "debug": false,
  "priority": 5,
  "timeout": "30m",
  "inputs": [
    {
      "id": "main_video",
      "source": "s3://bucket/raw/meeting-2024-12-14.mp4",
      "type": "video"
    },
    {
      "id": "background_music",
      "source": "https://cdn.example.com/music/calm-ambient.mp3",
      "type": "audio",
      "start_offset": "10s",
      "duration": "5m"
    },
    {
      "id": "logo",
      "source": "s3://bucket/assets/logo.png",
      "type": "image"
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
    },
    {
      "op": "loudnorm",
      "input": "trimmed",
      "output": "normalized",
      "params": {
        "target_lufs": -16,
        "target_tp": -1.5,
        "target_lra": 11
      }
    },
    {
      "op": "mix",
      "inputs": ["normalized", "background_music"],
      "output": "mixed_audio",
      "params": {
        "mode": "ducking",
        "main_weight": 1.0,
        "bg_weight": 0.3,
        "ducking_threshold": -20
      }
    },
    {
      "op": "overlay",
      "input": "trimmed",
      "output": "with_logo",
      "params": {
        "overlay_source": "logo",
        "position": "top_right",
        "margin": 20,
        "opacity": 0.8
      }
    },
    {
      "op": "combine",
      "inputs": ["with_logo", "mixed_audio"],
      "output": "final"
    }
  ],
  "outputs": [
    {
      "id": "final",
      "destination": "s3://bucket/output/meeting-edited-2024-12-14.mp4",
      "format": "mp4",
      "codec": {
        "video": {
          "codec": "libx264",
          "preset": "medium",
          "crf": 23
        },
        "audio": {
          "codec": "aac",
          "bitrate": "128k"
        }
      }
    }
  ],
  "limits": {
    "max_duration": "2h",
    "max_resolution": "3840x2160"
  }
}
```

### Validation Rules

**Inputs**:
- `id` must be unique within job
- `source` must use whitelisted protocol (http, https, s3, gs, file)
- For http/https sources, IP must not be in blocklist (localhost, private networks, link-local)

**Operations**:
- `op` must be a registered operator name
- `input`/`inputs` must reference existing input IDs or upstream operation outputs
- `output` must be unique within job
- Operations must form a valid DAG (no cycles)
- `params` validated by operator's `Validate()` method

**Outputs**:
- `id` must reference an operation output
- `destination` must use whitelisted protocol
- At least one output required

---

## 2. ProcessingPlan

Intermediate representation after JobSpec compilation. Contains execution plan with dependency graph and resource estimates.

### Go Struct Definition

```go
package schemas

import "time"

type ProcessingPlan struct {
    // Metadata
    PlanID      string            `json:"plan_id"`
    JobID       string            `json:"job_id"`
    CreatedAt   time.Time         `json:"created_at"`

    // Execution Plan
    Nodes       []PlanNode        `json:"nodes"`               // Execution nodes (DAG vertices)
    Edges       []PlanEdge        `json:"edges"`               // Dependencies (DAG edges)

    // Resource Estimates
    Estimates   ResourceEstimates `json:"estimates"`

    // Generated Artifacts
    FFmpegVersion string          `json:"ffmpeg_version"`      // e.g., "6.0"
    Commands      []FFmpegCommand `json:"commands"`            // Generated FFmpeg commands
}

type PlanNode struct {
    ID          string                 `json:"id"`              // Unique node ID
    Type        string                 `json:"type"`            // "input", "operation", "output"

    // For input nodes
    InputID     string                 `json:"input_id,omitempty"`
    SourceURI   string                 `json:"source_uri,omitempty"`

    // For operation nodes
    Operator    string                 `json:"operator,omitempty"`
    Params      map[string]interface{} `json:"params,omitempty"`

    // For output nodes
    OutputID    string                 `json:"output_id,omitempty"`
    DestURI     string                 `json:"dest_uri,omitempty"`

    // Metadata
    MediaInfo   *MediaInfo             `json:"media_info,omitempty"` // Detected after input download
    Estimates   *NodeEstimates         `json:"estimates,omitempty"`
}

type PlanEdge struct {
    From        string `json:"from"`                        // Source node ID
    To          string `json:"to"`                          // Target node ID
    StreamType  string `json:"stream_type,omitempty"`       // "video", "audio", "both"
}

type MediaInfo struct {
    Duration    float64       `json:"duration"`              // Seconds
    Format      string        `json:"format"`                // e.g., "mov,mp4,m4a,3gp,3g2,mj2"
    FileSize    int64         `json:"file_size"`             // Bytes
    Bitrate     int64         `json:"bitrate"`               // Bits per second
    VideoStreams []VideoStream `json:"video_streams,omitempty"`
    AudioStreams []AudioStream `json:"audio_streams,omitempty"`
}

type VideoStream struct {
    Index       int    `json:"index"`
    Codec       string `json:"codec"`                       // e.g., "h264"
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    FrameRate   string `json:"frame_rate"`                  // e.g., "30/1", "29.97"
    PixelFormat string `json:"pixel_format"`                // e.g., "yuv420p"
    Bitrate     int64  `json:"bitrate,omitempty"`
}

type AudioStream struct {
    Index       int    `json:"index"`
    Codec       string `json:"codec"`                       // e.g., "aac"
    SampleRate  int    `json:"sample_rate"`                 // e.g., 48000
    Channels    int    `json:"channels"`                    // e.g., 2
    Bitrate     int64  `json:"bitrate,omitempty"`
}

type NodeEstimates struct {
    CPUTime     time.Duration `json:"cpu_time"`              // Estimated processing time
    MemoryUsage int64         `json:"memory_usage"`          // Estimated peak memory (bytes)
    DiskUsage   int64         `json:"disk_usage"`            // Estimated disk space (bytes)
}

type ResourceEstimates struct {
    TotalCPUTime     time.Duration `json:"total_cpu_time"`
    PeakMemory       int64         `json:"peak_memory"`
    TotalDiskSpace   int64         `json:"total_disk_space"`
    EstimatedDuration time.Duration `json:"estimated_duration"` // Wall-clock time
}

type FFmpegCommand struct {
    ID          string            `json:"id"`                  // Unique command ID
    Stage       string            `json:"stage"`               // "probe", "loudnorm_pass1", "main", etc.
    Command     string            `json:"command"`             // Full FFmpeg command line
    Args        []string          `json:"args"`                // Parsed arguments
    WorkDir     string            `json:"work_dir"`            // Working directory
    DependsOn   []string          `json:"depends_on,omitempty"` // Previous command IDs
    Filtergraph string            `json:"filtergraph,omitempty"` // Complex filtergraph (if used)
}
```

### Example ProcessingPlan

```json
{
  "plan_id": "plan_abc123",
  "job_id": "job_abc123",
  "created_at": "2025-12-15T10:00:00Z",
  "nodes": [
    {
      "id": "input_main_video",
      "type": "input",
      "input_id": "main_video",
      "source_uri": "s3://bucket/raw/meeting.mp4",
      "media_info": {
        "duration": 3600.5,
        "format": "mov,mp4,m4a",
        "file_size": 524288000,
        "bitrate": 1165066,
        "video_streams": [{
          "index": 0,
          "codec": "h264",
          "width": 1920,
          "height": 1080,
          "frame_rate": "30/1",
          "pixel_format": "yuv420p"
        }],
        "audio_streams": [{
          "index": 1,
          "codec": "aac",
          "sample_rate": 48000,
          "channels": 2
        }]
      }
    },
    {
      "id": "op_trim",
      "type": "operation",
      "operator": "trim",
      "params": {
        "start": "00:00:10",
        "duration": "00:05:00"
      },
      "estimates": {
        "cpu_time": "30s",
        "memory_usage": 104857600,
        "disk_usage": 52428800
      }
    },
    {
      "id": "output_final",
      "type": "output",
      "output_id": "final",
      "dest_uri": "s3://bucket/output/meeting-edited.mp4"
    }
  ],
  "edges": [
    {"from": "input_main_video", "to": "op_trim", "stream_type": "both"},
    {"from": "op_trim", "to": "output_final", "stream_type": "both"}
  ],
  "estimates": {
    "total_cpu_time": "2m30s",
    "peak_memory": 524288000,
    "total_disk_space": 1073741824,
    "estimated_duration": "3m15s"
  },
  "ffmpeg_version": "6.0",
  "commands": [
    {
      "id": "cmd_main",
      "stage": "main",
      "command": "ffmpeg -i /tmp/media-pipeline/job_abc123/main_video.mp4 -filter_complex '[0:v]trim=start=10:duration=300[v];[0:a]atrim=start=10:duration=300[a]' -map '[v]' -map '[a]' -c:v libx264 -preset medium -crf 23 -c:a aac -b:a 128k -y /tmp/media-pipeline/job_abc123/final.mp4",
      "args": ["-i", "/tmp/...", "-filter_complex", "...", "..."],
      "work_dir": "/tmp/media-pipeline/job_abc123",
      "filtergraph": "[0:v]trim=start=10:duration=300[v];[0:a]atrim=start=10:duration=300[a]"
    }
  ]
}
```

---

## 3. JobStatus

Real-time job status and progress information.

### Go Struct Definition

```go
package schemas

import "time"

type JobStatus struct {
    JobID       string            `json:"job_id"`
    Status      JobState          `json:"status"`              // Current state
    Progress    *Progress         `json:"progress,omitempty"`  // Progress details
    Error       *ProcessingError  `json:"error,omitempty"`     // Error details (if failed)
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
    StartedAt   *time.Time        `json:"started_at,omitempty"`
    CompletedAt *time.Time        `json:"completed_at,omitempty"`

    // Results (when completed)
    OutputFiles []OutputFileInfo  `json:"output_files,omitempty"`
}

type JobState string

const (
    JobStatePending            JobState = "pending"
    JobStateValidating         JobState = "validating"
    JobStatePlanning           JobState = "planning"
    JobStateDownloadingInputs  JobState = "downloading_inputs"
    JobStateProcessing         JobState = "processing"
    JobStateUploadingOutputs   JobState = "uploading_outputs"
    JobStateCompleted          JobState = "completed"
    JobStateFailed             JobState = "failed"
    JobStateCancelled          JobState = "cancelled"
)

type Progress struct {
    OverallPercent   float64            `json:"overall_percent"`    // 0-100
    CurrentStep      string             `json:"current_step"`       // Human-readable step name
    StepProgress     *StepProgress      `json:"step_progress,omitempty"`
    EstimatedCompletion *time.Time      `json:"estimated_completion,omitempty"`
}

type StepProgress struct {
    // For downloading_inputs
    DownloadProgress *DownloadProgress  `json:"download_progress,omitempty"`

    // For processing
    FFmpegProgress   *FFmpegProgress    `json:"ffmpeg_progress,omitempty"`

    // For uploading_outputs
    UploadProgress   *UploadProgress    `json:"upload_progress,omitempty"`
}

type DownloadProgress struct {
    TotalFiles       int                `json:"total_files"`
    CompletedFiles   int                `json:"completed_files"`
    CurrentFile      string             `json:"current_file"`
    BytesDownloaded  int64              `json:"bytes_downloaded"`
    TotalBytes       int64              `json:"total_bytes"`
}

type FFmpegProgress struct {
    Frame            int                `json:"frame"`
    FPS              float64            `json:"fps"`
    CurrentTime      string             `json:"current_time"`       // HH:MM:SS.ms
    TotalTime        string             `json:"total_time"`         // HH:MM:SS.ms
    Speed            string             `json:"speed"`              // e.g., "1.2x"
    Bitrate          string             `json:"bitrate"`            // e.g., "1024kbits/s"
    TotalSize        int64              `json:"total_size"`         // Bytes written so far
}

type UploadProgress struct {
    TotalFiles       int                `json:"total_files"`
    CompletedFiles   int                `json:"completed_files"`
    CurrentFile      string             `json:"current_file"`
    BytesUploaded    int64              `json:"bytes_uploaded"`
    TotalBytes       int64              `json:"total_bytes"`
}

type OutputFileInfo struct {
    OutputID         string             `json:"output_id"`
    Destination      string             `json:"destination"`
    FileSize         int64              `json:"file_size"`
    MD5              string             `json:"md5,omitempty"`
    Duration         float64            `json:"duration,omitempty"`
    MediaInfo        *MediaInfo         `json:"media_info,omitempty"`
}
```

### Example JobStatus

```json
{
  "job_id": "job_abc123",
  "status": "processing",
  "progress": {
    "overall_percent": 45.5,
    "current_step": "Processing video (trim + loudnorm + mix)",
    "step_progress": {
      "ffmpeg_progress": {
        "frame": 1350,
        "fps": 30.2,
        "current_time": "00:00:45.000",
        "total_time": "00:05:00.000",
        "speed": "1.2x",
        "bitrate": "2048kbits/s",
        "total_size": 11534336
      }
    },
    "estimated_completion": "2025-12-15T10:03:45Z"
  },
  "created_at": "2025-12-15T10:00:00Z",
  "updated_at": "2025-12-15T10:01:23Z",
  "started_at": "2025-12-15T10:00:05Z"
}
```

---

## 4. JobLog

Complete execution record for debugging, reproducibility, and auditing.

### Go Struct Definition

```go
package schemas

import "time"

type JobLog struct {
    JobID            string            `json:"job_id"`
    CreatedAt        time.Time         `json:"created_at"`
    CompletedAt      *time.Time        `json:"completed_at,omitempty"`

    // Input Data
    JobSpec          *JobSpec          `json:"job_spec"`
    ProcessingPlan   *ProcessingPlan   `json:"processing_plan,omitempty"`

    // Execution
    ExecutionSteps   []ExecutionStep   `json:"execution_steps"`
    FFmpegVersion    string            `json:"ffmpeg_version"`

    // Results
    Status           JobState          `json:"status"`
    Error            *ProcessingError  `json:"error,omitempty"`
    OutputFiles      []OutputFileInfo  `json:"output_files,omitempty"`

    // Metrics
    Metrics          JobMetrics        `json:"metrics"`

    // System Info
    WorkerID         string            `json:"worker_id,omitempty"`
    WorkerHostname   string            `json:"worker_hostname,omitempty"`
}

type ExecutionStep struct {
    StepID           string            `json:"step_id"`
    Type             string            `json:"type"`                // "download", "ffmpeg", "upload"
    StartedAt        time.Time         `json:"started_at"`
    CompletedAt      *time.Time        `json:"completed_at,omitempty"`
    Duration         time.Duration     `json:"duration,omitempty"`

    // For download/upload steps
    URI              string            `json:"uri,omitempty"`
    BytesTransferred int64             `json:"bytes_transferred,omitempty"`

    // For ffmpeg steps
    CommandID        string            `json:"command_id,omitempty"`
    Command          string            `json:"command,omitempty"`
    ExitCode         int               `json:"exit_code,omitempty"`
    Stdout           string            `json:"stdout,omitempty"`
    Stderr           string            `json:"stderr,omitempty"`

    // Result
    Success          bool              `json:"success"`
    Error            string            `json:"error,omitempty"`
}

type JobMetrics struct {
    TotalDuration      time.Duration `json:"total_duration"`
    ValidationTime     time.Duration `json:"validation_time"`
    PlanningTime       time.Duration `json:"planning_time"`
    DownloadTime       time.Duration `json:"download_time"`
    ProcessingTime     time.Duration `json:"processing_time"`
    UploadTime         time.Duration `json:"upload_time"`

    InputSize          int64         `json:"input_size"`          // Total bytes downloaded
    OutputSize         int64         `json:"output_size"`         // Total bytes uploaded
    TempDiskUsage      int64         `json:"temp_disk_usage"`     // Peak temp disk usage

    ProcessingSpeed    float64       `json:"processing_speed"`    // e.g., 1.5x realtime
    CPUUsagePercent    float64       `json:"cpu_usage_percent,omitempty"`
    MemoryUsageMB      int64         `json:"memory_usage_mb,omitempty"`
}
```

---

## 5. ProcessingError

Structured error information.

### Go Struct Definition

```go
package schemas

type ProcessingError struct {
    Code             ErrorCode              `json:"code"`
    Message          string                 `json:"message"`
    Details          map[string]interface{} `json:"details,omitempty"`

    // FFmpeg-specific
    FFmpegStderr     string                 `json:"ffmpeg_stderr,omitempty"`
    FFmpegExitCode   int                    `json:"ffmpeg_exit_code,omitempty"`

    // Stack trace (for internal errors)
    StackTrace       string                 `json:"stack_trace,omitempty"`

    // Retry info
    Retryable        bool                   `json:"retryable"`
    RetryAfter       *time.Duration         `json:"retry_after,omitempty"`
}

type ErrorCode string

const (
    // Validation Errors (4xx - client errors)
    ErrInvalidJobSpec         ErrorCode = "INVALID_JOB_SPEC"
    ErrInvalidOperation       ErrorCode = "INVALID_OPERATION"
    ErrInvalidInput           ErrorCode = "INVALID_INPUT"
    ErrResourceLimitExceeded  ErrorCode = "RESOURCE_LIMIT_EXCEEDED"
    ErrUnsupportedFormat      ErrorCode = "UNSUPPORTED_FORMAT"

    // Input Errors (4xx)
    ErrInputDownloadFailed    ErrorCode = "INPUT_DOWNLOAD_FAILED"
    ErrInputNotFound          ErrorCode = "INPUT_NOT_FOUND"
    ErrInputForbidden         ErrorCode = "INPUT_FORBIDDEN"
    ErrSSRFBlocked            ErrorCode = "SSRF_BLOCKED"

    // Processing Errors (5xx - server errors)
    ErrFFmpegFailed           ErrorCode = "FFMPEG_FAILED"
    ErrFFmpegTimeout          ErrorCode = "FFMPEG_TIMEOUT"
    ErrOutputUploadFailed     ErrorCode = "OUTPUT_UPLOAD_FAILED"
    ErrInsufficientDiskSpace  ErrorCode = "INSUFFICIENT_DISK_SPACE"
    ErrInsufficientMemory     ErrorCode = "INSUFFICIENT_MEMORY"

    // System Errors (5xx)
    ErrInternalError          ErrorCode = "INTERNAL_ERROR"
    ErrTimeout                ErrorCode = "TIMEOUT"
    ErrCancelled              ErrorCode = "CANCELLED"
)
```

---

## Design Notes

### 1. Type System

**Duration Handling**:
- Use custom `Duration` type that supports multiple formats:
  - ISO 8601: "PT1H30M" (1 hour 30 minutes)
  - Timecode: "01:30:00.500" (HH:MM:SS.ms)
  - Human-readable: "1h30m", "90s"
- Internally stores as `time.Duration`

**Flexible Parameters**:
- `Operation.Params` uses `map[string]interface{}` for flexibility
- Each operator validates and type-converts parameters
- Allows adding new operators without changing core schema

### 2. Validation Strategy

**Three-Level Validation**:
1. **Schema validation**: JSON schema validation (optional, via JSON Schema)
2. **Semantic validation**: JobSpec validator checks references, DAG, security
3. **Operator validation**: Each operator validates its specific parameters

### 3. Versioning

**API Versioning**:
- JobSpec includes implicit version (inferred from fields used)
- Future: Add explicit `spec_version: "v1"` field
- Support multiple versions via adapter pattern

### 4. Extensibility

**Adding New Fields**:
- Use `omitempty` for optional fields
- Use pointers for nullable fields
- Unknown fields ignored during unmarshaling (for forward compatibility)

**Adding New Operators**:
- Only requires implementing `Operator` interface
- No schema changes needed

---

## Next Steps

1. Implement Go structs in `pkg/schemas/`
2. Add JSON marshaling tests
3. Add validation logic in `pkg/compiler/validator/`
4. Generate JSON Schema definitions (for external validation)
5. Write OpenAPI spec for API endpoints

---

**Status**: Ready for review and implementation
