package operators

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

// TypeConverter converts values to specific parameter types
type TypeConverter struct{}

// NewTypeConverter creates a new type converter
func NewTypeConverter() *TypeConverter {
	return &TypeConverter{}
}

// Convert converts a value to the target type
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
	case TypeString:
		return tc.toString(value)
	default:
		return value, nil
	}
}

// toDuration converts to time.Duration
func (tc *TypeConverter) toDuration(value interface{}) (time.Duration, error) {
	switch v := value.(type) {
	case string:
		return schemas.ParseDuration(v)
	case float64:
		return time.Duration(v * float64(time.Second)), nil
	case int:
		return time.Duration(v) * time.Second, nil
	case time.Duration:
		return v, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to duration", value)
	}
}

// toTimecode converts to time.Duration (timecode format)
func (tc *TypeConverter) toTimecode(value interface{}) (time.Duration, error) {
	return tc.toDuration(value)
}

// toResolution converts to Resolution
func (tc *TypeConverter) toResolution(value interface{}) (*Resolution, error) {
	switch v := value.(type) {
	case string:
		// Parse "1920x1080"
		parts := strings.Split(v, "x")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid resolution format: %s", v)
		}
		width, err1 := strconv.Atoi(parts[0])
		height, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid resolution format: %s", v)
		}
		return &Resolution{Width: width, Height: height}, nil
	case map[string]interface{}:
		// Parse {"width": 1920, "height": 1080}
		width, _ := v["width"].(float64)
		height, _ := v["height"].(float64)
		return &Resolution{Width: int(width), Height: int(height)}, nil
	case *Resolution:
		return v, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to resolution", value)
	}
}

// toInt converts to int
func (tc *TypeConverter) toInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// toFloat converts to float64
func (tc *TypeConverter) toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// toBool converts to bool
func (tc *TypeConverter) toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// toString converts to string
func (tc *TypeConverter) toString(value interface{}) (string, error) {
	return fmt.Sprintf("%v", value), nil
}
