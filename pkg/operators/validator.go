package operators

import (
	"fmt"
	"reflect"
	"time"
)

// ParameterValidator validates operator parameters
type ParameterValidator struct {
	converter *TypeConverter
}

// NewParameterValidator creates a new parameter validator
func NewParameterValidator() *ParameterValidator {
	return &ParameterValidator{
		converter: NewTypeConverter(),
	}
}

// ValidateParameter validates a single parameter
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

// applyRules applies validation rules to a value
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

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Message   string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("parameter '%s': %s", e.Parameter, e.Message)
}

// toFloat64 converts a value to float64
func toFloat64(value interface{}) (float64, error) {
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
	case time.Duration:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// StandardValidation performs standard validation for an operator
func StandardValidation(op Operator, params map[string]interface{}) error {
	validator := NewParameterValidator()
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
