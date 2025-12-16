# Distributed State Management Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md), [Schemas Design](./schemas-detailed-design.md), [API Design](./api-interface-design.md)

---

## Overview

This document defines the distributed state management system for media-pipeline, covering:
- **State Storage**: Database schema and consistency guarantees
- **Job State Machine**: State transitions and invariants
- **Distributed Locks**: Preventing race conditions and duplicate processing
- **Job Queue**: Task distribution and priority scheduling
- **Worker Coordination**: Worker registration, health checks, graceful shutdown
- **Failure Recovery**: Crash detection, job retry, orphan cleanup
- **Observability**: Metrics, tracing, and debugging

---

## System Architecture

### Components

```
┌─────────────┐
│  API Server │ ──┬──> PostgreSQL (Job State, Metadata)
└─────────────┘   │
                  ├──> Redis (Locks, Queue, Cache)
                  │
┌─────────────┐   │
│   Worker 1  │ ──┤
└─────────────┘   │
                  │
┌─────────────┐   │
│   Worker 2  │ ──┤
└─────────────┘   │
                  │
┌─────────────┐   │
│   Worker N  │ ──┘
└─────────────┘
```

### Technology Stack

- **PostgreSQL**: Persistent state storage, ACID guarantees
- **Redis**: Distributed locks, job queue, caching
- **S3/GCS**: Media files, intermediate artifacts
- **Prometheus**: Metrics collection
- **OpenTelemetry**: Distributed tracing

---

## State Storage

### Database Schema (PostgreSQL)

#### Jobs Table

```sql
CREATE TABLE jobs (
    job_id          VARCHAR(64) PRIMARY KEY,
    user_id         VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL,
    priority        INTEGER DEFAULT 5,

    -- Spec and Plan
    spec            JSONB NOT NULL,
    plan            JSONB,

    -- Timestamps
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,

    -- Progress
    progress        JSONB,

    -- Results
    output_files    JSONB,
    error           JSONB,

    -- Worker assignment
    worker_id       VARCHAR(64),
    worker_claimed_at TIMESTAMP,

    -- Retry tracking
    retry_count     INTEGER DEFAULT 0,
    max_retries     INTEGER DEFAULT 3,

    -- Indexes
    INDEX idx_status (status),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at DESC),
    INDEX idx_worker_id (worker_id),
    INDEX idx_priority_created (priority DESC, created_at ASC)
);

-- Trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

#### Job Logs Table

```sql
CREATE TABLE job_logs (
    id              BIGSERIAL PRIMARY KEY,
    job_id          VARCHAR(64) NOT NULL REFERENCES jobs(job_id) ON DELETE CASCADE,
    timestamp       TIMESTAMP NOT NULL DEFAULT NOW(),
    level           VARCHAR(16) NOT NULL,  -- debug, info, warn, error
    stage           VARCHAR(32),           -- validation, planning, processing
    message         TEXT NOT NULL,
    metadata        JSONB,

    INDEX idx_job_id (job_id),
    INDEX idx_timestamp (timestamp DESC)
);
```

#### Workers Table

```sql
CREATE TABLE workers (
    worker_id       VARCHAR(64) PRIMARY KEY,
    hostname        VARCHAR(256) NOT NULL,
    version         VARCHAR(32),

    -- Status
    status          VARCHAR(32) NOT NULL,  -- active, draining, offline
    started_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_heartbeat  TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Capacity
    max_concurrent  INTEGER DEFAULT 1,
    current_load    INTEGER DEFAULT 0,

    -- Metrics
    total_jobs      INTEGER DEFAULT 0,
    failed_jobs     INTEGER DEFAULT 0,

    INDEX idx_status (status),
    INDEX idx_last_heartbeat (last_heartbeat DESC)
);
```

#### Execution Steps Table

```sql
CREATE TABLE execution_steps (
    id              BIGSERIAL PRIMARY KEY,
    job_id          VARCHAR(64) NOT NULL REFERENCES jobs(job_id) ON DELETE CASCADE,
    step_id         VARCHAR(64) NOT NULL,
    type            VARCHAR(32) NOT NULL,  -- download, ffmpeg, upload

    -- Timing
    started_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMP,
    duration_ms     INTEGER,

    -- Details
    command         TEXT,
    exit_code       INTEGER,
    stdout          TEXT,
    stderr          TEXT,

    -- Result
    success         BOOLEAN,
    error           TEXT,

    INDEX idx_job_id (job_id),
    INDEX idx_started_at (started_at DESC)
);
```

### Data Consistency Guarantees

**ACID Properties**:
- **Atomicity**: State transitions are atomic (use transactions)
- **Consistency**: State machine invariants enforced by DB constraints
- **Isolation**: Serializable isolation for critical operations
- **Durability**: Write-ahead logging ensures persistence

**Example: Atomic State Transition**
```go
func (s *Store) ClaimJob(ctx context.Context, jobID, workerID string) error {
    tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Claim job atomically
    result, err := tx.ExecContext(ctx, `
        UPDATE jobs
        SET status = 'processing',
            worker_id = $1,
            worker_claimed_at = NOW(),
            started_at = COALESCE(started_at, NOW())
        WHERE job_id = $2
          AND status = 'pending'
          AND (worker_id IS NULL OR worker_claimed_at < NOW() - INTERVAL '5 minutes')
    `, workerID, jobID)

    if err != nil {
        return err
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return ErrJobNotAvailable
    }

    return tx.Commit()
}
```

---

## Job State Machine

### States

```
pending ──────────> processing ──────────> completed
   │                    │
   │                    │
   │                    v
   │                 failed ──────> pending (if retryable)
   │                                   │
   │                                   │
   └──────────────> cancelled <────────┘
