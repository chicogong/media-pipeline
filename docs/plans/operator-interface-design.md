# Operator Interface and Type System Design

**Date**: 2025-12-15
**Status**: Draft
**Related**: [Architecture Design](./2025-12-14-media-pipeline-architecture-design.md), [Schemas Design](./schemas-detailed-design.md), [Planner Design](./planner-detailed-design.md)

---

## Overview

The Operator system is the core extensibility mechanism of media-pipeline. It defines:
- **Operator Interface**: Contract that all operators must implement
- **Type System**: Parameter types, validation rules, and type conversion
- **Registration Mechanism**: How operators are registered and discovered
- **Parameter Validation**: Declarative validation with detailed error messages

---

## Operator Interface

### Core Interface

```go
package operators

type Operator interface {
    // Name returns the unique operator identifier (e.g., "trim", "loudnorm")
    Name() string

    // Category returns the operator category for documentation
    Category() Category

    // Describe returns human-readable description and parameter schema
    Describe() *OperatorDescriptor

    // ValidateParams validates operation parameters
    ValidateParams(params map[string]interface{}) error

    // ComputeOutputMetadata calculates output media properties based on inputs
    // Used during planning phase for metadata propagation
    ComputeOutputMetadata(params map[string]interface{}, inputs []*schemas.MediaInfo) (*schemas.MediaInfo, error)

    // EstimateResources estimates CPU time, memory, and disk usage
    EstimateResources(params map[string]interface{}, inputs []*schemas.MediaInfo) (*schemas.NodeEstimates, error)

    // Compile generates FFmpeg filter syntax or command arguments
    // Returns filtergraph fragment or complete command
    Compile(ctx *CompileContext) (*CompileResult, error)
}

type Category string

const (
    CategoryTimeline   Category = "timeline"    // trim, concat, split
    CategoryAudio      Category = "audio"       // loudnorm, mix, volume
    CategoryVideo      Category = "video"       // scale, crop, rotate
    CategoryGraphics   Category = "graphics"    // overlay, drawtext, subtitles
    CategoryOutput     Category = "output"      // export, thumbnail, waveform
    CategoryAdvanced   Category = "advanced"    // custom filters
)

type OperatorDescriptor struct {
    Name        string
    Category    Category
    Description string

    // Parameter schema (for validation and documentation)
    Parameters  []ParameterDescriptor

    // Input requirements
    MinInputs   int
    MaxInputs   int
    InputTypes  []MediaType  // e.g., [Video, Audio], [VideoAudio]

    // Output types
    OutputTypes []MediaType

    // Special requirements
    RequiresTwoPass bool  // e.g., loudnorm
    SupportsStreaming bool
}

type MediaType string

const (
    MediaTypeVideo      MediaType = "video"
    MediaTypeAudio      MediaType = "audio"
    MediaTypeVideoAudio MediaType = "video+audio"
    MediaTypeImage      MediaType = "image"
    MediaTypeAny        MediaType = "any"
)
```

### Compile Context and Result

```go
type CompileContext struct {
    // Inputs
    InputStreams  []StreamRef      // Input stream references
    Params        map[string]interface{}

    // Environment
    WorkDir       string
    TempDir       string

    // Metadata
    InputMetadata []*schemas.MediaInfo

    // Options
    Debug         bool
}

type StreamRef struct {
    SourceID      string  // Node ID that produces this stream
    StreamIndex   int     // FFmpeg stream index
    StreamType    string  // "video", "audio"
    Label         string  // FFmpeg label (e.g., "[v0]")
}

type CompileResult struct {
    // Filtergraph fragment (for filter-based operators)
    FilterExpression string

    // Or complete command (for operators requiring separate pass)
    Command          *Command

    // Output stream labels
    OutputLabels     []string

    // Temporary files generated
    TempFiles        []string

    // Dependencies (for multi-pass operations)
    DependsOn        []string
}

type Command struct {
    Stage       string    // "probe", "loudnorm_pass1", "main"
    Args        []string
    Stdin       string    // Optional stdin content
    WorkDir     string
}
```

---

## Type System

### Parameter Types

