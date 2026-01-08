package builtin

import (
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch_Basic(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok, "match function should be registered")

	tests := []struct {
		name     string
		text     string
		pattern  string
		expected map[string]interface{}
	}{
		{
			name:    "simple single capture",
			text:    "Hello World",
			pattern: "Hello {name}",
			expected: map[string]interface{}{
				"name": "World",
			},
		},
		{
			name:    "multiple captures",
			text:    "The sum of 5 and 10 is 15.",
			pattern: "The sum of {x} and {y} is {z}.",
			expected: map[string]interface{}{
				"x": "5",
				"y": "10",
				"z": "15",
			},
		},
		{
			name:     "no match returns empty map",
			text:     "Hello World",
			pattern:  "Goodbye {name}",
			expected: map[string]interface{}{},
		},
		{
			name:    "literal only match",
			text:    "Hello World",
			pattern: "Hello World",
			expected: map[string]interface{}{},
		},
		{
			name:    "capture at start",
			text:    "foo is great",
			pattern: "{word} is great",
			expected: map[string]interface{}{
				"word": "foo",
			},
		},
		{
			name:    "capture at end",
			text:    "value: bar",
			pattern: "value: {v}",
			expected: map[string]interface{}{
				"v": "bar",
			},
		},
		{
			name:    "multiple adjacent captures",
			text:    "abc123xyz",
			pattern: "{a}{b}{c}",
			expected: map[string]interface{}{
				"a": "a",       // non-greedy gets minimal
				"b": "b",       // non-greedy gets minimal
				"c": "c123xyz", // last capture gets the rest
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []evaluator.Value{
				&evaluator.StringValue{Value: tt.text},
				&evaluator.StringValue{Value: tt.pattern},
			}
			result, err := fn(args, nil)
			require.NoError(t, err)

			mv, ok := result.(*evaluator.MapValue)
			require.True(t, ok, "result should be a MapValue")

			// Check number of entries
			assert.Equal(t, len(tt.expected), len(mv.Pairs))

			// Check each expected value
			for k, v := range tt.expected {
				actual, exists := mv.Pairs[k]
				require.True(t, exists, "key %s should exist", k)
				sv, ok := actual.(*evaluator.StringValue)
				require.True(t, ok, "value should be StringValue")
				assert.Equal(t, v, sv.Value)
			}
		})
	}
}

func TestMatch_TypeHints(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("int conversion", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "value: 42"},
			&evaluator.StringValue{Value: "value: {n:int}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		n, exists := mv.Pairs["n"]
		require.True(t, exists)
		iv, ok := n.(*evaluator.IntValue)
		require.True(t, ok, "n should be IntValue")
		assert.Equal(t, int64(42), iv.Value)
	})

	t.Run("negative int", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "temp: -5"},
			&evaluator.StringValue{Value: "temp: {t:int}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		iv := mv.Pairs["t"].(*evaluator.IntValue)
		assert.Equal(t, int64(-5), iv.Value)
	})

	t.Run("float conversion", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "pi: 3.14159"},
			&evaluator.StringValue{Value: "pi: {v:float}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		fv := mv.Pairs["v"].(*evaluator.FloatValue)
		assert.InDelta(t, 3.14159, fv.Value, 0.00001)
	})

	t.Run("scientific notation float", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "val: 1.5e10"},
			&evaluator.StringValue{Value: "val: {n:float}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		fv := mv.Pairs["n"].(*evaluator.FloatValue)
		assert.InDelta(t, 1.5e10, fv.Value, 1e5)
	})

	t.Run("word extraction", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "name: John Doe"},
			&evaluator.StringValue{Value: "name: {first:word} {last:word}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "John", mv.Pairs["first"].(*evaluator.StringValue).Value)
		assert.Equal(t, "Doe", mv.Pairs["last"].(*evaluator.StringValue).Value)
	})

	t.Run("rest extraction (greedy)", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "prefix: everything else here"},
			&evaluator.StringValue{Value: "prefix: {content:rest}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "everything else here", mv.Pairs["content"].(*evaluator.StringValue).Value)
	})
}

