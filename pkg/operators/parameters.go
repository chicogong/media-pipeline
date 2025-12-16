package operators

// ParameterDescriptor describes an operator parameter
type ParameterDescriptor struct {
	Name        string
	Type        ParameterType
	Required    bool
	Default     interface{}
	Description string

	// Validation rules
	Validation *ValidationRules

	// Examples
	Examples []interface{}
}

// ParameterType represents parameter type
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

// ValidationRules defines parameter validation rules
type ValidationRules struct {
	// Numeric constraints
	Min        *float64
	Max        *float64
	MultipleOf *float64

	// String constraints
	MinLength *int
	MaxLength *int
	Pattern   *string

	// Enum values
	Enum []interface{}

	// Array constraints
	MinItems *int
	MaxItems *int
	ItemType ParameterType

	// Custom validator
	CustomValidator func(interface{}) error
}

// Resolution represents video resolution
type Resolution struct {
	Width  int
	Height int
}
