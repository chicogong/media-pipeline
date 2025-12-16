# Error Handling and Recovery Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md), [API Design](./api-interface-design.md), [State Management](./distributed-state-management-design.md)

---

## Overview

This document defines the comprehensive error handling and recovery system for media-pipeline, covering:
- **Error Classification**: Categorizing errors by type and severity
- **Error Codes**: Structured error code system
- **Error Context**: Rich error information for debugging
- **User-Facing Messages**: Clear, actionable error messages
- **FFmpeg Error Parsing**: Interpreting FFmpeg stderr output
- **Retry Strategies**: When and how to retry failed operations
- **Error Monitoring**: Alerting and observability
- **Debugging Tools**: Tools for troubleshooting failures

---

## Error Classification

### Error Categories

```go
type ErrorCategory string

const (
    // Client Errors (4xx) - User can fix
    CategoryValidation    ErrorCategory = "validation"     // Invalid input
    CategoryPermission    ErrorCategory = "permission"     // Access denied
    CategoryRateLimit     ErrorCategory = "rate_limit"     // Quota exceeded
    CategoryNotFound      ErrorCategory = "not_found"      // Resource doesn't exist

    // Server Errors (5xx) - System issue
    CategoryProcessing    ErrorCategory = "processing"     // FFmpeg/processing failure
    CategoryInfrastructure ErrorCategory = "infrastructure" // Database, network, disk
    CategoryTimeout       ErrorCategory = "timeout"        // Operation timed out
    CategoryInternal      ErrorCategory = "internal"       // Unexpected error
)
```

### Error Severity

```go
type ErrorSeverity string

const (
    SeverityInfo     ErrorSeverity = "info"     // Informational, no action needed
    SeverityWarning  ErrorSeverity = "warning"  // Degraded, but can continue
    SeverityError    ErrorSeverity = "error"    // Failed, cannot continue
    SeverityCritical ErrorSeverity = "critical" // System-wide issue
)
```

### Retryability

```go
type RetryableStatus string

const (
    Retryable         RetryableStatus = "retryable"          // Can retry
    NotRetryable      RetryableStatus = "not_retryable"      // Don't retry
    RetryableExternal RetryableStatus = "retryable_external" // User can resubmit
)
```

---

## Error Structure

### Core Error Type

```go
type ProcessingError struct {
    // Error identification
    Code     ErrorCode     `json:"code"`
    Category ErrorCategory `json:"category"`
    Severity ErrorSeverity `json:"severity"`

    // Human-readable message
    Message       string            `json:"message"`
    UserMessage   string            `json:"user_message,omitempty"` // User-friendly
    Documentation string            `json:"documentation,omitempty"` // Link to docs

    // Context
    Details   map[string]interface{} `json:"details,omitempty"`
    RequestID string                 `json:"request_id,omitempty"`
    JobID     string                 `json:"job_id,omitempty"`
    Timestamp time.Time              `json:"timestamp"`

    // FFmpeg-specific (if applicable)
    FFmpegError *FFmpegError `json:"ffmpeg_error,omitempty"`

    // Retry info
    Retryable  RetryableStatus `json:"retryable"`
    RetryAfter *time.Duration  `json:"retry_after,omitempty"`

    // Stack trace (internal errors only)
    StackTrace string `json:"stack_trace,omitempty"`

    // Cause chain
    Cause *ProcessingError `json:"cause,omitempty"`
}

type FFmpegError struct {
    Command    string `json:"command"`
    ExitCode   int    `json:"exit_code"`
    Stderr     string `json:"stderr"`
    StderrTail string `json:"stderr_tail"` // Last 20 lines
    ParsedError *ParsedFFmpegError `json:"parsed_error,omitempty"`
}

type ParsedFFmpegError struct {
    Type         string `json:"type"`          // "codec", "format", "io", "filter"
    Component    string `json:"component"`     // Which FFmpeg component failed
    Reason       string `json:"reason"`        // Human-readable reason
    Suggestion   string `json:"suggestion"`    // How to fix
}
```

---

