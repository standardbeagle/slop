// Package evaluator implements the SLOP interpreter.
package evaluator

import (
	"fmt"
	"strings"

	"github.com/standardbeagle/slop/internal/ast"
)

// Value represents a runtime value in SLOP.
type Value interface {
	Type() string
	String() string
	// IsTruthy returns true if the value is considered "true" in boolean context
	IsTruthy() bool
}

// NoneValue represents the absence of a value.
type NoneValue struct{}

func (n *NoneValue) Type() string    { return "none" }
func (n *NoneValue) String() string  { return "none" }
func (n *NoneValue) IsTruthy() bool  { return false }

// BoolValue represents a boolean.
type BoolValue struct {
	Value bool
}

func (b *BoolValue) Type() string   { return "bool" }
func (b *BoolValue) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}
func (b *BoolValue) IsTruthy() bool { return b.Value }

// IntValue represents an integer.
type IntValue struct {
	Value int64
}

func (i *IntValue) Type() string    { return "int" }
func (i *IntValue) String() string  { return fmt.Sprintf("%d", i.Value) }
func (i *IntValue) IsTruthy() bool  { return i.Value != 0 }

// FloatValue represents a floating-point number.
type FloatValue struct {
	Value float64
}

func (f *FloatValue) Type() string   { return "float" }
func (f *FloatValue) String() string { return fmt.Sprintf("%g", f.Value) }
func (f *FloatValue) IsTruthy() bool { return f.Value != 0 }

// StringValue represents a string.
type StringValue struct {
	Value string
}

func (s *StringValue) Type() string   { return "string" }
func (s *StringValue) String() string { return s.Value }
func (s *StringValue) IsTruthy() bool { return len(s.Value) > 0 }

// ListValue represents a list of values.
type ListValue struct {
	Elements []Value
}

func (l *ListValue) Type() string { return "list" }
func (l *ListValue) String() string {
	elements := make([]string, len(l.Elements))
	for i, e := range l.Elements {
		elements[i] = e.String()
	}
	return "[" + strings.Join(elements, ", ") + "]"
}
func (l *ListValue) IsTruthy() bool { return len(l.Elements) > 0 }

// MapValue represents a map/dictionary.
type MapValue struct {
	Pairs map[string]Value
	Order []string // Maintain insertion order
}

func NewMapValue() *MapValue {
	return &MapValue{
		Pairs: make(map[string]Value),
		Order: []string{},
	}
}

func (m *MapValue) Set(key string, value Value) {
	if _, exists := m.Pairs[key]; !exists {
		m.Order = append(m.Order, key)
	}
	m.Pairs[key] = value
}

func (m *MapValue) Get(key string) (Value, bool) {
	v, ok := m.Pairs[key]
	return v, ok
}

func (m *MapValue) Delete(key string) {
	delete(m.Pairs, key)
	// Remove from Order
	for i, k := range m.Order {
		if k == key {
			m.Order = append(m.Order[:i], m.Order[i+1:]...)
			break
		}
	}
}

