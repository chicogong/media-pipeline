package prober_test

import (
	"context"
	"fmt"
	"log"

	"github.com/chicogong/media-pipeline/pkg/prober"
)

// Example_basic demonstrates basic media probing
func Example_basic() {
	// Create a new prober
	p := prober.NewProber()

	// Probe a video file
	ctx := context.Background()
	info, err := p.Probe(ctx, "input.mp4")
	if err != nil {
		log.Fatal(err)
	}

	// Print media information
	fmt.Printf("Format: %s\n", info.Format.Format)
	fmt.Printf("Duration: %v\n", info.Format.Duration)
	fmt.Printf("Size: %d bytes\n", info.Format.Size)
	fmt.Printf("Video streams: %d\n", len(info.VideoStreams))
	fmt.Printf("Audio streams: %d\n", len(info.AudioStreams))

	// Access video stream details
	if len(info.VideoStreams) > 0 {
		video := info.VideoStreams[0]
		fmt.Printf("Video codec: %s\n", video.Codec)
		fmt.Printf("Resolution: %dx%d\n", video.Width, video.Height)
		fmt.Printf("Frame rate: %.2f fps\n", video.FrameRate)
	}

	// Access audio stream details
	if len(info.AudioStreams) > 0 {
		audio := info.AudioStreams[0]
		fmt.Printf("Audio codec: %s\n", audio.Codec)
		fmt.Printf("Sample rate: %d Hz\n", audio.SampleRate)
		fmt.Printf("Channels: %d\n", audio.Channels)
	}
}

// Example_customPath demonstrates using a custom ffprobe path
func Example_customPath() {
	// Create a prober with custom ffprobe path
	p := prober.NewProber(prober.WithFFprobePath("/usr/local/bin/ffprobe"))

	ctx := context.Background()
	info, err := p.Probe(ctx, "video.mp4")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Probed: %s\n", info.Format.Filename)
}

// Example_contextCancellation demonstrates cancelling a probe operation
func Example_contextCancellation() {
	p := prober.NewProber()

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start probing in a goroutine
	go func() {
		_, err := p.Probe(ctx, "large_file.mp4")
		if err != nil {
			fmt.Println("Probe cancelled:", err)
		}
	}()

	// Cancel the operation
	cancel()
}
