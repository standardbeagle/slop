package slop

import (
	"context"
	"testing"
	"time"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeBasic(t *testing.T) {
	rt := NewRuntime()

	result, err := rt.Execute("1 + 2")
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.(*evaluator.IntValue).Value)
}

func TestRuntimeBuiltinFunctions(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"len of string", "len(\"hello\")", int64(5)},
		{"len of list", "len([1, 2, 3])", int64(3)},
		{"abs of negative", "abs(-5)", int64(5)},
		{"max of list", "max([1, 5, 3])", int64(5)},
		{"min of list", "min([1, 5, 3])", int64(1)},
		{"sum of list", "sum([1, 2, 3, 4])", int64(10)},
		{"type of int", "type(42)", "int"},
		{"type of string", "type(\"hello\")", "string"},
		{"type of list", "type([1, 2, 3])", "list"},
		{"int from string", "int(\"42\")", int64(42)},
		{"float from string", "float(\"3.14\")", 3.14},
		{"str from int", "str(42)", "42"},
		{"bool from 0", "bool(0)", false},
		{"bool from 1", "bool(1)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRuntime()
			result, err := rt.Execute(tt.code)
			require.NoError(t, err)

			switch expected := tt.expected.(type) {
			case int64:
				assert.Equal(t, expected, result.(*evaluator.IntValue).Value)
			case float64:
				assert.Equal(t, expected, result.(*evaluator.FloatValue).Value)
			case string:
				assert.Equal(t, expected, result.(*evaluator.StringValue).Value)
			case bool:
				assert.Equal(t, expected, result.(*evaluator.BoolValue).Value)
			}
		})
	}
}

func TestRuntimeStringMethods(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"upper", "\"hello\".upper()", "HELLO"},
		{"lower", "\"HELLO\".lower()", "hello"},
		{"strip", "\"  hello  \".strip()", "hello"},
		{"repeat", "\"ab\".repeat(3)", "ababab"},
		{"reverse", "\"hello\".reverse()", "olleh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := NewRuntime()
			result, err := rt.Execute(tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.(*evaluator.StringValue).Value)
		})
	}
}

func TestRuntimeStringMethodsWithArgs(t *testing.T) {
	rt := NewRuntime()

	// Test split
	result, err := rt.Execute("\"a,b,c\".split(\",\")")
	require.NoError(t, err)
	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, "a", list.Elements[0].(*evaluator.StringValue).Value)
	assert.Equal(t, "b", list.Elements[1].(*evaluator.StringValue).Value)
	assert.Equal(t, "c", list.Elements[2].(*evaluator.StringValue).Value)

	// Test join
	result, err = rt.Execute("\",\".join([\"a\", \"b\", \"c\"])")
	require.NoError(t, err)
	assert.Equal(t, "a,b,c", result.(*evaluator.StringValue).Value)

	// Test replace
	result, err = rt.Execute("\"hello world\".replace(\"world\", \"SLOP\")")
	require.NoError(t, err)
	assert.Equal(t, "hello SLOP", result.(*evaluator.StringValue).Value)

	// Test startswith
	result, err = rt.Execute("\"hello\".startswith(\"he\")")
	require.NoError(t, err)
	assert.True(t, result.(*evaluator.BoolValue).Value)

	// Test endswith
	result, err = rt.Execute("\"hello\".endswith(\"lo\")")
	require.NoError(t, err)
	assert.True(t, result.(*evaluator.BoolValue).Value)

	// Test contains
	result, err = rt.Execute("\"hello\".contains(\"ell\")")
	require.NoError(t, err)
	assert.True(t, result.(*evaluator.BoolValue).Value)
}

func TestRuntimeListMethods(t *testing.T) {
	rt := NewRuntime()

	// Test append
	result, err := rt.Execute(`
x = [1, 2]
x.append(3)
x
`)
	require.NoError(t, err)
	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, int64(3), list.Elements[2].(*evaluator.IntValue).Value)

	// Test pop
	rt2 := NewRuntime()
	result, err = rt2.Execute(`
x = [1, 2, 3]
x.pop()
`)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.(*evaluator.IntValue).Value)

	// Test copy
	rt3 := NewRuntime()
	result, err = rt3.Execute(`
x = [1, 2, 3]
x.copy()
`)
	require.NoError(t, err)
	list = result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
}

