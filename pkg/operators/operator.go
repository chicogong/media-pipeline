package operators

import (
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Operator is the interface all operators must implement
type Operator interface {
	// Name returns the unique operator identifier
	Name() string

	// Category returns the operator category
	Category() Category

	// Describe returns operator description and parameter schema
	Describe() *OperatorDescriptor

	// ValidateParams validates operation parameters
	ValidateParams(params map[string]interface{}) error

	// ComputeOutputMetadata calculates output media properties
	ComputeOutputMetadata(params map[string]interface{}, inputs []*schemas.MediaInfo) (*schemas.MediaInfo, error)

	// EstimateResources estimates CPU time, memory, and disk usage
	EstimateResources(params map[string]interface{}, inputs []*schemas.MediaInfo) (*schemas.NodeEstimates, error)

	// Compile generates FFmpeg filter syntax or command arguments
	Compile(ctx *CompileContext) (*CompileResult, error)
}

// Category represents operator category
type Category string

const (
	CategoryTimeline Category = "timeline" // trim, concat, split
	CategoryAudio    Category = "audio"    // loudnorm, mix, volume
	CategoryVideo    Category = "video"    // scale, crop, rotate
	CategoryGraphics Category = "graphics" // overlay, drawtext, subtitles
	CategoryOutput   Category = "output"   // export, thumbnail, waveform
	CategoryAdvanced Category = "advanced" // custom filters
)

// OperatorDescriptor describes an operator
type OperatorDescriptor struct {
	Name        string
	Category    Category
	Description string

	// Parameter schema
	Parameters []ParameterDescriptor

	// Input requirements
	MinInputs   int
	MaxInputs   int
	InputTypes  []MediaType

	// Output types
	OutputTypes []MediaType

	// Special requirements
	RequiresTwoPass   bool
	SupportsStreaming bool
}

// MediaType represents media type
type MediaType string

const (
	MediaTypeVideo      MediaType = "video"
	MediaTypeAudio      MediaType = "audio"
	MediaTypeVideoAudio MediaType = "video+audio"
	MediaTypeImage      MediaType = "image"
	MediaTypeAny        MediaType = "any"
)

// CompileContext contains context for compilation
type CompileContext struct {
	// Inputs
	InputStreams  []StreamRef
	Params        map[string]interface{}

	// Environment
	WorkDir string
	TempDir string

	// Metadata
	InputMetadata []*schemas.MediaInfo

	// Options
	Debug bool
}

// StreamRef references an input stream
type StreamRef struct {
	SourceID    string
	StreamIndex int
	StreamType  string // "video", "audio"
	Label       string // FFmpeg label (e.g., "[v0]")
}

// CompileResult contains compilation result
type CompileResult struct {
	// Filtergraph fragment
	FilterExpression string

	// Or complete command
	Command *Command

	// Output stream labels
	OutputLabels []string

	// Temporary files
	TempFiles []string

	// Dependencies
	DependsOn []string
}

// Command represents an FFmpeg command
type Command struct {
	Stage   string   // "probe", "loudnorm_pass1", "main"
	Args    []string
	Stdin   string
	WorkDir string
}