func (m *MapValue) Type() string { return "map" }
func (m *MapValue) String() string {
	pairs := make([]string, 0, len(m.Pairs))
	for _, k := range m.Order {
		v := m.Pairs[k]
		pairs = append(pairs, fmt.Sprintf("%s: %s", k, v.String()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}
func (m *MapValue) IsTruthy() bool { return len(m.Pairs) > 0 }

// SetValue represents a set of unique values.
type SetValue struct {
	Elements map[string]Value // Key is string representation for dedup
}

func NewSetValue() *SetValue {
	return &SetValue{
		Elements: make(map[string]Value),
	}
}

func (s *SetValue) Add(value Value) {
	s.Elements[value.String()] = value
}

func (s *SetValue) Has(value Value) bool {
	_, ok := s.Elements[value.String()]
	return ok
}

func (s *SetValue) Remove(value Value) {
	delete(s.Elements, value.String())
}

func (s *SetValue) Type() string { return "set" }
func (s *SetValue) String() string {
	elements := make([]string, 0, len(s.Elements))
	for k := range s.Elements {
		elements = append(elements, k)
	}
	return "{" + strings.Join(elements, ", ") + "}"
}
func (s *SetValue) IsTruthy() bool { return len(s.Elements) > 0 }

// FunctionValue represents a user-defined function.
type FunctionValue struct {
	Name       string
	Parameters []*ast.Parameter
	Body       *ast.Block
	Env        *Scope // Closure environment
}

func (f *FunctionValue) Type() string   { return "function" }
func (f *FunctionValue) String() string { return fmt.Sprintf("<function %s>", f.Name) }
func (f *FunctionValue) IsTruthy() bool { return true }

// LambdaValue represents a lambda function.
type LambdaValue struct {
	Parameters []*ast.Identifier
	Body       ast.Expression
	Env        *Scope
}

func (l *LambdaValue) Type() string   { return "lambda" }
func (l *LambdaValue) String() string { return "<lambda>" }
func (l *LambdaValue) IsTruthy() bool { return true }

// BuiltinFunction is the signature for built-in functions.
type BuiltinFunction func(args []Value, kwargs map[string]Value) (Value, error)

// BuiltinValue represents a built-in function.
type BuiltinValue struct {
	Name string
	Fn   BuiltinFunction
}

func (b *BuiltinValue) Type() string   { return "builtin" }
func (b *BuiltinValue) String() string { return fmt.Sprintf("<builtin %s>", b.Name) }
func (b *BuiltinValue) IsTruthy() bool { return true }

// ServiceValue represents an MCP service.
type ServiceValue struct {
	Name    string
	Service Service
}

func (s *ServiceValue) Type() string   { return "service" }
func (s *ServiceValue) String() string { return fmt.Sprintf("<service %s>", s.Name) }
func (s *ServiceValue) IsTruthy() bool { return true }

// Service is the interface for MCP services.
type Service interface {
	Call(method string, args []Value, kwargs map[string]Value) (Value, error)
}

// SlopError represents a runtime error value.
type SlopError struct {
	Message string
	Data    Value
}

func (e *SlopError) Type() string   { return "error" }
func (e *SlopError) String() string { return fmt.Sprintf("Error: %s", e.Message) }
func (e *SlopError) IsTruthy() bool { return false }
func (e *SlopError) Error() string  { return e.Message }

// IteratorValue represents an iterator for lazy evaluation.
type IteratorValue struct {
	Type_   string
	Current int
	End     int
	Step    int
	Items   []Value // For list-based iteration
}

func (i *IteratorValue) Type() string   { return "iterator" }
func (i *IteratorValue) String() string { return "<iterator>" }
func (i *IteratorValue) IsTruthy() bool { return true }

func (i *IteratorValue) Next() (Value, bool) {
	if i.Items != nil {
		// List-based iteration
		if i.Current >= len(i.Items) {
			return nil, false
		}
		val := i.Items[i.Current]
		i.Current++
		return val, true
	}
	// Range-based iteration
	if i.Step > 0 && i.Current >= i.End {
		return nil, false
	}
	if i.Step < 0 && i.Current <= i.End {
		return nil, false
	}
	val := &IntValue{Value: int64(i.Current)}
	i.Current += i.Step
	return val, true
}

// Singleton values
var (
	NONE  = &NoneValue{}
	TRUE  = &BoolValue{Value: true}
	FALSE = &BoolValue{Value: false}
)

// NewBool returns a boolean value (uses singletons).
func NewBool(value bool) *BoolValue {
	if value {
		return TRUE
	}
	return FALSE
}

// Helper functions for type checking

func IsNone(v Value) bool {
	_, ok := v.(*NoneValue)
	return ok
}

func IsBool(v Value) bool {
	_, ok := v.(*BoolValue)
	return ok
}

func IsInt(v Value) bool {
	_, ok := v.(*IntValue)
	return ok
}

func IsFloat(v Value) bool {
	_, ok := v.(*FloatValue)
	return ok
}

func IsNumber(v Value) bool {
	return IsInt(v) || IsFloat(v)
}

func IsString(v Value) bool {
	_, ok := v.(*StringValue)
	return ok
}

func IsList(v Value) bool {
	_, ok := v.(*ListValue)
	return ok
}

func IsMap(v Value) bool {
	_, ok := v.(*MapValue)
	return ok
}

func IsCallable(v Value) bool {
	switch v.(type) {
	case *FunctionValue, *LambdaValue, *BuiltinValue:
		return true
	default:
		return false
	}
}

// ToFloat converts a numeric value to float64.
func ToFloat(v Value) (float64, bool) {
	switch val := v.(type) {
	case *IntValue:
		return float64(val.Value), true
	case *FloatValue:
		return val.Value, true
	default:
		return 0, false
	}
}

// ToInt converts a numeric value to int64 if possible.
func ToInt(v Value) (int64, bool) {
	switch val := v.(type) {
	case *IntValue:
		return val.Value, true
	case *FloatValue:
		return int64(val.Value), true
	default:
		return 0, false
	}
}

// Compare compares two values and returns:
// -1 if a < b
// 0 if a == b
// 1 if a > b
// error if not comparable
func Compare(a, b Value) (int, error) {
	// Handle numeric comparison
	if IsNumber(a) && IsNumber(b) {
		af, _ := ToFloat(a)
		bf, _ := ToFloat(b)
		if af < bf {
			return -1, nil
		}
		if af > bf {
			return 1, nil
		}
		return 0, nil
	}

	// Handle string comparison
	if as, ok := a.(*StringValue); ok {
		if bs, ok := b.(*StringValue); ok {
			if as.Value < bs.Value {
				return -1, nil
			}
			if as.Value > bs.Value {
				return 1, nil
			}
			return 0, nil
		}
	}

	// Handle boolean comparison
	if ab, ok := a.(*BoolValue); ok {
		if bb, ok := b.(*BoolValue); ok {
			if !ab.Value && bb.Value {
				return -1, nil
			}
			if ab.Value && !bb.Value {
				return 1, nil
			}
			return 0, nil
		}
	}

	// Handle none comparison
	if IsNone(a) && IsNone(b) {
		return 0, nil
	}

	return 0, fmt.Errorf("cannot compare %s and %s", a.Type(), b.Type())
}

// Equal checks if two values are equal.
func Equal(a, b Value) bool {
	cmp, err := Compare(a, b)
	if err != nil {
		// For non-comparable types, check identity
		return a == b
	}
	return cmp == 0
}
