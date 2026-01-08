package builtin

import (
	"strings"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)

	// Check that common functions are registered
	names := r.Names()
	assert.Contains(t, names, "int")
	assert.Contains(t, names, "float")
	assert.Contains(t, names, "str")
	assert.Contains(t, names, "len")
	assert.Contains(t, names, "print")
	assert.Contains(t, names, "range")
}

func TestTypeConversion(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		fn       string
		args     []evaluator.Value
		expected evaluator.Value
	}{
		// int()
		{"int from int", "int", []evaluator.Value{&evaluator.IntValue{Value: 42}}, &evaluator.IntValue{Value: 42}},
		{"int from float", "int", []evaluator.Value{&evaluator.FloatValue{Value: 3.7}}, &evaluator.IntValue{Value: 3}},
		{"int from string", "int", []evaluator.Value{&evaluator.StringValue{Value: "123"}}, &evaluator.IntValue{Value: 123}},
		{"int from true", "int", []evaluator.Value{evaluator.TRUE}, &evaluator.IntValue{Value: 1}},
		{"int from false", "int", []evaluator.Value{evaluator.FALSE}, &evaluator.IntValue{Value: 0}},

		// float()
		{"float from float", "float", []evaluator.Value{&evaluator.FloatValue{Value: 3.14}}, &evaluator.FloatValue{Value: 3.14}},
		{"float from int", "float", []evaluator.Value{&evaluator.IntValue{Value: 42}}, &evaluator.FloatValue{Value: 42.0}},
		{"float from string", "float", []evaluator.Value{&evaluator.StringValue{Value: "3.14"}}, &evaluator.FloatValue{Value: 3.14}},

		// str()
		{"str from int", "str", []evaluator.Value{&evaluator.IntValue{Value: 42}}, &evaluator.StringValue{Value: "42"}},
		{"str from float", "str", []evaluator.Value{&evaluator.FloatValue{Value: 3.14}}, &evaluator.StringValue{Value: "3.14"}},
		{"str from string", "str", []evaluator.Value{&evaluator.StringValue{Value: "hello"}}, &evaluator.StringValue{Value: "hello"}},

		// bool()
		{"bool from true", "bool", []evaluator.Value{evaluator.TRUE}, evaluator.TRUE},
		{"bool from false", "bool", []evaluator.Value{evaluator.FALSE}, evaluator.FALSE},
		{"bool from 0", "bool", []evaluator.Value{&evaluator.IntValue{Value: 0}}, evaluator.FALSE},
		{"bool from 1", "bool", []evaluator.Value{&evaluator.IntValue{Value: 1}}, evaluator.TRUE},
		{"bool from empty string", "bool", []evaluator.Value{&evaluator.StringValue{Value: ""}}, evaluator.FALSE},
		{"bool from string", "bool", []evaluator.Value{&evaluator.StringValue{Value: "hello"}}, evaluator.TRUE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := r.Get(tt.fn)
			require.True(t, ok)
			result, err := fn(tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeChecking(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		fn       string
		arg      evaluator.Value
		expected bool
	}{
		{"is_none with none", "is_none", evaluator.NONE, true},
		{"is_none with int", "is_none", &evaluator.IntValue{Value: 1}, false},
		{"is_bool with true", "is_bool", evaluator.TRUE, true},
		{"is_bool with int", "is_bool", &evaluator.IntValue{Value: 1}, false},
		{"is_int with int", "is_int", &evaluator.IntValue{Value: 1}, true},
		{"is_int with float", "is_int", &evaluator.FloatValue{Value: 1.0}, false},
		{"is_float with float", "is_float", &evaluator.FloatValue{Value: 1.0}, true},
		{"is_number with int", "is_number", &evaluator.IntValue{Value: 1}, true},
		{"is_number with float", "is_number", &evaluator.FloatValue{Value: 1.0}, true},
		{"is_number with string", "is_number", &evaluator.StringValue{Value: "1"}, false},
		{"is_string with string", "is_string", &evaluator.StringValue{Value: "hello"}, true},
		{"is_list with list", "is_list", &evaluator.ListValue{Elements: []evaluator.Value{}}, true},
		{"is_map with map", "is_map", &evaluator.MapValue{Pairs: map[string]evaluator.Value{}}, true},
		{"is_set with set", "is_set", &evaluator.SetValue{Elements: map[string]evaluator.Value{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := r.Get(tt.fn)
			require.True(t, ok)
			result, err := fn([]evaluator.Value{tt.arg}, nil)
			require.NoError(t, err)
			if tt.expected {
				assert.Equal(t, evaluator.TRUE, result)
			} else {
				assert.Equal(t, evaluator.FALSE, result)
			}
		})
	}
}

func TestMathFunctions(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		fn       string
		args     []evaluator.Value
		expected evaluator.Value
	}{
		{"abs of positive", "abs", []evaluator.Value{&evaluator.IntValue{Value: 5}}, &evaluator.IntValue{Value: 5}},
		{"abs of negative", "abs", []evaluator.Value{&evaluator.IntValue{Value: -5}}, &evaluator.IntValue{Value: 5}},
		{"min of list", "min", []evaluator.Value{&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 3}, &evaluator.IntValue{Value: 1}, &evaluator.IntValue{Value: 2},
		}}}, &evaluator.IntValue{Value: 1}},
		{"max of list", "max", []evaluator.Value{&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 3}, &evaluator.IntValue{Value: 1}, &evaluator.IntValue{Value: 2},
		}}}, &evaluator.IntValue{Value: 3}},
		{"sum of list", "sum", []evaluator.Value{&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 1}, &evaluator.IntValue{Value: 2}, &evaluator.IntValue{Value: 3},
		}}}, &evaluator.IntValue{Value: 6}},
		{"pow", "pow", []evaluator.Value{&evaluator.IntValue{Value: 2}, &evaluator.IntValue{Value: 3}}, &evaluator.IntValue{Value: 8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := r.Get(tt.fn)
			require.True(t, ok)
			result, err := fn(tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringFunctions(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		fn       string
		args     []evaluator.Value
		expected evaluator.Value
	}{
		{"upper", "upper", []evaluator.Value{&evaluator.StringValue{Value: "hello"}}, &evaluator.StringValue{Value: "HELLO"}},
		{"lower", "lower", []evaluator.Value{&evaluator.StringValue{Value: "HELLO"}}, &evaluator.StringValue{Value: "hello"}},
		{"strip", "strip", []evaluator.Value{&evaluator.StringValue{Value: "  hello  "}}, &evaluator.StringValue{Value: "hello"}},
		{"startswith true", "startswith", []evaluator.Value{
			&evaluator.StringValue{Value: "hello world"},
			&evaluator.StringValue{Value: "hello"},
		}, evaluator.TRUE},
		{"startswith false", "startswith", []evaluator.Value{
			&evaluator.StringValue{Value: "hello world"},
			&evaluator.StringValue{Value: "world"},
		}, evaluator.FALSE},
		{"endswith true", "endswith", []evaluator.Value{
			&evaluator.StringValue{Value: "hello world"},
			&evaluator.StringValue{Value: "world"},
		}, evaluator.TRUE},
		{"contains true", "contains", []evaluator.Value{
			&evaluator.StringValue{Value: "hello world"},
			&evaluator.StringValue{Value: "lo wo"},
		}, evaluator.TRUE},
		{"repeat", "repeat", []evaluator.Value{
			&evaluator.StringValue{Value: "ab"},
			&evaluator.IntValue{Value: 3},
		}, &evaluator.StringValue{Value: "ababab"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := r.Get(tt.fn)
			require.True(t, ok)
			result, err := fn(tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplit(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("split")
	require.True(t, ok)

	result, err := fn([]evaluator.Value{
		&evaluator.StringValue{Value: "a,b,c"},
		&evaluator.StringValue{Value: ","},
	}, nil)
	require.NoError(t, err)

	list, ok := result.(*evaluator.ListValue)
	require.True(t, ok)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, "a", list.Elements[0].(*evaluator.StringValue).Value)
	assert.Equal(t, "b", list.Elements[1].(*evaluator.StringValue).Value)
	assert.Equal(t, "c", list.Elements[2].(*evaluator.StringValue).Value)
}

func TestJoin(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("join")
	require.True(t, ok)

	result, err := fn([]evaluator.Value{
		&evaluator.StringValue{Value: ","},
		&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.StringValue{Value: "a"},
			&evaluator.StringValue{Value: "b"},
			&evaluator.StringValue{Value: "c"},
		}},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "a,b,c", result.(*evaluator.StringValue).Value)
}

func TestLen(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("len")
	require.True(t, ok)

	tests := []struct {
		name     string
		arg      evaluator.Value
		expected int64
	}{
		{"string", &evaluator.StringValue{Value: "hello"}, 5},
		{"list", &evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 1},
			&evaluator.IntValue{Value: 2},
		}}, 2},
		{"map", &evaluator.MapValue{Pairs: map[string]evaluator.Value{
			"a": &evaluator.IntValue{Value: 1},
			"b": &evaluator.IntValue{Value: 2},
		}}, 2},
		{"empty list", &evaluator.ListValue{Elements: []evaluator.Value{}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fn([]evaluator.Value{tt.arg}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.(*evaluator.IntValue).Value)
		})
	}
}

func TestRange(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("range")
	require.True(t, ok)

	// range(5)
	result, err := fn([]evaluator.Value{&evaluator.IntValue{Value: 5}}, nil)
	require.NoError(t, err)
	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, int64(i), list.Elements[i].(*evaluator.IntValue).Value)
	}

	// range(2, 5)
	result, err = fn([]evaluator.Value{
		&evaluator.IntValue{Value: 2},
		&evaluator.IntValue{Value: 5},
	}, nil)
	require.NoError(t, err)
	list = result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)
	assert.Equal(t, int64(2), list.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(3), list.Elements[1].(*evaluator.IntValue).Value)
	assert.Equal(t, int64(4), list.Elements[2].(*evaluator.IntValue).Value)

	// range(0, 10, 2)
	result, err = fn([]evaluator.Value{
		&evaluator.IntValue{Value: 0},
		&evaluator.IntValue{Value: 10},
		&evaluator.IntValue{Value: 2},
	}, nil)
	require.NoError(t, err)
	list = result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, int64(i*2), list.Elements[i].(*evaluator.IntValue).Value)
	}
}

