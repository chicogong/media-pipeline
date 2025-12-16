package schemas

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Duration wraps time.Duration with custom JSON marshaling
type Duration struct {
	time.Duration
}

// MarshalJSON converts Duration to JSON string
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON parses Duration from multiple formats
func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	parsed, err := ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = parsed
	return nil
}

// ParseDuration parses duration from multiple formats:
// - Go duration: "1h30m", "90s"
// - Timecode: "01:30:00", "00:05:30.500"
// - ISO 8601: "PT1H30M"
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Try Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try timecode format (HH:MM:SS or HH:MM:SS.mmm)
	if d, err := parseTimecode(s); err == nil {
		return d, nil
	}

	// Try ISO 8601 format (PT1H30M)
	if strings.HasPrefix(s, "PT") {
		return parseISO8601(s)
	}

	return 0, fmt.Errorf("invalid duration format: %s", s)
}

// parseTimecode parses "HH:MM:SS" or "HH:MM:SS.mmm" format
func parseTimecode(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d{1,2}):(\d{2}):(\d{2})(?:\.(\d{1,3}))?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid timecode format")
	}

	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])

	d := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second

	if matches[4] != "" {
		// Pad milliseconds to 3 digits
		ms := matches[4]
		for len(ms) < 3 {
			ms += "0"
		}
		millis, _ := strconv.Atoi(ms)
		d += time.Duration(millis) * time.Millisecond
	}

	return d, nil
}

// parseISO8601 parses "PT1H30M" format
func parseISO8601(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "PT") {
		return 0, fmt.Errorf("invalid ISO 8601 format")
	}

	s = s[2:] // Remove "PT"
	var d time.Duration

	re := regexp.MustCompile(`(\d+)([HMS])`)
	matches := re.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		value, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "H":
			d += time.Duration(value) * time.Hour
		case "M":
			d += time.Duration(value) * time.Minute
		case "S":
			d += time.Duration(value) * time.Second
		}
	}

	return d, nil
}
