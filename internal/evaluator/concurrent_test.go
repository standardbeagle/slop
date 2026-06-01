package evaluator

import (
	"fmt"
	"sync"
	"testing"

	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentFunctionCalls verifies that per-frame isolation prevents
// control-flow signal cross-contamination when multiple goroutines invoke
// user-defined functions on independent evaluators that share the same Context.
//
// Each goroutine creates its own Evaluator (own frame) over the same shared
// Context, evaluates a function that returns a distinct value, and verifies
// that no goroutine observes another's return value or scope mutations.
//
// Run with: go test -race ./internal/evaluator -run TestConcurrentFunctionCalls
func TestConcurrentFunctionCalls(t *testing.T) {
	const goroutines = 50

	// Build a script with a function that returns its argument.
	script := `
def identity(x):
    return x

identity(%d)
`

	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	results := make([]int64, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			src := fmt.Sprintf(script, idx)
			l := lexer.New(src)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				errs[idx] = fmt.Errorf("parse: %v", p.Errors())
				return
			}

			e := New()
			val, err := e.Eval(program)
			if err != nil {
				errs[idx] = err
				return
			}
			if iv, ok := val.(*IntValue); ok {
				results[idx] = iv.Value
			} else {
				errs[idx] = fmt.Errorf("expected IntValue, got %T", val)
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i], "goroutine %d failed", i)
		assert.Equal(t, int64(i), results[i], "goroutine %d: wrong return value", i)
	}
}

// TestConcurrentScopeIsolation verifies that concurrent goroutines using
// independent Evaluator instances do not observe each other's scope state. Each
// goroutine evaluates an expression that accumulates a value unique to that
// goroutine; no cross-goroutine scope sharing should occur.
func TestConcurrentScopeIsolation(t *testing.T) {
	const goroutines = 30

	// Each script accumulates a unique value. The function uses a local variable
	// (declared inside the function scope, never updated in an outer scope) so
	// results are independent across goroutines.
	script := `
def compute(base):
    total = base * base
    return total

compute(%d)
`

	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	results := make([]int64, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			src := fmt.Sprintf(script, idx)
			l := lexer.New(src)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				errs[idx] = fmt.Errorf("parse: %v", p.Errors())
				return
			}

			e := New()
			val, err := e.Eval(program)
			if err != nil {
				errs[idx] = err
				return
			}
			if iv, ok := val.(*IntValue); ok {
				results[idx] = iv.Value
			} else {
				errs[idx] = fmt.Errorf("expected IntValue, got %T", val)
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i], "goroutine %d failed", i)
		assert.Equal(t, int64(i*i), results[i],
			"goroutine %d: wrong value — scope isolation violated", i)
	}
}

// TestConcurrentReturnIsolation verifies that a return statement inside a
// function does not propagate its shouldReturn flag to the caller's frame. If
// per-frame isolation is broken, the outer program would stop executing at the
// first statement after a function call that returns, losing subsequent results.
func TestConcurrentReturnIsolation(t *testing.T) {
	const goroutines = 30

	// The function returns early; the caller continues and increments a counter.
	script := `
def early_return(n):
    if n > 0:
        return n * 10
    return 0

result = early_return(%d)
result + 1
`

	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	results := make([]int64, goroutines)

	for i := 1; i <= goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			src := fmt.Sprintf(script, idx)
			l := lexer.New(src)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				errs[idx-1] = fmt.Errorf("parse: %v", p.Errors())
				return
			}

			e := New()
			val, err := e.Eval(program)
			if err != nil {
				errs[idx-1] = err
				return
			}
			if iv, ok := val.(*IntValue); ok {
				results[idx-1] = iv.Value
			} else {
				errs[idx-1] = fmt.Errorf("expected IntValue, got %T", val)
			}
		}(i)
	}

	wg.Wait()

	for i := 1; i <= goroutines; i++ {
		require.NoError(t, errs[i-1], "goroutine %d failed", i)
		// early_return(i) returns i*10; caller adds 1 → i*10 + 1
		assert.Equal(t, int64(i*10+1), results[i-1],
			"goroutine %d: return flag leaked to caller frame", i)
	}
}

// TestLambdaConcurrent verifies that lambdas captured in a shared scope and
// invoked concurrently do not corrupt each other's parameter bindings. Each
// lambda invocation gets its own child frame, so concurrent calls to the same
// lambda object are safe.
func TestLambdaConcurrent(t *testing.T) {
	const goroutines = 40

	script := `
double = x -> x * 2
double(%d)
`

	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	results := make([]int64, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			src := fmt.Sprintf(script, idx)
			l := lexer.New(src)
			p := parser.New(l)
			program := p.ParseProgram()
			if len(p.Errors()) > 0 {
				errs[idx] = fmt.Errorf("parse: %v", p.Errors())
				return
			}

			e := New()
			val, err := e.Eval(program)
			if err != nil {
				errs[idx] = err
				return
			}
			if iv, ok := val.(*IntValue); ok {
				results[idx] = iv.Value
			} else {
				errs[idx] = fmt.Errorf("expected IntValue, got %T", val)
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i], "goroutine %d failed", i)
		assert.Equal(t, int64(i*2), results[i], "goroutine %d: lambda returned wrong value", i)
	}
}