```go
type ParameterDescriptor struct {
    Name        string
    Type        ParameterType
    Required    bool
    Default     interface{}
    Description string

    // Validation rules
    Validation  *ValidationRules

    // Examples
    Examples    []interface{}
}

type ParameterType string

const (
    TypeString     ParameterType = "string"
    TypeInt        ParameterType = "int"
    TypeFloat      ParameterType = "float"
    TypeBool       ParameterType = "bool"
    TypeDuration   ParameterType = "duration"   // "1h30m", "00:05:30"
    TypeTimecode   ParameterType = "timecode"   // "00:05:30.500"
    TypeResolution ParameterType = "resolution" // "1920x1080"
    TypeEnum       ParameterType = "enum"       // One of predefined values
    TypeArray      ParameterType = "array"
    TypeObject     ParameterType = "object"
)

type ValidationRules struct {
    // Numeric constraints
    Min         *float64
    Max         *float64
    MultipleOf  *float64

    // String constraints
    MinLength   *int
    MaxLength   *int
    Pattern     *string  // Regex pattern

    // Enum values
    Enum        []interface{}

    // Array constraints
    MinItems    *int
    MaxItems    *int
    ItemType    ParameterType

    // Custom validator function
    CustomValidator func(interface{}) error
}
```

### Type Conversion

```go
package operators

type TypeConverter struct{}

func (tc *TypeConverter) Convert(value interface{}, targetType ParameterType) (interface{}, error) {
    switch targetType {
    case TypeDuration:
        return tc.toDuration(value)
    case TypeTimecode:
        return tc.toTimecode(value)
    case TypeResolution:
        return tc.toResolution(value)
    case TypeInt:
        return tc.toInt(value)
    case TypeFloat:
        return tc.toFloat(value)
    case TypeBool:
        return tc.toBool(value)
    default:
        return value, nil
    }
}

func (tc *TypeConverter) toDuration(value interface{}) (time.Duration, error) {
    switch v := value.(type) {
    case string:
        // Support multiple formats:
        // - "1h30m", "90s" (Go duration)
        // - "01:30:00" (timecode)
        // - "PT1H30M" (ISO 8601)
        return parseDuration(v)
    case float64:
        // Treat as seconds
        return time.Duration(v * float64(time.Second)), nil
    case int:
        return time.Duration(v) * time.Second, nil
    default:
        return 0, fmt.Errorf("cannot convert %T to duration", value)
    }
}

func (tc *TypeConverter) toResolution(value interface{}) (*Resolution, error) {
    switch v := value.(type) {
    case string:
        // Parse "1920x1080"
        parts := strings.Split(v, "x")
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid resolution format: %s", v)
        }
        width, _ := strconv.Atoi(parts[0])
        height, _ := strconv.Atoi(parts[1])
        return &Resolution{Width: width, Height: height}, nil
    case map[string]interface{}:
        // Parse {"width": 1920, "height": 1080}
        width := int(v["width"].(float64))
        height := int(v["height"].(float64))
        return &Resolution{Width: width, Height: height}, nil
    default:
        return nil, fmt.Errorf("cannot convert %T to resolution", value)
    }
}

type Resolution struct {
    Width  int
    Height int
}
```

### Parameter Validator

```go
type ParameterValidator struct {
    converter *TypeConverter
}

func (pv *ParameterValidator) ValidateParameter(
    name string,
    value interface{},
    descriptor *ParameterDescriptor,
) error {
    // Type conversion
    converted, err := pv.converter.Convert(value, descriptor.Type)
    if err != nil {
        return &ValidationError{
            Parameter: name,
            Message:   fmt.Sprintf("type conversion failed: %v", err),
        }
    }

    // Apply validation rules
    if descriptor.Validation != nil {
        if err := pv.applyRules(converted, descriptor.Validation); err != nil {
            return &ValidationError{
                Parameter: name,
                Message:   err.Error(),
            }
        }
    }

    return nil
}

func (pv *ParameterValidator) applyRules(value interface{}, rules *ValidationRules) error {
    // Numeric constraints
    if rules.Min != nil || rules.Max != nil {
        numValue, err := toFloat64(value)
        if err != nil {
            return err
        }
        if rules.Min != nil && numValue < *rules.Min {
            return fmt.Errorf("value %v is less than minimum %v", numValue, *rules.Min)
        }
        if rules.Max != nil && numValue > *rules.Max {
            return fmt.Errorf("value %v is greater than maximum %v", numValue, *rules.Max)
        }
    }

    // Enum constraint
    if rules.Enum != nil {
        found := false
        for _, enumValue := range rules.Enum {
            if reflect.DeepEqual(value, enumValue) {
                found = true
                break
            }
        }
        if !found {
            return fmt.Errorf("value %v is not in allowed values %v", value, rules.Enum)
        }
    }

    // Custom validator
    if rules.CustomValidator != nil {
        if err := rules.CustomValidator(value); err != nil {
            return err
        }
    }

    return nil
}

type ValidationError struct {
    Parameter string
    Message   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("parameter '%s': %s", e.Parameter, e.Message)
}
```

