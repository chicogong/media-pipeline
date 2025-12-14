# Media Pipeline Architecture Design

**Date**: 2025-12-14
**Status**: Approved
**Target**: SaaS developers building video editing platforms

---

## Overview

media-pipeline is an open-source cloud media processing engine that transforms declarative job specifications into deterministic FFmpeg pipelines with comprehensive progress tracking, structured logging, and artifact management.

### Core Value Proposition

- **Declarative Interface**: High-level operators (trim, concat, loudnorm) instead of raw FFmpeg commands
- **Production Ready**: Built-in security, resource limits, error handling, and observability
- **Flexible Deployment**: Single-process mode for quick start, distributed mode for scale
- **Developer Friendly**: REST API, structured errors, real-time progress, detailed logs

---

## Design Decisions

### 1. Target Users and Use Cases

**Primary Target**: SaaS developers building video editing platforms (like Descript, Riverside)

**Key Requirements**:
- Embeddable media processing engine
- API-first design for easy integration
- Strong security for multi-tenant scenarios
- Comprehensive progress reporting
- Scalable architecture

### 2. Deployment Architecture

**Chosen**: Hybrid Mode (single-process + API/Worker separation)

**Rationale**:
- MVP can start quickly with single-process mode (zero dependencies)
- Production can switch to separated architecture (better resource isolation and scalability)
- Core design follows separation of concerns, single-process is just a runtime variation

**Modes**:
- **Single-process**: API server executes jobs in goroutines (development, demo, small-scale)
- **Distributed**: API server + Worker pool with queue (production, horizontal scaling)

### 3. API Design and Job Management

**Chosen**: REST API + Built-in Queue

**Endpoints**:
- `POST /jobs` - Submit job, returns job_id
- `GET /jobs/{id}` - Query status and progress
- `GET /jobs/{id}/logs` - Retrieve execution logs
- `DELETE /jobs/{id}` - Cancel job

**Queue Options**:
- Memory queue (single-process mode)
- Redis queue (distributed mode)
- Configurable via environment variables

### 4. Storage Strategy

**Chosen**: Local + Object Storage (abstracted interface)

**Input Sources**:
- Local filesystem paths
- HTTP/HTTPS URLs
- S3 URIs (`s3://bucket/key`)
- Google Cloud Storage (`gs://bucket/key`)

**Output Destinations**:
- Local filesystem
- Direct upload to object storage

**Implementation**: `storage.Storage` interface with pluggable backends

### 5. Feature Scope

**Chosen**: Complete Feature Set (8 major categories)

1. **Input Processing**: ffprobe detection, multi-source support, auto-alignment
2. **Timeline Editing**: trim, concat, split, transitions (xfade, acrossfade)
3. **Audio Processing**: loudnorm, mixing, ducking, silence detection/removal
4. **Video Processing**: crop, scale, rotate, deinterlace, color space conversion
5. **Graphics/Text**: subtitle burn-in, watermarks, overlay, drawtext
6. **Output/Packaging**: transcoding, HLS/DASH, thumbnails, waveforms
7. **Progress/Logs**: Machine-readable progress, structured logs, reproducibility
8. **Security/Limits**: SSRF protection, resource limits, operator whitelist

### 6. JobSpec Design Style

**Chosen**: Declarative + High-Level Operators

**Example**:
```json
{
  "inputs": [
    {"id": "video1", "source": "s3://bucket/raw/meeting.mp4"},
    {"id": "bgm", "source": "https://cdn.example.com/music.mp3"}
  ],
  "operations": [
    {"op": "trim", "input": "video1", "start": "00:00:10", "duration": "00:05:00", "output": "trimmed"},
    {"op": "loudnorm", "input": "trimmed", "target_lufs": -16, "output": "normalized"},
    {"op": "mix", "inputs": ["normalized", "bgm"], "mode": "ducking", "output": "final_audio"},
    {"op": "export", "input": "final_audio", "format": "mp4", "output": "result.mp4"}
  ],
  "outputs": [
    {"id": "result.mp4", "destination": "s3://bucket/output/meeting-edited.mp4"}
  ]
}
```

**Rationale**:
- Users declare "what to do", not "how to do it"
- Compiler generates optimal FFmpeg filtergraph
- Enables validation, optimization, resource planning
- Encapsulates best practices (e.g., loudnorm two-pass)

---

## Architecture

### Three-Layer Compilation Architecture

```
User JobSpec (JSON)
    ↓
[1. Validator] - Type checking, security checks, resource limits
    ↓
[2. Planner] - Generate ProcessingPlan (DAG, resource estimation)
    ↓
[3. Codegen] - ProcessingPlan → FFmpeg command + filtergraph
    ↓
[4. Runner] - Execute FFmpeg, parse progress, manage artifacts
```

