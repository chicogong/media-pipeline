package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		// Localhost
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		// Private networks
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"192.168.255.255", true},
		// Link-local (AWS metadata)
		{"169.254.169.254", true},
		// Public IPs (not blocked)
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			assert.Equal(t, tt.blocked, IsBlockedIP(tt.ip))
		})
	}
}

func TestValidateHTTPURI(t *testing.T) {
	tests := []struct {
		uri     string
		wantErr bool
		errMsg  string
	}{
		{"https://example.com/video.mp4", false, ""},
		{"http://google.com/file.mp4", false, ""},
		{"https://127.0.0.1/video.mp4", true, "localhost"},
		{"http://10.0.0.1/internal.mp4", true, "private network"},
		{"https://192.168.1.1/file.mp4", true, "private network"},
		{"http://169.254.169.254/metadata", true, "link-local"},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			err := ValidateHTTPURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