---

## Operator Registry

```go
package operators

type Registry struct {
    operators map[string]Operator
    mu        sync.RWMutex
}

var globalRegistry = &Registry{
    operators: make(map[string]Operator),
}

func Register(op Operator) {
    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()

    globalRegistry.operators[op.Name()] = op
}

func Get(name string) (Operator, error) {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    op, ok := globalRegistry.operators[name]
    if !ok {
        return nil, fmt.Errorf("operator '%s' not found", name)
    }
    return op, nil
}

func List() []Operator {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    result := make([]Operator, 0, len(globalRegistry.operators))
    for _, op := range globalRegistry.operators {
        result = append(result, op)
    }
    return result
}

func ListByCategory(category Category) []Operator {
    all := List()
    result := []Operator{}
    for _, op := range all {
        if op.Category() == category {
            result = append(result, op)
        }
    }
    return result
}
```

---

## Example Operator Implementations

### 1. Trim Operator

```go
package operators

type TrimOperator struct{}

func init() {
    Register(&TrimOperator{})
}

func (o *TrimOperator) Name() string {
    return "trim"
}

func (o *TrimOperator) Category() Category {
    return CategoryTimeline
}

func (o *TrimOperator) Describe() *OperatorDescriptor {
    return &OperatorDescriptor{
        Name:        "trim",
        Category:    CategoryTimeline,
        Description: "Trim video/audio to specified time range",
        Parameters: []ParameterDescriptor{
            {
                Name:        "start",
                Type:        TypeTimecode,
                Required:    false,
                Default:     "00:00:00",
                Description: "Start time",
                Examples:    []interface{}{"00:00:10", "10s", "00:00:10.500"},
            },
            {
                Name:        "duration",
                Type:        TypeDuration,
                Required:    false,
                Description: "Duration (if not specified, trim to end)",
                Examples:    []interface{}{"00:05:00", "5m", "300s"},
            },
            {
                Name:        "end",
                Type:        TypeTimecode,
                Required:    false,
                Description: "End time (alternative to duration)",
            },
        },
        MinInputs:   1,
        MaxInputs:   1,
        InputTypes:  []MediaType{MediaTypeVideoAudio, MediaTypeVideo, MediaTypeAudio},
        OutputTypes: []MediaType{MediaTypeVideoAudio},
    }
}

func (o *TrimOperator) ValidateParams(params map[string]interface{}) error {
    validator := &ParameterValidator{converter: &TypeConverter{}}
    descriptor := o.Describe()

    // Validate each parameter
    for _, paramDesc := range descriptor.Parameters {
        if value, ok := params[paramDesc.Name]; ok {
            if err := validator.ValidateParameter(paramDesc.Name, value, &paramDesc); err != nil {
                return err
            }
        } else if paramDesc.Required {
            return fmt.Errorf("required parameter '%s' is missing", paramDesc.Name)
        }
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

    input := inputs[0]
    output := *input // Copy

    // Update duration
    if duration, ok := params["duration"]; ok {
        d := duration.(time.Duration)
        output.Duration = d.Seconds()
    } else if end, ok := params["end"]; ok {
        start := params["start"].(time.Duration)
        endTime := end.(time.Duration)
        output.Duration = (endTime - start).Seconds()
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

    duration := inputs[0].Duration
    if d, ok := params["duration"]; ok {
        duration = d.(time.Duration).Seconds()
    }

    // Trim is fast (mostly copy operation if on keyframes)
    cpuTime := time.Duration(duration * 0.1) * time.Second

    return &schemas.NodeEstimates{
        CPUTime:     cpuTime,
        MemoryUsage: 100 * 1024 * 1024, // 100MB
        DiskUsage:   int64(inputs[0].Bitrate * int64(duration) / 8),
    }, nil
}

func (o *TrimOperator) Compile(ctx *CompileContext) (*CompileResult, error) {
    start := ctx.Params["start"].(time.Duration)

    var filterVideo, filterAudio string

    if duration, ok := ctx.Params["duration"]; ok {
        d := duration.(time.Duration)
        filterVideo = fmt.Sprintf("[%s]trim=start=%.3f:duration=%.3f[v]",
            ctx.InputStreams[0].Label, start.Seconds(), d.Seconds())
        filterAudio = fmt.Sprintf("[%s]atrim=start=%.3f:duration=%.3f[a]",
            ctx.InputStreams[0].Label, start.Seconds(), d.Seconds())
    } else {
        filterVideo = fmt.Sprintf("[%s]trim=start=%.3f[v]",
            ctx.InputStreams[0].Label, start.Seconds())
        filterAudio = fmt.Sprintf("[%s]atrim=start=%.3f[a]",
            ctx.InputStreams[0].Label, start.Seconds())
    }

    return &CompileResult{
        FilterExpression: filterVideo + ";" + filterAudio,
        OutputLabels:     []string{"[v]", "[a]"},
    }, nil
}
```

