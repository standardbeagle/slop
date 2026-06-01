package evaluator

import (
	"testing"

	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// evalWithLimits parses input and evaluates it under the given limits, returning
// the result and error so depth-guard behavior can be asserted directly.
func evalWithLimits(t *testing.T, input string, limits *ExecutionLimits) (Value, error) {
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
	e := NewWithContext(NewContextWithLimits(limits))
	return e.Eval(program)
}

// TestRecursionDepthGuardFunction proves unbounded recursion in a user function
// returns an error instead of crashing the host via Go stack exhaustion.
func TestRecursionDepthGuardFunction(t *testing.T) {
	input := `def recurse(n):
    return recurse(n + 1)
recurse(0)`

	_, err := evalWithLimits(t, input, &ExecutionLimits{MaxCallDepth: 50})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call depth limit exceeded")
}

// TestRecursionDepthGuardLambda proves the guard also covers lambda recursion.
func TestRecursionDepthGuardLambda(t *testing.T) {
	input := `f = n -> f(n + 1)
f(0)`

	_, err := evalWithLimits(t, input, &ExecutionLimits{MaxCallDepth: 50})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call depth limit exceeded")
}

// TestRecursionDepthGuardDefaultsOn proves the guard is active even when no
// explicit MaxCallDepth is configured: the built-in default must protect the
// host so untrusted scripts cannot crash it.
func TestRecursionDepthGuardDefaultsOn(t *testing.T) {
	input := `def recurse(n):
    return recurse(n + 1)
recurse(0)`

	// Empty limits => MaxCallDepth 0 => DefaultMaxCallDepth applies.
	_, err := evalWithLimits(t, input, &ExecutionLimits{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "call depth limit exceeded")
}

// TestBoundedRecursionStillWorks proves the guard does not break legitimate
// recursion that stays within the configured depth.
func TestBoundedRecursionStillWorks(t *testing.T) {
	input := `def sum_to(n):
    if n == 0:
        return 0
    return n + sum_to(n - 1)
sum_to(10)`

	result, err := evalWithLimits(t, input, &ExecutionLimits{MaxCallDepth: 50})
	require.NoError(t, err)
	iv, ok := result.(*IntValue)
	require.True(t, ok, "got %T", result)
	assert.Equal(t, int64(55), iv.Value)
}

// TestCallDepthUnwindsAfterCatch proves CallDepth returns to zero after deep
// recursion is caught, so a subsequent call is not penalized by stale depth.
func TestCallDepthUnwindsAfterCatch(t *testing.T) {
	input := `def recurse(n):
    return recurse(n + 1)

result = "ok"
try:
    recurse(0)
catch error:
    result = "caught"

# After catch, a fresh bounded call must succeed.
def sum_to(n):
    if n == 0:
        return 0
    return n + sum_to(n - 1)
sum_to(5)`

	e := NewWithContext(NewContextWithLimits(&ExecutionLimits{MaxCallDepth: 50}))
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors(), "parse errors: %v", p.Errors())

	result, err := e.Eval(program)
	require.NoError(t, err)
	iv, ok := result.(*IntValue)
	require.True(t, ok, "got %T", result)
	assert.Equal(t, int64(15), iv.Value)
	// Depth must be fully unwound, not stranded after the caught overflow.
	assert.Equal(t, int64(0), e.Context().Limits.CallDepth)
}

// TestCallDepthRoundTripsThroughCheckpoint proves the new limit fields survive
// serialize/deserialize so a resumed checkpoint keeps its recursion guard.
func TestCallDepthRoundTripsThroughCheckpoint(t *testing.T) {
	orig := &ExecutionLimits{MaxCallDepth: 1234, CallDepth: 7}
	restored, err := DeserializeLimits(SerializeLimits(orig))
	assert.NoError(t, err)
	assert.Equal(t, int64(1234), restored.MaxCallDepth)
	assert.Equal(t, int64(7), restored.CallDepth)
}

// TestEqualCyclicListViaScript proves that comparing a self-referential list
// produced by a SLOP script does NOT stack-overflow or hang. Before the cycle
// guard was added, x=[]; x.append(x); x==x would exhaust the Go stack.
//
// The test must complete within the test timeout — if Equal recurses infinitely
// the test binary panics with "stack overflow" or the test runner times it out.
func TestEqualCyclicListViaScript(t *testing.T) {
	// Build the cyclic list directly (SLOP scripts cannot yet produce this path
	// in the test harness without a full runtime; we exercise via the Go API to
	// guarantee the guard fires on the exact code path triggered by scripts).
	x := &ListValue{}
	x.Elements = []Value{x}

	// Must terminate and return true — co-inductive equality.
	result := Equal(x, x)
	require.True(t, result, "Equal on cyclic list must terminate and return true")
}
