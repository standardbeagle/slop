package parser

import (
	"testing"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIntegerLiteral(t *testing.T) {
	input := `42`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	lit := stmt.Expression.(*ast.IntegerLiteral)
	assert.Equal(t, int64(42), lit.Value)
}

func TestParseFloatLiteral(t *testing.T) {
	input := `3.14`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	lit := stmt.Expression.(*ast.FloatLiteral)
	assert.InDelta(t, 3.14, lit.Value, 0.001)
}

func TestParseStringLiteral(t *testing.T) {
	input := `"hello world"`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	lit := stmt.Expression.(*ast.StringLiteral)
	assert.Equal(t, "hello world", lit.Value)
}

func TestParseBooleanLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lit := stmt.Expression.(*ast.BooleanLiteral)
		assert.Equal(t, tt.expected, lit.Value)
	}
}

func TestParseNoneLiteral(t *testing.T) {
	input := `none`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.NoneLiteral)
	assert.True(t, ok)
}

func TestParseIdentifier(t *testing.T) {
	input := `foobar`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ident := stmt.Expression.(*ast.Identifier)
	assert.Equal(t, "foobar", ident.Value)
}

func TestParsePrefixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"-5", "-", int64(5)},
		{"not true", "not", true},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		exp := stmt.Expression.(*ast.PrefixExpression)
		assert.Equal(t, tt.operator, exp.Operator)
	}
}

func TestParseInfixExpressions(t *testing.T) {
	tests := []struct {
		input      string
		leftValue  interface{}
		operator   string
		rightValue interface{}
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 % 5", 5, "%", 5},
		{"5 ** 2", 5, "**", 2},
		{"5 > 5", 5, ">", 5},
		{"5 < 5", 5, "<", 5},
		{"5 >= 5", 5, ">=", 5},
		{"5 <= 5", 5, "<=", 5},
		{"5 == 5", 5, "==", 5},
		{"5 != 5", 5, "!=", 5},
		{"true and false", true, "and", false},
		{"true or false", true, "or", false},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		exp := stmt.Expression.(*ast.InfixExpression)
		assert.Equal(t, tt.operator, exp.Operator, "input: %s", tt.input)
	}
}

func TestParseListLiteral(t *testing.T) {
	input := `[1, 2, 3]`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	list := stmt.Expression.(*ast.ListLiteral)
	assert.Len(t, list.Elements, 3)
}

func TestParseMapLiteral(t *testing.T) {
	input := `{a: 1, b: 2}`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	m := stmt.Expression.(*ast.MapLiteral)
	assert.Len(t, m.Pairs, 2)
}

func TestParseEmptyMap(t *testing.T) {
	input := `{}`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	m := stmt.Expression.(*ast.MapLiteral)
	assert.Len(t, m.Pairs, 0)
}

func TestParseSetLiteral(t *testing.T) {
	input := `{1, 2, 3}`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	s := stmt.Expression.(*ast.SetLiteral)
	assert.Len(t, s.Elements, 3)
}

func TestParseAssignment(t *testing.T) {
	input := `x = 42`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.AssignStatement)
	assert.Equal(t, "=", stmt.Operator)
	assert.Len(t, stmt.Targets, 1)

	target := stmt.Targets[0].(*ast.Identifier)
	assert.Equal(t, "x", target.Value)
}

func TestParseCompoundAssignment(t *testing.T) {
	tests := []struct {
		input    string
		operator string
	}{
		{"x += 1", "+="},
		{"x -= 1", "-="},
		{"x *= 2", "*="},
		{"x /= 2", "/="},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.AssignStatement)
		assert.Equal(t, tt.operator, stmt.Operator, "input: %s", tt.input)
	}
}

func TestParseCallExpression(t *testing.T) {
	input := `add(1, 2, 3)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call := stmt.Expression.(*ast.CallExpression)

	fn := call.Function.(*ast.Identifier)
	assert.Equal(t, "add", fn.Value)
	assert.Len(t, call.Arguments, 3)
}

func TestParseCallWithKwargs(t *testing.T) {
	input := `llm.call(prompt: "hello", model: "claude")`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call := stmt.Expression.(*ast.CallExpression)

	assert.Len(t, call.Arguments, 0)
	assert.Len(t, call.Kwargs, 2)
	assert.Contains(t, call.Kwargs, "prompt")
	assert.Contains(t, call.Kwargs, "model")
}

func TestParseCallMixedArgsAndKwargs(t *testing.T) {
	input := `api.query("users", limit: 10, offset: 5)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call := stmt.Expression.(*ast.CallExpression)

	// Should have 1 positional arg
	assert.Len(t, call.Arguments, 1)
	arg := call.Arguments[0].(*ast.StringLiteral)
	assert.Equal(t, "users", arg.Value)

	// Should have 2 kwargs
	assert.Len(t, call.Kwargs, 2)
	assert.Contains(t, call.Kwargs, "limit")
	assert.Contains(t, call.Kwargs, "offset")
}

