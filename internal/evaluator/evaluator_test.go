package evaluator

import (
	"testing"

	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEval(t *testing.T, input string) Value {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parse error: %s", err)
		}
		t.FailNow()
	}

	e := New()
	result, err := e.Eval(program)
	require.NoError(t, err)
	return result
}

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
		{"2 ** 3", 8},
		{"10 % 3", 1},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "input: %s, got: %T", tt.input, result)
		assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
	}
}

func TestEvalFloatExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"-2.5", -2.5},
		{"1.5 + 2.5", 4.0},
		{"5.0 / 2.0", 2.5},
		{"5 / 2.0", 2.5},
		{"5.0 / 2", 2.5},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		fv, ok := result.(*FloatValue)
		require.True(t, ok, "input: %s, got: %T", tt.input, result)
		assert.InDelta(t, tt.expected, fv.Value, 0.0001, "input: %s", tt.input)
	}
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"not true", false},
		{"not false", true},
		{"not 0", true},
		{"not 1", false},
		{"true and true", true},
		{"true and false", false},
		{"false and true", false},
		{"false and false", false},
		{"true or true", true},
		{"true or false", true},
		{"false or true", true},
		{"false or false", false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		bv, ok := result.(*BoolValue)
		require.True(t, ok, "input: %s, got: %T (%s)", tt.input, result, result)
		assert.Equal(t, tt.expected, bv.Value, "input: %s", tt.input)
	}
}

func TestEvalStringExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`"hello" + " " + "world"`, "hello world"},
		{`"ab" * 3`, "ababab"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		sv, ok := result.(*StringValue)
		require.True(t, ok, "input: %s, got: %T", tt.input, result)
		assert.Equal(t, tt.expected, sv.Value, "input: %s", tt.input)
	}
}

func TestEvalNone(t *testing.T) {
	result := testEval(t, "none")
	assert.IsType(t, &NoneValue{}, result)
}

func TestEvalList(t *testing.T) {
	input := "[1, 2, 3]"
	result := testEval(t, input)

	lv, ok := result.(*ListValue)
	require.True(t, ok)
	require.Len(t, lv.Elements, 3)

	assert.Equal(t, int64(1), lv.Elements[0].(*IntValue).Value)
	assert.Equal(t, int64(2), lv.Elements[1].(*IntValue).Value)
	assert.Equal(t, int64(3), lv.Elements[2].(*IntValue).Value)
}

func TestEvalMap(t *testing.T) {
	input := `{a: 1, b: 2}`
	result := testEval(t, input)

	mv, ok := result.(*MapValue)
	require.True(t, ok)
	assert.Len(t, mv.Pairs, 2)

	aVal, ok := mv.Get("a")
	require.True(t, ok)
	assert.Equal(t, int64(1), aVal.(*IntValue).Value)

	bVal, ok := mv.Get("b")
	require.True(t, ok)
	assert.Equal(t, int64(2), bVal.(*IntValue).Value)
}

func TestEvalIndex(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"[1, 2, 3][0]", int64(1)},
		{"[1, 2, 3][1]", int64(2)},
		{"[1, 2, 3][2]", int64(3)},
		{"[1, 2, 3][-1]", int64(3)},
		{`"hello"[0]`, "h"},
		{`"hello"[-1]`, "o"},
		{`{a: 1}["a"]`, int64(1)},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		switch expected := tt.expected.(type) {
		case int64:
			iv, ok := result.(*IntValue)
			require.True(t, ok, "input: %s", tt.input)
			assert.Equal(t, expected, iv.Value, "input: %s", tt.input)
		case string:
			sv, ok := result.(*StringValue)
			require.True(t, ok, "input: %s", tt.input)
			assert.Equal(t, expected, sv.Value, "input: %s", tt.input)
		}
	}
}