## Error Codes

### Error Code System

Format: `CATEGORY_SPECIFIC_REASON`

```go
type ErrorCode string

// Validation Errors (4xx)
const (
    ErrInvalidJobSpec        ErrorCode = "INVALID_JOB_SPEC"
    ErrInvalidOperation      ErrorCode = "INVALID_OPERATION"
    ErrInvalidOperatorParams ErrorCode = "INVALID_OPERATOR_PARAMS"
    ErrInvalidInput          ErrorCode = "INVALID_INPUT"
    ErrInvalidOutput         ErrorCode = "INVALID_OUTPUT"
    ErrCyclicDependency      ErrorCode = "CYCLIC_DEPENDENCY"
    ErrUnreachableNode       ErrorCode = "UNREACHABLE_NODE"
)

// Input Errors (4xx)
const (
    ErrInputNotFound         ErrorCode = "INPUT_NOT_FOUND"
    ErrInputAccessDenied     ErrorCode = "INPUT_ACCESS_DENIED"
    ErrInputDownloadFailed   ErrorCode = "INPUT_DOWNLOAD_FAILED"
    ErrInputCorrupted        ErrorCode = "INPUT_CORRUPTED"
    ErrInputTooLarge         ErrorCode = "INPUT_TOO_LARGE"
    ErrUnsupportedFormat     ErrorCode = "UNSUPPORTED_FORMAT"
    ErrUnsupportedCodec      ErrorCode = "UNSUPPORTED_CODEC"
    ErrSSRFBlocked           ErrorCode = "SSRF_BLOCKED"
)

// Resource Errors (4xx)
const (
    ErrResourceLimitExceeded ErrorCode = "RESOURCE_LIMIT_EXCEEDED"
    ErrDurationTooLong       ErrorCode = "DURATION_TOO_LONG"
    ErrResolutionTooHigh     ErrorCode = "RESOLUTION_TOO_HIGH"
    ErrOutputTooLarge        ErrorCode = "OUTPUT_TOO_LARGE"
    ErrQuotaExceeded         ErrorCode = "QUOTA_EXCEEDED"
    ErrRateLimitExceeded     ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// Processing Errors (5xx)
const (
    ErrFFmpegFailed          ErrorCode = "FFMPEG_FAILED"
    ErrFFmpegTimeout         ErrorCode = "FFMPEG_TIMEOUT"
    ErrFFmpegCrashed         ErrorCode = "FFMPEG_CRASHED"
    ErrEncodingFailed        ErrorCode = "ENCODING_FAILED"
    ErrDecodingFailed        ErrorCode = "DECODING_FAILED"
    ErrFilterFailed          ErrorCode = "FILTER_FAILED"
    ErrMuxingFailed          ErrorCode = "MUXING_FAILED"
)

// Output Errors (5xx)
const (
    ErrOutputUploadFailed    ErrorCode = "OUTPUT_UPLOAD_FAILED"
    ErrOutputAccessDenied    ErrorCode = "OUTPUT_ACCESS_DENIED"
    ErrOutputWriteFailed     ErrorCode = "OUTPUT_WRITE_FAILED"
)

// Infrastructure Errors (5xx)
const (
    ErrInsufficientDiskSpace ErrorCode = "INSUFFICIENT_DISK_SPACE"
    ErrInsufficientMemory    ErrorCode = "INSUFFICIENT_MEMORY"
    ErrDatabaseError         ErrorCode = "DATABASE_ERROR"
    ErrQueueError            ErrorCode = "QUEUE_ERROR"
    ErrStorageError          ErrorCode = "STORAGE_ERROR"
    ErrNetworkError          ErrorCode = "NETWORK_ERROR"
)

// System Errors (5xx)
const (
    ErrInternalError         ErrorCode = "INTERNAL_ERROR"
    ErrTimeout               ErrorCode = "TIMEOUT"
    ErrCancelled             ErrorCode = "CANCELLED"
    ErrWorkerCrashed         ErrorCode = "WORKER_CRASHED"
    ErrUnexpectedState       ErrorCode = "UNEXPECTED_STATE"
)
```

