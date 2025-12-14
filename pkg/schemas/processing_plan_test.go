package schemas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessingPlan_JSON(t *testing.T) {
	plan := &ProcessingPlan{
		JobID: "test-job-123",
		Nodes: []PlanNode{
			{
				ID:       "node1",
				Op:       "trim",
				Inputs:   []string{"video1"},
				Params:   map[string]interface{}{"start": "00:00:10"},
				Outputs:  []string{"trimmed"},
				DependsOn: []string{},
			},
		},
		ResourceEstimate: ResourceEstimate{
			EstimatedDuration: 120 * time.Second,
			CPUIntensive:      true,
		},
		FFmpegCommand: "ffmpeg -i input.mp4 ...",
		CreatedAt:     time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(plan)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-job-123")

	// Test JSON unmarshaling
	var decoded ProcessingPlan
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, plan.JobID, decoded.JobID)
	assert.Len(t, decoded.Nodes, 1)
	assert.Equal(t, "node1", decoded.Nodes[0].ID)
}

func TestPlanNode_HasDependency(t *testing.T) {
	node := &PlanNode{
		ID:        "node2",
		DependsOn: []string{"node1", "node3"},
	}

	assert.True(t, node.HasDependency("node1"))
	assert.True(t, node.HasDependency("node3"))
	assert.False(t, node.HasDependency("node4"))
}
