package executor

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Progress represents FFmpeg encoding progress
type Progress struct {
	Frame   int           // Current frame number
	FPS     float64       // Frames per second
	Time    time.Duration // Current position in media
	Size    int64         // Output size in bytes
	Bitrate float64       // Bitrate in kbits/s
	Speed   float64       // Encoding speed multiplier (1.0 = realtime)
}

// ProgressParser parses FFmpeg progress output
type ProgressParser struct {
	totalDuration time.Duration
	frameRegex    *regexp.Regexp
	fpsRegex      *regexp.Regexp
	timeRegex     *regexp.Regexp
	sizeRegex     *regexp.Regexp
	bitrateRegex  *regexp.Regexp
	speedRegex    *regexp.Regexp
}

// NewProgressParser creates a new progress parser
func NewProgressParser() *ProgressParser {
	return &ProgressParser{
		frameRegex:   regexp.MustCompile(`frame=\s*(\d+)`),
		fpsRegex:     regexp.MustCompile(`fps=\s*([\d.]+)`),
		timeRegex:    regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`),
		sizeRegex:    regexp.MustCompile(`size=\s*(\d+)kB`),
		bitrateRegex: regexp.MustCompile(`bitrate=\s*([\d.]+)kbits/s`),
		speedRegex:   regexp.MustCompile(`speed=\s*([\d.]+)x`),
	}
}

// SetTotalDuration sets the total duration for percentage calculation
func (pp *ProgressParser) SetTotalDuration(duration time.Duration) {
	pp.totalDuration = duration
}

// ParseLine parses a single line of FFmpeg output
// Returns nil if the line doesn't contain progress information
func (pp *ProgressParser) ParseLine(line string) *Progress {
	// Check if this line contains progress info (has "frame=" field)
	if !strings.Contains(line, "frame=") {
		return nil
	}

	progress := &Progress{}

	// Parse frame
	if matches := pp.frameRegex.FindStringSubmatch(line); len(matches) > 1 {
		frame, _ := strconv.Atoi(matches[1])
		progress.Frame = frame
	}

	// Parse fps
	if matches := pp.fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
		fps, _ := strconv.ParseFloat(matches[1], 64)
		progress.FPS = fps
	}

	// Parse time
	if matches := pp.timeRegex.FindStringSubmatch(line); len(matches) > 4 {
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.Atoi(matches[3])
		centiseconds, _ := strconv.Atoi(matches[4])

		progress.Time = time.Duration(hours)*time.Hour +
			time.Duration(minutes)*time.Minute +
			time.Duration(seconds)*time.Second +
			time.Duration(centiseconds)*10*time.Millisecond
	}

	// Parse size
	if matches := pp.sizeRegex.FindStringSubmatch(line); len(matches) > 1 {
		sizeKB, _ := strconv.ParseInt(matches[1], 10, 64)
		progress.Size = sizeKB * 1024
	}

	// Parse bitrate
	if matches := pp.bitrateRegex.FindStringSubmatch(line); len(matches) > 1 {
		bitrate, _ := strconv.ParseFloat(matches[1], 64)
		progress.Bitrate = bitrate
	}

	// Parse speed
	if matches := pp.speedRegex.FindStringSubmatch(line); len(matches) > 1 {
		speed, _ := strconv.ParseFloat(matches[1], 64)
		progress.Speed = speed
	}

	return progress
}

// ComputePercentage computes completion percentage based on time
func (pp *ProgressParser) ComputePercentage(progress *Progress) float64 {
	if pp.totalDuration == 0 {
		return 0.0
	}

	percentage := float64(progress.Time) / float64(pp.totalDuration) * 100.0
	if percentage > 100.0 {
		percentage = 100.0
	}

	return percentage
}

// parseFFmpegTime parses FFmpeg time format (HH:MM:SS.CS)
func parseFFmpegTime(timeStr string) time.Duration {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])

	secParts := strings.Split(parts[2], ".")
	seconds, _ := strconv.Atoi(secParts[0])

	var centiseconds int
	if len(secParts) > 1 {
		centiseconds, _ = strconv.Atoi(secParts[1])
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(centiseconds)*10*time.Millisecond
}