**Benefits**:
- Clear separation of concerns (validation, planning, compilation, execution)
- Each layer independently testable
- ProcessingPlan is machine-readable intermediate representation
- Supports "plan preview" feature (show execution plan before running)
- Easy to extend with new operators

### Core Components

**1. API Server** (`cmd/api-server`)
- Provides REST API
- Handles job submission and status queries
- Single-process mode: directly invokes Runner
- Distributed mode: pushes tasks to queue

**2. Worker** (`cmd/worker`, distributed mode only)
- Pulls tasks from queue
- Invokes Compiler and Runner
- Horizontally scalable

**3. Compiler** (`pkg/compiler`)
- `validator/`: Validates JobSpec (types, security, limits)
- `planner/`: Generates ProcessingPlan (dependency graph, resource estimation)
- `codegen/`: ProcessingPlan → FFmpeg commands + filtergraph

**4. Runner** (`pkg/runner`)
- Executes FFmpeg processes (via `exec.Command`)
- Parses `-progress pipe:1` output in real-time
- Manages temporary files, artifact uploads, cleanup

**5. Storage** (`pkg/storage`)
- Interface abstraction: `Get(uri)`, `Put(uri, data)`
- Implementations: LocalStorage, S3Storage, HTTPStorage, GCSStorage

**6. Schemas** (`pkg/schemas`)
- `JobSpec`: User-submitted job definition
- `ProcessingPlan`: Intermediate representation
- `JobStatus`: Job state and progress

---

## Data Flow

### From JobSpec to FFmpeg

**Step 1: User Submits JobSpec**
```json
POST /jobs
{
  "inputs": [...],
  "operations": [...],
  "outputs": [...]
}
```

