// Package slop provides the public API for the SLOP language runtime.
//
// This file exports the core types needed for external service integration.
// External packages can implement the Service interface to create custom
// services accessible from SLOP scripts.
package slop

import (
	"fmt"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// Service is the interface for services that can be called from SLOP scripts.
// Implement this interface to create custom services accessible as:
//
//	service_name.method_name(args...)
//
// Example implementation:
//
//	type MyService struct{}
//
//	func (s *MyService) Call(method string, args []slop.Value, kwargs map[string]slop.Value) (slop.Value, error) {
//	    switch method {
//	    case "greet":
//	        name := "World"
//	        if n, ok := kwargs["name"]; ok {
//	            if sv, ok := n.(*slop.StringValue); ok {
//	                name = sv.Value
//	            }
//	        }
//	        return &slop.StringValue{Value: "Hello, " + name + "!"}, nil
//	    default:
//	        return nil, fmt.Errorf("unknown method: %s", method)
//	    }
//	}
type Service interface {
	// Call invokes a method on the service.
	// method is the method name being called.
	// args are positional arguments (in order).
	// kwargs are keyword arguments (by name).
	Call(method string, args []Value, kwargs map[string]Value) (Value, error)
}

// Value represents a runtime value in SLOP.
// All SLOP values implement this interface.
type Value = evaluator.Value

// StringValue represents a string value.
type StringValue = evaluator.StringValue

// NumberValue represents a numeric value (float64).
// Note: SLOP uses float64 internally for all numbers.
type NumberValue = evaluator.FloatValue

// IntValue represents an integer value.
type IntValue = evaluator.IntValue

// BoolValue represents a boolean value.
type BoolValue = evaluator.BoolValue

// ListValue represents a list/array of values.
type ListValue = evaluator.ListValue

// MapValue represents a map/dictionary of values.
type MapValue = evaluator.MapValue

// NullValue represents a null/none value.
type NullValue = evaluator.NoneValue

// ErrorValue represents an error value.
type ErrorValue = evaluator.SlopError

// NewStringValue creates a new string value.
func NewStringValue(s string) *StringValue {
	return &evaluator.StringValue{Value: s}
}

// NewNumberValue creates a new number value.
func NewNumberValue(n float64) *NumberValue {
	return &evaluator.FloatValue{Value: n}
}

// NewIntValue creates a new integer value.
func NewIntValue(n int64) *IntValue {
	return &evaluator.IntValue{Value: n}
}

// NewBoolValue creates a new boolean value.
func NewBoolValue(b bool) *BoolValue {
	return evaluator.NewBool(b)
}

// NewListValue creates a new list value from elements.
func NewListValue(elements []Value) *ListValue {
	return &evaluator.ListValue{Elements: elements}
}

// NewMapValue creates a new empty map value.
func NewMapValue() *MapValue {
	return evaluator.NewMapValue()
}

// NewNullValue creates a new null value.
func NewNullValue() *NullValue {
	return evaluator.NONE
}

// NewErrorValue creates a new error value.
func NewErrorValue(message string) *ErrorValue {
	return &evaluator.SlopError{Message: message}
}

// ServiceAdapter wraps an external Service implementation for use with the runtime.
// This is used internally to bridge external services to the evaluator.
type serviceAdapter struct {
	external Service
}

func (s *serviceAdapter) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	return s.external.Call(method, args, kwargs)
}

// wrapService wraps an external Service for internal use.
func wrapService(svc Service) evaluator.Service {
	// Check if it's already an internal service (e.g., from MCP)
	if internal, ok := svc.(evaluator.Service); ok {
		return internal
	}
	return &serviceAdapter{external: svc}
}

// RegisterExternalService registers an external service implementation with the runtime.
// This allows custom services to be called from SLOP scripts.
//
// Example:
//
//	rt := slop.NewRuntime()
//	rt.RegisterExternalService("myservice", &MyService{})
//
//	// Now in SLOP scripts:
//	// result = myservice.method(arg1, key: value)
func (r *Runtime) RegisterExternalService(name string, service Service) {
	r.evaluator.Context().RegisterService(name, wrapService(service))
}

// ValueToGo converts a SLOP Value to a native Go type.
// Returns:
//   - string for StringValue
//   - float64 for NumberValue
//   - int64 for IntValue
//   - bool for BoolValue
//   - []any for ListValue
//   - map[string]any for MapValue
//   - nil for NullValue
//   - error for ErrorValue
func ValueToGo(v Value) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case *evaluator.StringValue:
		return val.Value
	case *evaluator.FloatValue:
		return val.Value
	case *evaluator.IntValue:
		return val.Value
	case *evaluator.BoolValue:
		return val.Value
	case *evaluator.NoneValue:
		return nil
	case *evaluator.ListValue:
		result := make([]any, len(val.Elements))
		for i, elem := range val.Elements {
			result[i] = ValueToGo(elem)
		}
		return result
	case *evaluator.MapValue:
		result := make(map[string]any)
		for k, elem := range val.Pairs {
			result[k] = ValueToGo(elem)
		}
		return result
	case *evaluator.SlopError:
		return fmt.Errorf("%s", val.Message)
	default:
		return v.String()
	}
}

// GoToValue converts a native Go value to a SLOP Value.
// Supports:
//   - string -> StringValue
//   - float64, float32, int, int64, int32 -> NumberValue/IntValue
//   - bool -> BoolValue
//   - []any -> ListValue
//   - map[string]any -> MapValue
//   - nil -> NullValue
//   - error -> ErrorValue
func GoToValue(v any) Value {
	if v == nil {
		return NewNullValue()
	}

	switch val := v.(type) {
	case string:
		return NewStringValue(val)
	case float64:
		return NewNumberValue(val)
	case float32:
		return NewNumberValue(float64(val))
	case int:
		return NewIntValue(int64(val))
	case int64:
		return NewIntValue(val)
	case int32:
		return NewIntValue(int64(val))
	case bool:
		return NewBoolValue(val)
	case []any:
		elements := make([]Value, len(val))
		for i, elem := range val {
			elements[i] = GoToValue(elem)
		}
		return NewListValue(elements)
	case map[string]any:
		m := NewMapValue()
		for k, elem := range val {
			m.Set(k, GoToValue(elem))
		}
		return m
	case error:
		return NewErrorValue(val.Error())
	case Value:
		return val // Already a SLOP value
	default:
		return NewStringValue(fmt.Sprintf("%v", val))
	}
}
