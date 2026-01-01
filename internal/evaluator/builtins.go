// Package evaluator provides the builtin integration.
package evaluator

import (
	"fmt"
	"strings"
)

// RegisterBuiltins registers all built-in functions from the builtin package.
// This is called by the runtime to populate the global scope with built-in functions.
func (c *Context) RegisterBuiltins(registry BuiltinRegistry) {
	for _, name := range registry.Names() {
		// Check if it's a constant first
		if constReg, ok := registry.(ConstantRegistry); ok {
			if val, ok := constReg.GetConstant(name); ok {
				c.Globals.Set(name, val)
				continue
			}
		}

		// Regular function
		if fn, ok := registry.GetAsValue(name); ok {
			c.Globals.Set(name, fn)
		}
	}
}

// BuiltinRegistry is the interface for accessing built-in functions.
type BuiltinRegistry interface {
	Names() []string
	GetAsValue(name string) (Value, bool)
}

// ConstantRegistry is the interface for accessing constant values.
type ConstantRegistry interface {
	GetConstant(name string) (Value, bool)
}

// getMethod returns a bound method for a value type, or nil if not found.
// This enables calling methods on built-in types like str.upper(), list.append(), etc.
func (e *Evaluator) getMethod(obj Value, name string) Value {
	switch v := obj.(type) {
	case *StringValue:
		return e.getStringMethod(v, name)
	case *ListValue:
		return e.getListMethod(v, name)
	case *MapValue:
		return e.getMapMethod(v, name)
	case *SetValue:
		return e.getSetMethod(v, name)
	}
	return nil
}

// getStringMethod returns string methods.
func (e *Evaluator) getStringMethod(s *StringValue, name string) Value {
	switch name {
	case "upper":
		return &BuiltinValue{
			Name: "upper",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				return &StringValue{Value: strings.ToUpper(s.Value)}, nil
			},
		}
	case "lower":
		return &BuiltinValue{
			Name: "lower",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				return &StringValue{Value: strings.ToLower(s.Value)}, nil
			},
		}
	case "strip":
		return &BuiltinValue{
			Name: "strip",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				return &StringValue{Value: strings.TrimSpace(s.Value)}, nil
			},
		}
	case "lstrip":
		return &BuiltinValue{
			Name: "lstrip",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				return &StringValue{Value: strings.TrimLeft(s.Value, " \t\n\r")}, nil
			},
		}
	case "rstrip":
		return &BuiltinValue{
			Name: "rstrip",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				return &StringValue{Value: strings.TrimRight(s.Value, " \t\n\r")}, nil
			},
		}
	case "split":
		return &BuiltinValue{
			Name: "split",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				var parts []string
				if len(args) > 0 {
					if sep, ok := args[0].(*StringValue); ok {
						parts = strings.Split(s.Value, sep.Value)
					} else {
						return nil, fmt.Errorf("split() separator must be string")
					}
				} else {
					parts = strings.Fields(s.Value)
				}
				elements := make([]Value, len(parts))
				for i, p := range parts {
					elements[i] = &StringValue{Value: p}
				}
				return &ListValue{Elements: elements}, nil
			},
		}
	case "join":
		return &BuiltinValue{
			Name: "join",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("join() requires 1 argument")
				}
				list, ok := args[0].(*ListValue)
				if !ok {
					return nil, fmt.Errorf("join() requires list argument")
				}
				parts := make([]string, len(list.Elements))
				for i, elem := range list.Elements {
					parts[i] = elem.String()
					if sv, ok := elem.(*StringValue); ok {
						parts[i] = sv.Value
					}
				}
				return &StringValue{Value: strings.Join(parts, s.Value)}, nil
			},
		}
	case "replace":
		return &BuiltinValue{
			Name: "replace",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) < 2 {
					return nil, fmt.Errorf("replace() requires 2 arguments")
				}
				old, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("replace() old must be string")
				}
				new, ok := args[1].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("replace() new must be string")
				}
				return &StringValue{Value: strings.ReplaceAll(s.Value, old.Value, new.Value)}, nil
			},
		}
	case "startswith":
		return &BuiltinValue{
			Name: "startswith",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("startswith() requires 1 argument")
				}
				prefix, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("startswith() requires string argument")
				}
				return NewBool(strings.HasPrefix(s.Value, prefix.Value)), nil
			},
		}
	case "endswith":
		return &BuiltinValue{
			Name: "endswith",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("endswith() requires 1 argument")
				}
				suffix, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("endswith() requires string argument")
				}
				return NewBool(strings.HasSuffix(s.Value, suffix.Value)), nil
			},
		}
	case "contains":
		return &BuiltinValue{
			Name: "contains",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("contains() requires 1 argument")
				}
				substr, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("contains() requires string argument")
				}
				return NewBool(strings.Contains(s.Value, substr.Value)), nil
			},
		}
	case "find":
		return &BuiltinValue{
			Name: "find",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("find() requires 1 argument")
				}
				substr, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("find() requires string argument")
				}
				return &IntValue{Value: int64(strings.Index(s.Value, substr.Value))}, nil
			},
		}
	case "count":
		return &BuiltinValue{
			Name: "count",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("count() requires 1 argument")
				}
				substr, ok := args[0].(*StringValue)
				if !ok {
					return nil, fmt.Errorf("count() requires string argument")
				}
				return &IntValue{Value: int64(strings.Count(s.Value, substr.Value))}, nil
			},
		}
	case "repeat":
		return &BuiltinValue{
			Name: "repeat",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("repeat() requires 1 argument")
				}
				n, ok := ToInt(args[0])
				if !ok {
					return nil, fmt.Errorf("repeat() requires int argument")
				}
				return &StringValue{Value: strings.Repeat(s.Value, int(n))}, nil
			},
		}
	case "reverse":
		return &BuiltinValue{
			Name: "reverse",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				runes := []rune(s.Value)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return &StringValue{Value: string(runes)}, nil
			},
		}
	case "lines":
		return &BuiltinValue{
			Name: "lines",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				lines := strings.Split(s.Value, "\n")
				elements := make([]Value, len(lines))
				for i, line := range lines {
					elements[i] = &StringValue{Value: line}
				}
				return &ListValue{Elements: elements}, nil
			},
		}
	case "words":
		return &BuiltinValue{
			Name: "words",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				words := strings.Fields(s.Value)
				elements := make([]Value, len(words))
				for i, word := range words {
					elements[i] = &StringValue{Value: word}
				}
				return &ListValue{Elements: elements}, nil
			},
		}
	}
	return nil
}

