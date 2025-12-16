package schemas

import "time"

// ProcessingPlan is the compiled execution plan
type ProcessingPlan struct {
	// Metadata
	PlanID    string    `json:"plan_id"`
	JobID     string    `json:"job_id"`
	CreatedAt time.Time `json:"created_at"`

	// Execution Plan
	Nodes            []*PlanNode        `json:"nodes"`
	Edges            []*PlanEdge        `json:"edges"`
	ExecutionOrder   []string           `json:"execution_order"`   // Topological sort order
	ExecutionStages  [][]string         `json:"execution_stages"`  // Parallel execution stages
	ResourceEstimate *ResourceEstimates `json:"resource_estimate,omitempty"` // Resource estimates

	// Generated Artifacts
	FFmpegVersion string          `json:"ffmpeg_version"`
	Commands      []FFmpegCommand `json:"commands"`
}

// PlanNode represents a node in the execution DAG
type PlanNode struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "input", "operation", "output"

	// For input nodes
	InputID   string `json:"input_id,omitempty"`
	SourceURI string `json:"source_uri,omitempty"`

	// For operation nodes
	Operator string                 `json:"operator,omitempty"`
	Params   map[string]interface{} `json:"params,omitempty"`

	// For output nodes
	OutputID string `json:"output_id,omitempty"`
	DestURI  string `json:"dest_uri,omitempty"`

	// Metadata (computed during planning)
	Metadata  *MediaInfo     `json:"metadata,omitempty"` // Computed output metadata
	Estimates *NodeEstimates `json:"estimates,omitempty"`
}

// PlanEdge represents a dependency between nodes
type PlanEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	StreamType string `json:"stream_type,omitempty"` // "video", "audio", "both"
}

// MediaInfo contains detected media properties
type MediaInfo struct {
	Format       FormatInfo    `json:"format"`
	VideoStreams []VideoStream `json:"video_streams,omitempty"`
	AudioStreams []AudioStream `json:"audio_streams,omitempty"`
}

// FormatInfo contains format-level information
type FormatInfo struct {
	Filename  string        `json:"filename,omitempty"`
	Format    string        `json:"format,omitempty"`
	Duration  time.Duration `json:"duration"`
	Size      int64         `json:"size"`
	BitRate   int64         `json:"bit_rate,omitempty"`
	StartTime time.Duration `json:"start_time,omitempty"`
}

// VideoStream represents a video stream
type VideoStream struct {
	Index       int           `json:"index"`
	Codec       string        `json:"codec"`
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	FrameRate   float64       `json:"frame_rate"`
	PixelFormat string        `json:"pixel_format,omitempty"`
	BitRate     int64         `json:"bit_rate,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// AudioStream represents an audio stream
type AudioStream struct {
	Index      int           `json:"index"`
	Codec      string        `json:"codec"`
	SampleRate int           `json:"sample_rate"`
	Channels   int           `json:"channels"`
	BitRate    int64         `json:"bit_rate,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
}

// NodeEstimates contains resource estimates for a node
type NodeEstimates struct {
	Duration time.Duration `json:"duration"` // Estimated processing time
	MemoryMB int64         `json:"memory_mb"` // Peak memory in MB
	DiskMB   int64         `json:"disk_mb"`   // Disk space in MB
	CPUCores float64       `json:"cpu_cores,omitempty"` // CPU cores utilized
}

// ResourceEstimates contains total resource estimates
type ResourceEstimates struct {
	NodeEstimates map[string]*NodeEstimates `json:"node_estimates"` // Per-node estimates
	TotalDuration time.Duration              `json:"total_duration"` // Total processing time
	PeakMemoryMB  int64                      `json:"peak_memory_mb"` // Peak memory across all stages
	TotalDiskMB   int64                      `json:"total_disk_mb"`  // Total disk space needed
}

// FFmpegCommand represents a generated FFmpeg command
type FFmpegCommand struct {
	ID          string   `json:"id"`
	Stage       string   `json:"stage"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	WorkDir     string   `json:"work_dir"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Filtergraph string   `json:"filtergraph,omitempty"`
}