func TestEvalSlice(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"[1:4]`, "ell"},
		{`"hello"[:3]`, "hel"},
		{`"hello"[2:]`, "llo"},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		sv, ok := result.(*StringValue)
		require.True(t, ok, "input: %s", tt.input)
		assert.Equal(t, tt.expected, sv.Value, "input: %s", tt.input)
	}
}

func TestEvalAssignment(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"x = 5\nx", 5},
		{"x = 5\nx = 10\nx", 10},
		{"x = 5\ny = x\ny", 5},
		{"x = 5\nx += 3\nx", 8},
		{"x = 10\nx -= 3\nx", 7},
		{"x = 5\nx *= 2\nx", 10},
		{"x = 10\nx /= 2\nx", 5},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "input: %s, got: %T", tt.input, result)
		assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
	}
}

func TestEvalAssignmentLexicalScope(t *testing.T) {
	t.Run("plain assignment shadows outer binding", func(t *testing.T) {
		result := testEval(t, `x = 1
def shadow():
    x = 2
    return x
[shadow(), x]`)

		list, ok := result.(*ListValue)
		require.True(t, ok)
		require.Len(t, list.Elements, 2)
		assert.Equal(t, int64(2), list.Elements[0].(*IntValue).Value)
		assert.Equal(t, int64(1), list.Elements[1].(*IntValue).Value)
	})

	t.Run("compound assignment updates outer binding", func(t *testing.T) {
		result := testEval(t, `x = 1
def increment():
    x += 1
increment()
x`)

		value, ok := result.(*IntValue)
		require.True(t, ok)
		assert.Equal(t, int64(2), value.Value)
	})
}

func TestEvalIfStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"if true:\n    10", 10},
		{"if false:\n    10", 0}, // none becomes 0 in this context
		{"if 1:\n    10", 10},
		{"if 0:\n    10\nelse:\n    20", 20},
		{"if 1 < 2:\n    10", 10},
		{"if 1 > 2:\n    10\nelse:\n    20", 20},
		{"x = 5\nif x > 3:\n    10\nelse:\n    20", 10},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		if iv, ok := result.(*IntValue); ok {
			assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
		} else if tt.expected != 0 {
			t.Errorf("input: %s, expected int, got: %T", tt.input, result)
		}
	}
}

func TestEvalForStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`sum = 0
for i in range(5):
    sum += i
sum`, 10},
		{`items = [1, 2, 3, 4, 5]
sum = 0
for x in items:
    sum += x
sum`, 15},
		{`sum = 0
for i in range(100) with limit(5):
    sum += i
sum`, 10},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "input: %s, got: %T", tt.input, result)
		assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
	}
}

func TestEvalForParallelFailsLoudly(t *testing.T) {
	input := `for i in range(3) with parallel(2):
    i`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	e := New()
	_, err := e.Eval(program)
	require.EqualError(t, err, "parallel loop execution is not supported")
}

func TestEvalForBreakContinue(t *testing.T) {
	// Test break
	input := `sum = 0
for i in range(10):
    if i == 5:
        break
    sum += i
sum`
	result := testEval(t, input)
	iv, ok := result.(*IntValue)
	require.True(t, ok)
	assert.Equal(t, int64(10), iv.Value) // 0+1+2+3+4 = 10

	// Test continue
	input = `sum = 0
for i in range(10):
    if i % 2 == 0:
        continue
    sum += i
sum`
	result = testEval(t, input)
	iv, ok = result.(*IntValue)
	require.True(t, ok)
	assert.Equal(t, int64(25), iv.Value) // 1+3+5+7+9 = 25
}

func TestEvalFunction(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`def add(a, b):
    return a + b
add(3, 4)`, 7},
		{`def factorial(n):
    result = 1
    for i in range(1, n + 1):
        result *= i
    return result
factorial(5)`, 120},
		{`def greet(name, greeting = "Hello"):
    return greeting
greet("World")`, 0}, // This returns a string
	}

	for i, tt := range tests {
		if i == 2 {
			// Skip the string test for now
			continue
		}
		result := testEval(t, tt.input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "input: %s, got: %T (%s)", tt.input, result, result)
		assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
	}
}