// getListMethod returns list methods.
func (e *Evaluator) getListMethod(l *ListValue, name string) Value {
	switch name {
	case "append":
		return &BuiltinValue{
			Name: "append",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("append() requires 1 argument")
				}
				l.Elements = append(l.Elements, args[0])
				return l, nil
			},
		}
	case "extend":
		return &BuiltinValue{
			Name: "extend",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("extend() requires 1 argument")
				}
				other, ok := args[0].(*ListValue)
				if !ok {
					return nil, fmt.Errorf("extend() requires list argument")
				}
				l.Elements = append(l.Elements, other.Elements...)
				return l, nil
			},
		}
	case "insert":
		return &BuiltinValue{
			Name: "insert",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("insert() requires 2 arguments")
				}
				i, ok := ToInt(args[0])
				if !ok {
					return nil, fmt.Errorf("insert() index must be int")
				}
				if i < 0 || i > int64(len(l.Elements)) {
					return nil, fmt.Errorf("insert() index out of range")
				}
				l.Elements = append(l.Elements[:i], append([]Value{args[1]}, l.Elements[i:]...)...)
				return l, nil
			},
		}
	case "remove":
		return &BuiltinValue{
			Name: "remove",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("remove() requires 1 argument")
				}
				for i, elem := range l.Elements {
					if Equal(elem, args[0]) {
						l.Elements = append(l.Elements[:i], l.Elements[i+1:]...)
						return l, nil
					}
				}
				return nil, fmt.Errorf("remove() element not found")
			},
		}
	case "pop":
		return &BuiltinValue{
			Name: "pop",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(l.Elements) == 0 {
					return nil, fmt.Errorf("pop() from empty list")
				}
				var i int64 = -1
				if len(args) > 0 {
					var ok bool
					i, ok = ToInt(args[0])
					if !ok {
						return nil, fmt.Errorf("pop() index must be int")
					}
				}
				if i < 0 {
					i = int64(len(l.Elements)) + i
				}
				if i < 0 || i >= int64(len(l.Elements)) {
					return nil, fmt.Errorf("pop() index out of range")
				}
				val := l.Elements[i]
				l.Elements = append(l.Elements[:i], l.Elements[i+1:]...)
				return val, nil
			},
		}
	case "clear":
		return &BuiltinValue{
			Name: "clear",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				l.Elements = []Value{}
				return l, nil
			},
		}
	case "index":
		return &BuiltinValue{
			Name: "index",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("index() requires 1 argument")
				}
				for i, elem := range l.Elements {
					if Equal(elem, args[0]) {
						return &IntValue{Value: int64(i)}, nil
					}
				}
				return nil, fmt.Errorf("index() element not found")
			},
		}
	case "count":
		return &BuiltinValue{
			Name: "count",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("count() requires 1 argument")
				}
				count := int64(0)
				for _, elem := range l.Elements {
					if Equal(elem, args[0]) {
						count++
					}
				}
				return &IntValue{Value: count}, nil
			},
		}
	case "sort":
		return &BuiltinValue{
			Name: "sort",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				// Simple in-place sort using bubble sort (for correctness, not performance)
				for i := 0; i < len(l.Elements); i++ {
					for j := i + 1; j < len(l.Elements); j++ {
						cmp, err := Compare(l.Elements[i], l.Elements[j])
						if err != nil {
							return nil, err
						}
						if cmp > 0 {
							l.Elements[i], l.Elements[j] = l.Elements[j], l.Elements[i]
						}
					}
				}
				return l, nil
			},
		}
	case "reverse":
		return &BuiltinValue{
			Name: "reverse",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				for i, j := 0, len(l.Elements)-1; i < j; i, j = i+1, j-1 {
					l.Elements[i], l.Elements[j] = l.Elements[j], l.Elements[i]
				}
				return l, nil
			},
		}
	case "copy":
		return &BuiltinValue{
			Name: "copy",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				elements := make([]Value, len(l.Elements))
				copy(elements, l.Elements)
				return &ListValue{Elements: elements}, nil
			},
		}
	}
	return nil
}