func TestRuntimeMapMethods(t *testing.T) {
	rt := NewRuntime()

	// Test keys
	result, err := rt.Execute(`
m = {a: 1, b: 2}
m.keys()
`)
	require.NoError(t, err)
	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 2)

	// Test values
	rt2 := NewRuntime()
	result, err = rt2.Execute(`
m = {a: 1, b: 2}
m.values()
`)
	require.NoError(t, err)
	list = result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 2)

	// Test get with default
	rt3 := NewRuntime()
	result, err = rt3.Execute(`
m = {a: 1}
m.get("b", 999)
`)
	require.NoError(t, err)
	assert.Equal(t, int64(999), result.(*evaluator.IntValue).Value)
}

func TestRuntimePipelineOperations(t *testing.T) {
	rt := NewRuntime()

	// Test map pipeline
	result, err := rt.Execute(`
items = [1, 2, 3]
items | map(x -> x * 2)
`)
	require.NoError(t, err)
	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, int64(2), list.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(4), list.Elements[1].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(6), list.Elements[2].(*evaluator.IntValue).Value)

	// Test filter pipeline
	rt2 := NewRuntime()
	result, err = rt2.Execute(`
items = [1, 2, 3, 4, 5]
items | filter(x -> x > 2)
`)
	require.NoError(t, err)
	list = result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, int64(3), list.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(4), list.Elements[1].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(5), list.Elements[2].(*evaluator.IntValue).Value)
}

func TestRuntimeForLoop(t *testing.T) {
	rt := NewRuntime()

	result, err := rt.Execute(`
total = 0
for i in range(5):
    total = total + i
total
`)
	require.NoError(t, err)
	assert.Equal(t, int64(10), result.(*evaluator.IntValue).Value)
}

func TestRuntimeFunction(t *testing.T) {
	rt := NewRuntime()

	result, err := rt.Execute(`
def add(a, b):
    return a + b

add(3, 4)
`)
	require.NoError(t, err)
	assert.Equal(t, int64(7), result.(*evaluator.IntValue).Value)
}

func TestRuntimeEmit(t *testing.T) {
	rt := NewRuntime()

	_, err := rt.Execute(`
emit(count: 42, message: "hello")
`)
	require.NoError(t, err)

	emitted := rt.Emitted()
	require.Len(t, emitted, 1)

	m := emitted[0].(*evaluator.MapValue)
	count, ok := m.Get("count")
	require.True(t, ok)
	assert.Equal(t, int64(42), count.(*evaluator.IntValue).Value)

	msg, ok := m.Get("message")
	require.True(t, ok)
	assert.Equal(t, "hello", msg.(*evaluator.StringValue).Value)
}

func TestRuntimeComplexScript(t *testing.T) {
	rt := NewRuntime()

	// A more complex script that exercises multiple features
	result, err := rt.Execute(`
# Process a list of items
items = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

# Filter even numbers and double them
processed = items | filter(x -> x % 2 == 0) | map(x -> x * 2)

# Sum the results
sum(processed)
`)
	require.NoError(t, err)
	assert.Equal(t, int64(60), result.(*evaluator.IntValue).Value) // (2+4+6+8+10) * 2 = 60
}

func TestRuntimeMatch(t *testing.T) {
	rt := NewRuntime()

	result, err := rt.Execute(`
x = 2
match x:
    1 -> "one"
    2 -> "two"
    _ -> "other"
`)
	require.NoError(t, err)
	assert.Equal(t, "two", result.(*evaluator.StringValue).Value)
}

// MockService implements evaluator.Service for testing.
type MockService struct {
	name        string
	returnValue evaluator.Value
}

func (s *MockService) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	// Return a value based on the method called
	if s.returnValue != nil {
		return s.returnValue, nil
	}

	// Default: return a map with method name and args count
	m := evaluator.NewMapValue()
	m.Set("method", &evaluator.StringValue{Value: method})
	m.Set("args", &evaluator.IntValue{Value: int64(len(args))})

	// Include first arg if present
	if len(args) > 0 {
		m.Set("firstArg", args[0])
	}

	// Include kwargs
	if len(kwargs) > 0 {
		kw := evaluator.NewMapValue()
		for k, v := range kwargs {
			kw.Set(k, v)
		}
		m.Set("kwargs", kw)
	}

	return m, nil
}