func TestMatch_NamedPatterns(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("IP address", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "server: 192.168.1.1"},
			&evaluator.StringValue{Value: "server: {addr:IP}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "192.168.1.1", mv.Pairs["addr"].(*evaluator.StringValue).Value)
	})

	t.Run("email address", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "contact: user@example.com"},
			&evaluator.StringValue{Value: "contact: {e:email}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "user@example.com", mv.Pairs["e"].(*evaluator.StringValue).Value)
	})

	t.Run("UUID", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "id: 550e8400-e29b-41d4-a716-446655440000"},
			&evaluator.StringValue{Value: "id: {id:uuid}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", mv.Pairs["id"].(*evaluator.StringValue).Value)
	})

	t.Run("ISO8601 timestamp", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "2024-01-15T10:30:00Z [INFO] message"},
			&evaluator.StringValue{Value: "{ts:iso8601} [{level:word}] {msg:rest}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "2024-01-15T10:30:00Z", mv.Pairs["ts"].(*evaluator.StringValue).Value)
		assert.Equal(t, "INFO", mv.Pairs["level"].(*evaluator.StringValue).Value)
		assert.Equal(t, "message", mv.Pairs["msg"].(*evaluator.StringValue).Value)
	})

	t.Run("URL", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "link: https://example.com/path?q=1"},
			&evaluator.StringValue{Value: "link: {url:url}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "https://example.com/path?q=1", mv.Pairs["url"].(*evaluator.StringValue).Value)
	})
}

func TestMatch_Optional(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("optional present", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "error: 404 Not Found"},
			&evaluator.StringValue{Value: "error: {code:int} {?message}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, int64(404), mv.Pairs["code"].(*evaluator.IntValue).Value)
		assert.Equal(t, "Not Found", mv.Pairs["message"].(*evaluator.StringValue).Value)
	})

	t.Run("optional absent", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "error: 404"},
			&evaluator.StringValue{Value: "error: {code:int}{?message}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, int64(404), mv.Pairs["code"].(*evaluator.IntValue).Value)
		_, exists := mv.Pairs["message"]
		assert.False(t, exists, "optional field should not be present when not matched")
	})
}

func TestMatch_JSON(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("json object", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: `data: {"key": "value"}`},
			&evaluator.StringValue{Value: "data: {obj:json}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		jsonVal, exists := mv.Pairs["obj"]
		require.True(t, exists)

		objMap, ok := jsonVal.(*evaluator.MapValue)
		require.True(t, ok, "json should parse to MapValue")
		assert.Equal(t, "value", objMap.Pairs["key"].(*evaluator.StringValue).Value)
	})

	t.Run("json array", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "items: [1, 2, 3]"},
			&evaluator.StringValue{Value: "items: {arr:json}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		arrVal, exists := mv.Pairs["arr"]
		require.True(t, exists)

		listVal, ok := arrVal.(*evaluator.ListValue)
		require.True(t, ok, "json array should parse to ListValue")
		assert.Len(t, listVal.Elements, 3)
	})
}

func TestMatch_StrictMode(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("strict mode - no match error", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "Hello"},
			&evaluator.StringValue{Value: "Goodbye {name}"},
		}
		kwargs := map[string]evaluator.Value{
			"strict": evaluator.TRUE,
		}
		_, err := fn(args, kwargs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not match")
	})

	t.Run("strict mode - validation error", func(t *testing.T) {
		// IP pattern regex matches "999.999.999.999" but validator fails (octet > 255)
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "ip: 999.999.999.999"},
			&evaluator.StringValue{Value: "ip: {addr:IP}"},
		}
		kwargs := map[string]evaluator.Value{
			"strict": evaluator.TRUE,
		}
		_, err := fn(args, kwargs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "conversion error")
	})

	t.Run("non-strict mode - validation fallback to string", func(t *testing.T) {
		// IP pattern regex matches "999.999.999.999" but validator fails
		// In non-strict mode, should fall back to string value
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "ip: 999.999.999.999"},
			&evaluator.StringValue{Value: "ip: {addr:IP}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		// Should fall back to string when validation fails
		sv := mv.Pairs["addr"].(*evaluator.StringValue)
		assert.Equal(t, "999.999.999.999", sv.Value)
	})

	t.Run("int pattern does not match non-digits", func(t *testing.T) {
		// When type hint is used, the regex must match - "abc" won't match int pattern
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "value: abc"},
			&evaluator.StringValue{Value: "value: {n:int}"},
		}
		kwargs := map[string]evaluator.Value{
			"strict": evaluator.TRUE,
		}
		_, err := fn(args, kwargs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not match") // Match failure, not conversion
	})
}

