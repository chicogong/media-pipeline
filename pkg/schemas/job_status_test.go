package schemas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobStatus_JSON(t *testing.T) {
	status := &JobStatus{
		JobID:     "job-123",
		Status:    StatusProcessing,
		Progress:  45.5,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FFmpegProgress: &FFmpegProgress{
			Frame:   1350,
			FPS:     30.2,
			Time:    "00:00:45.000",
			Speed:   "1.2x",
			Bitrate: "2500kbits/s",
		},
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)
	assert.Contains(t, string(data), "job-123")
	assert.Contains(t, string(data), "processing")

	var decoded JobStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, status.JobID, decoded.JobID)
	assert.Equal(t, StatusProcessing, decoded.Status)
	assert.Equal(t, 45.5, decoded.Progress)
}

func TestJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		terminal bool
	}{
		{StatusPending, false},
		{StatusProcessing, false},
		{StatusCompleted, true},
		{StatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			js := &JobStatus{Status: tt.status}
			assert.Equal(t, tt.terminal, js.IsTerminal())
		})
	}
}