func TestRuntimeServiceCall(t *testing.T) {
	t.Run("basic service call", func(t *testing.T) {
		rt := NewRuntime()
		svc := &MockService{name: "test"}

		// Register the service
		rt.RegisterService("test", svc)

		result, err := rt.Execute(`test.hello("world")`)
		require.NoError(t, err)

		m := result.(*evaluator.MapValue)
		method, ok := m.Get("method")
		require.True(t, ok)
		assert.Equal(t, "hello", method.(*evaluator.StringValue).Value)

		firstArg, ok := m.Get("firstArg")
		require.True(t, ok)
		assert.Equal(t, "world", firstArg.(*evaluator.StringValue).Value)
	})

	t.Run("service call with kwargs", func(t *testing.T) {
		rt := NewRuntime()
		svc := &MockService{name: "api"}

		rt.RegisterService("api", svc)

		result, err := rt.Execute(`api.query("users", limit: 10, offset: 5)`)
		require.NoError(t, err)

		m := result.(*evaluator.MapValue)

		kwargs, ok := m.Get("kwargs")
		require.True(t, ok)
		kw := kwargs.(*evaluator.MapValue)

		limit, ok := kw.Get("limit")
		require.True(t, ok)
		assert.Equal(t, int64(10), limit.(*evaluator.IntValue).Value)

		offset, ok := kw.Get("offset")
		require.True(t, ok)
		assert.Equal(t, int64(5), offset.(*evaluator.IntValue).Value)
	})

	t.Run("service call with return value", func(t *testing.T) {
		rt := NewRuntime()
		svc := &MockService{
			name:        "db",
			returnValue: &evaluator.ListValue{Elements: []evaluator.Value{&evaluator.IntValue{Value: 1}, &evaluator.IntValue{Value: 2}}},
		}

		rt.RegisterService("db", svc)

		result, err := rt.Execute(`db.query("SELECT * FROM users")`)
		require.NoError(t, err)

		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 2)
		assert.Equal(t, int64(1), list.Elements[0].(*evaluator.IntValue).Value)
		assert.Equal(t, int64(2), list.Elements[1].(*evaluator.IntValue).Value)
	})

	t.Run("chained service call with pipeline", func(t *testing.T) {
		rt := NewRuntime()

		// Service that returns a list
		svc := &MockService{
			name: "data",
			returnValue: &evaluator.ListValue{
				Elements: []evaluator.Value{
					&evaluator.IntValue{Value: 1},
					&evaluator.IntValue{Value: 2},
					&evaluator.IntValue{Value: 3},
					&evaluator.IntValue{Value: 4},
					&evaluator.IntValue{Value: 5},
				},
			},
		}

		rt.RegisterService("data", svc)

		result, err := rt.Execute(`
items = data.getItems()
items | filter(x -> x > 2) | map(x -> x * 10)
`)
		require.NoError(t, err)

		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 3)
		assert.Equal(t, int64(30), list.Elements[0].(*evaluator.IntValue).Value)
		assert.Equal(t, int64(40), list.Elements[1].(*evaluator.IntValue).Value)
		assert.Equal(t, int64(50), list.Elements[2].(*evaluator.IntValue).Value)
	})
}

// CustomLLMClient implements LLMClient for testing
type CustomLLMClient struct {
	responses map[string]*LLMResponse
}

func (c *CustomLLMClient) Complete(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if c.responses != nil {
		if resp, ok := c.responses[request.Prompt]; ok {
			return resp, nil
		}
	}
	// Default response based on schema
	return &LLMResponse{
		Parsed: map[string]any{
			"action": "search",
			"query":  "test query",
		},
	}, nil
}