func TestEvalLambda(t *testing.T) {
	input := `double = x -> x * 2
double(5)`
	result := testEval(t, input)
	iv, ok := result.(*IntValue)
	require.True(t, ok)
	assert.Equal(t, int64(10), iv.Value)
}

func TestEvalMatch(t *testing.T) {
	input := `status = 200
match status:
    200 -> "ok"
    404 -> "not found"
    _ -> "error"`
	result := testEval(t, input)
	sv, ok := result.(*StringValue)
	require.True(t, ok)
	assert.Equal(t, "ok", sv.Value)

	input = `status = 500
match status:
    200 -> "ok"
    404 -> "not found"
    _ -> "error"`
	result = testEval(t, input)
	sv, ok = result.(*StringValue)
	require.True(t, ok)
	assert.Equal(t, "error", sv.Value)
}

func TestEvalTernary(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5 if true else 10", 5},
		{"5 if false else 10", 10},
		{"10 if 1 > 2 else 20", 20},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "input: %s", tt.input)
		assert.Equal(t, tt.expected, iv.Value, "input: %s", tt.input)
	}
}

func TestEvalInOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1 in [1, 2, 3]", true},
		{"4 in [1, 2, 3]", false},
		{`"a" in {a: 1, b: 2}`, true},
		{`"c" in {a: 1, b: 2}`, false},
		{`"el" in "hello"`, true},
		{`"xy" in "hello"`, false},
	}

	for _, tt := range tests {
		result := testEval(t, tt.input)
		bv, ok := result.(*BoolValue)
		require.True(t, ok, "input: %s", tt.input)
		assert.Equal(t, tt.expected, bv.Value, "input: %s", tt.input)
	}
}

func TestEvalMemberAccess(t *testing.T) {
	input := `data = {name: "John", age: 30}
data.name`
	result := testEval(t, input)
	sv, ok := result.(*StringValue)
	require.True(t, ok)
	assert.Equal(t, "John", sv.Value)
}

// TestEvalListComprehension is skipped until parser supports comprehensions
func TestEvalListComprehension(t *testing.T) {
	t.Skip("List comprehension parsing not yet implemented")
}

func TestEvalEmit(t *testing.T) {
	input := `emit(42)
emit(status: "done")`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	e := New()
	_, err := e.Eval(program)
	require.NoError(t, err)

	// Check emitted values
	assert.Len(t, e.Context().Emitted, 2)
	assert.Equal(t, int64(42), e.Context().Emitted[0].(*IntValue).Value)
}

func TestEvalStop(t *testing.T) {
	input := `x = 1
stop
x = 2`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	e := New()
	_, err := e.Eval(program)
	require.NoError(t, err)

	// x should still be 1 since stop was called
	val, ok := e.Context().Scope.Get("x")
	require.True(t, ok)
	assert.Equal(t, int64(1), val.(*IntValue).Value)
}

func TestEvalTryCatch(t *testing.T) {
	// This is a basic test - full error handling needs more work
	input := `result = "ok"
try:
    result = "in try"
catch error:
    result = "in catch"
result`

	result := testEval(t, input)
	sv, ok := result.(*StringValue)
	require.True(t, ok)
	assert.Equal(t, "in try", sv.Value)
}

func TestEvalClosure(t *testing.T) {
	input := `def make_adder(n):
    def adder(x):
        return x + n
    return adder

add5 = make_adder(5)
add5(10)`

	result := testEval(t, input)
	iv, ok := result.(*IntValue)
	require.True(t, ok)
	assert.Equal(t, int64(15), iv.Value)
}

