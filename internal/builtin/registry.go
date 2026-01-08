// Package builtin provides built-in functions for the SLOP language.
package builtin

import (
	"fmt"
	"strings"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// BuiltinFunc is the signature for built-in functions.
type BuiltinFunc func(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error)

// Registry holds all registered built-in functions.
type Registry struct {
	functions map[string]BuiltinFunc
}

// NewRegistry creates a new registry with all built-in functions registered.
func NewRegistry() *Registry {
	r := &Registry{
		functions: make(map[string]BuiltinFunc),
	}

	// Register all built-in functions
	r.registerCoreFunctions()
	r.registerMathFunctions()
	r.registerStringFunctions()
	r.registerListFunctions()
	r.registerMapFunctions()
	r.registerSetFunctions()
	r.registerPipelineFunctions()
	r.registerUtilityFunctions()
	r.registerControlFunctions()
	r.registerGeneratorFunctions()

	return r
}

// Get returns a built-in function by name.
func (r *Registry) Get(name string) (BuiltinFunc, bool) {
	fn, ok := r.functions[name]
	return fn, ok
}

// GetAsValue returns a built-in function as an evaluator.Value.
func (r *Registry) GetAsValue(name string) (evaluator.Value, bool) {
	fn, ok := r.functions[name]
	if !ok {
		return nil, false
	}
	return &evaluator.BuiltinValue{
		Name: name,
		Fn:   evaluator.BuiltinFunction(fn),
	}, true
}

// Register adds a built-in function to the registry.
func (r *Registry) Register(name string, fn BuiltinFunc) {
	r.functions[name] = fn
}

// RegisterConstant registers a constant value.
// When retrieved with GetAsValue, it returns the value directly instead of a BuiltinValue.
func (r *Registry) RegisterConstant(name string, value evaluator.Value) {
	// Store as a special marker function
	r.functions[name+"__const__"] = func(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
		return value, nil
	}
}

// GetConstant retrieves a constant value by name.
func (r *Registry) GetConstant(name string) (evaluator.Value, bool) {
	fn, ok := r.functions[name+"__const__"]
	if !ok {
		return nil, false
	}
	val, _ := fn(nil, nil)
	return val, true
}

// Names returns all registered function names (including constants).
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		// Skip internal constant markers
		if !strings.HasSuffix(name, "__const__") {
			names = append(names, name)
		}
	}
	// Add constant names
	for name := range r.functions {
		if strings.HasSuffix(name, "__const__") {
			constName := strings.TrimSuffix(name, "__const__")
			names = append(names, constName)
		}
	}
	return names
}

// Helper functions for argument validation

func requireArgs(name string, args []evaluator.Value, count int) error {
	if len(args) != count {
		return fmt.Errorf("%s() requires exactly %d argument(s), got %d", name, count, len(args))
	}
	return nil
}

func requireMinArgs(name string, args []evaluator.Value, min int) error {
	if len(args) < min {
		return fmt.Errorf("%s() requires at least %d argument(s), got %d", name, min, len(args))
	}
	return nil
}

func requireRangeArgs(name string, args []evaluator.Value, min, max int) error {
	if len(args) < min || len(args) > max {
		return fmt.Errorf("%s() requires %d-%d argument(s), got %d", name, min, max, len(args))
	}
	return nil
}

func requireInt(name string, val evaluator.Value) (int64, error) {
	if iv, ok := val.(*evaluator.IntValue); ok {
		return iv.Value, nil
	}
	return 0, fmt.Errorf("%s() requires int argument, got %s", name, val.Type())
}

func requireFloat(name string, val evaluator.Value) (float64, error) {
	switch v := val.(type) {
	case *evaluator.IntValue:
		return float64(v.Value), nil
	case *evaluator.FloatValue:
		return v.Value, nil
	}
	return 0, fmt.Errorf("%s() requires numeric argument, got %s", name, val.Type())
}

func requireString(name string, val evaluator.Value) (string, error) {
	if sv, ok := val.(*evaluator.StringValue); ok {
		return sv.Value, nil
	}
	return "", fmt.Errorf("%s() requires string argument, got %s", name, val.Type())
}

func requireList(name string, val evaluator.Value) ([]evaluator.Value, error) {
	if lv, ok := val.(*evaluator.ListValue); ok {
		return lv.Elements, nil
	}
	return nil, fmt.Errorf("%s() requires list argument, got %s", name, val.Type())
}

func requireMap(name string, val evaluator.Value) (map[string]evaluator.Value, error) {
	if mv, ok := val.(*evaluator.MapValue); ok {
		return mv.Pairs, nil
	}
	return nil, fmt.Errorf("%s() requires map argument, got %s", name, val.Type())
}

func requireSet(name string, val evaluator.Value) (map[string]evaluator.Value, error) {
	if sv, ok := val.(*evaluator.SetValue); ok {
		return sv.Elements, nil
	}
	return nil, fmt.Errorf("%s() requires set argument, got %s", name, val.Type())
}

func requireCallable(name string, val evaluator.Value) error {
	if !evaluator.IsCallable(val) {
		return fmt.Errorf("%s() requires callable argument, got %s", name, val.Type())
	}
	return nil
}

func toFloat(val evaluator.Value) (float64, bool) {
	switch v := val.(type) {
	case *evaluator.IntValue:
		return float64(v.Value), true
	case *evaluator.FloatValue:
		return v.Value, true
	}
	return 0, false
}

func toInt(val evaluator.Value) (int64, bool) {
	switch v := val.(type) {
	case *evaluator.IntValue:
		return v.Value, true
	case *evaluator.FloatValue:
		return int64(v.Value), true
	}
	return 0, false
}
