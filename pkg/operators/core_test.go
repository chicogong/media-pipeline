package operators

import (
	"testing"
	"time"

	"github.com/chicogong/media-pipeline/pkg/schemas"
)

type testOperator struct{}

func (testOperator) Name() string     { return "test" }
func (testOperator) Category() Category { return CategoryVideo }

func (testOperator) Describe() *OperatorDescriptor {
	min := 0.0
	max := 10.0

	return &OperatorDescriptor{
		Name:        "test",
		Category:    CategoryVideo,
		Description: "test operator",
		Parameters: []ParameterDescriptor{
			{
				Name:        "width",
				Type:        TypeInt,
				Required:    true,
				Description: "required int with range",
				Validation: &ValidationRules{
					Min: &min,
					Max: &max,
				},
			},
			{
				Name:        "mode",
				Type:        TypeEnum,
				Required:    false,
				Description: "optional enum",
				Validation: &ValidationRules{
					Enum: []interface{}{"fast", "slow"},
				},
			},
		},
		MinInputs:   1,
		MaxInputs:   1,
		InputTypes:  []MediaType{MediaTypeAny},
		OutputTypes: []MediaType{MediaTypeAny},
	}
}

func (testOperator) ValidateParams(params map[string]interface{}) error { return StandardValidation(testOperator{}, params) }
func (testOperator) ComputeOutputMetadata(map[string]interface{}, []*schemas.MediaInfo) (*schemas.MediaInfo, error) {
	return nil, nil
}
func (testOperator) EstimateResources(map[string]interface{}, []*schemas.MediaInfo) (*schemas.NodeEstimates, error) {
	return nil, nil
}
func (testOperator) Compile(*CompileContext) (*CompileResult, error) { return &CompileResult{}, nil }

func TestTypeConverter(t *testing.T) {
	converter := NewTypeConverter()

	gotDuration, err := converter.Convert("00:00:01.5", TypeDuration)
	if err != nil {
		t.Fatalf("duration convert failed: %v", err)
	}
	if gotDuration.(time.Duration) != 1500*time.Millisecond {
		t.Fatalf("duration mismatch: got=%v want=%v", gotDuration, 1500*time.Millisecond)
	}

	gotResolution, err := converter.Convert("1920x1080", TypeResolution)
	if err != nil {
		t.Fatalf("resolution convert failed: %v", err)
	}
	if gotResolution.(*Resolution).Width != 1920 || gotResolution.(*Resolution).Height != 1080 {
		t.Fatalf("resolution mismatch: got=%+v want=1920x1080", gotResolution)
	}
}

func TestStandardValidation(t *testing.T) {
	op := testOperator{}

	if err := StandardValidation(op, map[string]interface{}{}); err == nil {
		t.Fatal("expected error for missing required parameter, got nil")
	}

	if err := StandardValidation(op, map[string]interface{}{"width": "5"}); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if err := StandardValidation(op, map[string]interface{}{"width": 11}); err == nil {
		t.Fatal("expected error for max constraint, got nil")
	}

	if err := StandardValidation(op, map[string]interface{}{"width": 5, "mode": "nope"}); err == nil {
		t.Fatal("expected error for enum constraint, got nil")
	}
}

func TestRegistry(t *testing.T) {
	r := &Registry{operators: make(map[string]Operator)}

	r.Register(testOperator{})

	if _, err := r.Get("test"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if _, err := r.Get("missing"); err == nil {
		t.Fatal("expected error for missing operator, got nil")
	}

	if got := len(r.List()); got != 1 {
		t.Fatalf("List mismatch: got=%d want=1", got)
	}
	if got := len(r.ListByCategory(CategoryVideo)); got != 1 {
		t.Fatalf("ListByCategory mismatch: got=%d want=1", got)
	}

	r.Reset()
	if got := len(r.List()); got != 0 {
		t.Fatalf("expected empty registry after reset, got=%d", got)
	}
}

