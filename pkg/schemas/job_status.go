package schemas

import (
	"time"
)

// Status represents the current state of a job
type Status string

const (
	StatusPending            Status = "pending"
	StatusDownloadingInputs  Status = "downloading_inputs"
	StatusProcessing         Status = "processing"
	StatusUploadingOutputs   Status = "uploading_outputs"
	StatusCompleted          Status = "completed"
	StatusFailed             Status = "failed"
)

// FFmpegProgress contains real-time progress from FFmpeg
type FFmpegProgress struct {
	Frame   int     `json:"frame"`
	FPS     float64 `json:"fps"`
	Time    string  `json:"time"`     // Current output time (HH:MM:SS.mmm)
	Speed   string  `json:"speed"`    // Processing speed (e.g., "1.2x")
	Bitrate string  `json:"bitrate"`
}

// JobStatus represents the current status of a job
type JobStatus struct {
	JobID           string          `json:"job_id"`
	Status          Status          `json:"status"`
	Progress        float64         `json:"progress"`         // 0-100
	CurrentStep     string          `json:"current_step"`     // Human-readable current step
	FFmpegProgress  *FFmpegProgress `json:"ffmpeg_progress,omitempty"`
	Error           *ProcessingError `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	EstimatedCompletion *time.Time  `json:"estimated_completion,omitempty"`
}

// IsTerminal returns true if the job is in a terminal state
func (js *JobStatus) IsTerminal() bool {
	return js.Status == StatusCompleted || js.Status == StatusFailed
}

// ProcessingError represents a structured error
type ProcessingError struct {
	Code           string                 `json:"code"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details,omitempty"`
	FFmpegStderr   string                 `json:"ffmpeg_stderr,omitempty"`
	FFmpegExitCode int                    `json:"ffmpeg_exit_code,omitempty"`
}

// Error implements the error interface
func (pe *ProcessingError) Error() string {
	return pe.Message
}
