package schemas

import (
	"time"
)

// PlanNode represents a single node in the processing DAG
type PlanNode struct {
	ID        string                 `json:"id"`         // Unique node ID
	Op        string                 `json:"op"`         // Operator name
	Inputs    []string               `json:"inputs"`     // Input IDs
	Params    map[string]interface{} `json:"params"`     // Operation parameters
	Outputs   []string               `json:"outputs"`    // Output IDs
	DependsOn []string               `json:"depends_on"` // Node IDs this depends on
}

// HasDependency checks if this node depends on the given node ID
func (pn *PlanNode) HasDependency(nodeID string) bool {
	for _, dep := range pn.DependsOn {
		if dep == nodeID {
			return true
		}
	}
	return false
}

// ResourceEstimate provides resource usage estimation
type ResourceEstimate struct {
	EstimatedDuration time.Duration `json:"estimated_duration"` // Estimated processing time
	CPUIntensive      bool          `json:"cpu_intensive"`      // Whether this is CPU-bound
	RequiresTwoPass   bool          `json:"requires_two_pass"`  // Whether FFmpeg needs two passes
}

// ProcessingPlan is the compiled intermediate representation
type ProcessingPlan struct {
	JobID            string           `json:"job_id"`
	Nodes            []PlanNode       `json:"nodes"`             // DAG nodes in topological order
	ResourceEstimate ResourceEstimate `json:"resource_estimate"`
	FFmpegCommand    string           `json:"ffmpeg_command"`    // Generated FFmpeg command
	FFmpegVersion    string           `json:"ffmpeg_version"`    // FFmpeg version used
	CreatedAt        time.Time        `json:"created_at"`
}