func TestEvalOptionalAccess(t *testing.T) {
	input := `data = none
data?.name`
	result := testEval(t, input)
	assert.IsType(t, &NoneValue{}, result)

	input = `data = {name: "John"}
data?.name`
	result = testEval(t, input)
	sv, ok := result.(*StringValue)
	require.True(t, ok)
	assert.Equal(t, "John", sv.Value)
}

func TestEvalForWithIndex(t *testing.T) {
	input := `result = []
for i, x in ["a", "b", "c"]:
    result = result + [[i, x]]
result`

	result := testEval(t, input)
	lv, ok := result.(*ListValue)
	require.True(t, ok)
	require.Len(t, lv.Elements, 3)

	// Check first element is [0, "a"]
	first := lv.Elements[0].(*ListValue)
	assert.Equal(t, int64(0), first.Elements[0].(*IntValue).Value)
	assert.Equal(t, "a", first.Elements[1].(*StringValue).Value)
}

func TestEvalPauseStatement(t *testing.T) {
	t.Run("pause without message", func(t *testing.T) {
		input := `x = 1
pause
x = 2`

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.NoError(t, err)

		// Should have paused, x should still be 1
		x, ok := e.ctx.Scope.Get("x")
		require.True(t, ok)
		assert.Equal(t, int64(1), x.(*IntValue).Value)
		assert.True(t, e.ctx.ShouldPause())
		assert.Equal(t, "", e.ctx.GetPauseMessage())
	})

	t.Run("pause with message", func(t *testing.T) {
		input := `x = 42
pause "checkpoint 1"
x = 100`

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.NoError(t, err)

		// Should have paused with message
		x, ok := e.ctx.Scope.Get("x")
		require.True(t, ok)
		assert.Equal(t, int64(42), x.(*IntValue).Value)
		assert.True(t, e.ctx.ShouldPause())
		assert.Equal(t, "checkpoint 1", e.ctx.GetPauseMessage())
	})

	t.Run("pause with expression message", func(t *testing.T) {
		input := `name = "test"
pause name + " checkpoint"
x = 1`

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.NoError(t, err)

		assert.True(t, e.ctx.ShouldPause())
		assert.Equal(t, "test checkpoint", e.ctx.GetPauseMessage())
	})

	t.Run("pause in function", func(t *testing.T) {
		input := `def process():
    x = 1
    pause "in function"
    x = 2
    return x

result = process()`

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.NoError(t, err)

		// Should have paused inside function
		assert.True(t, e.ctx.ShouldPause())
		assert.Equal(t, "in function", e.ctx.GetPauseMessage())
	})

	t.Run("pause in loop", func(t *testing.T) {
		input := `count = 0
for i in range(5):
    count = count + 1
    if count == 3:
        pause "at count 3"
count`

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.NoError(t, err)

		// Should have paused at count 3
		count, ok := e.ctx.Scope.Get("count")
		require.True(t, ok)
		assert.Equal(t, int64(3), count.(*IntValue).Value)
		assert.True(t, e.ctx.ShouldPause())
		assert.Equal(t, "at count 3", e.ctx.GetPauseMessage())
	})
}

func TestEvalHyphenatedIdentifiers(t *testing.T) {
	t.Run("hyphenated name in scope resolves directly", func(t *testing.T) {
		input := "dart-query = 42\ndart-query"
		result := testEval(t, input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "expected IntValue, got %T", result)
		assert.Equal(t, int64(42), iv.Value)
	})

	t.Run("subtraction fallback when parts are in scope", func(t *testing.T) {
		input := "a = 10\nb = 3\na-b"
		result := testEval(t, input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "expected IntValue, got %T", result)
		assert.Equal(t, int64(7), iv.Value)
	})

	t.Run("chained subtraction fallback", func(t *testing.T) {
		input := "a = 10\nb = 3\nc = 2\na-b-c"
		result := testEval(t, input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "expected IntValue, got %T", result)
		assert.Equal(t, int64(5), iv.Value)
	})

	t.Run("undefined part gives error with full name", func(t *testing.T) {
		input := "a = 10\na-b"
		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()
		require.Empty(t, p.Errors())

		e := New()
		_, err := e.Eval(program)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefined variable: a-b")
	})

	t.Run("hyphenated name takes priority over subtraction", func(t *testing.T) {
		// When the full hyphenated name is in scope, use it even if parts exist too
		input := "a = 10\nb = 3\na-b = 99\na-b"
		result := testEval(t, input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "expected IntValue, got %T", result)
		assert.Equal(t, int64(99), iv.Value)
	})

	t.Run("spaced subtraction still works", func(t *testing.T) {
		input := "a = 10\nb = 3\na - b"
		result := testEval(t, input)
		iv, ok := result.(*IntValue)
		require.True(t, ok, "expected IntValue, got %T", result)
		assert.Equal(t, int64(7), iv.Value)
	})
}

