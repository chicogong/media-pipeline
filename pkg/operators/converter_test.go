package operators

import (
	"testing"
	"time"
)

func TestConverter_Duration(t *testing.T) {
	tc := NewTypeConverter()

	cases := []struct {
		name    string
		input   interface{}
		want    time.Duration
		wantErr bool
	}{
		{"go_duration_string_1h30m", "1h30m", 90 * time.Minute, false},
		{"timecode_string_1.5s", "00:00:01.5", 1500 * time.Millisecond, false},
		{"timecode_string_90s", "00:01:30", 90 * time.Second, false},
		{"float64_seconds", float64(2.5), time.Duration(2.5 * float64(time.Second)), false},
		{"int_seconds", int(5), 5 * time.Second, false},
		{"passthrough_duration", 3 * time.Second, 3 * time.Second, false},
		{"invalid_string", "nonsense", 0, true},
		{"unsupported_type", []int{1}, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tc.Convert(c.input, TypeDuration)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			d, ok := got.(time.Duration)
			if !ok {
				t.Fatalf("result is not time.Duration: %T", got)
			}
			if d != c.want {
				t.Errorf("got %v, want %v", d, c.want)
			}
		})
	}
}

func TestConverter_Timecode(t *testing.T) {
	tc := NewTypeConverter()

	got, err := tc.Convert("00:00:02.500", TypeTimecode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := got.(time.Duration)
	if !ok {
		t.Fatalf("result is not time.Duration: %T", got)
	}
	if d != 2500*time.Millisecond {
		t.Errorf("got %v, want %v", d, 2500*time.Millisecond)
	}
}

func TestConverter_Resolution(t *testing.T) {
	tc := NewTypeConverter()

	t.Run("string_1280x720", func(t *testing.T) {
		got, err := tc.Convert("1280x720", TypeResolution)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := got.(*Resolution)
		if r.Width != 1280 || r.Height != 720 {
			t.Errorf("got %+v, want 1280x720", r)
		}
	})

	t.Run("map_640x480", func(t *testing.T) {
		got, err := tc.Convert(map[string]interface{}{"width": float64(640), "height": float64(480)}, TypeResolution)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		r := got.(*Resolution)
		if r.Width != 640 || r.Height != 480 {
			t.Errorf("got %+v, want 640x480", r)
		}
	})

	t.Run("passthrough_resolution", func(t *testing.T) {
		orig := &Resolution{Width: 1920, Height: 1080}
		got, err := tc.Convert(orig, TypeResolution)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != orig {
			t.Errorf("expected same pointer, got different value")
		}
	})

	invalidCases := []struct {
		name  string
		input interface{}
	}{
		{"missing_height", "1920"},
		{"non_numeric", "axb"},
		{"too_many_parts", "12x34x56"},
		{"int_type", 1920},
	}
	for _, c := range invalidCases {
		t.Run("invalid_"+c.name, func(t *testing.T) {
			_, err := tc.Convert(c.input, TypeResolution)
			if err == nil {
				t.Fatalf("expected error for input %v, got nil", c.input)
			}
		})
	}
}

func TestConverter_Int(t *testing.T) {
	tc := NewTypeConverter()

	cases := []struct {
		name    string
		input   interface{}
		want    int
		wantErr bool
	}{
		{"int", int(7), 7, false},
		{"int32", int32(8), 8, false},
		{"int64", int64(9), 9, false},
		{"float64_truncate", float64(3.9), 3, false},
		{"string_42", "42", 42, false},
		{"invalid_string", "abc", 0, true},
		{"bool_unsupported", true, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tc.Convert(c.input, TypeInt)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.(int) != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestConverter_Float(t *testing.T) {
	tc := NewTypeConverter()

	cases := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", float64(1.5), 1.5, false},
		{"float32", float32(2.5), 2.5, false},
		{"int", int(3), 3.0, false},
		{"int32", int32(4), 4.0, false},
		{"int64", int64(5), 5.0, false},
		{"string_4.5", "4.5", 4.5, false},
		{"invalid_string", "xyz", 0, true},
		{"bool_unsupported", false, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tc.Convert(c.input, TypeFloat)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			f, ok := got.(float64)
			if !ok {
				t.Fatalf("result is not float64: %T", got)
			}
			if f != c.want {
				t.Errorf("got %v, want %v", f, c.want)
			}
		})
	}
}

func TestConverter_Bool(t *testing.T) {
	tc := NewTypeConverter()

	cases := []struct {
		name    string
		input   interface{}
		want    bool
		wantErr bool
	}{
		{"bool_true", true, true, false},
		{"bool_false", false, false, false},
		{"string_true", "true", true, false},
		{"string_0", "0", false, false},
		{"int_1_true", int(1), true, false},
		{"int_0_false", int(0), false, false},
		{"float64_2_true", float64(2.0), true, false},
		{"invalid_string", "maybe", false, true},
		{"unsupported_type", []int{}, false, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tc.Convert(c.input, TypeBool)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := got.(bool)
			if !ok {
				t.Fatalf("result is not bool: %T", got)
			}
			if b != c.want {
				t.Errorf("got %v, want %v", b, c.want)
			}
		})
	}
}

func TestConverter_String(t *testing.T) {
	tc := NewTypeConverter()

	cases := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string_passthrough", "hello", "hello"},
		{"int_42", 42, "42"},
		{"bool_true", true, "true"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := tc.Convert(c.input, TypeString)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			s, ok := got.(string)
			if !ok {
				t.Fatalf("result is not string: %T", got)
			}
			if s != c.want {
				t.Errorf("got %q, want %q", s, c.want)
			}
		})
	}
}

func TestConverter_DefaultPassthrough(t *testing.T) {
	tc := NewTypeConverter()

	val := []string{"a", "b"}
	got, err := tc.Convert(val, TypeArray)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSlice, ok := got.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", got)
	}
	if len(gotSlice) != 2 || gotSlice[0] != "a" || gotSlice[1] != "b" {
		t.Errorf("passthrough value mismatch: got %v", gotSlice)
	}
}
