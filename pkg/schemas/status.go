package schemas

import "time"

// JobState represents the current state of a job
type JobState string

const (
	JobStatePending           JobState = "pending"
	JobStateValidating        JobState = "validating"
	JobStatePlanning          JobState = "planning"
	JobStateDownloadingInputs JobState = "downloading_inputs"
	JobStateProcessing        JobState = "processing"
	JobStateUploadingOutputs  JobState = "uploading_outputs"
	JobStateCompleted         JobState = "completed"
	JobStateFailed            JobState = "failed"
	JobStateCancelled         JobState = "cancelled"
)

// JobStatus represents real-time job status
type JobStatus struct {
	JobID       string       `json:"job_id"`
	Status      JobState     `json:"status"`
	Progress    *Progress    `json:"progress,omitempty"`
	Error       *ErrorInfo   `json:"error,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	StartedAt   *time.Time   `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	OutputFiles []OutputFile `json:"output_files,omitempty"`
}

// Progress represents job progress information
type Progress struct {
	OverallPercent      float64           `json:"overall_percent"`
	CurrentStep         string            `json:"current_step"`
	StepProgress        *StepProgress     `json:"step_progress,omitempty"`
	EstimatedCompletion *time.Time        `json:"estimated_completion,omitempty"`
}

// StepProgress contains detailed progress for current step
type StepProgress struct {
	DownloadProgress *DownloadProgress `json:"download_progress,omitempty"`
	FFmpegProgress   *FFmpegProgress   `json:"ffmpeg_progress,omitempty"`
	UploadProgress   *UploadProgress   `json:"upload_progress,omitempty"`
}

// DownloadProgress tracks input download progress
type DownloadProgress struct {
	TotalFiles      int    `json:"total_files"`
	CompletedFiles  int    `json:"completed_files"`
	CurrentFile     string `json:"current_file"`
	BytesDownloaded int64  `json:"bytes_downloaded"`
	TotalBytes      int64  `json:"total_bytes"`
}

// FFmpegProgress tracks FFmpeg execution progress
type FFmpegProgress struct {
	Frame       int     `json:"frame"`
	FPS         float64 `json:"fps"`
	CurrentTime string  `json:"current_time"`
	TotalTime   string  `json:"total_time"`
	Speed       string  `json:"speed"`
	Bitrate     string  `json:"bitrate"`
	TotalSize   int64   `json:"total_size"`
}

// UploadProgress tracks output upload progress
type UploadProgress struct {
	TotalFiles     int    `json:"total_files"`
	CompletedFiles int    `json:"completed_files"`
	CurrentFile    string `json:"current_file"`
	BytesUploaded  int64  `json:"bytes_uploaded"`
	TotalBytes     int64  `json:"total_bytes"`
}

// OutputFile contains information about an output file
type OutputFile struct {
	OutputID    string     `json:"output_id"`
	Destination string     `json:"destination"`
	FileSize    int64      `json:"file_size"`
	MD5         string     `json:"md5,omitempty"`
	Duration    float64    `json:"duration,omitempty"`
	MediaInfo   *MediaInfo `json:"media_info,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code             string                 `json:"code"`
	Message          string                 `json:"message"`
	Details          map[string]interface{} `json:"details,omitempty"`
	FFmpegStderr     string                 `json:"ffmpeg_stderr,omitempty"`
	FFmpegExitCode   int                    `json:"ffmpeg_exit_code,omitempty"`
	StackTrace       string                 `json:"stack_trace,omitempty"`
	Retryable        bool                   `json:"retryable"`
	RetryAfter       *time.Duration         `json:"retry_after,omitempty"`
}