### 2. Loudnorm Operator (Two-Pass)

```go
package operators

type LoudnormOperator struct{}

func init() {
    Register(&LoudnormOperator{})
}

func (o *LoudnormOperator) Name() string {
    return "loudnorm"
}

func (o *LoudnormOperator) Category() Category {
    return CategoryAudio
}

func (o *LoudnormOperator) Describe() *OperatorDescriptor {
    return &OperatorDescriptor{
        Name:        "loudnorm",
        Category:    CategoryAudio,
        Description: "EBU R128 loudness normalization (two-pass)",
        Parameters: []ParameterDescriptor{
            {
                Name:        "target_lufs",
                Type:        TypeFloat,
                Required:    false,
                Default:     -16.0,
                Description: "Target integrated loudness (LUFS)",
                Validation: &ValidationRules{
                    Min: floatPtr(-70.0),
                    Max: floatPtr(-5.0),
                },
                Examples: []interface{}{-16.0, -23.0},
            },
            {
                Name:        "target_tp",
                Type:        TypeFloat,
                Required:    false,
                Default:     -1.5,
                Description: "Target true peak (dBTP)",
                Validation: &ValidationRules{
                    Min: floatPtr(-9.0),
                    Max: floatPtr(0.0),
                },
            },
            {
                Name:        "target_lra",
                Type:        TypeFloat,
                Required:    false,
                Default:     11.0,
                Description: "Target loudness range (LU)",
                Validation: &ValidationRules{
                    Min: floatPtr(1.0),
                    Max: floatPtr(50.0),
                },
            },
        },
        MinInputs:      1,
        MaxInputs:      1,
        InputTypes:     []MediaType{MediaTypeVideoAudio, MediaTypeAudio},
        OutputTypes:    []MediaType{MediaTypeVideoAudio},
        RequiresTwoPass: true,
    }
}

func (o *LoudnormOperator) ValidateParams(params map[string]interface{}) error {
    // Use standard validation
    return standardValidation(o, params)
}

func (o *LoudnormOperator) ComputeOutputMetadata(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.MediaInfo, error) {
    output := *inputs[0]
    // Audio properties remain same (sample rate, channels)
    // Only loudness changes (not reflected in MediaInfo)
    return &output, nil
}

func (o *LoudnormOperator) EstimateResources(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.NodeEstimates, error) {
    duration := inputs[0].Duration

    // Two-pass operation takes ~2x realtime
    cpuTime := time.Duration(duration * 2.0) * time.Second

    return &schemas.NodeEstimates{
        CPUTime:     cpuTime,
        MemoryUsage: 150 * 1024 * 1024, // 150MB
        DiskUsage:   int64(inputs[0].Bitrate * int64(duration) / 8),
    }, nil
}

func (o *LoudnormOperator) Compile(ctx *CompileContext) (*CompileResult, error) {
    targetLUFS := ctx.Params["target_lufs"].(float64)
    targetTP := ctx.Params["target_tp"].(float64)
    targetLRA := ctx.Params["target_lra"].(float64)

    input := ctx.InputStreams[0]

    // Pass 1: Measure loudness
    pass1Cmd := &Command{
        Stage: "loudnorm_pass1",
        Args: []string{
            "-i", input.SourceID,
            "-af", fmt.Sprintf("loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:print_format=json",
                targetLUFS, targetTP, targetLRA),
            "-f", "null",
            "-",
        },
    }

    // Pass 2: Apply normalization
    // (This will use measured_I, measured_LRA, measured_TP from pass 1)
    filter := fmt.Sprintf("[%s]loudnorm=I=%.1f:TP=%.1f:LRA=%.1f:measured_I=${measured_I}:measured_LRA=${measured_LRA}:measured_TP=${measured_TP}:offset=${offset}:linear=true[a]",
        input.Label, targetLUFS, targetTP, targetLRA)

    return &CompileResult{
        Command: pass1Cmd,
        FilterExpression: filter,
        OutputLabels: []string{"[a]"},
        DependsOn: []string{"loudnorm_pass1"},
    }, nil
}

func floatPtr(f float64) *float64 { return &f }
```

