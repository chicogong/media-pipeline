package operators

import (
	"testing"
)

func TestGlobalRegistry_Singleton(t *testing.T) {
	r1 := GlobalRegistry()
	r2 := GlobalRegistry()

	if r1 == nil {
		t.Fatal("GlobalRegistry() returned nil")
	}
	if r1 != r2 {
		t.Errorf("GlobalRegistry() returned different instances on repeated calls")
	}
}

func TestGlobalRegistry_RegisterAndGet(t *testing.T) {
	GlobalRegistry().Reset()
	t.Cleanup(func() { GlobalRegistry().Reset() })

	Register(testOperator{})

	op, err := Get("test")
	if err != nil {
		t.Fatalf("Get(\"test\") failed: %v", err)
	}
	if op == nil {
		t.Fatal("Get(\"test\") returned nil operator")
	}

	_, err = Get("missing")
	if err == nil {
		t.Fatal("Get(\"missing\") expected error, got nil")
	}

	if got := len(List()); got != 1 {
		t.Fatalf("List() len mismatch: got=%d want=1", got)
	}

	if got := len(ListByCategory(CategoryVideo)); got != 1 {
		t.Fatalf("ListByCategory(CategoryVideo) len mismatch: got=%d want=1", got)
	}

	if got := len(ListByCategory(CategoryAudio)); got != 0 {
		t.Fatalf("ListByCategory(CategoryAudio) expected 0, got=%d", got)
	}
}

func TestRegistry_ReRegister(t *testing.T) {
	r := &Registry{operators: make(map[string]Operator)}

	r.Register(testOperator{})
	r.Register(testOperator{})

	if got := len(r.List()); got != 1 {
		t.Fatalf("re-registering same name should keep count at 1, got=%d", got)
	}
}
