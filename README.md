# Media Pipeline

A declarative, scalable media processing pipeline built on FFmpeg.

[中文文档](README.zh-CN.md)

## Overview

Media Pipeline is a core engine for building declarative video/audio workflows on top of FFmpeg. The repository currently focuses on schemas, operators, planning, and execution; API/queue/store/worker components are planned.

### Key Features

- **Declarative API**: Describe what you want, not how to do it
- **Operator System**: Extensible operator interface (currently: `trim`, `scale`)
- **Distributed (planned)**: Horizontal scaling with multiple workers
- **Type-Safe**: Strong parameter validation and type conversion
- **Extensible**: Add custom operators without modifying core code
- **Observable (planned)**: Metrics, tracing, and structured logging
- **Reliable (planned)**: Retry, failure recovery, and richer error handling

## Architecture

```
┌─────────────┐
│ REST API    │  JobSpec submission, status queries
└─────────────┘
       │
       ▼
┌─────────────┐
│  Validator  │  Parameter validation, SSRF protection
└─────────────┘
       │
       ▼
┌─────────────┐
│   Planner   │  DAG construction, resource estimation
└─────────────┘
       │
       ▼
┌─────────────┐
│ Job Queue   │  Priority scheduling (Redis)
└─────────────┘
       │
       ▼
┌─────────────┐
│   Workers   │  FFmpeg execution, progress tracking
└─────────────┘
       │
       ▼
┌─────────────┐
│   Storage   │  S3/GCS output upload
└─────────────┘
```

## Quick Start

### Example: Trim and Scale Video

```json
{
  "inputs": [
    {
      "id": "video",
      "source": "s3://bucket/input.mp4"
    }
  ],
  "operations": [
    {
      "op": "trim",
      "input": "video",
      "output": "trimmed",
      "params": {
        "start": "00:00:10",
        "duration": "00:05:00"
      }
    },
    {
      "op": "scale",
      "input": "trimmed",
      "output": "scaled",
      "params": {
        "width": 1280,
        "height": 720,
        "algorithm": "lanczos"
      }
    }
  ],
  "outputs": [
    {
      "id": "scaled",
      "destination": "s3://bucket/output.mp4",
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
  ]
}
```

## Project Structure

```
media-pipeline/
├── cmd/
│   ├── api/          # API server
│   └── worker/       # Worker process
├── pkg/
│   ├── schemas/      # Data structures (JobSpec, ProcessingPlan, etc.)
│   ├── operators/    # Operator interface and built-in operators
│   ├── planner/      # DAG builder and resource estimator
│   ├── executor/     # FFmpeg executor
│   ├── store/        # Database layer (PostgreSQL/Redis)
│   └── api/          # HTTP handlers
├── internal/
│   └── config/       # Configuration
└── docs/
    └── plans/        # Design documents
```

## Implementation Status

### ✅ Completed (60%)

- **Schemas Package** (`pkg/schemas/`) - 4 files, 400 lines
  - JobSpec, ProcessingPlan, JobStatus structures
  - Duration type with multiple format support (Go duration, timecode, ISO 8601)
  - MediaInfo structures for video/audio metadata
  - Resource estimation structures (NodeEstimates, ResourceEstimates)

- **Operators Package** (`pkg/operators/`) - 7 files, 800 lines
  - Operator interface (6 core methods)
  - Type system (11 parameter types)
  - Parameter validation framework with declarative rules
  - Type conversion (automatic conversion between formats)
  - Registry mechanism (global operator registration)

- **Built-in Operators** (`pkg/operators/builtin/`)
  - `trim` - Trim video/audio to time range with flexible time formats
  - `scale` - Scale video resolution with algorithm selection (lanczos, bicubic, etc.)

- **Planner Module** (`pkg/planner/`) - 13 files, 1,400 lines, 43 tests
  - DAG construction with cycle detection
  - Topological sorting (Kahn's algorithm)
  - Execution stage computation for parallelization
  - Metadata propagation through operations
  - Resource estimation (CPU, memory, disk)
  - Integrated planner with validation

- **Executor Module** (`pkg/executor/`) - 7 files, 600 lines, 14 tests
  - FFmpeg command builder from ProcessingPlan
  - Real-time progress parsing
  - Process execution with cancellation support
  - Comprehensive error handling

**Total**: 31 files, 3,200 lines of code + 1,900 lines of tests

### 📋 Next Steps

- **Media Prober** - FFprobe wrapper and parallel probing
- **Store Module** - Database layer (PostgreSQL/Redis)
- **Error Handling** - Error taxonomy, FFmpeg parsing, retries
- **API Server** - RESTful endpoints, authentication, webhooks
- **Worker Coordination** - Distributed job execution
- **More Operators** - loudnorm, mix, concat, overlay, etc.

## Design Documents

Comprehensive design documents available in `docs/plans/`:

1. [Architecture Design](docs/plans/2025-12-14-media-pipeline-architecture-design.md)
2. [Schemas Detailed Design](docs/plans/schemas-detailed-design.md)
3. [Planner Module Design](docs/plans/planner-detailed-design.md)
4. [Operator Interface Design](docs/plans/operator-interface-design.md)
5. [API Interface Design](docs/plans/api-interface-design.md)
6. [Distributed State Management](docs/plans/distributed-state-management-design.md)
7. [Error Handling Design](docs/plans/error-handling-design.md)

## Contributing

See [IMPLEMENTATION_GUIDE.md](IMPLEMENTATION_GUIDE.md) for detailed implementation roadmap.

## Testing

Run unit tests:

```bash
go test ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Status**: Core Engine complete (60%). Schemas, Operators, Planner, and Executor modules implemented with comprehensive tests. Ready for media probing, state management, and error handling.
