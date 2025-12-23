# Media Pipeline Architecture

Comprehensive architecture documentation with detailed diagrams.

## Table of Contents

- [System Overview](#system-overview)
- [Module Architecture](#module-architecture)
- [Data Flow](#data-flow)
- [Operator System](#operator-system)
- [Planner Architecture](#planner-architecture)
- [Executor Architecture](#executor-architecture)
- [API Layer](#api-layer)
- [Future Distributed Architecture](#future-distributed-architecture)

## System Overview

### High-Level Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        WebApp[Web Application]
        Mobile[Mobile App]
        CLI[CLI Tool]
    end

    subgraph "API Layer"
        Gateway[API Gateway<br/>REST Endpoints]
        Auth[Authentication<br/>Future]
        RateLimit[Rate Limiting<br/>Future]
    end

    subgraph "Business Logic Layer"
        direction TB
        Validator[Validator<br/>JobSpec Validation]
        Prober[Media Prober<br/>FFprobe Wrapper]
        Planner[Planner<br/>DAG Builder]
        Executor[Executor<br/>FFmpeg Runner]
    end

    subgraph "Data Layer"
        Store[Store Interface]
        MemStore[Memory Store<br/>Current]
        PostgresStore[PostgreSQL Store<br/>Future]
        RedisCache[Redis Cache<br/>Future]
    end

    subgraph "Storage Layer"
        Local[Local Filesystem]
        S3[S3/Object Storage<br/>Future]
        NFS[Network Storage<br/>Future]
    end

    WebApp --> Gateway
    Mobile --> Gateway
    CLI --> Gateway

    Gateway --> Auth
    Auth --> RateLimit
    RateLimit --> Validator

    Validator --> Prober
    Prober --> Planner
    Planner --> Executor

    Gateway --> Store
    Store --> MemStore
    Store -.->|Future| PostgresStore
    Store -.->|Future| RedisCache

    Executor --> Local
    Executor -.->|Future| S3
    Executor -.->|Future| NFS

    style Gateway fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Validator fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Prober fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Planner fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Executor fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style MemStore fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
```

## Module Architecture

### Core Modules

```mermaid
graph LR
    subgraph "pkg/schemas"
        JobSpec[JobSpec<br/>Job Definition]
        ProcessingPlan[ProcessingPlan<br/>Execution Plan]
        JobStatus[JobStatus<br/>Job State]
        MediaInfo[MediaInfo<br/>Media Metadata]
    end

    subgraph "pkg/operators"
        Interface[Operator Interface<br/>6 Core Methods]
        Registry[Global Registry<br/>Operator Lookup]
        TypeSystem[Type System<br/>11 Parameter Types]
        Validation[Validation Framework<br/>Declarative Rules]

        subgraph "Built-in Operators"
            Trim[Trim<br/>Time Range]
            Scale[Scale<br/>Resolution]
        end
    end

    subgraph "pkg/planner"
        DAGBuilder[DAG Builder<br/>Graph Construction]
        Topo[Topological Sort<br/>Kahn's Algorithm]
        Estimator[Resource Estimator<br/>CPU/Memory/Disk]
        Metadata[Metadata Propagation<br/>Type Inference]
    end

    subgraph "pkg/executor"
        CommandBuilder[Command Builder<br/>FFmpeg Args]
        ProcessManager[Process Manager<br/>Execution & Cancel]
        ProgressParser[Progress Parser<br/>Real-time Updates]
    end

    JobSpec --> DAGBuilder
    DAGBuilder --> ProcessingPlan
    ProcessingPlan --> CommandBuilder

    Interface --> Trim
    Interface --> Scale
    Registry --> Interface

    DAGBuilder --> Registry
    CommandBuilder --> Registry

    style JobSpec fill:#FFC107,stroke:#333,stroke-width:2px
    style ProcessingPlan fill:#FFC107,stroke:#333,stroke-width:2px
    style Interface fill:#9C27B0,stroke:#333,stroke-width:2px,color:#fff
    style DAGBuilder fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style CommandBuilder fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
```

## Data Flow

### Job Processing Data Flow

```mermaid
flowchart TD
    Start([Client Submits JobSpec])

    subgraph "Validation Phase"
        V1[Parse JSON]
        V2[Validate Schema]
        V3[Check Parameters]
        V4[SSRF Protection]
    end

    subgraph "Probing Phase"
        P1[Extract Input URLs]
        P2[Download Samples<br/>Future]
        P3[Run FFprobe]
        P4[Parse MediaInfo]
    end

    subgraph "Planning Phase"
        PL1[Build Dependency Graph]
        PL2[Detect Cycles]
        PL3[Topological Sort]
        PL4[Propagate Metadata]
        PL5[Estimate Resources]
    end

    subgraph "Execution Phase"
        E1[Generate FFmpeg Command]
        E2[Start Process]
        E3[Parse Progress]
        E4[Update Status]
    end

    Finish([Job Completed])
    Error([Job Failed])

    Start --> V1
    V1 --> V2
    V2 --> V3
    V3 --> V4
    V4 -->|Valid| P1
    V4 -->|Invalid| Error

    P1 --> P2
    P2 --> P3
    P3 --> P4
    P4 -->|Success| PL1
    P4 -->|Failure| Error

    PL1 --> PL2
    PL2 -->|No Cycles| PL3
    PL2 -->|Cycle Detected| Error
    PL3 --> PL4
    PL4 --> PL5
    PL5 --> E1

    E1 --> E2
    E2 --> E3
    E3 --> E4
    E4 -->|Success| Finish
    E4 -->|Error| Error

    style Start fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Finish fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Error fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### Store Data Model

```mermaid
erDiagram
    JOB {
        string JobID PK
        timestamp Created
        timestamp Updated
        JobState Status
        JobSpec Spec
        ProcessingPlan Plan
        Progress Progress
        ErrorInfo Error
        timestamp StartedAt
        timestamp CompletedAt
        OutputFile[] OutputFiles
        int RetryCount
        string WorkerID
    }

    JOB_SPEC {
        Input[] Inputs
        Operation[] Operations
        Output[] Outputs
    }

    INPUT {
        string ID
        string Source
    }

    OPERATION {
        string Op
        string Input
        string Output
        map Params
    }

    OUTPUT {
        string ID
        string Destination
        CodecConfig Codec
    }

    PROCESSING_PLAN {
        Node[] Nodes
        Edge[] Edges
        Stage[] Stages
        ResourceEstimates Resources
    }

    JOB ||--|| JOB_SPEC : contains
    JOB ||--o| PROCESSING_PLAN : generates
    JOB_SPEC ||--|{ INPUT : has
    JOB_SPEC ||--|{ OPERATION : has
    JOB_SPEC ||--|{ OUTPUT : has
```

## Operator System

### Operator Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Registration: System Startup

    Registration --> Idle: Registered in GlobalRegistry
    Idle --> Validation: Plan() called

    Validation --> MetadataGen: Validate() success
    Validation --> Error: Validate() failure

    MetadataGen --> CommandGen: EstimateOutputMetadata()
    CommandGen --> Idle: BuildCommand()

    Error --> [*]
    Idle --> [*]: Shutdown

    note right of Registration
        init() function
        registers operator
    end note

    note right of Validation
        Type checking,
        parameter validation,
        range checks
    end note

    note right of MetadataGen
        Infer output format,
        resolution, duration
    end note

    note right of CommandGen
        Generate FFmpeg
        filter arguments
    end note
```

### Type System

```mermaid
graph TB
    subgraph "Type Hierarchy"
        Type[Parameter Type]

        String[String<br/>Text values]
        Int[Int<br/>Integers]
        Float[Float<br/>Decimals]
        Bool[Bool<br/>true/false]
        Duration[Duration<br/>Time spans]

        subgraph "Complex Types"
            Object[Object<br/>Nested maps]
            Array[Array<br/>Lists]
            Enum[Enum<br/>Fixed choices]
        end

        subgraph "Media Types"
            Resolution[Resolution<br/>1920x1080]
            Timecode[Timecode<br/>HH:MM:SS.mmm]
            Codec[Codec<br/>Video/Audio config]
        end
    end

    subgraph "Validation Rules"
        Required[Required]
        Range[Range<br/>Min/Max]
        Pattern[Pattern<br/>Regex]
        Custom[Custom<br/>Validation Func]
    end

    Type --> String
    Type --> Int
    Type --> Float
    Type --> Bool
    Type --> Duration
    Type --> Object
    Type --> Array
    Type --> Enum
    Type --> Resolution
    Type --> Timecode
    Type --> Codec

    String --> Required
    Int --> Range
    String --> Pattern
    Type --> Custom

    style Type fill:#9C27B0,stroke:#333,stroke-width:2px,color:#fff
    style Required fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style Range fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
    style Pattern fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### Built-in Operators

```mermaid
graph LR
    subgraph "Input Operators"
        Download[Download<br/>Future]
        Probe[Probe<br/>Metadata]
    end

    subgraph "Transform Operators"
        Trim[Trim<br/>✅ Implemented]
        Scale[Scale<br/>✅ Implemented]
        Loudnorm[Loudnorm<br/>Future]
        Mix[Mix Audio<br/>Future]
        Overlay[Overlay Video<br/>Future]
        Concat[Concatenate<br/>Future]
    end

    subgraph "Output Operators"
        Encode[Encode<br/>Codec Config]
        Upload[Upload<br/>S3/GCS Future]
    end

    Input[Input Media] --> Download
    Download --> Probe
    Probe --> Trim
    Trim --> Scale
    Scale --> Loudnorm
    Loudnorm --> Mix
    Mix --> Overlay
    Overlay --> Concat
    Concat --> Encode
    Encode --> Upload
    Upload --> Output[Output Media]

    style Trim fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Scale fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Loudnorm fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Mix fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Overlay fill:#9E9E9E,stroke:#333,stroke-width:2px
    style Concat fill:#9E9E9E,stroke:#333,stroke-width:2px
```

## Planner Architecture

### DAG Construction Process

```mermaid
flowchart TD
    Start([JobSpec Input])

    subgraph "Graph Building"
        B1[Create Input Nodes]
        B2[Create Operation Nodes]
        B3[Create Output Nodes]
        B4[Build Edges from Dependencies]
    end

    subgraph "Validation"
        V1{Has Cycles?}
        V2{All Inputs Resolved?}
        V3{Valid Operators?}
    end

    subgraph "Optimization"
        O1[Topological Sort]
        O2[Compute Stages]
        O3[Parallel Groups]
    end

    subgraph "Metadata Propagation"
        M1[Input Metadata]
        M2[Operator Transform]
        M3[Output Metadata]
    end

    subgraph "Resource Estimation"
        R1[Estimate CPU Usage]
        R2[Estimate Memory]
        R3[Estimate Disk I/O]
        R4[Estimate Duration]
    end

    Finish([ProcessingPlan Output])
    Error([Planning Error])

    Start --> B1
    B1 --> B2
    B2 --> B3
    B3 --> B4

    B4 --> V1
    V1 -->|No| V2
    V1 -->|Yes| Error
    V2 -->|Yes| V3
    V2 -->|No| Error
    V3 -->|Yes| O1
    V3 -->|No| Error

    O1 --> O2
    O2 --> O3

    O3 --> M1
    M1 --> M2
    M2 --> M3

    M3 --> R1
    R1 --> R2
    R2 --> R3
    R3 --> R4

    R4 --> Finish

    style Start fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Finish fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style Error fill:#F44336,stroke:#333,stroke-width:2px,color:#fff
```

### Execution Stages

```mermaid
gantt
    title Parallel Execution Stages
    dateFormat  X
    axisFormat %s

    section Stage 0
    Input Node 1    :0, 10
    Input Node 2    :0, 10

    section Stage 1
    Trim Operation  :10, 30

    section Stage 2
    Scale Operation :40, 50

    section Stage 3
    Encode Output 1 :90, 110
    Encode Output 2 :90, 110

    section Dependencies
    Stage 1 waits for Stage 0 :crit, 10, 0
    Stage 2 waits for Stage 1 :crit, 40, 0
    Stage 3 waits for Stage 2 :crit, 90, 0
```

## Executor Architecture

### FFmpeg Execution Flow

```mermaid
sequenceDiagram
    participant E as Executor
    participant CB as CommandBuilder
    participant PM as ProcessManager
    participant FF as FFmpeg Process
    participant PP as ProgressParser
    participant C as Callback

    E->>CB: BuildCommand(plan)
    CB->>CB: Generate FFmpeg args
    CB-->>E: []string (command)

    E->>PM: Start(command)
    PM->>FF: exec.CommandContext()
    FF-->>PM: Process started

    loop Every stderr line
        FF->>PP: stderr output
        PP->>PP: Parse progress
        PP->>C: OnProgress(frame, fps, bitrate)
        C-->>E: Update job status
    end

    FF-->>PM: Process exit
    PM-->>E: Success/Error

    alt Success
        E->>E: Verify output files
        E-->>E: Complete
    else Error
        E->>E: Parse error message
        E-->>E: Failed
    end
```

### Command Builder

```mermaid
graph TB
    Plan[ProcessingPlan]

    subgraph "Input Processing"
        I1[Parse Input Nodes]
        I2[Generate -i flags]
        I3[Map input streams]
    end

    subgraph "Filter Graph"
        F1[Build filter_complex]
        F2[Chain operators]
        F3[Label streams]
    end

    subgraph "Output Processing"
        O1[Parse Output Nodes]
        O2[Apply codec settings]
        O3[Set output paths]
    end

    subgraph "Optimization"
        Opt1[Enable hardware accel]
        Opt2[Thread configuration]
        Opt3[Progress reporting]
    end

    Result[FFmpeg Command]

    Plan --> I1
    I1 --> I2
    I2 --> I3
    I3 --> F1
    F1 --> F2
    F2 --> F3
    F3 --> O1
    O1 --> O2
    O2 --> O3
    O3 --> Opt1
    Opt1 --> Opt2
    Opt2 --> Opt3
    Opt3 --> Result

    style Plan fill:#FFC107,stroke:#333,stroke-width:2px
    style Result fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
```

## API Layer

### API Request Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant M as Middleware Chain
    participant H as Handler
    participant S as Store
    participant BG as Background Worker

    C->>M: HTTP Request

    Note over M: Logging Middleware
    M->>M: Log request details

    Note over M: CORS Middleware
    M->>M: Add CORS headers

    Note over M: Recovery Middleware
    M->>M: Setup panic recovery

    M->>H: Forward to handler
    H->>H: Parse request body
    H->>H: Validate input

    H->>S: CreateJob(job)
    S-->>H: job_id

    H->>BG: Start async processing
    Note over BG: goroutine

    H-->>M: Response (201 Created)
    M->>M: Log response
    M-->>C: HTTP Response

    Note over BG: Background Processing
    BG->>S: UpdateStatus(validating)
    BG->>BG: Probe → Plan → Execute
    BG->>S: UpdateStatus(completed)
```

### Middleware Chain

```mermaid
graph LR
    Request[HTTP Request]

    subgraph "Middleware Chain"
        Log[Logging<br/>Request/Response]
        CORS[CORS<br/>Headers]
        Recovery[Panic Recovery<br/>Error Handler]
        Auth[Authentication<br/>Future]
        RateLimit[Rate Limiting<br/>Future]
    end

    Handler[Route Handler]
    Response[HTTP Response]

    Request --> Log
    Log --> CORS
    CORS --> Recovery
    Recovery --> Auth
    Auth --> RateLimit
    RateLimit --> Handler
    Handler --> Response

    style Log fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style CORS fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Recovery fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style Auth fill:#9E9E9E,stroke:#333,stroke-width:2px
    style RateLimit fill:#9E9E9E,stroke:#333,stroke-width:2px
```

## Future Distributed Architecture

### Worker Pool Architecture

```mermaid
graph TB
    subgraph "API Layer"
        API1[API Server 1]
        API2[API Server 2]
        API3[API Server 3]
    end

    subgraph "Message Queue"
        Queue[(Redis Queue<br/>Priority + FIFO)]
    end

    subgraph "Worker Pool"
        W1[Worker 1<br/>Executor]
        W2[Worker 2<br/>Executor]
        W3[Worker 3<br/>Executor]
        W4[Worker 4<br/>Executor]
    end

    subgraph "Shared State"
        DB[(PostgreSQL<br/>Job State)]
        Cache[(Redis<br/>Hot Data)]
    end

    subgraph "Shared Storage"
        NFS[NFS/Network Storage<br/>Media Files]
    end

    Client([Clients]) --> API1
    Client --> API2
    Client --> API3

    API1 --> Queue
    API2 --> Queue
    API3 --> Queue

    Queue --> W1
    Queue --> W2
    Queue --> W3
    Queue --> W4

    W1 --> DB
    W2 --> DB
    W3 --> DB
    W4 --> DB

    W1 --> Cache
    W2 --> Cache
    W3 --> Cache
    W4 --> Cache

    W1 <--> NFS
    W2 <--> NFS
    W3 <--> NFS
    W4 <--> NFS

    style API1 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style API2 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style API3 fill:#4CAF50,stroke:#333,stroke-width:2px,color:#fff
    style W1 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W2 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W3 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
    style W4 fill:#2196F3,stroke:#333,stroke-width:2px,color:#fff
```

### Horizontal Scaling Strategy

```mermaid
graph TB
    subgraph "Traffic Growth"
        T1[100 req/s] --> T2[500 req/s] --> T3[1000 req/s]
    end

    subgraph "Scaling Strategy"
        S1[1 API + 2 Workers]
        S2[3 API + 5 Workers]
        S3[5 API + 10 Workers]
    end

    subgraph "Resource Allocation"
        R1[Light: 2 CPU, 4GB RAM]
        R2[Medium: 8 CPU, 16GB RAM]
        R3[Heavy: 16 CPU, 32GB RAM]
    end

    T1 --> S1
    T2 --> S2
    T3 --> S3

    S1 --> R1
    S2 --> R2
    S3 --> R3

    style T1 fill:#FFC107,stroke:#333,stroke-width:2px
    style T2 fill:#FF9800,stroke:#333,stroke-width:2px,color:#fff
    style T3 fill:#FF5722,stroke:#333,stroke-width:2px,color:#fff
```

## Technology Stack

```mermaid
mindmap
  root((Media Pipeline))
    Backend
      Go 1.21
        net/http
        context
        encoding/json
      FFmpeg 8.0
        libx264
        libx265
        aac
    Data
      PostgreSQL 15
        ACID
        Replication
      Redis 7
        Cache
        Queue
        Pub/Sub
      In-Memory Store
        Thread-safe
        MVP
    Infrastructure
      Docker
        Multi-stage build
        Alpine Linux
      Docker Compose
        Service orchestration
        Health checks
      Kubernetes
        Future scaling
        HA deployment
    Monitoring
      Prometheus
        Metrics
        Alerting
      Grafana
        Dashboards
        Visualization
      Loki
        Log aggregation
        Query
```

---

**Document Version**: 1.0
**Last Updated**: 2024-12-22
**Status**: Production Ready