func TestEnumerate(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("enumerate")
	require.True(t, ok)

	result, err := fn([]evaluator.Value{
		&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.StringValue{Value: "a"},
			&evaluator.StringValue{Value: "b"},
			&evaluator.StringValue{Value: "c"},
		}},
	}, nil)
	require.NoError(t, err)

	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 3)

	pair0 := list.Elements[0].(*evaluator.ListValue)
	assert.Equal(t, int64(0), pair0.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, "a", pair0.Elements[1].(*evaluator.StringValue).Value)

	pair1 := list.Elements[1].(*evaluator.ListValue)
	assert.Equal(t, int64(1), pair1.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, "b", pair1.Elements[1].(*evaluator.StringValue).Value)
}

func TestZip(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("zip")
	require.True(t, ok)

	result, err := fn([]evaluator.Value{
		&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 1},
			&evaluator.IntValue{Value: 2},
		}},
		&evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.StringValue{Value: "a"},
			&evaluator.StringValue{Value: "b"},
		}},
	}, nil)
	require.NoError(t, err)

	list := result.(*evaluator.ListValue)
	require.Len(t, list.Elements, 2)

	pair0 := list.Elements[0].(*evaluator.ListValue)
	assert.Equal(t, int64(1), pair0.Elements[0].(*evaluator.IntValue).Value)
	assert.Equal(t, "a", pair0.Elements[1].(*evaluator.StringValue).Value)
}