### Error Code Metadata

```go
type ErrorCodeMetadata struct {
    Code          ErrorCode
    Category      ErrorCategory
    HTTPStatus    int
    Retryable     RetryableStatus
    UserMessage   string
    Documentation string
}

var errorCodeMetadata = map[ErrorCode]*ErrorCodeMetadata{
    ErrInvalidJobSpec: {
        Code:        ErrInvalidJobSpec,
        Category:    CategoryValidation,
        HTTPStatus:  400,
        Retryable:   NotRetryable,
        UserMessage: "The job specification is invalid. Please check the documentation and try again.",
        Documentation: "https://docs.example.com/errors/invalid-job-spec",
    },
    ErrInputNotFound: {
        Code:        ErrInputNotFound,
        Category:    CategoryNotFound,
        HTTPStatus:  404,
        Retryable:   NotRetryable,
        UserMessage: "The input file could not be found. Please verify the URL and permissions.",
        Documentation: "https://docs.example.com/errors/input-not-found",
    },
    ErrFFmpegFailed: {
        Code:        ErrFFmpegFailed,
        Category:    CategoryProcessing,
        HTTPStatus:  500,
        Retryable:   Retryable,
        UserMessage: "Video processing failed. Our team has been notified and will investigate.",
        Documentation: "https://docs.example.com/errors/ffmpeg-failed",
    },
    ErrInsufficientDiskSpace: {
        Code:        ErrInsufficientDiskSpace,
        Category:    CategoryInfrastructure,
        HTTPStatus:  503,
        Retryable:   Retryable,
        UserMessage: "The server is temporarily out of disk space. Please try again in a few minutes.",
        Documentation: "https://docs.example.com/errors/insufficient-disk-space",
    },
}

func NewError(code ErrorCode, message string) *ProcessingError {
    metadata := errorCodeMetadata[code]
    if metadata == nil {
        metadata = &ErrorCodeMetadata{
            Code:       code,
            Category:   CategoryInternal,
            HTTPStatus: 500,
            Retryable:  NotRetryable,
        }
    }

    return &ProcessingError{
        Code:        code,
        Category:    metadata.Category,
        Severity:    SeverityError,
        Message:     message,
        UserMessage: metadata.UserMessage,
        Documentation: metadata.Documentation,
        Retryable:   metadata.Retryable,
        Timestamp:   time.Now(),
    }
}
```

---

## FFmpeg Error Parsing

### Common FFmpeg Errors