```

### State Definitions

| State | Description | Terminal? |
|-------|-------------|-----------|
| `pending` | Job queued, waiting for worker | No |
| `validating` | Validating JobSpec | No |
| `planning` | Compiling execution plan | No |
| `downloading_inputs` | Downloading input files | No |
| `processing` | Executing FFmpeg commands | No |
| `uploading_outputs` | Uploading output files | No |
| `completed` | Successfully completed | Yes |
| `failed` | Failed (not retryable or max retries exceeded) | Yes |
| `cancelled` | Cancelled by user | Yes |

### State Transition Rules

```go
type StateTransition struct {
    From    JobState
    To      JobState
    Allowed bool
}

var allowedTransitions = map[JobState][]JobState{
    StatePending: {
        StateValidating,
        StateCancelled,
    },
    StateValidating: {
        StatePlanning,
        StateFailed,
        StateCancelled,
    },
    StatePlanning: {
        StateDownloadingInputs,
        StateFailed,
        StateCancelled,
    },
    StateDownloadingInputs: {
        StateProcessing,
        StateFailed,
        StateCancelled,
    },
    StateProcessing: {
        StateUploadingOutputs,
        StateFailed,
        StateCancelled,
    },
    StateUploadingOutputs: {
        StateCompleted,
        StateFailed,
        StateCancelled,
    },
    StateFailed: {
        StatePending,  // Only if retryable and retries < max
    },
}

func (s *StateMachine) CanTransition(from, to JobState) bool {
    allowed, ok := allowedTransitions[from]
    if !ok {
        return false
    }
    for _, state := range allowed {
        if state == to {
            return true
        }
    }
    return false
}

func (s *StateMachine) Transition(ctx context.Context, jobID string, to JobState) error {
    return s.store.Transaction(ctx, func(tx *sql.Tx) error {
        // Get current state
        var currentState JobState
        err := tx.QueryRowContext(ctx,
            "SELECT status FROM jobs WHERE job_id = $1 FOR UPDATE",
            jobID,
        ).Scan(&currentState)
        if err != nil {
            return err
        }

        // Check if transition is allowed
        if !s.CanTransition(currentState, to) {
            return fmt.Errorf("invalid state transition: %s -> %s", currentState, to)
        }

        // Update state
        _, err = tx.ExecContext(ctx,
            "UPDATE jobs SET status = $1 WHERE job_id = $2",
            to, jobID,
        )
        return err
    })
}
```

### Invariants

1. **Once terminal, cannot change**: Terminal states (completed, failed, cancelled) cannot transition to other states
2. **Worker assignment**: Only processing jobs have worker_id set
3. **Timestamps monotonic**: started_at ≤ completed_at
4. **Retry limit**: retry_count ≤ max_retries

---

## Distributed Locks

Use Redis for distributed locking to prevent race conditions.

### Lock Implementation

```go
type DistributedLock struct {
    redis  *redis.Client
    key    string
    token  string
    ttl    time.Duration
}

