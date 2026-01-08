package runtime

import (
	"context"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockService is a test service that echoes arguments.
type MockService struct {
	name        string
	callCount   int
	lastMethod  string
	lastArgs    []evaluator.Value
	lastKwargs  map[string]evaluator.Value
	returnValue evaluator.Value
	returnError error
}

func NewMockService(name string) *MockService {
	return &MockService{
		name:        name,
		returnValue: evaluator.NONE,
	}
}

func (s *MockService) Name() string {
	return s.name
}

func (s *MockService) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	return s.CallWithContext(context.Background(), method, args, kwargs)
}

func (s *MockService) CallWithContext(_ context.Context, method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	s.callCount++
	s.lastMethod = method
	s.lastArgs = args
	s.lastKwargs = kwargs

	if s.returnError != nil {
		return nil, s.returnError
	}
	return s.returnValue, nil
}

func (s *MockService) Methods() []string {
	return []string{"test", "echo"}
}

func (s *MockService) Close() error {
	return nil
}

func TestServiceRegistry(t *testing.T) {
	t.Run("register and get service", func(t *testing.T) {
		registry := NewServiceRegistry()
		svc := NewMockService("test")

		err := registry.Register(svc)
		require.NoError(t, err)

		got, ok := registry.Get("test")
		assert.True(t, ok)
		assert.Equal(t, svc, got)
	})

	t.Run("get unknown service returns false", func(t *testing.T) {
		registry := NewServiceRegistry()

		_, ok := registry.Get("unknown")
		assert.False(t, ok)
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		registry := NewServiceRegistry()
		svc1 := NewMockService("test")
		svc2 := NewMockService("test")

		err := registry.Register(svc1)
		require.NoError(t, err)

		err = registry.Register(svc2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("list services", func(t *testing.T) {
		registry := NewServiceRegistry()
		assert.NoError(t, registry.Register(NewMockService("svc1")))
		assert.NoError(t, registry.Register(NewMockService("svc2")))
		assert.NoError(t, registry.Register(NewMockService("svc3")))

		names := registry.List()
		assert.Len(t, names, 3)
		assert.Contains(t, names, "svc1")
		assert.Contains(t, names, "svc2")
		assert.Contains(t, names, "svc3")
	})

	t.Run("unregister service", func(t *testing.T) {
		registry := NewServiceRegistry()
		svc := NewMockService("test")
		assert.NoError(t, registry.Register(svc))

		registry.Unregister("test")

		_, ok := registry.Get("test")
		assert.False(t, ok)
	})

	t.Run("create service value", func(t *testing.T) {
		registry := NewServiceRegistry()
		svc := NewMockService("test")
		assert.NoError(t, registry.Register(svc))

		sv, ok := registry.CreateServiceValue("test")
		assert.True(t, ok)
		assert.Equal(t, "test", sv.Name)
		assert.NotNil(t, sv.Service)
	})
}

func TestMockServiceCall(t *testing.T) {
	t.Run("call with args", func(t *testing.T) {
		svc := NewMockService("test")
		svc.returnValue = &evaluator.StringValue{Value: "result"}

		args := []evaluator.Value{
			&evaluator.StringValue{Value: "arg1"},
			&evaluator.IntValue{Value: 42},
		}

		result, err := svc.Call("method", args, nil)
		require.NoError(t, err)

		assert.Equal(t, "method", svc.lastMethod)
		assert.Equal(t, args, svc.lastArgs)
		assert.Equal(t, 1, svc.callCount)
		assert.Equal(t, "result", result.(*evaluator.StringValue).Value)
	})

	t.Run("call with kwargs", func(t *testing.T) {
		svc := NewMockService("test")

		kwargs := map[string]evaluator.Value{
			"name": &evaluator.StringValue{Value: "test"},
			"age":  &evaluator.IntValue{Value: 30},
		}

		_, err := svc.Call("method", nil, kwargs)
		require.NoError(t, err)

		assert.Equal(t, kwargs, svc.lastKwargs)
	})
}

func TestValueConversion(t *testing.T) {
	t.Run("valueToAny conversions", func(t *testing.T) {
		tests := []struct {
			value    evaluator.Value
			expected any
		}{
			{evaluator.NONE, nil},
			{&evaluator.BoolValue{Value: true}, true},
			{&evaluator.BoolValue{Value: false}, false},
			{&evaluator.IntValue{Value: 42}, int64(42)},
			{&evaluator.FloatValue{Value: 3.14}, 3.14},
			{&evaluator.StringValue{Value: "hello"}, "hello"},
		}

		for _, tt := range tests {
			result := valueToAny(tt.value)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("valueToAny list", func(t *testing.T) {
		list := &evaluator.ListValue{
			Elements: []evaluator.Value{
				&evaluator.IntValue{Value: 1},
				&evaluator.IntValue{Value: 2},
				&evaluator.IntValue{Value: 3},
			},
		}

		result := valueToAny(list)
		expected := []any{int64(1), int64(2), int64(3)}
		assert.Equal(t, expected, result)
	})

	t.Run("valueToAny map", func(t *testing.T) {
		m := evaluator.NewMapValue()
		m.Set("name", &evaluator.StringValue{Value: "test"})
		m.Set("value", &evaluator.IntValue{Value: 42})

		result := valueToAny(m)
		expected := map[string]any{
			"name":  "test",
			"value": int64(42),
		}
		assert.Equal(t, expected, result)
	})

	t.Run("anyToValue conversions", func(t *testing.T) {
		tests := []struct {
			input    any
			expected evaluator.Value
		}{
			{nil, evaluator.NONE},
			{true, evaluator.TRUE},
			{false, evaluator.FALSE},
			{42, &evaluator.IntValue{Value: 42}},
			{int64(100), &evaluator.IntValue{Value: 100}},
			{3.14, &evaluator.FloatValue{Value: 3.14}},
			{"hello", &evaluator.StringValue{Value: "hello"}},
		}

		for _, tt := range tests {
			result := anyToValue(tt.input)
			switch expected := tt.expected.(type) {
			case *evaluator.IntValue:
				assert.Equal(t, expected.Value, result.(*evaluator.IntValue).Value)
			case *evaluator.FloatValue:
				assert.Equal(t, expected.Value, result.(*evaluator.FloatValue).Value)
			case *evaluator.StringValue:
				assert.Equal(t, expected.Value, result.(*evaluator.StringValue).Value)
			default:
				assert.Equal(t, tt.expected, result)
			}
		}
	})

	t.Run("anyToValue list", func(t *testing.T) {
		input := []any{1, 2, 3}
		result := anyToValue(input)

		list := result.(*evaluator.ListValue)
		assert.Len(t, list.Elements, 3)
	})

	t.Run("anyToValue map", func(t *testing.T) {
		input := map[string]any{
			"key": "value",
		}
		result := anyToValue(input)

		m := result.(*evaluator.MapValue)
		val, ok := m.Get("key")
		assert.True(t, ok)
		assert.Equal(t, "value", val.(*evaluator.StringValue).Value)
	})
}

func TestMCPManager(t *testing.T) {
	t.Run("create manager", func(t *testing.T) {
		manager := NewMCPManager()
		assert.NotNil(t, manager.Registry())
	})

	t.Run("registry access", func(t *testing.T) {
		manager := NewMCPManager()
		registry := manager.Registry()

		svc := NewMockService("test")
		err := registry.Register(svc)
		require.NoError(t, err)

		sv, ok := manager.GetService("test")
		assert.True(t, ok)
		assert.Equal(t, "test", sv.Name)
	})
}