func TestRuntimeLLMService(t *testing.T) {
	t.Run("basic llm.call with mock", func(t *testing.T) {
		rt := NewRuntime()

		// Test that LLM service is registered by default with mock client
		result, err := rt.Execute(`llm.call(prompt: "What should I do?", schema: {action: "string", query: "string"})`)
		require.NoError(t, err)

		m := result.(*evaluator.MapValue)
		action, ok := m.Get("action")
		require.True(t, ok)
		// Mock returns "mock_string" for string types
		assert.NotNil(t, action)
	})

	t.Run("llm.call with custom client", func(t *testing.T) {
		rt := NewRuntime()

		// Set a custom LLM client
		client := &CustomLLMClient{
			responses: map[string]*LLMResponse{
				"What is 2+2?": {
					Parsed: map[string]any{
						"answer":     "4",
						"confidence": 0.99,
					},
				},
			},
		}
		rt.SetLLMClient(client)

		result, err := rt.Execute(`llm.call(prompt: "What is 2+2?", schema: {answer: "string", confidence: "float"})`)
		require.NoError(t, err)

		m := result.(*evaluator.MapValue)
		answer, ok := m.Get("answer")
		require.True(t, ok)
		assert.Equal(t, "4", answer.(*evaluator.StringValue).Value)
	})

	t.Run("llm.call result used in control flow", func(t *testing.T) {
		rt := NewRuntime()

		// Set a custom LLM client that returns structured decisions
		client := &CustomLLMClient{
			responses: map[string]*LLMResponse{
				"Should I continue?": {
					Parsed: map[string]any{
						"action": "done",
						"result": "All tasks completed",
					},
				},
			},
		}
		rt.SetLLMClient(client)

		result, err := rt.Execute(`
decision = llm.call(prompt: "Should I continue?", schema: {action: "string", result: "string"})

if decision.action == "done":
    output = decision.result
else:
    output = "Still working"

output
`)
		require.NoError(t, err)
		assert.Equal(t, "All tasks completed", result.(*evaluator.StringValue).Value)
	})

	t.Run("llm.call in a loop", func(t *testing.T) {
		rt := NewRuntime()

		// Track call count
		callCount := 0
		client := &CustomLLMClient{
			responses: map[string]*LLMResponse{},
		}

		// Override Complete to track calls and return appropriate values
		rt.SetLLMClient(&trackingLLMClient{
			callCount: &callCount,
		})

		result, err := rt.Execute(`
results = []
for i in range(3):
    result = llm.call(prompt: "Process item", schema: {processed: "bool"})
    results.append(result.processed)
results
`)
		require.NoError(t, err)

		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 3)

		// Each call should return true (from trackingLLMClient)
		for _, elem := range list.Elements {
			assert.True(t, elem.(*evaluator.BoolValue).Value)
		}

		// Verify 3 LLM calls were made
		assert.Equal(t, 3, callCount)

		_ = client // avoid unused warning
	})
}

// trackingLLMClient tracks LLM calls for testing
type trackingLLMClient struct {
	callCount *int
}

func (c *trackingLLMClient) Complete(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	*c.callCount++
	return &LLMResponse{
		Parsed: map[string]any{
			"processed": true,
		},
	}, nil
}

func TestRuntimeLimits(t *testing.T) {
	t.Run("for loop with limit modifier", func(t *testing.T) {
		rt := NewRuntime()

		result, err := rt.Execute(`
count = 0
for i in range(100) with limit(5):
    count = count + 1
count
`)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.(*evaluator.IntValue).Value)
	})

	t.Run("for loop limit on list", func(t *testing.T) {
		rt := NewRuntime()

		result, err := rt.Execute(`
items = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
total = 0
for item in items with limit(3):
    total = total + item
total
`)
		require.NoError(t, err)
		assert.Equal(t, int64(6), result.(*evaluator.IntValue).Value) // 1+2+3
	})

	t.Run("rate limiting enforced", func(t *testing.T) {
		rt := NewRuntime()

		// Use a high rate but verify timing
		start := time.Now()

		result, err := rt.Execute(`
count = 0
for i in range(3) with rate(20):
    count = count + 1
count
`)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result.(*evaluator.IntValue).Value)
		// 3 iterations at 20/sec = 100ms total minimum (2 waits of 50ms each)
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(80))
	})

	t.Run("timeout stops loop", func(t *testing.T) {
		rt := NewRuntime()

		// Loop should stop before completing due to timeout
		// Use rate limiting to slow down the loop so timeout can trigger
		result, err := rt.Execute(`
count = 0
for i in range(1000) with timeout("100ms"), rate(50):
    count = count + 1
count
`)
		require.NoError(t, err)

		// Should have terminated early due to timeout
		// At rate 50/sec, 100ms should allow ~5 iterations
		count := result.(*evaluator.IntValue).Value
		assert.Less(t, count, int64(20)) // With some tolerance
		assert.Greater(t, count, int64(0))
	})

	t.Run("combined limit and rate", func(t *testing.T) {
		rt := NewRuntime()

		start := time.Now()

		result, err := rt.Execute(`
count = 0
for i in range(100) with limit(5), rate(100):
    count = count + 1
count
`)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, int64(5), result.(*evaluator.IntValue).Value)

		// 5 iterations at 100/sec = 40ms minimum (4 waits of 10ms each)
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(30))
	})

	t.Run("global iteration limit", func(t *testing.T) {
		rt := NewRuntimeWithConfig(Config{
			MaxIterations: 10,
		})

		_, err := rt.Execute(`
count = 0
for i in range(100):
    count = count + 1
count
`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "iteration limit")
	})
}