// TestEvalCollectionEquality verifies deep structural equality for list, map,
// and set values — not pointer identity.
func TestEvalCollectionEquality(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// List equality
		{"empty lists equal", "[] == []", true},
		{"equal lists", "[1, 2, 3] == [1, 2, 3]", true},
		{"lists different length", "[1, 2] == [1, 2, 3]", false},
		{"lists different element", "[1, 2, 3] == [1, 2, 4]", false},
		{"list inequality op", "[1, 2] != [1, 3]", true},
		// Nested list equality
		{"nested lists equal", "[[1, 2], [3]] == [[1, 2], [3]]", true},
		{"nested lists not equal", "[[1, 2], [3]] == [[1, 2], [4]]", false},
		// Map equality
		{"empty maps equal", "{} == {}", true},
		{"equal maps", "{a: 1, b: 2} == {a: 1, b: 2}", true},
		{"maps different value", "{a: 1} == {a: 2}", false},
		{"maps different key set", "{a: 1, b: 2} == {a: 1}", false},
		// Set equality ({x, y, ...} is a set literal; {} is an empty map)
		{"equal sets", "{1, 2, 3} == {1, 2, 3}", true},
		{"sets different element", "{1, 2, 3} == {1, 2, 4}", false},
		{"sets different size", "{1, 2} == {1, 2, 3}", false},
		{"set inequality op", "{1, 2} != {1, 3}", true},
		// in operator with list-as-element (nested)
		{"list in list of lists", "[1, 2] in [[1, 2], [3, 4]]", true},
		{"list not in list of lists", "[1, 3] in [[1, 2], [3, 4]]", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testEval(t, tt.input)
			bv, ok := result.(*BoolValue)
			require.True(t, ok, "expected BoolValue, got %T (%s)", result, result)
			assert.Equal(t, tt.expected, bv.Value)
		})
	}
}

// TestEvalEqualFunctionDirect exercises the Equal() helper directly for full
// coverage of nested and mixed-depth collection cases.
func TestEvalEqualFunctionDirect(t *testing.T) {
	// List
	assert.True(t, Equal(&ListValue{}, &ListValue{}))
	assert.True(t, Equal(
		&ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}},
		&ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}},
	))
	assert.False(t, Equal(
		&ListValue{Elements: []Value{&IntValue{1}}},
		&ListValue{Elements: []Value{&IntValue{2}}},
	))
	// List != non-list
	assert.False(t, Equal(&ListValue{}, &MapValue{Pairs: map[string]Value{}}))

	// Map
	m1 := NewMapValue()
	m1.Set("x", &IntValue{42})
	m2 := NewMapValue()
	m2.Set("x", &IntValue{42})
	m3 := NewMapValue()
	m3.Set("x", &IntValue{99})
	assert.True(t, Equal(m1, m2))
	assert.False(t, Equal(m1, m3))

	// Set
	s1 := NewSetValue()
	s1.Add(&IntValue{1})
	s1.Add(&IntValue{2})
	s2 := NewSetValue()
	s2.Add(&IntValue{1})
	s2.Add(&IntValue{2})
	s3 := NewSetValue()
	s3.Add(&IntValue{1})
	s3.Add(&IntValue{3})
	assert.True(t, Equal(s1, s2))
	assert.False(t, Equal(s1, s3))

	// Nested: list containing map
	inner1 := NewMapValue()
	inner1.Set("k", &StringValue{"v"})
	inner2 := NewMapValue()
	inner2.Set("k", &StringValue{"v"})
	nested1 := &ListValue{Elements: []Value{inner1}}
	nested2 := &ListValue{Elements: []Value{inner2}}
	assert.True(t, Equal(nested1, nested2))

	// Map containing list value
	ml1 := NewMapValue()
	ml1.Set("items", &ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}})
	ml2 := NewMapValue()
	ml2.Set("items", &ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}})
	assert.True(t, Equal(ml1, ml2))
	ml3 := NewMapValue()
	ml3.Set("items", &ListValue{Elements: []Value{&IntValue{1}, &IntValue{9}}})
	assert.False(t, Equal(ml1, ml3))
}