**Step 2: Validator Validates**
- Check input references exist (dependency graph completeness)
- Validate parameter types (e.g., `start` is valid timecode, `target_lufs` in range)
- Security checks: source URIs use whitelisted protocols (http/https/s3, block file:// internal paths)
- Resource limits: total duration < 2 hours, resolution < 4K

**Step 3: Planner Generates ProcessingPlan**
- Build dependency graph (DAG): `video1 → trimmed → normalized → final_audio → result.mp4`
- Resource estimation: estimate CPU time based on input duration and operation complexity
- Determine execution order (topological sort)

**Step 4: Codegen Generates FFmpeg Command**
- Generate filtergraph from ProcessingPlan:
  ```
  [0:v]trim=start=10:duration=300[v_trimmed];
  [0:a]atrim=start=10:duration=300,loudnorm=I=-16:TP=-1.5[a_norm];
  [a_norm][1:a]amix=inputs=2:weights=1 0.3[a_final]
  ```
- Generate complete command:
  ```bash
  ffmpeg -i s3://bucket/raw/meeting.mp4 -i https://cdn.example.com/music.mp3 \
    -filter_complex "[filtergraph]" \
    -map "[v_trimmed]" -map "[a_final]" \
    -c:v libx264 -c:a aac \
    -progress pipe:1 \
    /tmp/result.mp4
  ```

**Step 5: Runner Executes**
- Download inputs from sources
- Execute FFmpeg with progress pipe
- Parse progress in real-time (update job status)
- Upload outputs to destinations
- Cleanup temporary files

---

## Operator System

### Extensible Architecture

**Operator Interface**:
```go
type Operator interface {
    Name() string
    Validate(params map[string]interface{}) error
    Plan(params map[string]interface{}, inputs []string) (*PlanNode, error)
    Compile(node *PlanNode) (string, error)
}
```

**Registration Mechanism**:
```go
var registry = map[string]Operator{}

func Register(op Operator) {
    registry[op.Name()] = op
}

func init() {
    Register(&TrimOperator{})
    Register(&LoudnormOperator{})
    Register(&ConcatOperator{})
    // ... other operators
}
```

**Example: Loudnorm Operator**:
```go
type LoudnormOperator struct{}

func (o *LoudnormOperator) Name() string { return "loudnorm" }

func (o *LoudnormOperator) Validate(params map[string]interface{}) error {
    lufs, ok := params["target_lufs"].(float64)
    if !ok || lufs < -24 || lufs > -16 {
        return fmt.Errorf("target_lufs must be between -24 and -16")
    }
    return nil
}

func (o *LoudnormOperator) Compile(node *PlanNode) (string, error) {
    lufs := node.Params["target_lufs"].(float64)
    return fmt.Sprintf("loudnorm=I=%.1f:TP=-1.5:LRA=11", lufs), nil
}
```

### MVP Operator List

**Timeline Editing**:
- `trim` - Trim by time range
- `concat` - Concatenate (lossless or re-encode)
- `split` - Split into segments
- `xfade` - Video transitions (crossfade, wipe, slide, etc.)
- `acrossfade` - Audio transitions

**Audio Processing**:
- `loudnorm` - EBU R128 loudness normalization
- `mix` - Audio mixing
- `ducking` - Auto-ducking (lower background when voice present)
- `volume` - Volume adjustment
- `silencedetect` / `silenceremove` - Silence detection and removal

**Video Processing**:
- `crop` - Crop video
- `scale` - Resize video
- `pad` - Add padding
- `rotate` - Rotate video
- `fps` - Change frame rate

**Graphics/Text**:
- `subtitles` - Burn-in subtitles (libass)
- `overlay` - Overlay images/videos (watermarks, picture-in-picture)
- `drawtext` - Draw text (titles, timestamps)

**Output**:
- `export` - Transcode and export
- `thumbnail` - Generate thumbnail images
- `waveform` - Generate audio waveform

---

## Security and Limits

### Multi-Tenant Security

**1. Input Source Whitelist (Prevent SSRF)**
```go
var AllowedProtocols = []string{"https", "http", "s3", "gs", "azure"}
var BlockedNetworks = []string{
    "127.0.0.0/8",    // localhost
    "10.0.0.0/8",     // private network
    "172.16.0.0/12",  // private network
    "192.168.0.0/16", // private network
    "169.254.0.0/16", // link-local (AWS metadata)
}

func ValidateSourceURI(uri string) error {
    // Check protocol whitelist
    // Check IP blocklist for HTTP/HTTPS
    // Prevent internal network access
}
```

**2. Resource Limits (Prevent Abuse)**
```go
type Limits struct {
    MaxDuration       time.Duration // e.g., 2 hours
    MaxResolution     string        // e.g., "3840x2160"
    MaxOutputSize     int64         // e.g., 5GB
    MaxConcurrentJobs int           // e.g., 5 per user
    Timeout           time.Duration // e.g., 30 minutes
}
```

Validated in Validator phase after ffprobe detection.

**3. Operator Whitelist**
- Users can only use registered operators
- Cannot directly inject arbitrary FFmpeg commands
- All parameters validated by operator's `Validate()` method

**4. Temporary Disk Quota**
- Check available disk space before execution
- Reserve 3x estimated size (input + intermediate + output)
- Clean up temporary files after job completion/failure

---

## Runner Execution

### Process Management

```go
cmd := exec.Command("ffmpeg",
    "-progress", "pipe:1",  // Progress to stdout
    "-i", inputPath,
    "-filter_complex", filtergraph,
    "-y", outputPath,
)

stdout, _ := cmd.StdoutPipe()  // Capture progress
stderr, _ := cmd.StderrPipe()  // Capture error logs

cmd.Start()

go parseProgress(stdout, jobID)  // Parse and update status
go captureErrors(stderr, jobID)  // Save error logs

cmd.Wait()
```

### Progress Parsing

FFmpeg `-progress` output format (updated every second):
```
frame=150
fps=30.5
total_size=5242880
out_time_us=5000000
out_time=00:00:05.000000
speed=1.2x
progress=continue
```

Parsing logic:
- Extract `out_time_us` ÷ total duration = percentage
- Extract `speed` (processing speed, e.g., 1.2x = 20% faster than realtime)
- Update job status in database/memory

### State Machine

Job states:
```
pending → downloading_inputs → processing → uploading_outputs → completed
                                    ↓
                                 failed
```

Error handling:
- **Input download failure**: Return 4xx error (client issue)
- **FFmpeg execution failure**: Save full stderr, parse common errors, return structured error
- **Output upload failure**: Retry up to 3 times (S3 transient network issues)

### Artifact Management

- Temporary files in `/tmp/media-pipeline/{job_id}/`
- On success: upload to `outputs.destination`, delete temp files
- On failure: keep temp files for 1 hour (debugging), then clean up

---

## Testing Strategy

### Layered Testing

**1. Unit Tests**
- Test each operator independently (Validate, Plan, Compile)
- Test validator security checks
- Test planner DAG construction

**2. Integration Tests**
- Test complete compilation flow (JobSpec → ProcessingPlan → FFmpeg command)
- Verify generated filtergraph correctness

**3. E2E Tests**
- Execute real FFmpeg commands with test fixtures
- Verify output file correctness (duration, resolution, codec)
- Use `ffprobe` to validate outputs

**4. Golden Tests**
- Save known-correct FFmpeg commands as "golden" references
- Detect regressions in code generation

**5. Security Tests**
- Verify SSRF protection (block localhost, private networks, AWS metadata)
- Test resource limit enforcement

**6. Performance Benchmarks**
- Benchmark compilation speed for large jobs (100+ operations)

---

## Project Structure

```
media-pipeline/
├── cmd/
│   ├── api-server/          # API service entry point
│   ├── worker/              # Worker process (distributed mode)
│   └── mpctl/               # CLI tool
│
├── pkg/
│   ├── schemas/             # Data structures
│   ├── compiler/            # JobSpec → FFmpeg compiler
│   │   ├── validator/
│   │   ├── planner/
│   │   ├── codegen/
│   │   └── operators/       # Operator implementations
│   ├── runner/              # Execution engine
│   ├── storage/             # Storage abstraction
│   ├── executor/            # Task executor (shared logic)
│   ├── queue/               # Queue abstraction
│   └── api/                 # API handlers
│
├── presets/                 # Preset templates (podcast, meeting, etc.)
├── docs/
│   ├── plans/               # Design documents
│   ├── api.md
│   └── operators.md
│
├── tests/
│   ├── e2e/
│   ├── fixtures/
│   └── golden/
│
├── scripts/
├── deployments/
│   ├── docker/
│   └── k8s/
│
├── go.mod
└── README.md
```

---

## Error Handling and Observability

### Structured Error System

```go
type ErrorCode string

const (
    ErrInvalidJobSpec     ErrorCode = "INVALID_JOB_SPEC"
    ErrInputDownloadFailed ErrorCode = "INPUT_DOWNLOAD_FAILED"
    ErrFFmpegFailed       ErrorCode = "FFMPEG_FAILED"
    ErrOutputUploadFailed ErrorCode = "OUTPUT_UPLOAD_FAILED"
    ErrTimeout            ErrorCode = "TIMEOUT"
    ErrResourceLimit      ErrorCode = "RESOURCE_LIMIT_EXCEEDED"
)

type ProcessingError struct {
    Code           ErrorCode              `json:"code"`
    Message        string                 `json:"message"`
    Details        map[string]interface{} `json:"details"`
    FFmpegStderr   string                 `json:"ffmpeg_stderr,omitempty"`
    FFmpegExitCode int                    `json:"ffmpeg_exit_code,omitempty"`
}
```

### Complete Execution Logs

Every job saves full execution context for debugging and reproducibility:

```go
type JobLog struct {
    JobID          string              `json:"job_id"`
    CreatedAt      time.Time           `json:"created_at"`
    JobSpec        *JobSpec            `json:"job_spec"`
    ProcessingPlan *ProcessingPlan     `json:"processing_plan"`
    FFmpegCommand  string              `json:"ffmpeg_command"`
    FFmpegVersion  string              `json:"ffmpeg_version"`
    ExecutionSteps []ExecutionStep     `json:"execution_steps"`
    Status         string              `json:"status"`
    Error          *ProcessingError    `json:"error,omitempty"`
    OutputFiles    []OutputFileInfo    `json:"output_files,omitempty"`
    Metrics        JobMetrics          `json:"metrics"`
}

type JobMetrics struct {
    TotalDuration   time.Duration `json:"total_duration"`
    DownloadTime    time.Duration `json:"download_time"`
    ProcessingTime  time.Duration `json:"processing_time"`
    UploadTime      time.Duration `json:"upload_time"`
    ProcessingSpeed float64       `json:"processing_speed"` // e.g., 1.5x realtime
}
```

### Real-Time Progress API

```
GET /jobs/{id}
{
  "job_id": "abc123",
  "status": "processing",
  "progress": {
    "percent": 45.5,
    "current_step": "processing",
    "ffmpeg_progress": {
      "frame": 1350,
      "fps": 30.2,
      "time": "00:00:45.000",
      "speed": "1.2x"
    }
  },
  "estimated_completion": "2025-12-14T10:03:45Z"
}

GET /jobs/{id}/logs
{
  "job_id": "abc123",
  "logs": [
    {"timestamp": "...", "level": "INFO", "message": "Job created"},
    {"timestamp": "...", "level": "INFO", "message": "Processing 45% complete"}
  ]
}
```

### Health Check Endpoint

```
GET /health
{
  "status": "healthy",
  "checks": {
    "ffmpeg": "ok",
    "disk_space": "ok",
    "queue": "ok",
    "storage": "ok"
  },
  "version": "1.0.0",
  "ffmpeg_version": "6.0"
}
```

### Debug Mode

JobSpec supports `debug` flag:
```json
{
  "debug": true,
  "operations": [...]
}
```

In debug mode:
- Keep all intermediate files
- Output full FFmpeg commands (for manual reproduction)
- Log detailed metadata for each operator

---

## Deployment Modes

### Single-Process Mode

**Use Case**: Development, demo, small-scale usage

**How It Works**:
- API server executes jobs directly in goroutines
- No external dependencies (queue, database)
- Job state kept in memory

**Start Command**:
```bash
./api-server --mode=standalone
```

### Distributed Mode

**Use Case**: Production, large-scale usage

**Components**:
- API server: handles HTTP requests, pushes jobs to queue
- Worker pool: pulls jobs from queue, executes them
- Queue: Redis or similar message queue
- Optional: Database for job persistence

**Start Commands**:
```bash
# API server
./api-server --mode=distributed --queue=redis://localhost:6379

# Workers (can run multiple instances)
./worker --queue=redis://localhost:6379
```

---

## Implementation Roadmap

### Phase 1: Core Infrastructure (Weeks 1-2)
- [ ] Project structure and Go modules setup
- [ ] Schemas: JobSpec, ProcessingPlan, JobStatus
- [ ] Validator: basic type checking and security checks
- [ ] Storage interface: Local and HTTP implementations
- [ ] Basic API server (single-process mode)

### Phase 2: Compilation Pipeline (Weeks 3-4)
- [ ] Planner: DAG construction and dependency resolution
- [ ] Codegen: filtergraph generation
- [ ] Operator interface and registration
- [ ] MVP operators: trim, concat, export

### Phase 3: Execution Engine (Weeks 5-6)
- [ ] Runner: FFmpeg process management
- [ ] Progress parsing (-progress pipe:1)
- [ ] Artifact management (temp files, cleanup)
- [ ] Error handling and structured errors

### Phase 4: Audio Operators (Week 7)
- [ ] loudnorm (with two-pass support)
- [ ] mix (basic audio mixing)
- [ ] volume, silencedetect, silenceremove

### Phase 5: Video Operators (Week 8)
- [ ] crop, scale, pad, rotate
- [ ] xfade (video transitions)
- [ ] fps (frame rate conversion)

### Phase 6: Graphics Operators (Week 9)
- [ ] subtitles (burn-in with libass)
- [ ] overlay (watermarks, picture-in-picture)
- [ ] drawtext (titles, timestamps)

### Phase 7: Advanced Features (Week 10)
- [ ] HLS/DASH output
- [ ] Thumbnail generation
- [ ] Waveform generation
- [ ] Presets (podcast, meeting, social video)

### Phase 8: Distributed Mode (Week 11)
- [ ] Queue abstraction (memory, Redis)
- [ ] Worker implementation
- [ ] Job persistence (optional database)

### Phase 9: Observability (Week 12)
- [ ] Structured logging
- [ ] JobLog with metrics
- [ ] Health check endpoint
- [ ] Debug mode

### Phase 10: Testing and Documentation (Weeks 13-14)
- [ ] Comprehensive unit tests
- [ ] Integration tests
- [ ] E2E tests with fixtures
- [ ] API documentation
- [ ] Operator documentation
- [ ] Deployment guides

### Phase 11: Polish and Release (Week 15)
- [ ] Docker images
- [ ] Kubernetes manifests
- [ ] Example projects
- [ ] README with quickstart
- [ ] CONTRIBUTING guide
- [ ] CI/CD setup

---

## Success Criteria

**Functional**:
- ✅ All 8 feature categories implemented
- ✅ Support for complete workflow (input → edit → output)
- ✅ Both single-process and distributed modes working

**Non-Functional**:
- ✅ Security: SSRF protection, resource limits enforced
- ✅ Reliability: Structured errors, retry logic, cleanup on failure
- ✅ Observability: Real-time progress, detailed logs, health checks
- ✅ Performance: Processing speed close to FFmpeg native performance
- ✅ Developer Experience: Clear API, good documentation, easy setup

**Community**:
- ✅ MIT license
- ✅ Comprehensive README
- ✅ Example use cases
- ✅ Active issue tracking

---

## References

- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)
- [FFmpeg Progress Output Format](https://ffmpeg.org/ffmpeg.html#Main-options)
- [FFmpeg Filtergraph Syntax](https://ffmpeg.org/ffmpeg-filters.html)
- [EBU R128 Loudness Normalization](https://ffmpeg.org/ffmpeg-filters.html#loudnorm)
- [xfade Transitions](https://trac.ffmpeg.org/wiki/Xfade)

---

**Next Steps**: Ready to set up for implementation?
