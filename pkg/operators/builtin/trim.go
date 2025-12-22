package builtin

import (
	"fmt"
	"time"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// TrimOperator implements the trim operation
type TrimOperator struct{}

func init() {
	operators.Register(&TrimOperator{})
}

func (o *TrimOperator) Name() string {
	return "trim"
}

func (o *TrimOperator) Category() operators.Category {
	return operators.CategoryTimeline
}

func (o *TrimOperator) Describe() *operators.OperatorDescriptor {
	return &operators.OperatorDescriptor{
		Name:        "trim",
		Category:    operators.CategoryTimeline,
		Description: "Trim video/audio to specified time range",
		Parameters: []operators.ParameterDescriptor{
			{
				Name:        "start",
				Type:        operators.TypeTimecode,
				Required:    false,
				Default:     "00:00:00",
				Description: "Start time",
				Examples:    []interface{}{"00:00:10", "10s", "00:00:10.500"},
			},
			{
				Name:        "duration",
				Type:        operators.TypeDuration,
				Required:    false,
				Description: "Duration (if not specified, trim to end)",
				Examples:    []interface{}{"00:05:00", "5m", "300s"},
			},
			{
				Name:        "end",
				Type:        operators.TypeTimecode,
				Required:    false,
				Description: "End time (alternative to duration)",
			},
		},
		MinInputs:         1,
		MaxInputs:         1,
		InputTypes:        []operators.MediaType{operators.MediaTypeVideoAudio, operators.MediaTypeVideo, operators.MediaTypeAudio},
		OutputTypes:       []operators.MediaType{operators.MediaTypeVideoAudio},
		SupportsStreaming: true,
	}
}

func (o *TrimOperator) ValidateParams(params map[string]interface{}) error {
	if err := operators.StandardValidation(o, params); err != nil {
		return err
	}

	// Business logic validation
	_, hasDuration := params["duration"]
	_, hasEnd := params["end"]
	if hasDuration && hasEnd {
		return fmt.Errorf("cannot specify both 'duration' and 'end'")
	}

	return nil
}

func (o *TrimOperator) ComputeOutputMetadata(
	params map[string]interface{},
	inputs []*schemas.MediaInfo,
) (*schemas.MediaInfo, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("trim requires at least one input")
	}

	converter := operators.NewTypeConverter()
	input := inputs[0]
	output := *input // Copy

	// Update duration
	if duration, ok := params["duration"]; ok {
		d, err := converter.Convert(duration, operators.TypeDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
		output.Format.Duration = d.(time.Duration)
	} else if end, ok := params["end"]; ok {
		startVal, err := converter.Convert(params["start"], operators.TypeDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid start: %w", err)
		}
		endVal, err := converter.Convert(end, operators.TypeDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid end: %w", err)
		}
		output.Format.Duration = endVal.(time.Duration) - startVal.(time.Duration)
	}

	return &output, nil
}

func (o *TrimOperator) EstimateResources(
	params map[string]interface{},
	inputs []*schemas.MediaInfo,
) (*schemas.NodeEstimates, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input metadata")
	}

	converter := operators.NewTypeConverter()
	duration := inputs[0].Format.Duration
	if d, ok := params["duration"]; ok {
		converted, err := converter.Convert(d, operators.TypeDuration)
		if err == nil {
			duration = converted.(time.Duration)
		}
	}

	// Trim is fast (mostly copy operation if on keyframes)
	// Estimate 10% of realtime
	cpuTime := duration / 10

	bitrate := inputs[0].Format.BitRate
	if bitrate == 0 {
		bitrate = 5000000 // Default 5 Mbps
	}

	return &schemas.NodeEstimates{
		Duration: cpuTime,
		MemoryMB: 100, // 100MB
		DiskMB:   int64(bitrate * int64(duration.Seconds()) / 8 / 1024 / 1024),
	}, nil
}

func (o *TrimOperator) Compile(ctx *operators.CompileContext) (*operators.CompileResult, error) {
	converter := operators.NewTypeConverter()

	startValue, ok := ctx.Params["start"]
	if !ok {
		startValue = "00:00:00"
	}
	start, err := converter.Convert(startValue, operators.TypeDuration)
	if err != nil {
		return nil, err
	}
	startDuration := start.(time.Duration)

	var videoInputLabel string
	var audioInputLabel string
	for _, stream := range ctx.InputStreams {
		switch stream.StreamType {
		case "video":
			if videoInputLabel == "" {
				videoInputLabel = stream.Label
			}
		case "audio":
			if audioInputLabel == "" {
				audioInputLabel = stream.Label
			}
		}
	}
	if videoInputLabel == "" && audioInputLabel == "" {
		return nil, fmt.Errorf("trim requires at least one input stream")
	}

	var filterVideo, filterAudio string

	if duration, ok := ctx.Params["duration"]; ok {
		d, err := converter.Convert(duration, operators.TypeDuration)
		if err != nil {
			return nil, err
		}
		durationValue := d.(time.Duration)

		if videoInputLabel != "" {
			filterVideo = fmt.Sprintf("%strim=start=%.3f:duration=%.3f[v]",
				videoInputLabel, startDuration.Seconds(), durationValue.Seconds())
		}
		if audioInputLabel != "" {
			filterAudio = fmt.Sprintf("%satrim=start=%.3f:duration=%.3f[a]",
				audioInputLabel, startDuration.Seconds(), durationValue.Seconds())
		}
	} else {
		if videoInputLabel != "" {
			filterVideo = fmt.Sprintf("%strim=start=%.3f[v]",
				videoInputLabel, startDuration.Seconds())
		}
		if audioInputLabel != "" {
			filterAudio = fmt.Sprintf("%satrim=start=%.3f[a]",
				audioInputLabel, startDuration.Seconds())
		}
	}

	filterExpression := ""
	outputLabels := []string{}

	if filterVideo != "" {
		filterExpression = filterVideo
		outputLabels = append(outputLabels, "[v]")
	}
	if filterAudio != "" {
		if filterExpression != "" {
			filterExpression += ";"
		}
		filterExpression += filterAudio
		outputLabels = append(outputLabels, "[a]")
	}

	return &operators.CompileResult{
		FilterExpression: filterExpression,
		OutputLabels:     outputLabels,
	}, nil
}