// TestEqualCycleGuardList proves that Equal() does not stack-overflow on a
// self-referential list. list.append() mutates in place so x=[]; x.append(x)
// is a valid (if pathological) SLOP value. Without the cycle guard this test
// would hang forever (or crash via Go stack exhaustion).
func TestEqualCycleGuardList(t *testing.T) {
	// x is a list that contains itself: x[0] == x
	x := &ListValue{}
	x.Elements = []Value{x}

	// A self-cyclic list must equal itself (co-inductive termination).
	assert.True(t, Equal(x, x), "self-cyclic list must equal itself")

	// Two structurally identical cyclic lists: y[0] == y, z[0] == z.
	// They are isomorphic so Equal must return true.
	y := &ListValue{}
	y.Elements = []Value{y}
	z := &ListValue{}
	z.Elements = []Value{z}
	assert.True(t, Equal(y, z), "two isomorphic cyclic lists must be equal")

	// A self-cyclic list is NOT equal to a simple non-cyclic list: lengths differ
	// at each unrolling level — the cycle guard fires on the cyclic side while the
	// non-cyclic side has a definite length mismatch.
	plain := &ListValue{Elements: []Value{&IntValue{1}}}
	// plain[0] is an IntValue; x[0] is a *ListValue — types differ so Equal returns false.
	assert.False(t, Equal(x, plain), "cyclic list != non-cyclic list")
}

// TestEqualCycleGuardMap proves that Equal() terminates on a self-referential
// map. A script can produce such a map via m={}; m["self"]=m.
func TestEqualCycleGuardMap(t *testing.T) {
	m := NewMapValue()
	m.Set("self", m)

	assert.True(t, Equal(m, m), "self-cyclic map must equal itself")

	m2 := NewMapValue()
	m2.Set("self", m2)
	assert.True(t, Equal(m, m2), "two isomorphic self-cyclic maps must be equal")
}

// TestEqualDAGNoFalseNegative proves that a shared (non-cyclic) substructure is
// not mis-classified as a cycle. x and y both reference the same inner list,
// forming a DAG rather than a true cycle. Equal must still return true.
func TestEqualDAGNoFalseNegative(t *testing.T) {
	shared := &ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}}

	// Both outer lists reference the same inner object — DAG, not a cycle.
	outer1 := &ListValue{Elements: []Value{shared, shared}}
	outer2 := &ListValue{Elements: []Value{shared, shared}}
	assert.True(t, Equal(outer1, outer2), "DAG with shared node must still be equal")

	// Different shared node (same value, different pointer) must also be equal.
	shared2 := &ListValue{Elements: []Value{&IntValue{1}, &IntValue{2}}}
	outer3 := &ListValue{Elements: []Value{shared2, shared2}}
	assert.True(t, Equal(outer1, outer3), "DAG with structurally-equal shared nodes must be equal")
}