### 3. Scale Operator

```go
package operators

type ScaleOperator struct{}

func init() {
    Register(&ScaleOperator{})
}

func (o *ScaleOperator) Name() string {
    return "scale"
}

func (o *ScaleOperator) Category() Category {
    return CategoryVideo
}

func (o *ScaleOperator) Describe() *OperatorDescriptor {
    return &OperatorDescriptor{
        Name:        "scale",
        Category:    CategoryVideo,
        Description: "Scale video to specified resolution",
        Parameters: []ParameterDescriptor{
            {
                Name:        "width",
                Type:        TypeInt,
                Required:    true,
                Description: "Target width (or -1 to maintain aspect ratio)",
                Validation: &ValidationRules{
                    Min: floatPtr(-1),
                    Max: floatPtr(7680),
                },
            },
            {
                Name:        "height",
                Type:        TypeInt,
                Required:    true,
                Description: "Target height (or -1 to maintain aspect ratio)",
                Validation: &ValidationRules{
                    Min: floatPtr(-1),
                    Max: floatPtr(4320),
                },
            },
            {
                Name:        "algorithm",
                Type:        TypeEnum,
                Required:    false,
                Default:     "bicubic",
                Description: "Scaling algorithm",
                Validation: &ValidationRules{
                    Enum: []interface{}{"bilinear", "bicubic", "lanczos", "neighbor"},
                },
            },
        },
        MinInputs:   1,
        MaxInputs:   1,
        InputTypes:  []MediaType{MediaTypeVideo, MediaTypeVideoAudio},
        OutputTypes: []MediaType{MediaTypeVideo},
    }
}

func (o *ScaleOperator) ValidateParams(params map[string]interface{}) error {
    width := int(params["width"].(float64))
    height := int(params["height"].(float64))

    if width == -1 && height == -1 {
        return fmt.Errorf("both width and height cannot be -1")
    }

    return standardValidation(o, params)
}

func (o *ScaleOperator) ComputeOutputMetadata(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.MediaInfo, error) {
    output := *inputs[0]

    width := int(params["width"].(float64))
    height := int(params["height"].(float64))

    inputWidth := output.VideoStreams[0].Width
    inputHeight := output.VideoStreams[0].Height

    // Calculate actual dimensions
    if width == -1 {
        width = inputWidth * height / inputHeight
    } else if height == -1 {
        height = inputHeight * width / inputWidth
    }

    output.VideoStreams[0].Width = width
    output.VideoStreams[0].Height = height

    return &output, nil
}

func (o *ScaleOperator) EstimateResources(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.NodeEstimates, error) {
    duration := inputs[0].Duration

    // Scaling is moderately expensive
    cpuTime := time.Duration(duration * 0.5) * time.Second

    return &schemas.NodeEstimates{
        CPUTime:     cpuTime,
        MemoryUsage: 200 * 1024 * 1024, // 200MB
        DiskUsage:   int64(inputs[0].Bitrate * int64(duration) / 8),
    }, nil
}

func (o *ScaleOperator) Compile(ctx *CompileContext) (*CompileResult, error) {
    width := int(ctx.Params["width"].(float64))
    height := int(ctx.Params["height"].(float64))
    algorithm := ctx.Params["algorithm"].(string)

    // Map algorithm to FFmpeg flag
    algorithmFlag := map[string]string{
        "bilinear": "bilinear",
        "bicubic":  "bicubic",
        "lanczos":  "lanczos",
        "neighbor": "neighbor",
    }[algorithm]

    filter := fmt.Sprintf("[%s]scale=%d:%d:flags=%s[v]",
        ctx.InputStreams[0].Label, width, height, algorithmFlag)

    return &CompileResult{
        FilterExpression: filter,
        OutputLabels:     []string{"[v]"},
    }, nil
}
```