func TestListFunctions(t *testing.T) {
	r := NewRegistry()

	t.Run("first", func(t *testing.T) {
		fn, _ := r.Get("first")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.IntValue{Value: 1},
				&evaluator.IntValue{Value: 2},
			}},
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.(*evaluator.IntValue).Value)
	})

	t.Run("last", func(t *testing.T) {
		fn, _ := r.Get("last")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.IntValue{Value: 1},
				&evaluator.IntValue{Value: 2},
			}},
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.(*evaluator.IntValue).Value)
	})

	t.Run("reversed", func(t *testing.T) {
		fn, _ := r.Get("reversed")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.IntValue{Value: 1},
				&evaluator.IntValue{Value: 2},
				&evaluator.IntValue{Value: 3},
			}},
		}, nil)
		require.NoError(t, err)
		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 3)
		assert.Equal(t, int64(3), list.Elements[0].(*evaluator.IntValue).Value)
		assert.Equal(t, int64(2), list.Elements[1].(*evaluator.IntValue).Value)
		assert.Equal(t, int64(1), list.Elements[2].(*evaluator.IntValue).Value)
	})

	t.Run("flatten", func(t *testing.T) {
		fn, _ := r.Get("flatten")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.ListValue{Elements: []evaluator.Value{
					&evaluator.IntValue{Value: 1},
					&evaluator.IntValue{Value: 2},
				}},
				&evaluator.ListValue{Elements: []evaluator.Value{
					&evaluator.IntValue{Value: 3},
					&evaluator.IntValue{Value: 4},
				}},
			}},
		}, nil)
		require.NoError(t, err)
		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 4)
	})
}

