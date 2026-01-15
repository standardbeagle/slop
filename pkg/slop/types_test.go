package slop

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExternalTestService is a test service implementation.
type ExternalTestService struct {
	calls []externalTestCall
}

type externalTestCall struct {
	method string
	args   []Value
	kwargs map[string]Value
}

func (m *ExternalTestService) Call(method string, args []Value, kwargs map[string]Value) (Value, error) {
	m.calls = append(m.calls, externalTestCall{method: method, args: args, kwargs: kwargs})

	switch method {
	case "echo":
		if len(args) > 0 {
			return args[0], nil
		}
		if msg, ok := kwargs["message"]; ok {
			return msg, nil
		}
		return NewStringValue(""), nil

	case "add":
		a := 0.0
		b := 0.0
		if av, ok := kwargs["a"]; ok {
			if nv, ok := av.(*IntValue); ok {
				a = float64(nv.Value)
			} else if nv, ok := av.(*NumberValue); ok {
				a = nv.Value
			}
		}
		if bv, ok := kwargs["b"]; ok {
			if nv, ok := bv.(*IntValue); ok {
				b = float64(nv.Value)
			} else if nv, ok := bv.(*NumberValue); ok {
				b = nv.Value
			}
		}
		return NewNumberValue(a + b), nil

	case "greet":
		name := "World"
		if n, ok := kwargs["name"]; ok {
			if sv, ok := n.(*StringValue); ok {
				name = sv.Value
			}
		}
		return NewStringValue("Hello, " + name + "!"), nil

	case "get_list":
		return NewListValue([]Value{
			NewIntValue(1),
			NewIntValue(2),
			NewIntValue(3),
		}), nil

	case "get_map":
		m := NewMapValue()
		m.Set("foo", NewStringValue("bar"))
		m.Set("count", NewIntValue(42))
		return m, nil

	case "fail":
		return NewErrorValue("intentional error"), nil

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func TestRegisterExternalService(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("test", svc)

	// Verify service is registered
	services := rt.Services()
	_, ok := services["test"]
	assert.True(t, ok, "service should be registered")
}

func TestExternalService_Echo(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("mock", svc)

	// Execute script that calls the service
	result, err := rt.Execute(`result = mock.echo(message: "Hello!")
emit(result)`)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check service was called
	require.Len(t, svc.calls, 1)
	assert.Equal(t, "echo", svc.calls[0].method)

	// Check emitted value
	emitted := rt.Emitted()
	require.Len(t, emitted, 1)
	if sv, ok := emitted[0].(*StringValue); ok {
		assert.Equal(t, "Hello!", sv.Value)
	}
}

func TestExternalService_Add(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("calc", svc)

	result, err := rt.Execute(`sum = calc.add(a: 10, b: 25)
emit(sum)`)

	require.NoError(t, err)
	assert.NotNil(t, result)

	emitted := rt.Emitted()
	require.Len(t, emitted, 1)
	if nv, ok := emitted[0].(*NumberValue); ok {
		assert.Equal(t, 35.0, nv.Value)
	}
}

func TestExternalService_Greet(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("greeter", svc)

	result, err := rt.Execute(`msg = greeter.greet(name: "Claude")
emit(msg)`)

	require.NoError(t, err)
	assert.NotNil(t, result)

	emitted := rt.Emitted()
	require.Len(t, emitted, 1)
	if sv, ok := emitted[0].(*StringValue); ok {
		assert.Equal(t, "Hello, Claude!", sv.Value)
	}
}

func TestExternalService_ReturnList(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("data", svc)

	result, err := rt.Execute(`items = data.get_list()
emit(items)`)

	require.NoError(t, err)
	assert.NotNil(t, result)

	emitted := rt.Emitted()
	require.Len(t, emitted, 1)
	if lv, ok := emitted[0].(*ListValue); ok {
		assert.Len(t, lv.Elements, 3)
	}
}

func TestExternalService_ReturnMap(t *testing.T) {
	rt := NewRuntime()
	defer rt.Close()

	svc := &ExternalTestService{}
	rt.RegisterExternalService("data", svc)

	result, err := rt.Execute(`info = data.get_map()
emit(info)`)

	require.NoError(t, err)
	assert.NotNil(t, result)

	emitted := rt.Emitted()
	require.Len(t, emitted, 1)
	if mv, ok := emitted[0].(*MapValue); ok {
		foo, _ := mv.Get("foo")
		if sv, ok := foo.(*StringValue); ok {
			assert.Equal(t, "bar", sv.Value)
		}
	}
}

func TestValueToGo(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected any
	}{
		{"string", NewStringValue("hello"), "hello"},
		{"number", NewNumberValue(3.14), 3.14},
		{"int", NewIntValue(42), int64(42)},
		{"bool_true", NewBoolValue(true), true},
		{"bool_false", NewBoolValue(false), false},
		{"null", NewNullValue(), nil},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValueToGo(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValueToGo_List(t *testing.T) {
	list := NewListValue([]Value{
		NewIntValue(1),
		NewStringValue("two"),
		NewBoolValue(true),
	})

	result := ValueToGo(list)
	arr, ok := result.([]any)
	require.True(t, ok)
	assert.Len(t, arr, 3)
	assert.Equal(t, int64(1), arr[0])
	assert.Equal(t, "two", arr[1])
	assert.Equal(t, true, arr[2])
}

func TestValueToGo_Map(t *testing.T) {
	m := NewMapValue()
	m.Set("name", NewStringValue("test"))
	m.Set("count", NewIntValue(10))

	result := ValueToGo(m)
	mp, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", mp["name"])
	assert.Equal(t, int64(10), mp["count"])
}

func TestGoToValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string // Type name
	}{
		{"string", "hello", "string"},
		{"float64", 3.14, "float"},
		{"float32", float32(3.14), "float"},
		{"int", 42, "int"},
		{"int64", int64(42), "int"},
		{"bool", true, "bool"},
		{"nil", nil, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GoToValue(tt.input)
			assert.Equal(t, tt.expected, result.Type())
		})
	}
}

func TestGoToValue_Slice(t *testing.T) {
	input := []any{1, "two", true}
	result := GoToValue(input)

	lv, ok := result.(*ListValue)
	require.True(t, ok)
	assert.Len(t, lv.Elements, 3)
}

func TestGoToValue_Map(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"count": 10,
	}
	result := GoToValue(input)

	mv, ok := result.(*MapValue)
	require.True(t, ok)

	name, _ := mv.Get("name")
	if sv, ok := name.(*StringValue); ok {
		assert.Equal(t, "test", sv.Value)
	}
}

func TestGoToValue_Error(t *testing.T) {
	input := fmt.Errorf("test error")
	result := GoToValue(input)

	ev, ok := result.(*ErrorValue)
	require.True(t, ok)
	assert.Equal(t, "test error", ev.Message)
}

func TestGoToValue_RoundTrip(t *testing.T) {
	// Test that Go -> Value -> Go preserves data
	original := map[string]any{
		"name":   "test",
		"count":  int64(42),
		"active": true,
		"items":  []any{int64(1), int64(2), int64(3)},
	}

	value := GoToValue(original)
	result := ValueToGo(value)

	mp, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", mp["name"])
	assert.Equal(t, int64(42), mp["count"])
	assert.Equal(t, true, mp["active"])

	items, ok := mp["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 3)
}