### 4. Mix Operator (Multi-Input)

```go
package operators

type MixOperator struct{}

func init() {
    Register(&MixOperator{})
}

func (o *MixOperator) Name() string {
    return "mix"
}

func (o *MixOperator) Category() Category {
    return CategoryAudio
}

func (o *MixOperator) Describe() *OperatorDescriptor {
    return &OperatorDescriptor{
        Name:        "mix",
        Category:    CategoryAudio,
        Description: "Mix multiple audio streams with optional ducking",
        Parameters: []ParameterDescriptor{
            {
                Name:        "mode",
                Type:        TypeEnum,
                Required:    false,
                Default:     "simple",
                Description: "Mixing mode",
                Validation: &ValidationRules{
                    Enum: []interface{}{"simple", "ducking"},
                },
            },
            {
                Name:        "weights",
                Type:        TypeArray,
                Required:    false,
                Default:     []float64{1.0, 1.0},
                Description: "Volume weights for each input (0.0-1.0)",
            },
            {
                Name:        "ducking_threshold",
                Type:        TypeFloat,
                Required:    false,
                Default:     -20.0,
                Description: "Threshold for ducking (dB)",
                Validation: &ValidationRules{
                    Min: floatPtr(-60.0),
                    Max: floatPtr(0.0),
                },
            },
        },
        MinInputs:   2,
        MaxInputs:   10,
        InputTypes:  []MediaType{MediaTypeAudio},
        OutputTypes: []MediaType{MediaTypeAudio},
    }
}

func (o *MixOperator) ValidateParams(params map[string]interface{}) error {
    return standardValidation(o, params)
}

func (o *MixOperator) ComputeOutputMetadata(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.MediaInfo, error) {
    // Output inherits properties from first input
    output := *inputs[0]

    // Duration is max of all inputs
    maxDuration := 0.0
    for _, input := range inputs {
        if input.Duration > maxDuration {
            maxDuration = input.Duration
        }
    }
    output.Duration = maxDuration

    return &output, nil
}

func (o *MixOperator) EstimateResources(
    params map[string]interface{},
    inputs []*schemas.MediaInfo,
) (*schemas.NodeEstimates, error) {
    maxDuration := 0.0
    for _, input := range inputs {
        if input.Duration > maxDuration {
            maxDuration = input.Duration
        }
    }

    cpuTime := time.Duration(maxDuration * 0.3) * time.Second

    return &schemas.NodeEstimates{
        CPUTime:     cpuTime,
        MemoryUsage: 100 * 1024 * 1024,
        DiskUsage:   int64(inputs[0].Bitrate * int64(maxDuration) / 8),
    }, nil
}

func (o *MixOperator) Compile(ctx *CompileContext) (*CompileResult, error) {
    mode := ctx.Params["mode"].(string)
    weights := ctx.Params["weights"].([]interface{})

    inputCount := len(ctx.InputStreams)

    var filter string
    if mode == "simple" {
        // Simple mixing
        inputs := make([]string, inputCount)
        for i, stream := range ctx.InputStreams {
            inputs[i] = stream.Label
        }

        weightStr := ""
        for i, w := range weights {
            if i > 0 {
                weightStr += " "
            }
            weightStr += fmt.Sprintf("%.2f", w.(float64))
        }

        filter = fmt.Sprintf("%s amix=inputs=%d:weights=%s[a]",
            strings.Join(inputs, ""), inputCount, weightStr)
    } else {
        // Ducking mode (first input is main, rest are ducked)
        threshold := ctx.Params["ducking_threshold"].(float64)

        filter = fmt.Sprintf("[%s][%s]sidechaincompress=threshold=%.1f:ratio=4:attack=20:release=200[a]",
            ctx.InputStreams[0].Label, ctx.InputStreams[1].Label, threshold)
    }

    return &CompileResult{
        FilterExpression: filter,
        OutputLabels:     []string{"[a]"},
    }, nil
}
```