```go
type FFmpegErrorPattern struct {
    Pattern    *regexp.Regexp
    Type       string
    Reason     string
    Suggestion string
    Retryable  bool
}

var ffmpegErrorPatterns = []FFmpegErrorPattern{
    // Codec errors
    {
        Pattern:    regexp.MustCompile(`(?i)Unknown encoder '(.+?)'`),
        Type:       "codec",
        Reason:     "Encoder not available",
        Suggestion: "Use a different codec or check FFmpeg build configuration",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Unknown decoder '(.+?)'`),
        Type:       "codec",
        Reason:     "Decoder not available",
        Suggestion: "The input file uses an unsupported codec",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Could not find codec parameters`),
        Type:       "codec",
        Reason:     "Unable to detect codec parameters",
        Suggestion: "The input file may be corrupted or incomplete",
        Retryable:  false,
    },

    // Format errors
    {
        Pattern:    regexp.MustCompile(`(?i)Invalid data found when processing input`),
        Type:       "format",
        Reason:     "Input file is corrupted or invalid",
        Suggestion: "Verify the input file is a valid media file",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)moov atom not found`),
        Type:       "format",
        Reason:     "MP4 file is incomplete or corrupted",
        Suggestion: "The MP4 file may not have been fully uploaded",
        Retryable:  false,
    },

    // I/O errors
    {
        Pattern:    regexp.MustCompile(`(?i)No such file or directory`),
        Type:       "io",
        Reason:     "File not found",
        Suggestion: "Verify the file path is correct",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Permission denied`),
        Type:       "io",
        Reason:     "Permission denied",
        Suggestion: "Check file permissions",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Connection refused|Connection timed out`),
        Type:       "io",
        Reason:     "Network error",
        Suggestion: "Check network connectivity and URL",
        Retryable:  true,
    },

    // Filter errors
    {
        Pattern:    regexp.MustCompile(`(?i)No such filter: '(.+?)'`),
        Type:       "filter",
        Reason:     "Filter not available",
        Suggestion: "Use a different filter or check FFmpeg build",
        Retryable:  false,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Cannot find a matching stream`),
        Type:       "filter",
        Reason:     "Stream mismatch in filtergraph",
        Suggestion: "Check filtergraph configuration",
        Retryable:  false,
    },

    // Resource errors
    {
        Pattern:    regexp.MustCompile(`(?i)Cannot allocate memory`),
        Type:       "resource",
        Reason:     "Out of memory",
        Suggestion: "Reduce resolution or file size",
        Retryable:  true,
    },
    {
        Pattern:    regexp.MustCompile(`(?i)Disk quota exceeded|No space left on device`),
        Type:       "resource",
        Reason:     "Out of disk space",
        Suggestion: "Free up disk space and retry",
        Retryable:  true,
    },
}

func ParseFFmpegError(stderr string, exitCode int) *ParsedFFmpegError {
    for _, pattern := range ffmpegErrorPatterns {
        if matches := pattern.Pattern.FindStringSubmatch(stderr); matches != nil {
            return &ParsedFFmpegError{
                Type:       pattern.Type,
                Component:  matches[0],
                Reason:     pattern.Reason,
                Suggestion: pattern.Suggestion,
            }
        }
    }

    // Fallback: generic error based on exit code
    return &ParsedFFmpegError{
        Type:   "unknown",
        Reason: fmt.Sprintf("FFmpeg exited with code %d", exitCode),
        Suggestion: "Check FFmpeg stderr output for details",
    }
}
```

### FFmpeg Error Wrapper

```go
func WrapFFmpegError(cmd string, exitCode int, stderr string) *ProcessingError {
    parsed := ParseFFmpegError(stderr, exitCode)

    // Determine error code based on parsed type
    var code ErrorCode
    var retryable RetryableStatus
    switch parsed.Type {
    case "codec":
        code = ErrUnsupportedCodec
        retryable = NotRetryable
    case "format":
        code = ErrInputCorrupted
        retryable = NotRetryable
    case "io":
        code = ErrInputDownloadFailed
        retryable = Retryable
    case "filter":
        code = ErrFilterFailed
        retryable = NotRetryable
    case "resource":
        code = ErrInsufficientMemory
        retryable = Retryable
    default:
        code = ErrFFmpegFailed
        retryable = Retryable
    }

    return &ProcessingError{
        Code:      code,
        Category:  CategoryProcessing,
        Severity:  SeverityError,
        Message:   parsed.Reason,
        UserMessage: parsed.Suggestion,
        Retryable: retryable,
        FFmpegError: &FFmpegError{
            Command:     cmd,
            ExitCode:    exitCode,
            Stderr:      stderr,
            StderrTail:  getLastLines(stderr, 20),
            ParsedError: parsed,
        },
        Timestamp: time.Now(),
    }
}

func getLastLines(s string, n int) string {
    lines := strings.Split(s, "\n")
    if len(lines) <= n {
        return s
    }
    return strings.Join(lines[len(lines)-n:], "\n")
}
```

---

## Error Context and Enrichment

### Context Capture

```go
type ErrorContext struct {
    JobID     string
    WorkerID  string
    Stage     string // "validation", "planning", "processing"
    StepID    string
    RequestID string
    UserID    string

    // Execution context
    InputFiles  []string
    Command     string
    Environment map[string]string

    // System context
    Hostname    string
    WorkerLoad  int
    DiskUsage   int64
    MemoryUsage int64
}

func (e *ProcessingError) WithContext(ctx *ErrorContext) *ProcessingError {
    e.JobID = ctx.JobID
    e.RequestID = ctx.RequestID

    if e.Details == nil {
        e.Details = make(map[string]interface{})
    }

    e.Details["worker_id"] = ctx.WorkerID
    e.Details["stage"] = ctx.Stage
    e.Details["step_id"] = ctx.StepID
    e.Details["hostname"] = ctx.Hostname
    e.Details["worker_load"] = ctx.WorkerLoad

    return e
}
```

### Error Chain

```go
func (e *ProcessingError) Wrap(cause error) *ProcessingError {
    if causeErr, ok := cause.(*ProcessingError); ok {
        e.Cause = causeErr
    } else {
        e.Cause = &ProcessingError{
            Code:     ErrInternalError,
            Category: CategoryInternal,
            Severity: SeverityError,
            Message:  cause.Error(),
        }
    }
    return e
}

func (e *ProcessingError) Unwrap() error {
    if e.Cause != nil {
        return e.Cause
    }
    return nil
}

// Example usage
func processVideo(ctx context.Context, jobID string) error {
    if err := downloadInput(ctx, jobID); err != nil {
        return NewError(ErrInputDownloadFailed, "Failed to download input").
            WithContext(&ErrorContext{JobID: jobID, Stage: "downloading_inputs"}).
            Wrap(err)
    }
    return nil
}
```

---

## Error Recovery Strategies

### Retry Decision Tree

```go
type RetryStrategy interface {
    ShouldRetry(err *ProcessingError, attempt int) bool
    BackoffDuration(attempt int) time.Duration
}

type DefaultRetryStrategy struct {
    MaxAttempts      int
    InitialBackoff   time.Duration
    MaxBackoff       time.Duration
    BackoffMultiplier float64
}

func (s *DefaultRetryStrategy) ShouldRetry(err *ProcessingError, attempt int) bool {
    // Don't retry if max attempts reached
    if attempt >= s.MaxAttempts {
        return false
    }

    // Check if error is retryable
    if err.Retryable == NotRetryable {
        return false
    }

    // Retry certain categories
    switch err.Category {
    case CategoryValidation, CategoryPermission, CategoryNotFound:
        return false // Client errors, don't retry
    case CategoryRateLimit:
        return true // Always retry rate limits
    case CategoryTimeout, CategoryInfrastructure:
        return true // Retry transient errors
    case CategoryProcessing:
        // Retry only if explicitly marked retryable
        return err.Retryable == Retryable
    }

    return false
}

func (s *DefaultRetryStrategy) BackoffDuration(attempt int) time.Duration {
    backoff := s.InitialBackoff * time.Duration(math.Pow(s.BackoffMultiplier, float64(attempt)))
    if backoff > s.MaxBackoff {
        backoff = s.MaxBackoff
    }
    return backoff
}
```

### Retry Execution

```go
func RetryWithBackoff(ctx context.Context, strategy RetryStrategy, fn func() error) error {
    var lastErr *ProcessingError

    for attempt := 0; attempt < 5; attempt++ {
        // Execute function
        err := fn()
        if err == nil {
            return nil
        }

        // Convert to ProcessingError if needed
        if procErr, ok := err.(*ProcessingError); ok {
            lastErr = procErr
        } else {
            lastErr = NewError(ErrInternalError, err.Error())
        }

        // Check if should retry
        if !strategy.ShouldRetry(lastErr, attempt) {
            return lastErr
        }

        // Calculate backoff
        backoff := strategy.BackoffDuration(attempt)
        lastErr.Details["retry_attempt"] = attempt + 1
        lastErr.Details["retry_backoff"] = backoff.String()

        // Wait before retry
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            continue
        }
    }

    return lastErr
}
```

### Circuit Breaker

```go
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    lastFailure  time.Time
    state        string // "closed", "open", "half-open"
    mu           sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    // Check state
    switch cb.state {
    case "open":
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = "half-open"
            cb.failures = 0
        } else {
            return NewError(ErrTimeout, "Circuit breaker is open")
        }
    }

    // Execute function
    err := fn()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()

        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }

    // Success: reset
    cb.failures = 0
    cb.state = "closed"
    return nil
}
```

---

## Error Logging and Monitoring

### Structured Logging

```go
func LogError(ctx context.Context, err *ProcessingError) {
    logger := log.WithFields(log.Fields{
        "error_code":    err.Code,
        "error_category": err.Category,
        "error_severity": err.Severity,
        "job_id":        err.JobID,
        "request_id":    err.RequestID,
        "retryable":     err.Retryable,
    })

    // Add FFmpeg context if available
    if err.FFmpegError != nil {
        logger = logger.WithFields(log.Fields{
            "ffmpeg_exit_code": err.FFmpegError.ExitCode,
            "ffmpeg_type":      err.FFmpegError.ParsedError.Type,
        })
    }

    // Log based on severity
    switch err.Severity {
    case SeverityInfo:
        logger.Info(err.Message)
    case SeverityWarning:
        logger.Warn(err.Message)
    case SeverityError:
        logger.Error(err.Message)
    case SeverityCritical:
        logger.Fatal(err.Message)
    }
}
```

### Error Metrics

```go
var (
    errorsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "errors_total",
            Help: "Total number of errors",
        },
        []string{"code", "category", "severity", "retryable"},
    )

    errorsByStage = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "errors_by_stage_total",
            Help: "Errors by processing stage",
        },
        []string{"stage", "code"},
    )

    ffmpegErrors = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ffmpeg_errors_total",
            Help: "FFmpeg errors by type",
        },
        []string{"type", "exit_code"},
    )
)

func RecordError(err *ProcessingError) {
    errorsTotal.WithLabelValues(
        string(err.Code),
        string(err.Category),
        string(err.Severity),
        string(err.Retryable),
    ).Inc()

    if stage, ok := err.Details["stage"].(string); ok {
        errorsByStage.WithLabelValues(stage, string(err.Code)).Inc()
    }

    if err.FFmpegError != nil {
        ffmpegErrors.WithLabelValues(
            err.FFmpegError.ParsedError.Type,
            fmt.Sprintf("%d", err.FFmpegError.ExitCode),
        ).Inc()
    }
}
```

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: media_pipeline_errors
    interval: 1m
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: |
          rate(errors_total{severity="error"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec"

      # Critical errors
      - alert: CriticalError
        expr: |
          errors_total{severity="critical"} > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Critical error detected"

      # FFmpeg failures
      - alert: FrequentFFmpegFailures
        expr: |
          rate(ffmpeg_errors_total[10m]) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Frequent FFmpeg failures"
          description: "FFmpeg error rate is {{ $value }} errors/sec"

      # Infrastructure issues
      - alert: InfrastructureErrors
        expr: |
          rate(errors_total{category="infrastructure"}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Infrastructure errors detected"
```

---

## Error Response Formatting

### API Error Response

```go
type APIErrorResponse struct {
    Error APIError `json:"error"`
}

type APIError struct {
    Code          string                 `json:"code"`
    Message       string                 `json:"message"`
    Details       map[string]interface{} `json:"details,omitempty"`
    Documentation string                 `json:"documentation,omitempty"`
    RequestID     string                 `json:"request_id,omitempty"`
}

func FormatAPIError(err *ProcessingError) *APIErrorResponse {
    return &APIErrorResponse{
        Error: APIError{
            Code:          string(err.Code),
            Message:       err.UserMessage,
            Details:       formatDetails(err),
            Documentation: err.Documentation,
            RequestID:     err.RequestID,
        },
    }
}

func formatDetails(err *ProcessingError) map[string]interface{} {
    details := make(map[string]interface{})

    // Add relevant details (filter sensitive info)
    if err.FFmpegError != nil && err.FFmpegError.ParsedError != nil {
        details["reason"] = err.FFmpegError.ParsedError.Reason
        details["suggestion"] = err.FFmpegError.ParsedError.Suggestion
    }

    // Add retry info
    if err.RetryAfter != nil {
        details["retry_after"] = err.RetryAfter.Seconds()
    }

    return details
}
```

### HTTP Status Code Mapping

```go
func ErrorToHTTPStatus(err *ProcessingError) int {
    metadata := errorCodeMetadata[err.Code]
    if metadata != nil {
        return metadata.HTTPStatus
    }

    // Fallback based on category
    switch err.Category {
    case CategoryValidation:
        return 400 // Bad Request
    case CategoryPermission:
        return 403 // Forbidden
    case CategoryNotFound:
        return 404 // Not Found
    case CategoryRateLimit:
        return 429 // Too Many Requests
    case CategoryTimeout:
        return 504 // Gateway Timeout
    default:
        return 500 // Internal Server Error
    }
}
```

---

## Debugging Tools

### Error Debugger

```go
type ErrorDebugger struct {
    store *Store
}

func (d *ErrorDebugger) AnalyzeJobFailure(ctx context.Context, jobID string) (*FailureAnalysis, error) {
    // Get job status
    job, err := d.store.GetJob(ctx, jobID)
    if err != nil {
        return nil, err
    }

    // Get execution logs
    logs, err := d.store.GetJobLogs(ctx, jobID)
    if err != nil {
        return nil, err
    }

    // Analyze error patterns
    analysis := &FailureAnalysis{
        JobID:       jobID,
        ErrorCode:   job.Error.Code,
        FailedStage: getFailedStage(logs),
        RootCause:   identifyRootCause(job.Error, logs),
        Timeline:    buildTimeline(logs),
        Suggestions: generateSuggestions(job.Error, logs),
    }

    return analysis, nil
}

type FailureAnalysis struct {
    JobID       string
    ErrorCode   ErrorCode
    FailedStage string
    RootCause   string
    Timeline    []TimelineEvent
    Suggestions []string
}

type TimelineEvent struct {
    Timestamp time.Time
    Stage     string
    Event     string
    Details   map[string]interface{}
}

func identifyRootCause(err *ProcessingError, logs []Log) string {
    // Walk error chain
    current := err
    for current.Cause != nil {
        current = current.Cause
    }

    // Analyze logs for context
    contextLogs := findContextLogs(logs, err)

    return fmt.Sprintf("%s: %s", current.Code, current.Message)
}

func generateSuggestions(err *ProcessingError, logs []Log) []string {
    suggestions := []string{}

    // Add error-specific suggestions
    if err.UserMessage != "" {
        suggestions = append(suggestions, err.UserMessage)
    }

    // Add FFmpeg-specific suggestions
    if err.FFmpegError != nil && err.FFmpegError.ParsedError != nil {
        suggestions = append(suggestions, err.FFmpegError.ParsedError.Suggestion)
    }

    // Add log-based suggestions
    if hasMemoryError(logs) {
        suggestions = append(suggestions, "Consider reducing video resolution or splitting into smaller segments")
    }

    if hasDiskSpaceError(logs) {
        suggestions = append(suggestions, "Free up disk space or use a smaller output format")
    }

    return suggestions
}
```

### Error Replay

```go
func (d *ErrorDebugger) ReplayJob(ctx context.Context, jobID string) error {
    // Get original job spec
    job, err := d.store.GetJob(ctx, jobID)
    if err != nil {
        return err
    }

    // Create replay job with debug mode
    replaySpec := job.Spec
    replaySpec.Debug = true

    // Submit new job
    replayJob, err := d.store.CreateJob(ctx, &replaySpec)
    if err != nil {
        return err
    }

    log.Printf("Created replay job: %s", replayJob.JobID)
    return nil
}
```

---

## Error Aggregation and Reporting

### Error Aggregation

```go
type ErrorAggregator struct {
    store *Store
}

func (a *ErrorAggregator) GetErrorStats(ctx context.Context, period time.Duration) (*ErrorStats, error) {
    since := time.Now().Add(-period)

    stats := &ErrorStats{
        Period:     period,
        TotalJobs:  0,
        FailedJobs: 0,
        ErrorsByCode: make(map[ErrorCode]int),
        ErrorsByCategory: make(map[ErrorCategory]int),
        TopErrors: []ErrorSummary{},
    }

    // Query database for error stats
    rows, err := a.store.db.QueryContext(ctx, `
        SELECT
            COUNT(*) as total,
            COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
            error->>'code' as error_code,
            COUNT(*) as count
        FROM jobs
        WHERE created_at >= $1
        GROUP BY error->>'code'
        ORDER BY count DESC
    `, since)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var errorCode string
        var count int
        rows.Scan(&stats.TotalJobs, &stats.FailedJobs, &errorCode, &count)

        code := ErrorCode(errorCode)
        stats.ErrorsByCode[code] = count

        if metadata := errorCodeMetadata[code]; metadata != nil {
            stats.ErrorsByCategory[metadata.Category] += count
        }
    }

    return stats, nil
}

type ErrorStats struct {
    Period     time.Duration
    TotalJobs  int
    FailedJobs int
    ErrorsByCode map[ErrorCode]int
    ErrorsByCategory map[ErrorCategory]int
    TopErrors  []ErrorSummary
}

type ErrorSummary struct {
    Code     ErrorCode
    Category ErrorCategory
    Count    int
    Examples []string // Job IDs
}
```

### Daily Error Report

```go
func GenerateDailyErrorReport(ctx context.Context, agg *ErrorAggregator) (*ErrorReport, error) {
    stats, err := agg.GetErrorStats(ctx, 24*time.Hour)
    if err != nil {
        return nil, err
    }

    report := &ErrorReport{
        Date:       time.Now(),
        TotalJobs:  stats.TotalJobs,
        FailedJobs: stats.FailedJobs,
        FailureRate: float64(stats.FailedJobs) / float64(stats.TotalJobs),
        TopErrors:  getTopErrors(stats, 10),
        Trends:     analyzeErrorTrends(stats),
    }

    return report, nil
}

type ErrorReport struct {
    Date       time.Time
    TotalJobs  int
    FailedJobs int
    FailureRate float64
    TopErrors  []ErrorSummary
    Trends     []ErrorTrend
}

type ErrorTrend struct {
    ErrorCode   ErrorCode
    Change      float64 // +/-% from previous period
    Impact      string  // "increasing", "decreasing", "stable"
}
```

---

## Testing Strategy

### Error Injection

```go
type ErrorInjector struct {
    enabled    bool
    errorRates map[ErrorCode]float64
}

func (e *ErrorInjector) ShouldInjectError(code ErrorCode) bool {
    if !e.enabled {
        return false
    }

    rate := e.errorRates[code]
    return rand.Float64() < rate
}

// Usage in tests
func TestErrorHandling(t *testing.T) {
    injector := &ErrorInjector{
        enabled: true,
        errorRates: map[ErrorCode]float64{
            ErrFFmpegFailed: 0.1, // 10% failure rate
        },
    }

    executor := NewJobExecutor(store, injector)
    err := executor.Execute(ctx, jobID)

    // Verify error handling
    assert.Error(t, err)
    procErr := err.(*ProcessingError)
    assert.Equal(t, ErrFFmpegFailed, procErr.Code)
}
```

---

## Summary

The error handling system provides:

1. **Comprehensive Error Classification**: Categories, severity, retryability
2. **Structured Error Codes**: 50+ error codes with metadata
3. **FFmpeg Error Parsing**: Pattern-based parsing with suggestions
4. **Rich Error Context**: Capture execution context for debugging
5. **Smart Retry Logic**: Decision tree with exponential backoff
6. **Circuit Breaker**: Prevent cascading failures
7. **Observability**: Metrics, logging, alerting
8. **Debugging Tools**: Error analysis, replay, aggregation
9. **User-Friendly Messages**: Clear, actionable error messages

Key features:
- Automatic FFmpeg error interpretation
- Retry strategies based on error type
- Detailed error context for debugging
- Prometheus metrics and alerts
- Error aggregation and reporting
- Error replay for debugging

---

**Status**: Ready for implementation