func TestMapFunctions(t *testing.T) {
	r := NewRegistry()

	m := &evaluator.MapValue{Pairs: map[string]evaluator.Value{
		"a": &evaluator.IntValue{Value: 1},
		"b": &evaluator.IntValue{Value: 2},
	}}

	t.Run("keys", func(t *testing.T) {
		fn, _ := r.Get("keys")
		result, err := fn([]evaluator.Value{m}, nil)
		require.NoError(t, err)
		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 2)
	})

	t.Run("values", func(t *testing.T) {
		fn, _ := r.Get("values")
		result, err := fn([]evaluator.Value{m}, nil)
		require.NoError(t, err)
		list := result.(*evaluator.ListValue)
		require.Len(t, list.Elements, 2)
	})

	t.Run("get with default", func(t *testing.T) {
		fn, _ := r.Get("get")
		result, err := fn([]evaluator.Value{
			m,
			&evaluator.StringValue{Value: "c"},
			&evaluator.IntValue{Value: 99},
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(99), result.(*evaluator.IntValue).Value)
	})

	t.Run("has_key", func(t *testing.T) {
		fn, _ := r.Get("has_key")
		result, err := fn([]evaluator.Value{m, &evaluator.StringValue{Value: "a"}}, nil)
		require.NoError(t, err)
		assert.Equal(t, evaluator.TRUE, result)

		result, err = fn([]evaluator.Value{m, &evaluator.StringValue{Value: "c"}}, nil)
		require.NoError(t, err)
		assert.Equal(t, evaluator.FALSE, result)
	})
}

func TestUtilityFunctions(t *testing.T) {
	r := NewRegistry()

	t.Run("json_stringify and json_parse", func(t *testing.T) {
		stringify, _ := r.Get("json_stringify")
		parse, _ := r.Get("json_parse")

		data := &evaluator.MapValue{Pairs: map[string]evaluator.Value{
			"name": &evaluator.StringValue{Value: "test"},
			"age":  &evaluator.IntValue{Value: 42},
		}}

		jsonStr, err := stringify([]evaluator.Value{data}, nil)
		require.NoError(t, err)
		assert.Contains(t, jsonStr.(*evaluator.StringValue).Value, "name")

		parsed, err := parse([]evaluator.Value{jsonStr}, nil)
		require.NoError(t, err)
		m := parsed.(*evaluator.MapValue)
		assert.Equal(t, "test", m.Pairs["name"].(*evaluator.StringValue).Value)
	})

	t.Run("base64", func(t *testing.T) {
		encode, _ := r.Get("base64_encode")
		decode, _ := r.Get("base64_decode")

		encoded, err := encode([]evaluator.Value{&evaluator.StringValue{Value: "hello"}}, nil)
		require.NoError(t, err)
		assert.Equal(t, "aGVsbG8=", encoded.(*evaluator.StringValue).Value)

		decoded, err := decode([]evaluator.Value{encoded}, nil)
		require.NoError(t, err)
		assert.Equal(t, "hello", decoded.(*evaluator.StringValue).Value)
	})

	t.Run("hash_sha256", func(t *testing.T) {
		fn, _ := r.Get("hash_sha256")
		result, err := fn([]evaluator.Value{&evaluator.StringValue{Value: "hello"}}, nil)
		require.NoError(t, err)
		assert.Len(t, result.(*evaluator.StringValue).Value, 64) // SHA256 produces 64 hex chars
	})

	t.Run("regex_test", func(t *testing.T) {
		fn, _ := r.Get("regex_test")
		result, err := fn([]evaluator.Value{
			&evaluator.StringValue{Value: `\d+`},
			&evaluator.StringValue{Value: "abc123"},
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, evaluator.TRUE, result)

		result, err = fn([]evaluator.Value{
			&evaluator.StringValue{Value: `\d+`},
			&evaluator.StringValue{Value: "abc"},
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, evaluator.FALSE, result)
	})
}

func TestValidation(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		fn       string
		input    string
		expected bool
	}{
		{"valid email", "validate_email", "test@example.com", true},
		{"invalid email", "validate_email", "not-an-email", false},
		{"valid url", "validate_url", "https://example.com", true},
		{"invalid url", "validate_url", "not-a-url", false},
		{"valid uuid", "validate_uuid", "550e8400-e29b-41d4-a716-446655440000", true},
		{"invalid uuid", "validate_uuid", "not-a-uuid", false},
		{"valid json", "validate_json", `{"key": "value"}`, true},
		{"invalid json", "validate_json", "not json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := r.Get(tt.fn)
			require.True(t, ok)
			result, err := fn([]evaluator.Value{&evaluator.StringValue{Value: tt.input}}, nil)
			require.NoError(t, err)
			if tt.expected {
				assert.Equal(t, evaluator.TRUE, result)
			} else {
				assert.Equal(t, evaluator.FALSE, result)
			}
		})
	}
}