// getMapMethod returns map methods.
func (e *Evaluator) getMapMethod(m *MapValue, name string) Value {
	switch name {
	case "keys":
		return &BuiltinValue{
			Name: "keys",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				keys := make([]Value, 0, len(m.Pairs))
				for _, key := range m.Order {
					keys = append(keys, &StringValue{Value: key})
				}
				return &ListValue{Elements: keys}, nil
			},
		}
	case "values":
		return &BuiltinValue{
			Name: "values",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				values := make([]Value, 0, len(m.Pairs))
				for _, key := range m.Order {
					values = append(values, m.Pairs[key])
				}
				return &ListValue{Elements: values}, nil
			},
		}
	case "items":
		return &BuiltinValue{
			Name: "items",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				items := make([]Value, 0, len(m.Pairs))
				for _, key := range m.Order {
					pair := &ListValue{Elements: []Value{
						&StringValue{Value: key},
						m.Pairs[key],
					}}
					items = append(items, pair)
				}
				return &ListValue{Elements: items}, nil
			},
		}
	case "get":
		return &BuiltinValue{
			Name: "get",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("get() requires at least 1 argument")
				}
				key := args[0].String()
				if sv, ok := args[0].(*StringValue); ok {
					key = sv.Value
				}
				val, ok := m.Get(key)
				if !ok {
					if len(args) > 1 {
						return args[1], nil
					}
					return NONE, nil
				}
				return val, nil
			},
		}
	case "pop":
		return &BuiltinValue{
			Name: "pop",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("pop() requires at least 1 argument")
				}
				key := args[0].String()
				if sv, ok := args[0].(*StringValue); ok {
					key = sv.Value
				}
				val, ok := m.Get(key)
				if !ok {
					if len(args) > 1 {
						return args[1], nil
					}
					return nil, fmt.Errorf("key not found: %s", key)
				}
				m.Delete(key)
				return val, nil
			},
		}
	case "update":
		return &BuiltinValue{
			Name: "update",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("update() requires 1 argument")
				}
				other, ok := args[0].(*MapValue)
				if !ok {
					return nil, fmt.Errorf("update() requires map argument")
				}
				for _, key := range other.Order {
					m.Set(key, other.Pairs[key])
				}
				return m, nil
			},
		}
	case "clear":
		return &BuiltinValue{
			Name: "clear",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				m.Pairs = make(map[string]Value)
				m.Order = []string{}
				return m, nil
			},
		}
	case "copy":
		return &BuiltinValue{
			Name: "copy",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				newMap := NewMapValue()
				for _, key := range m.Order {
					newMap.Set(key, m.Pairs[key])
				}
				return newMap, nil
			},
		}
	case "merge":
		return &BuiltinValue{
			Name: "merge",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("merge() requires 1 argument")
				}
				other, ok := args[0].(*MapValue)
				if !ok {
					return nil, fmt.Errorf("merge() requires map argument")
				}
				newMap := NewMapValue()
				for _, key := range m.Order {
					newMap.Set(key, m.Pairs[key])
				}
				for _, key := range other.Order {
					newMap.Set(key, other.Pairs[key])
				}
				return newMap, nil
			},
		}
	}
	return nil
}

