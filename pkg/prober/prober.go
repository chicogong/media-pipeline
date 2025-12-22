// Package prober provides media file probing using ffprobe
package prober

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// Prober probes media files using ffprobe
type Prober struct {
	ffprobePath string
}

// ProberOption is a functional option for Prober
type ProberOption func(*Prober)

// WithFFprobePath sets a custom ffprobe binary path
func WithFFprobePath(path string) ProberOption {
	return func(p *Prober) {
		p.ffprobePath = path
	}
}

// NewProber creates a new Prober instance
func NewProber(opts ...ProberOption) *Prober {
	p := &Prober{
		ffprobePath: findFFprobe(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Probe probes a media file and returns its metadata
func (p *Prober) Probe(ctx context.Context, filePath string) (*schemas.MediaInfo, error) {
	if p.ffprobePath == "" {
		return nil, fmt.Errorf("ffprobe not found in PATH")
	}

	// Build ffprobe command
	args := []string{
		"-v", "quiet",                    // Suppress logs
		"-print_format", "json",          // Output JSON
		"-show_format",                   // Show format info
		"-show_streams",                  // Show stream info
		filePath,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)

	// Execute command
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffprobe failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffprobe execution error: %w", err)
	}

	// Parse output
	return parseFFprobeOutput(output)
}

// findFFprobe locates ffprobe in PATH
func findFFprobe() string {
	// Try common paths
	candidates := []string{
		"ffprobe",                         // In PATH
		"/usr/local/bin/ffprobe",         // Homebrew on macOS
		"/opt/homebrew/bin/ffprobe",      // Apple Silicon Homebrew
		"/usr/bin/ffprobe",               // Linux
	}

	for _, path := range candidates {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	return ""
}

// ffprobeOutput represents the raw JSON output from ffprobe
type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Filename   string `json:"filename"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
	StartTime  string `json:"start_time"`
}

type ffprobeStream struct {
	Index       int    `json:"index"`
	CodecType   string `json:"codec_type"`
	CodecName   string `json:"codec_name"`

	// Video fields
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	RFrameRate  string `json:"r_frame_rate"`
	PixelFormat string `json:"pix_fmt"`

	// Audio fields
	SampleRate  string `json:"sample_rate"`
	Channels    int    `json:"channels"`

	// Common fields
	BitRate     string `json:"bit_rate"`
	Duration    string `json:"duration"`
}

// parseFFprobeOutput parses ffprobe JSON output into MediaInfo
func parseFFprobeOutput(data []byte) (*schemas.MediaInfo, error) {
	var output ffprobeOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &schemas.MediaInfo{}

	// Parse format info
	info.Format = schemas.FormatInfo{
		Filename: output.Format.Filename,
		Format:   output.Format.FormatName,
		Duration: parseDuration(output.Format.Duration),
		Size:     parseInt64(output.Format.Size),
		BitRate:  parseInt64(output.Format.BitRate),
		StartTime: parseDuration(output.Format.StartTime),
	}

	// Parse streams
	for _, stream := range output.Streams {
		switch stream.CodecType {
		case "video":
			info.VideoStreams = append(info.VideoStreams, schemas.VideoStream{
				Index:       stream.Index,
				Codec:       stream.CodecName,
				Width:       stream.Width,
				Height:      stream.Height,
				FrameRate:   parseFrameRate(stream.RFrameRate),
				PixelFormat: stream.PixelFormat,
				BitRate:     parseInt64(stream.BitRate),
				Duration:    parseDuration(stream.Duration),
			})
		case "audio":
			info.AudioStreams = append(info.AudioStreams, schemas.AudioStream{
				Index:      stream.Index,
				Codec:      stream.CodecName,
				SampleRate: parseInt(stream.SampleRate),
				Channels:   stream.Channels,
				BitRate:    parseInt64(stream.BitRate),
				Duration:   parseDuration(stream.Duration),
			})
		}
	}

	return info, nil
}

// parseDuration parses a duration string from ffprobe (seconds as float)
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}

	seconds, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}

// parseInt64 parses an int64 from string
func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return v
}

// parseInt parses an int from string
func parseInt(s string) int {
	if s == "" {
		return 0
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}

	return v
}

// parseFrameRate parses a frame rate from ffprobe format (e.g., "30/1" or "30000/1001")
func parseFrameRate(s string) float64 {
	if s == "" {
		return 0
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		// Try parsing as plain float
		rate, _ := strconv.ParseFloat(s, 64)
		return rate
	}

	numerator, err1 := strconv.ParseFloat(parts[0], 64)
	denominator, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}