func TestMatch_EscapedBraces(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("escaped opening brace", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "use {name} for vars"},
			&evaluator.StringValue{Value: "use {{name}} for vars"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Len(t, mv.Pairs, 0, "escaped braces should not create captures")
	})

	t.Run("mixed escaped and captures", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "value is {42}"},
			&evaluator.StringValue{Value: "value is {{{n:int}}}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, int64(42), mv.Pairs["n"].(*evaluator.IntValue).Value)
	})
}

func TestMatch_PatternErrors(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("unclosed brace", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "text"},
			&evaluator.StringValue{Value: "{unclosed"},
		}
		_, err := fn(args, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unclosed")
	})

	t.Run("empty placeholder name", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "text"},
			&evaluator.StringValue{Value: "value: {}"},
		}
		_, err := fn(args, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("invalid placeholder name", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "text"},
			&evaluator.StringValue{Value: "value: {my-name}"},
		}
		_, err := fn(args, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

func TestMatch_EdgeCases(t *testing.T) {
	r := NewRegistry()
	fn, ok := r.Get("match")
	require.True(t, ok)

	t.Run("empty text", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: ""},
			&evaluator.StringValue{Value: "{content}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Len(t, mv.Pairs, 0, "empty text should return empty map")
	})

	t.Run("empty pattern", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: ""},
			&evaluator.StringValue{Value: ""},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Len(t, mv.Pairs, 0, "empty pattern with empty text should match but have no captures")
	})

	t.Run("unicode text", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "name: test"},
			&evaluator.StringValue{Value: "name: {n}"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "test", mv.Pairs["n"].(*evaluator.StringValue).Value)
	})

	t.Run("special regex characters in literal", func(t *testing.T) {
		args := []evaluator.Value{
			&evaluator.StringValue{Value: "file.txt (1)"},
			&evaluator.StringValue{Value: "{name}.txt ({num:int})"},
		}
		result, err := fn(args, nil)
		require.NoError(t, err)

		mv := result.(*evaluator.MapValue)
		assert.Equal(t, "file", mv.Pairs["name"].(*evaluator.StringValue).Value)
		assert.Equal(t, int64(1), mv.Pairs["num"].(*evaluator.IntValue).Value)
	})
}

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantCount int
		wantErr   bool
	}{
		{"empty", "", 0, false},
		{"literal only", "hello world", 1, false},
		{"single placeholder", "{name}", 1, false},
		{"multiple placeholders", "{a} and {b}", 3, false},
		{"placeholder with type", "{x:int}", 1, false},
		{"optional placeholder", "{?opt}", 1, false},
		{"escaped braces", "{{name}}", 2, false},
		{"unclosed brace", "{name", 0, true},
		{"empty name", "{}", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := parseTemplate(tt.template)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, tokens, tt.wantCount)
			}
		})
	}
}

func TestCompileTemplate(t *testing.T) {
	t.Run("basic compilation", func(t *testing.T) {
		tokens, err := parseTemplate("Hello {name}")
		require.NoError(t, err)

		compiled, err := compileTemplate(tokens)
		require.NoError(t, err)
		assert.NotNil(t, compiled.Regex)
		assert.Len(t, compiled.Names, 1)
		assert.Equal(t, "name", compiled.Names[0])
	})

	t.Run("regex escaping", func(t *testing.T) {
		tokens, err := parseTemplate("file.txt ({n:int})")
		require.NoError(t, err)

		compiled, err := compileTemplate(tokens)
		require.NoError(t, err)

		// Should match literal dots and parentheses
		assert.True(t, compiled.Regex.MatchString("file.txt (42)"))
		assert.False(t, compiled.Regex.MatchString("filextxt (42)"))
	})
}