---

## Helper Functions

```go
func standardValidation(op Operator, params map[string]interface{}) error {
    validator := &ParameterValidator{converter: &TypeConverter{}}
    descriptor := op.Describe()

    for _, paramDesc := range descriptor.Parameters {
        if value, ok := params[paramDesc.Name]; ok {
            if err := validator.ValidateParameter(paramDesc.Name, value, &paramDesc); err != nil {
                return err
            }
        } else if paramDesc.Required {
            return fmt.Errorf("required parameter '%s' is missing", paramDesc.Name)
        }
    }

    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestTrimOperatorValidation(t *testing.T) {
    op := &TrimOperator{}

    tests := []struct {
        params  map[string]interface{}
        wantErr bool
    }{
        {
            params: map[string]interface{}{
                "start":    "00:00:10",
                "duration": "00:05:00",
            },
            wantErr: false,
        },
        {
            params: map[string]interface{}{
                "start":    "00:00:10",
                "duration": "5m",
                "end":      "00:05:10", // Error: both duration and end
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        err := op.ValidateParams(tt.params)
        if (err != nil) != tt.wantErr {
            t.Errorf("ValidateParams() error = %v, wantErr %v", err, tt.wantErr)
        }
    }
}
```

---

## Documentation Generation

Auto-generate operator documentation from descriptors:

```go
func GenerateOperatorDocs() string {
    operators := List()

    var sb strings.Builder
    sb.WriteString("# Operators Reference\n\n")

    categories := map[Category][]Operator{}
    for _, op := range operators {
        cat := op.Category()
        categories[cat] = append(categories[cat], op)
    }

    for category, ops := range categories {
        sb.WriteString(fmt.Sprintf("## %s\n\n", category))

        for _, op := range ops {
            desc := op.Describe()
            sb.WriteString(fmt.Sprintf("### %s\n\n", desc.Name))
            sb.WriteString(fmt.Sprintf("%s\n\n", desc.Description))

            sb.WriteString("**Parameters:**\n\n")
            for _, param := range desc.Parameters {
                required := ""
                if param.Required {
                    required = " (required)"
                }
                sb.WriteString(fmt.Sprintf("- `%s` (%s)%s: %s\n",
                    param.Name, param.Type, required, param.Description))
            }
            sb.WriteString("\n")
        }
    }

    return sb.String()
}
```

---

## Summary

The Operator system provides:

1. **Extensibility**: Add new operators by implementing a simple interface
2. **Type Safety**: Strong type system with automatic conversion and validation
3. **Self-Documentation**: Operators describe their own parameters and behavior
4. **Validation**: Declarative validation rules with detailed error messages
5. **Flexibility**: Support for single-pass and multi-pass operations
6. **Compilation**: Generate FFmpeg filtergraphs or complete commands

Adding a new operator requires:
1. Implement the `Operator` interface
2. Define parameter schema in `Describe()`
3. Implement validation, metadata computation, and compilation
4. Register with `Register()` in `init()`

---

**Status**: Ready for implementation