func TestParseMemberExpression(t *testing.T) {
	input := `foo.bar.baz`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	member := stmt.Expression.(*ast.MemberExpression)
	assert.Equal(t, "baz", member.Property.Value)
	assert.False(t, member.Optional)
}

func TestParseOptionalMemberExpression(t *testing.T) {
	input := `foo?.bar?.baz`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	member := stmt.Expression.(*ast.MemberExpression)
	assert.Equal(t, "baz", member.Property.Value)
	assert.True(t, member.Optional)
}

func TestParseIndexExpression(t *testing.T) {
	input := `arr[0]`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	idx := stmt.Expression.(*ast.IndexExpression)
	assert.False(t, idx.Optional)
}

func TestParseSliceExpression(t *testing.T) {
	tests := []struct {
		input    string
		hasStart bool
		hasEnd   bool
		hasStep  bool
	}{
		{"arr[1:3]", true, true, false},
		{"arr[:3]", false, true, false},
		{"arr[1:]", true, false, false},
		{"arr[::2]", false, false, true},
		{"arr[1:3:2]", true, true, true},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		slice := stmt.Expression.(*ast.SliceExpression)
		assert.Equal(t, tt.hasStart, slice.Start != nil, "input: %s start", tt.input)
		assert.Equal(t, tt.hasEnd, slice.End != nil, "input: %s end", tt.input)
		assert.Equal(t, tt.hasStep, slice.Step != nil, "input: %s step", tt.input)
	}
}

func TestParseLambdaExpression(t *testing.T) {
	tests := []struct {
		input     string
		numParams int
	}{
		{"x -> x * 2", 1},
		{"(x) -> x * 2", 1},
		{"(a, b) -> a + b", 2},
		{"() -> 42", 0},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lambda := stmt.Expression.(*ast.LambdaExpression)
		assert.Len(t, lambda.Parameters, tt.numParams, "input: %s", tt.input)
	}
}

func TestParsePipelineExpression(t *testing.T) {
	input := `items | filter(x -> x > 0) | map(x -> x * 2)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	pipe := stmt.Expression.(*ast.PipelineExpression)
	assert.NotNil(t, pipe.Left)
	assert.NotNil(t, pipe.Right)
}

func TestParseTernaryExpression(t *testing.T) {
	input := `"yes" if condition else "no"`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ternary := stmt.Expression.(*ast.TernaryExpression)
	assert.NotNil(t, ternary.Condition)
	assert.NotNil(t, ternary.Consequence)
	assert.NotNil(t, ternary.Alternative)
}

func TestParseRangeExpression(t *testing.T) {
	tests := []struct {
		input    string
		hasStart bool
		hasStep  bool
	}{
		{"range(10)", false, false},
		{"range(1, 10)", true, false},
		{"range(1, 10, 2)", true, true},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		r := stmt.Expression.(*ast.RangeExpression)
		assert.Equal(t, tt.hasStart, r.Start != nil, "input: %s", tt.input)
		assert.Equal(t, tt.hasStep, r.Step != nil, "input: %s", tt.input)
	}
}

func TestParseIfStatement(t *testing.T) {
	input := `if x > 0:
    print(x)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.IfStatement)
	assert.NotNil(t, stmt.Condition)
	assert.NotNil(t, stmt.Consequence)
	assert.Len(t, stmt.Consequence.Statements, 1)
}

func TestParseIfElseStatement(t *testing.T) {
	input := `if x > 0:
    print("positive")
else:
    print("non-positive")`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.IfStatement)
	assert.NotNil(t, stmt.Condition)
	assert.NotNil(t, stmt.Consequence)
	assert.NotNil(t, stmt.Alternative)
}

func TestParseIfElifElseStatement(t *testing.T) {
	input := `if x > 0:
    print("positive")
elif x < 0:
    print("negative")
else:
    print("zero")`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.IfStatement)
	assert.NotNil(t, stmt.Condition)
	assert.NotNil(t, stmt.Consequence)

	// Alternative should be another IfStatement (elif)
	elif, ok := stmt.Alternative.(*ast.IfStatement)
	require.True(t, ok)
	assert.NotNil(t, elif.Alternative)
}

func TestParseForStatement(t *testing.T) {
	input := `for item in items:
    print(item)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ForStatement)
	assert.Equal(t, "item", stmt.Variable.Value)
	assert.Nil(t, stmt.Index)
	assert.NotNil(t, stmt.Iterable)
	assert.Len(t, stmt.Body.Statements, 1)
}

func TestParseForWithIndex(t *testing.T) {
	input := `for i, item in items:
    print(i, item)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ForStatement)
	assert.Equal(t, "i", stmt.Index.Value)
	assert.Equal(t, "item", stmt.Variable.Value)
}

