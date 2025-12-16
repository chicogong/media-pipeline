package operators

import (
	"fmt"
	"sync"
)

// Registry stores registered operators
type Registry struct {
	operators map[string]Operator
	mu        sync.RWMutex
}

// globalRegistry is the global operator registry
var globalRegistry = &Registry{
	operators: make(map[string]Operator),
}

// GlobalRegistry returns the global operator registry
func GlobalRegistry() *Registry {
	return globalRegistry
}

// Register registers an operator globally
func Register(op Operator) {
	globalRegistry.Register(op)
}

// Get retrieves an operator by name
func Get(name string) (Operator, error) {
	return globalRegistry.Get(name)
}

// List returns all registered operators
func List() []Operator {
	return globalRegistry.List()
}

// ListByCategory returns operators in a specific category
func ListByCategory(category Category) []Operator {
	return globalRegistry.ListByCategory(category)
}

// Register registers an operator in this registry
func (r *Registry) Register(op Operator) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := op.Name()
	// Allow re-registration (useful for testing)
	r.operators[name] = op
}

// Reset clears all registered operators (for testing)
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operators = make(map[string]Operator)
}

// Get retrieves an operator by name
func (r *Registry) Get(name string) (Operator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	op, ok := r.operators[name]
	if !ok {
		return nil, fmt.Errorf("operator '%s' not found", name)
	}

	return op, nil
}

// List returns all registered operators
func (r *Registry) List() []Operator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Operator, 0, len(r.operators))
	for _, op := range r.operators {
		result = append(result, op)
	}

	return result
}

// ListByCategory returns operators in a specific category
func (r *Registry) ListByCategory(category Category) []Operator {
	all := r.List()
	result := []Operator{}

	for _, op := range all {
		if op.Category() == category {
			result = append(result, op)
		}
	}

	return result
}
