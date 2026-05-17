package operators

import (
	"errors"
	"testing"
	"time"
)

func TestValidationError_Error(t *testing.T) {
	e := &ValidationError{Parameter: "width", Message: "must be positive"}
	want := "parameter 'width': must be positive"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestToFloat64(t *testing.T) {
	cases := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", float64(3.14), 3.14, false},
		{"float32", float32(1.5), float64(float32(1.5)), false},
		{"int", int(7), 7.0, false},
		{"int32", int32(8), 8.0, false},
		{"int64", int64(9), 9.0, false},
		{"duration_second", time.Second, float64(time.Second), false},
		{"string_unsupported", "42", 0, true},
		{"bool_unsupported", true, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := toFloat64(c.input)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestParameterValidator_ValidateParameter(t *testing.T) {
	min := 1.0
	max := 10.0
	descriptor := &ParameterDescriptor{
		Name: "width",
		Type: TypeInt,
		Validation: &ValidationRules{
			Min: &min,
			Max: &max,
		},
	}

	pv := NewParameterValidator()

	t.Run("valid_value", func(t *testing.T) {
		err := pv.ValidateParameter("width", 5, descriptor)
		if err != nil {
			t.Fatalf("expected no error for value 5, got: %v", err)
		}
	})

	t.Run("below_min", func(t *testing.T) {
		err := pv.ValidateParameter("width", 0, descriptor)
		if err == nil {
			t.Fatal("expected error for value 0 (below min 1), got nil")
		}
	})

	t.Run("above_max", func(t *testing.T) {
		err := pv.ValidateParameter("width", 99, descriptor)
		if err == nil {
			t.Fatal("expected error for value 99 (above max 10), got nil")
		}
	})

	t.Run("unconvertible_type_is_validation_error", func(t *testing.T) {
		err := pv.ValidateParameter("width", []int{1}, descriptor)
		if err == nil {
			t.Fatal("expected error for []int input, got nil")
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("expected *ValidationError, got %T: %v", err, err)
		}
	})
}
