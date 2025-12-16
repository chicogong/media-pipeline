package schemas

import "time"

// JobSpec is the user-submitted job specification
type JobSpec struct {
	// Metadata
	JobID     string            `json:"job_id,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`

	// Configuration
	Debug    bool      `json:"debug,omitempty"`
	Priority int       `json:"priority,omitempty"`
	Timeout  *Duration `json:"timeout,omitempty"`

	// Core Specification
	Inputs     []Input     `json:"inputs"`
	Operations []Operation `json:"operations"`
	Outputs    []Output    `json:"outputs"`

	// Resource Limits
	Limits *ResourceLimits `json:"limits,omitempty"`

	// Webhook
	WebhookURL string `json:"webhook_url,omitempty"`
}

// Input represents an input source
type Input struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`
	Type        string            `json:"type,omitempty"`
	Format      string            `json:"format,omitempty"`
	StartOffset *Duration         `json:"start_offset,omitempty"`
	Duration    *Duration         `json:"duration,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Operation represents a processing operation
type Operation struct {
	Op     string                 `json:"op"`
	Input  string                 `json:"input,omitempty"`
	Inputs []string               `json:"inputs,omitempty"`
	Output string                 `json:"output"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// Output represents an output destination
type Output struct {
	ID          string            `json:"id"`
	Destination string            `json:"destination"`
	Format      string            `json:"format,omitempty"`
	Codec       *CodecParams      `json:"codec,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CodecParams specifies codec settings
type CodecParams struct {
	Video *VideoCodec `json:"video,omitempty"`
	Audio *AudioCodec `json:"audio,omitempty"`
}

// VideoCodec specifies video codec parameters
type VideoCodec struct {
	Codec       string `json:"codec,omitempty"`
	Bitrate     string `json:"bitrate,omitempty"`
	CRF         *int   `json:"crf,omitempty"`
	Preset      string `json:"preset,omitempty"`
	Profile     string `json:"profile,omitempty"`
	PixelFormat string `json:"pixel_format,omitempty"`
}

// AudioCodec specifies audio codec parameters
type AudioCodec struct {
	Codec      string `json:"codec,omitempty"`
	Bitrate    string `json:"bitrate,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
}

// ResourceLimits specifies resource constraints
type ResourceLimits struct {
	MaxDuration   *Duration `json:"max_duration,omitempty"`
	MaxResolution string    `json:"max_resolution,omitempty"`
	MaxOutputSize int64     `json:"max_output_size,omitempty"`
	MaxMemory     int64     `json:"max_memory,omitempty"`
}