// getSetMethod returns set methods.
func (e *Evaluator) getSetMethod(s *SetValue, name string) Value {
	switch name {
	case "add":
		return &BuiltinValue{
			Name: "add",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("add() requires 1 argument")
				}
				s.Add(args[0])
				return s, nil
			},
		}
	case "remove":
		return &BuiltinValue{
			Name: "remove",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("remove() requires 1 argument")
				}
				if !s.Has(args[0]) {
					return nil, fmt.Errorf("element not in set")
				}
				s.Remove(args[0])
				return s, nil
			},
		}
	case "discard":
		return &BuiltinValue{
			Name: "discard",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("discard() requires 1 argument")
				}
				s.Remove(args[0])
				return s, nil
			},
		}
	case "pop":
		return &BuiltinValue{
			Name: "pop",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(s.Elements) == 0 {
					return nil, fmt.Errorf("pop from empty set")
				}
				for key, val := range s.Elements {
					delete(s.Elements, key)
					return val, nil
				}
				return NONE, nil
			},
		}
	case "clear":
		return &BuiltinValue{
			Name: "clear",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				s.Elements = make(map[string]Value)
				return s, nil
			},
		}
	case "union":
		return &BuiltinValue{
			Name: "union",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("union() requires 1 argument")
				}
				other, ok := args[0].(*SetValue)
				if !ok {
					return nil, fmt.Errorf("union() requires set argument")
				}
				result := NewSetValue()
				for _, v := range s.Elements {
					result.Add(v)
				}
				for _, v := range other.Elements {
					result.Add(v)
				}
				return result, nil
			},
		}
	case "intersection":
		return &BuiltinValue{
			Name: "intersection",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("intersection() requires 1 argument")
				}
				other, ok := args[0].(*SetValue)
				if !ok {
					return nil, fmt.Errorf("intersection() requires set argument")
				}
				result := NewSetValue()
				for key, v := range s.Elements {
					if _, exists := other.Elements[key]; exists {
						result.Add(v)
					}
				}
				return result, nil
			},
		}
	case "difference":
		return &BuiltinValue{
			Name: "difference",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("difference() requires 1 argument")
				}
				other, ok := args[0].(*SetValue)
				if !ok {
					return nil, fmt.Errorf("difference() requires set argument")
				}
				result := NewSetValue()
				for key, v := range s.Elements {
					if _, exists := other.Elements[key]; !exists {
						result.Add(v)
					}
				}
				return result, nil
			},
		}
	case "issubset":
		return &BuiltinValue{
			Name: "issubset",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("issubset() requires 1 argument")
				}
				other, ok := args[0].(*SetValue)
				if !ok {
					return nil, fmt.Errorf("issubset() requires set argument")
				}
				for key := range s.Elements {
					if _, exists := other.Elements[key]; !exists {
						return FALSE, nil
					}
				}
				return TRUE, nil
			},
		}
	case "issuperset":
		return &BuiltinValue{
			Name: "issuperset",
			Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("issuperset() requires 1 argument")
				}
				other, ok := args[0].(*SetValue)
				if !ok {
					return nil, fmt.Errorf("issuperset() requires set argument")
				}
				for key := range other.Elements {
					if _, exists := s.Elements[key]; !exists {
						return FALSE, nil
					}
				}
				return TRUE, nil
			},
		}
	}
	return nil
}