func TestAssertions(t *testing.T) {
	r := NewRegistry()

	t.Run("assert passes", func(t *testing.T) {
		fn, _ := r.Get("assert")
		_, err := fn([]evaluator.Value{evaluator.TRUE}, nil)
		assert.NoError(t, err)
	})

	t.Run("assert fails", func(t *testing.T) {
		fn, _ := r.Get("assert")
		_, err := fn([]evaluator.Value{evaluator.FALSE}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "AssertionError")
	})

	t.Run("assert_eq passes", func(t *testing.T) {
		fn, _ := r.Get("assert_eq")
		_, err := fn([]evaluator.Value{
			&evaluator.IntValue{Value: 42},
			&evaluator.IntValue{Value: 42},
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("assert_eq fails", func(t *testing.T) {
		fn, _ := r.Get("assert_eq")
		_, err := fn([]evaluator.Value{
			&evaluator.IntValue{Value: 1},
			&evaluator.IntValue{Value: 2},
		}, nil)
		assert.Error(t, err)
	})
}

func TestRandomFunctions(t *testing.T) {
	r := NewRegistry()

	t.Run("random_int", func(t *testing.T) {
		fn, _ := r.Get("random_int")
		result, err := fn([]evaluator.Value{
			&evaluator.IntValue{Value: 1},
			&evaluator.IntValue{Value: 10},
		}, nil)
		require.NoError(t, err)
		v := result.(*evaluator.IntValue).Value
		assert.GreaterOrEqual(t, v, int64(1))
		assert.LessOrEqual(t, v, int64(10))
	})

	t.Run("random_choice", func(t *testing.T) {
		fn, _ := r.Get("random_choice")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.StringValue{Value: "a"},
				&evaluator.StringValue{Value: "b"},
				&evaluator.StringValue{Value: "c"},
			}},
		}, nil)
		require.NoError(t, err)
		s := result.(*evaluator.StringValue).Value
		assert.Contains(t, []string{"a", "b", "c"}, s)
	})

	t.Run("random_shuffle", func(t *testing.T) {
		fn, _ := r.Get("random_shuffle")
		result, err := fn([]evaluator.Value{
			&evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.IntValue{Value: 1},
				&evaluator.IntValue{Value: 2},
				&evaluator.IntValue{Value: 3},
			}},
		}, nil)
		require.NoError(t, err)
		list := result.(*evaluator.ListValue)
		assert.Len(t, list.Elements, 3)
	})
}

func TestGenerators(t *testing.T) {
	r := NewRegistry()

	t.Run("gen_name", func(t *testing.T) {
		fn, _ := r.Get("gen_name")
		result, err := fn([]evaluator.Value{}, nil)
		require.NoError(t, err)
		name := result.(*evaluator.StringValue).Value
		assert.Contains(t, name, " ") // First and last name separated by space
	})

	t.Run("gen_email", func(t *testing.T) {
		fn, _ := r.Get("gen_email")
		result, err := fn([]evaluator.Value{}, nil)
		require.NoError(t, err)
		email := result.(*evaluator.StringValue).Value
		assert.Contains(t, email, "@")
	})

	t.Run("gen_color", func(t *testing.T) {
		fn, _ := r.Get("gen_color")
		result, err := fn([]evaluator.Value{}, nil)
		require.NoError(t, err)
		color := result.(*evaluator.StringValue).Value
		assert.True(t, strings.HasPrefix(color, "#"))
		assert.Len(t, color, 7)
	})
}