func (l *DistributedLock) Acquire(ctx context.Context) error {
    // Generate unique token
    l.token = uuid.New().String()

    // SET NX EX (set if not exists with expiration)
    success, err := l.redis.SetNX(ctx, l.key, l.token, l.ttl).Result()
    if err != nil {
        return err
    }

    if !success {
        return ErrLockNotAcquired
    }

    // Start background goroutine to extend lock
    go l.keepAlive(ctx)

    return nil
}

func (l *DistributedLock) Release(ctx context.Context) error {
    // Lua script to ensure we only delete our own lock
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `

    result, err := l.redis.Eval(ctx, script, []string{l.key}, l.token).Result()
    if err != nil {
        return err
    }

    if result.(int64) == 0 {
        return ErrLockNotHeld
    }

    return nil
}

func (l *DistributedLock) keepAlive(ctx context.Context) {
    ticker := time.NewTicker(l.ttl / 2)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Extend lock expiration
            script := `
                if redis.call("get", KEYS[1]) == ARGV[1] then
                    return redis.call("expire", KEYS[1], ARGV[2])
                else
                    return 0
                end
            `
            l.redis.Eval(ctx, script, []string{l.key}, l.token, int(l.ttl.Seconds()))
        }
    }
}
```

### Lock Usage

**Job Processing Lock**:
```go
func (w *Worker) ProcessJob(ctx context.Context, jobID string) error {
    // Acquire lock
    lock := NewDistributedLock(w.redis, fmt.Sprintf("lock:job:%s", jobID), 5*time.Minute)
    if err := lock.Acquire(ctx); err != nil {
        return err
    }
    defer lock.Release(ctx)

    // Process job
    return w.executeJob(ctx, jobID)
}
```

**Worker Registration Lock**:
```go
func (w *Worker) Register(ctx context.Context) error {
    lock := NewDistributedLock(w.redis, fmt.Sprintf("lock:worker:%s", w.ID), 30*time.Second)
    if err := lock.Acquire(ctx); err != nil {
        return err
    }
    defer lock.Release(ctx)

    // Register in database
    return w.store.UpsertWorker(ctx, &Worker{
        ID:       w.ID,
        Hostname: w.hostname,
        Status:   StatusActive,
    })
}
```

---

## Job Queue

Use Redis sorted sets for priority queue.

### Queue Implementation

```go
type JobQueue struct {
    redis *redis.Client
    key   string
}

func (q *JobQueue) Enqueue(ctx context.Context, jobID string, priority int, createdAt time.Time) error {
    // Score = priority (high priority = low score) + timestamp for FIFO within priority
    // Format: priority * 1e10 + timestamp_seconds
    // Example: Priority 10 at time 1735689600 -> 10*1e10 + 1735689600 = 100000001735689600
    score := float64(priority)*1e10 + float64(createdAt.Unix())

    return q.redis.ZAdd(ctx, q.key, &redis.Z{
        Score:  score,
        Member: jobID,
    }).Err()
}

func (q *JobQueue) Dequeue(ctx context.Context) (string, error) {
    // Get job with lowest score (highest priority, oldest timestamp)
    result, err := q.redis.ZPopMin(ctx, q.key, 1).Result()
    if err != nil {
        return "", err
    }

    if len(result) == 0 {
        return "", ErrQueueEmpty
    }

    return result[0].Member.(string), nil
}

func (q *JobQueue) Remove(ctx context.Context, jobID string) error {
    return q.redis.ZRem(ctx, q.key, jobID).Err()
}

func (q *JobQueue) Size(ctx context.Context) (int64, error) {
    return q.redis.ZCard(ctx, q.key).Result()
}

func (q *JobQueue) Peek(ctx context.Context, limit int64) ([]string, error) {
    return q.redis.ZRange(ctx, q.key, 0, limit-1).Result()
}
```

### Priority Scheduling

Priority values (0-10):
- **10**: Critical/urgent jobs
- **7-9**: High priority
- **5-6**: Normal priority (default: 5)
- **1-4**: Low priority
- **0**: Background/batch jobs

Within same priority, jobs are processed in FIFO order (oldest first).

---

## Worker Coordination

### Worker Registration

```go
type Worker struct {
    ID              string
    Hostname        string
    Version         string
    MaxConcurrent   int
    store           *Store
    redis           *redis.Client
}

func (w *Worker) Start(ctx context.Context) error {
    // Register worker
    if err := w.register(ctx); err != nil {
        return err
    }

    // Start heartbeat
    go w.heartbeatLoop(ctx)

    // Start job processing loop
    go w.processLoop(ctx)

    return nil
}

func (w *Worker) register(ctx context.Context) error {
    return w.store.UpsertWorker(ctx, &schemas.Worker{
        WorkerID:      w.ID,
        Hostname:      w.Hostname,
        Version:       w.Version,
        Status:        "active",
        MaxConcurrent: w.MaxConcurrent,
        StartedAt:     time.Now(),
        LastHeartbeat: time.Now(),
    })
}

func (w *Worker) heartbeatLoop(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.heartbeat(ctx)
        }
    }
}

func (w *Worker) heartbeat(ctx context.Context) error {
    return w.store.UpdateWorkerHeartbeat(ctx, w.ID)
}
```

### Worker Status

| Status | Description |
|--------|-------------|
| `active` | Processing jobs |
| `draining` | No new jobs, finishing current jobs |
| `offline` | Not responding to heartbeats |

### Graceful Shutdown

```go
func (w *Worker) Shutdown(ctx context.Context) error {
    // 1. Mark as draining
    if err := w.store.UpdateWorkerStatus(ctx, w.ID, "draining"); err != nil {
        return err
    }

    // 2. Stop accepting new jobs
    close(w.stopCh)

    // 3. Wait for current jobs to complete (with timeout)
    done := make(chan struct{})
    go func() {
        w.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // All jobs completed
    case <-time.After(5 * time.Minute):
        // Timeout: cancel remaining jobs
        w.cancelAllJobs(ctx)
    }

    // 4. Mark as offline
    return w.store.UpdateWorkerStatus(ctx, w.ID, "offline")
}
```

### Job Processing Loop

```go
func (w *Worker) processLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-w.stopCh:
            return
        default:
            // Check if we can accept more jobs
            if w.getCurrentLoad() >= w.MaxConcurrent {
                time.Sleep(1 * time.Second)
                continue
            }

            // Dequeue job
            jobID, err := w.queue.Dequeue(ctx)
            if err == ErrQueueEmpty {
                time.Sleep(1 * time.Second)
                continue
            }
            if err != nil {
                log.Printf("Failed to dequeue: %v", err)
                continue
            }

            // Process job in goroutine
            w.wg.Add(1)
            go func(jid string) {
                defer w.wg.Done()
                if err := w.processJob(ctx, jid); err != nil {
                    log.Printf("Job %s failed: %v", jid, err)
                }
            }(jobID)
        }
    }
}

func (w *Worker) processJob(ctx context.Context, jobID string) error {
    // 1. Claim job
    if err := w.store.ClaimJob(ctx, jobID, w.ID); err != nil {
        return err
    }

    // 2. Execute job
    executor := NewJobExecutor(w.store, jobID)
    return executor.Execute(ctx)
}
```

---

## Failure Recovery

### Crash Detection

**Stale Worker Detection**:
```go
func (s *Store) DetectStaleWorkers(ctx context.Context) ([]string, error) {
    var workerIDs []string
    err := s.db.Select(&workerIDs, `
        SELECT worker_id
        FROM workers
        WHERE status = 'active'
          AND last_heartbeat < NOW() - INTERVAL '30 seconds'
    `)
    return workerIDs, err
}

func (s *Store) MarkWorkerOffline(ctx context.Context, workerID string) error {
    _, err := s.db.ExecContext(ctx, `
        UPDATE workers
        SET status = 'offline'
        WHERE worker_id = $1
    `, workerID)
    return err
}
```

**Orphaned Job Recovery**:
```go
func (s *Store) RecoverOrphanedJobs(ctx context.Context) error {
    // Find jobs claimed by offline workers
    _, err := s.db.ExecContext(ctx, `
        UPDATE jobs
        SET status = 'pending',
            worker_id = NULL,
            worker_claimed_at = NULL,
            retry_count = retry_count + 1
        WHERE status = 'processing'
          AND worker_id IN (
              SELECT worker_id
              FROM workers
              WHERE status = 'offline'
          )
          AND retry_count < max_retries
    `)
    return err
}
```

### Watchdog Process

```go
type Watchdog struct {
    store *Store
}

func (w *Watchdog) Run(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.check(ctx)
        }
    }
}

func (w *Watchdog) check(ctx context.Context) {
    // 1. Detect stale workers
    staleWorkers, err := w.store.DetectStaleWorkers(ctx)
    if err != nil {
        log.Printf("Failed to detect stale workers: %v", err)
        return
    }

    // 2. Mark them offline
    for _, workerID := range staleWorkers {
        if err := w.store.MarkWorkerOffline(ctx, workerID); err != nil {
            log.Printf("Failed to mark worker %s offline: %v", workerID, err)
        }
    }

    // 3. Recover orphaned jobs
    if err := w.store.RecoverOrphanedJobs(ctx); err != nil {
        log.Printf("Failed to recover orphaned jobs: %v", err)
    }

    // 4. Fail jobs exceeding max retries
    if err := w.store.FailJobsExceedingRetries(ctx); err != nil {
        log.Printf("Failed to fail jobs: %v", err)
    }
}
```

### Retry Policy

```go
type RetryPolicy struct {
    MaxRetries int
    Backoff    BackoffStrategy
}

type BackoffStrategy interface {
    NextDelay(retryCount int) time.Duration
}

type ExponentialBackoff struct {
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func (b *ExponentialBackoff) NextDelay(retryCount int) time.Duration {
    delay := b.InitialDelay * time.Duration(math.Pow(b.Multiplier, float64(retryCount)))
    if delay > b.MaxDelay {
        delay = b.MaxDelay
    }
    return delay
}

// Default retry policy
var DefaultRetryPolicy = &RetryPolicy{
    MaxRetries: 3,
    Backoff: &ExponentialBackoff{
        InitialDelay: 10 * time.Second,
        MaxDelay:     5 * time.Minutes,
        Multiplier:   2.0,
    },
}

func (s *Store) RetryJob(ctx context.Context, jobID string) error {
    return s.Transaction(ctx, func(tx *sql.Tx) error {
        var job Job
        err := tx.QueryRowContext(ctx,
            "SELECT retry_count, max_retries FROM jobs WHERE job_id = $1 FOR UPDATE",
            jobID,
        ).Scan(&job.RetryCount, &job.MaxRetries)
        if err != nil {
            return err
        }

        if job.RetryCount >= job.MaxRetries {
            // Mark as failed
            _, err = tx.ExecContext(ctx,
                "UPDATE jobs SET status = 'failed' WHERE job_id = $1",
                jobID,
            )
            return err
        }

        // Calculate backoff delay
        delay := DefaultRetryPolicy.Backoff.NextDelay(job.RetryCount)

        // Re-queue job
        _, err = tx.ExecContext(ctx, `
            UPDATE jobs
            SET status = 'pending',
                worker_id = NULL,
                worker_claimed_at = NULL,
                retry_count = retry_count + 1,
                updated_at = NOW() + $1
            WHERE job_id = $2
        `, delay, jobID)

        return err
    })
}
```

---

## Concurrency Control

### Optimistic Locking

Use row versioning to detect concurrent modifications:

```sql
ALTER TABLE jobs ADD COLUMN version INTEGER DEFAULT 1;

-- Update with version check
UPDATE jobs
SET status = $1,
    version = version + 1
WHERE job_id = $2
  AND version = $3
RETURNING version;
```

```go
func (s *Store) UpdateJobWithVersion(ctx context.Context, jobID string, expectedVersion int, update func(*Job)) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    var job Job
    err = tx.QueryRowContext(ctx,
        "SELECT * FROM jobs WHERE job_id = $1 FOR UPDATE",
        jobID,
    ).Scan(&job)
    if err != nil {
        return err
    }

    if job.Version != expectedVersion {
        return ErrVersionMismatch
    }

    update(&job)

    result, err := tx.ExecContext(ctx, `
        UPDATE jobs
        SET status = $1,
            progress = $2,
            version = version + 1
        WHERE job_id = $3
          AND version = $4
    `, job.Status, job.Progress, job.JobID, expectedVersion)

    if err != nil {
        return err
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return ErrVersionMismatch
    }

    return tx.Commit()
}
```

### Advisory Locks

Use PostgreSQL advisory locks for critical sections:

```go
func (s *Store) WithAdvisoryLock(ctx context.Context, key int64, fn func() error) error {
    // Acquire lock
    _, err := s.db.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key)
    if err != nil {
        return err
    }

    // Ensure unlock on exit
    defer s.db.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", key)

    return fn()
}

// Example: Prevent duplicate job creation with same idempotency key
func (s *Store) CreateJobIdempotent(ctx context.Context, idempotencyKey string, spec *JobSpec) (*Job, error) {
    lockKey := int64(crc32.ChecksumIEEE([]byte(idempotencyKey)))

    var job *Job
    err := s.WithAdvisoryLock(ctx, lockKey, func() error {
        // Check if job already exists
        existing, err := s.GetJobByIdempotencyKey(ctx, idempotencyKey)
        if err == nil {
            job = existing
            return nil
        }

        // Create new job
        job, err = s.CreateJob(ctx, spec)
        if err != nil {
            return err
        }

        // Store idempotency mapping
        return s.StoreIdempotencyKey(ctx, idempotencyKey, job.JobID)
    })

    return job, err
}
```

---

## Observability

### Metrics

**Job Metrics**:
```go
var (
    jobsCreated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "jobs_created_total",
            Help: "Total number of jobs created",
        },
        []string{"user_id"},
    )

    jobsCompleted = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "jobs_completed_total",
            Help: "Total number of jobs completed",
        },
        []string{"status"}, // completed, failed, cancelled
    )

    jobDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "job_duration_seconds",
            Help:    "Job execution duration",
            Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to 512s
        },
        []string{"status"},
    )

    jobsInProgress = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "jobs_in_progress",
            Help: "Number of jobs currently processing",
        },
    )

    queueSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "queue_size",
            Help: "Number of jobs in queue",
        },
    )
)
```

**Worker Metrics**:
```go
var (
    workersActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "workers_active",
            Help: "Number of active workers",
        },
    )

    workerLoad = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "worker_load",
            Help: "Current load per worker",
        },
        []string{"worker_id"},
    )
)
```

### Distributed Tracing

Use OpenTelemetry for distributed tracing:

```go
func (w *Worker) processJob(ctx context.Context, jobID string) error {
    tracer := otel.Tracer("media-pipeline")
    ctx, span := tracer.Start(ctx, "process_job",
        trace.WithAttributes(
            attribute.String("job_id", jobID),
            attribute.String("worker_id", w.ID),
        ),
    )
    defer span.End()

    // Validation phase
    if err := w.validate(ctx, jobID); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "validation failed")
        return err
    }

    // Planning phase
    if err := w.plan(ctx, jobID); err != nil {
        span.RecordError(err)
        return err
    }

    // Processing phase
    if err := w.execute(ctx, jobID); err != nil {
        span.RecordError(err)
        return err
    }

    span.SetStatus(codes.Ok, "job completed")
    return nil
}
```

### Logging

Structured logging with correlation IDs:

```go
type Logger struct {
    logger *zap.Logger
}

func (l *Logger) WithJobID(jobID string) *Logger {
    return &Logger{
        logger: l.logger.With(zap.String("job_id", jobID)),
    }
}

func (l *Logger) WithWorkerID(workerID string) *Logger {
    return &Logger{
        logger: l.logger.With(zap.String("worker_id", workerID)),
    }
}

// Usage
log := logger.WithJobID(jobID).WithWorkerID(workerID)
log.Info("Starting job processing")
log.Error("Job failed", zap.Error(err))
```

---

## Testing Strategy

### Unit Tests

```go
func TestStateTransitions(t *testing.T) {
    sm := NewStateMachine()

    tests := []struct {
        from    JobState
        to      JobState
        allowed bool
    }{
        {StatePending, StateProcessing, true},
        {StateProcessing, StateCompleted, true},
        {StateCompleted, StatePending, false}, // Terminal state
        {StateFailed, StatePending, true},     // Retry
    }

    for _, tt := range tests {
        allowed := sm.CanTransition(tt.from, tt.to)
        if allowed != tt.allowed {
            t.Errorf("Transition %s -> %s: expected %v, got %v",
                tt.from, tt.to, tt.allowed, allowed)
        }
    }
}
```

### Integration Tests

```go
func TestJobProcessing(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    redis := setupTestRedis(t)
    worker := NewWorker(db, redis)

    // Create job
    job := createTestJob(t, db)

    // Process job
    err := worker.ProcessJob(context.Background(), job.JobID)
    assert.NoError(t, err)

    // Verify state
    status, err := db.GetJobStatus(context.Background(), job.JobID)
    assert.NoError(t, err)
    assert.Equal(t, StateCompleted, status.Status)
}
```

### Chaos Testing

Simulate failures:

```go
func TestWorkerCrashRecovery(t *testing.T) {
    // Start worker
    worker := NewWorker(db, redis)
    go worker.Start(context.Background())

    // Create job
    job := createTestJob(t, db)

    // Wait for job to start
    time.Sleep(1 * time.Second)

    // Simulate worker crash (stop heartbeat)
    worker.Shutdown(context.Background())

    // Start watchdog
    watchdog := NewWatchdog(db)
    watchdog.check(context.Background())

    // Verify job is recovered
    status, _ := db.GetJobStatus(context.Background(), job.JobID)
    assert.Equal(t, StatePending, status.Status)
}
```

---

## Performance Considerations

### Database Optimization

**Indexes**:
```sql
-- Job queries by status
CREATE INDEX CONCURRENTLY idx_jobs_status_created
    ON jobs(status, created_at DESC);

-- Job queries by user
CREATE INDEX CONCURRENTLY idx_jobs_user_created
    ON jobs(user_id, created_at DESC);

-- Worker health checks
CREATE INDEX CONCURRENTLY idx_workers_last_heartbeat
    ON workers(last_heartbeat DESC)
    WHERE status = 'active';
```

**Partitioning** (for high volume):
```sql
-- Partition jobs table by created_at (monthly)
CREATE TABLE jobs_2025_12 PARTITION OF jobs
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
```

**Archiving**:
```sql
-- Move old completed jobs to archive table
INSERT INTO jobs_archive
SELECT * FROM jobs
WHERE status IN ('completed', 'failed', 'cancelled')
  AND completed_at < NOW() - INTERVAL '30 days';

DELETE FROM jobs
WHERE status IN ('completed', 'failed', 'cancelled')
  AND completed_at < NOW() - INTERVAL '30 days';
```

### Redis Optimization

**Connection Pooling**:
```go
redis := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    PoolSize:     100,
    MinIdleConns: 10,
})
```

**Pipeline Batching**:
```go
pipe := redis.Pipeline()
for _, jobID := range jobIDs {
    pipe.ZAdd(ctx, queueKey, &redis.Z{Score: score, Member: jobID})
}
_, err := pipe.Exec(ctx)
```

---

## Summary

The distributed state management system provides:

1. **Reliable State Storage**: PostgreSQL with ACID guarantees
2. **State Machine**: Clear state transitions and invariants
3. **Distributed Locks**: Redis-based locking for coordination
4. **Priority Queue**: Redis sorted sets for efficient job scheduling
5. **Worker Coordination**: Registration, heartbeat, graceful shutdown
6. **Failure Recovery**: Crash detection, orphan recovery, retry logic
7. **Concurrency Control**: Optimistic locking and advisory locks
8. **Observability**: Metrics, tracing, structured logging

Key features:
- Atomic state transitions with database transactions
- Distributed locking to prevent race conditions
- Priority-based job scheduling with FIFO within priority
- Automatic failure detection and recovery
- Worker health monitoring with heartbeats
- Retry logic with exponential backoff
- Comprehensive metrics and tracing

---

**Status**: Ready for implementation
