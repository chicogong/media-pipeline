package builtin

import (
	"fmt"

	"github.com/chicogong/media-pipeline/pkg/operators"
	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// ScaleOperator implements the scale operation
type ScaleOperator struct{}

func init() {
	operators.Register(&ScaleOperator{})
}

func (o *ScaleOperator) Name() string {
	return "scale"
}

func (o *ScaleOperator) Category() operators.Category {
	return operators.CategoryVideo
}

func (o *ScaleOperator) Describe() *operators.OperatorDescriptor {
	return &operators.OperatorDescriptor{
		Name:        "scale",
		Category:    operators.CategoryVideo,
		Description: "Scale video to specified resolution",
		Parameters: []operators.ParameterDescriptor{
			{
				Name:        "width",
				Type:        operators.TypeInt,
				Required:    true,
				Description: "Target width (or -1 to maintain aspect ratio)",
				Validation: &operators.ValidationRules{
					Min: floatPtr(-1),
					Max: floatPtr(7680),
				},
			},
			{
				Name:        "height",
				Type:        operators.TypeInt,
				Required:    true,
				Description: "Target height (or -1 to maintain aspect ratio)",
				Validation: &operators.ValidationRules{
					Min: floatPtr(-1),
					Max: floatPtr(4320),
				},
			},
			{
				Name:        "algorithm",
				Type:        operators.TypeEnum,
				Required:    false,
				Default:     "bicubic",
				Description: "Scaling algorithm",
				Validation: &operators.ValidationRules{
					Enum: []interface{}{"bilinear", "bicubic", "lanczos", "neighbor"},
				},
			},
		},
		MinInputs:         1,
		MaxInputs:         1,
		InputTypes:        []operators.MediaType{operators.MediaTypeVideo, operators.MediaTypeVideoAudio},
		OutputTypes:       []operators.MediaType{operators.MediaTypeVideo},
		SupportsStreaming: true,
	}
}

func (o *ScaleOperator) ValidateParams(params map[string]interface{}) error {
	if err := operators.StandardValidation(o, params); err != nil {
		return err
	}

	converter := operators.NewTypeConverter()
	width, _ := converter.Convert(params["width"], operators.TypeInt)
	height, _ := converter.Convert(params["height"], operators.TypeInt)

	if width.(int) == -1 && height.(int) == -1 {
		return fmt.Errorf("both width and height cannot be -1")
	}

	return nil
}

func (o *ScaleOperator) ComputeOutputMetadata(
	params map[string]interface{},
	inputs []*schemas.MediaInfo,
) (*schemas.MediaInfo, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("scale requires at least one input")
	}

	input := inputs[0]
	output := *input
	output.VideoStreams = append([]schemas.VideoStream(nil), input.VideoStreams...)
	output.AudioStreams = append([]schemas.AudioStream(nil), input.AudioStreams...)

	converter := operators.NewTypeConverter()
	width, _ := converter.Convert(params["width"], operators.TypeInt)
	height, _ := converter.Convert(params["height"], operators.TypeInt)

	widthInt := width.(int)
	heightInt := height.(int)

	if len(output.VideoStreams) > 0 {
		inputWidth := output.VideoStreams[0].Width
		inputHeight := output.VideoStreams[0].Height

		// Calculate actual dimensions
		if widthInt == -1 {
			widthInt = inputWidth * heightInt / inputHeight
		} else if heightInt == -1 {
			heightInt = inputHeight * widthInt / inputWidth
		}

		output.VideoStreams[0].Width = widthInt
		output.VideoStreams[0].Height = heightInt
	}

	return &output, nil
}

func (o *ScaleOperator) EstimateResources(
	params map[string]interface{},
	inputs []*schemas.MediaInfo,
) (*schemas.NodeEstimates, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input metadata")
	}

	duration := inputs[0].Format.Duration

	// Scaling is moderately expensive (estimate 50% of realtime)
	cpuTime := duration / 2

	bitrate := inputs[0].Format.BitRate
	if bitrate == 0 {
		bitrate = 5000000 // Default 5 Mbps
	}

	return &schemas.NodeEstimates{
		Duration: cpuTime,
		MemoryMB: 200,  // 200MB
		DiskMB:   int64(bitrate * int64(duration.Seconds()) / 8 / 1024 / 1024),
	}, nil
}

func (o *ScaleOperator) Compile(ctx *operators.CompileContext) (*operators.CompileResult, error) {
	converter := operators.NewTypeConverter()

	width, _ := converter.Convert(ctx.Params["width"], operators.TypeInt)
	height, _ := converter.Convert(ctx.Params["height"], operators.TypeInt)

	algorithm := "bicubic"
	if algo, ok := ctx.Params["algorithm"]; ok {
		algorithm = algo.(string)
	}

	// Map algorithm to FFmpeg flag
	algorithmFlag := map[string]string{
		"bilinear": "bilinear",
		"bicubic":  "bicubic",
		"lanczos":  "lanczos",
		"neighbor": "neighbor",
	}[algorithm]

	var inputLabel string
	for _, stream := range ctx.InputStreams {
		if stream.StreamType == "video" {
			inputLabel = stream.Label
			break
		}
	}
	if inputLabel == "" && len(ctx.InputStreams) > 0 {
		inputLabel = ctx.InputStreams[0].Label
	}
	if inputLabel == "" {
		return nil, fmt.Errorf("scale requires a video input stream")
	}

	filter := fmt.Sprintf("%sscale=%d:%d:flags=%s[v]",
		inputLabel, width.(int), height.(int), algorithmFlag)

	return &operators.CompileResult{
		FilterExpression: filter,
		OutputLabels:     []string{"[v]"},
	}, nil
}

func floatPtr(f float64) *float64 {
	return &f
}