func TestParseForWithModifiers(t *testing.T) {
	input := `for item in items with limit(100), rate(10/s):
    process(item)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.ForStatement)
	assert.Len(t, stmt.Modifiers, 2)

	assert.Equal(t, "limit", stmt.Modifiers[0].Type)
	assert.Equal(t, "rate", stmt.Modifiers[1].Type)
	assert.Equal(t, "s", stmt.Modifiers[1].Unit)
}

func TestParseDefStatement(t *testing.T) {
	input := `def greet(name):
    print("Hello, " + name)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.DefStatement)
	assert.Equal(t, "greet", stmt.Name.Value)
	assert.Len(t, stmt.Parameters, 1)
	assert.Equal(t, "name", stmt.Parameters[0].Name.Value)
}

func TestParseDefWithDefaultParams(t *testing.T) {
	input := `def greet(name, greeting = "Hello"):
    print(greeting + ", " + name)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.DefStatement)
	assert.Len(t, stmt.Parameters, 2)
	assert.Nil(t, stmt.Parameters[0].Default)
	assert.NotNil(t, stmt.Parameters[1].Default)
}

func TestParseReturnStatement(t *testing.T) {
	tests := []struct {
		input    string
		hasValue bool
	}{
		{"return", false},
		{"return 42", true},
		{"return x + y", true},
	}

	for _, tt := range tests {
		// Wrap in function to be valid
		input := "def f():\n    " + tt.input
		program := parseProgram(t, input)

		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		def := program.Statements[0].(*ast.DefStatement)
		require.Len(t, def.Body.Statements, 1, "input: %s", tt.input)
		ret := def.Body.Statements[0].(*ast.ReturnStatement)
		assert.Equal(t, tt.hasValue, ret.Value != nil, "input: %s", tt.input)
	}
}

func TestParseEmitStatement(t *testing.T) {
	tests := []struct {
		input     string
		numValues int
		numNamed  int
	}{
		{"emit(result)", 1, 0},
		{"emit(x, y, z)", 3, 0},
		{"emit(status: done)", 0, 1},
		{"emit(result, count: 10)", 1, 1},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		emit := program.Statements[0].(*ast.EmitStatement)
		assert.Len(t, emit.Values, tt.numValues, "input: %s values", tt.input)
		assert.Len(t, emit.Named, tt.numNamed, "input: %s named", tt.input)
	}
}

func TestParseStopStatement(t *testing.T) {
	tests := []struct {
		input    string
		rollback bool
	}{
		{"stop", false},
		{"stop with rollback", true},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stop := program.Statements[0].(*ast.StopStatement)
		assert.Equal(t, tt.rollback, stop.Rollback, "input: %s", tt.input)
	}
}

func TestParseTryStatement(t *testing.T) {
	input := `try:
    risky()
catch error:
    handle(error)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.TryStatement)
	assert.NotNil(t, stmt.Body)
	assert.Len(t, stmt.Catches, 1)
}

func TestParseMatchStatement(t *testing.T) {
	input := `match status:
    200 -> print("ok")
    404 -> print("not found")
    _ -> print("error")`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	stmt := program.Statements[0].(*ast.MatchStatement)
	assert.NotNil(t, stmt.Subject)
	assert.Len(t, stmt.Arms, 3)
}

func TestParseBreakContinue(t *testing.T) {
	input := `for i in range(10):
    if i == 5:
        break
    if i % 2 == 0:
        continue
    print(i)`
	program := parseProgram(t, input)

	require.Len(t, program.Statements, 1)
	forStmt := program.Statements[0].(*ast.ForStatement)
	assert.Len(t, forStmt.Body.Statements, 3)
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"1 + 2 + 3", "((1 + 2) + 3)"},
		{"-1 * 2", "((-1) * 2)"},
		{"1 < 2 and 3 > 4", "((1 < 2) and (3 > 4))"},
		{"1 or 2 and 3", "(1 or (2 and 3))"},
		{"2 ** 3 ** 4", "(2 ** (3 ** 4))"},
	}

	for _, tt := range tests {
		program := parseProgram(t, tt.input)
		require.Len(t, program.Statements, 1, "input: %s", tt.input)
		stmt := program.Statements[0].(*ast.ExpressionStatement)
		assert.Equal(t, tt.expected, stmt.Expression.String(), "input: %s", tt.input)
	}
}

func TestComplexScript(t *testing.T) {
	input := `task = input.task

plan = llm.call(
    prompt: "Break into subtasks: {task}",
    schema: {subtasks: list}
)

for subtask in plan.subtasks with limit(10), rate(5/s):
    result = tools.execute(subtask)

    if result.needs_help:
        guidance = llm.call(
            prompt: "How to handle: {result.error}",
            schema: {action: string}
        )
        match guidance.action:
            retry -> tools.execute(subtask)
            skip -> continue
            abort -> break

emit(results)`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Logf("parse error: %s", err)
		}
	}

	// Should parse without errors
	assert.Empty(t, p.Errors(), "parse errors")
	assert.NotEmpty(t, program.Statements, "should have statements")
}

// Helper function to parse and check for errors
func parseProgram(t *testing.T, input string) *ast.Program {
	t.Helper()
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parse error: %s", err)
		}
		t.FailNow()
	}

	return program
}
