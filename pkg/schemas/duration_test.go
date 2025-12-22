package schemas

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    time.Duration
		wantErr bool
	}{
		{name: "go_duration", in: "1h30m", want: 90 * time.Minute},
		{name: "timecode_hms", in: "01:02:03", want: time.Hour + 2*time.Minute + 3*time.Second},
		{name: "timecode_millis_padding", in: "00:00:01.5", want: 1500 * time.Millisecond},
		{name: "iso8601", in: "PT1H30M", want: 90 * time.Minute},
		{name: "invalid", in: "nope", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDuration(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (duration=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("duration mismatch: got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	// Unmarshal from timecode format
	var d Duration
	if err := json.Unmarshal([]byte(`"00:01:30"`), &d); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if d.Duration != 90*time.Second {
		t.Fatalf("duration mismatch: got=%v want=%v", d.Duration, 90*time.Second)
	}

	// Marshal uses Go duration string (e.g., "1m30s")
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var d2 Duration
	if err := json.Unmarshal(b, &d2); err != nil {
		t.Fatalf("unmarshal roundtrip failed: %v", err)
	}
	if d2.Duration != 90*time.Second {
		t.Fatalf("roundtrip mismatch: got=%v want=%v", d2.Duration, 90*time.Second)
	}
}

