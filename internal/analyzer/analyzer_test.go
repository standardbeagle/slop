package analyzer

import (
	"strings"
	"testing"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeNoRecursion(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		hasError bool
		errorMsg string
	}{
		{
			name: "simple function no recursion",
			code: `
def add(a, b):
    return a + b

add(1, 2)
`,
			hasError: false,
		},
		{
			name: "direct self-recursion",
			code: `
def factorial(n):
    if n <= 1:
        return 1
    return n * factorial(n - 1)
`,
			hasError: true,
			errorMsg: "calls itself",
		},
		{
			name: "indirect mutual recursion",
			code: `
def is_even(n):
    if n == 0:
        return true
    return is_odd(n - 1)

def is_odd(n):
    if n == 0:
        return false
    return is_even(n - 1)
`,
			hasError: true,
			errorMsg: "cycle",
		},
		{
			name: "non-recursive call chain",
			code: `
def step1(x):
    return step2(x + 1)

def step2(x):
    return step3(x + 1)

def step3(x):
    return x * 2
`,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			a := New()
			errors := a.Analyze(program)

			if tt.hasError {
				require.NotEmpty(t, errors, "expected analysis error")
				found := false
				for _, err := range errors {
					if strings.Contains(err.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected error containing %q, got %v", tt.errorMsg, errors)
			} else {
				// Filter out false positives from cycle detection
				var realErrors []AnalysisError
				for _, err := range errors {
					if err.Kind == ErrorRecursion || err.Kind == ErrorCycle {
						realErrors = append(realErrors, err)
					}
				}
				assert.Empty(t, realErrors, "unexpected errors: %v", realErrors)
			}
		})
	}
}

func TestBoundsComputation(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		maxIterations int64
		maxLLMCalls   int64
		maxAPICalls   int64
	}{
		{
			name: "simple loop",
			code: `
for i in range(10):
    x = i * 2
`,
			maxIterations: 10,
			maxLLMCalls:   0,
			maxAPICalls:   0,
		},
		{
			name: "loop with limit modifier",
			code: `
for item in items with limit(50):
    process(item)
`,
			maxIterations: 50,
			maxLLMCalls:   0,
			maxAPICalls:   0,
		},
		{
			name: "llm call in loop",
			code: `
for i in range(5):
    result = llm.call(prompt: "test", schema: {answer: "string"})
`,
			maxIterations: 5,
			maxLLMCalls:   1,
			maxAPICalls:   0,
		},
		{
			name: "api call",
			code: `
result = api.query("SELECT * FROM users")
`,
			maxIterations: 0,
			maxLLMCalls:   0,
			maxAPICalls:   1,
		},
		{
			name: "multiple loops",
			code: `
for i in range(10):
    x = i

for j in range(5):
    y = j
`,
			maxIterations: 15, // 10 + 5
			maxLLMCalls:   0,
			maxAPICalls:   0,
		},
		{
			name: "llm call outside loop",
			code: `
plan = llm.call(prompt: "make plan", schema: {steps: "list"})
answer = llm.call(prompt: "answer", schema: {text: "string"})
`,
			maxIterations: 0,
			maxLLMCalls:   2,
			maxAPICalls:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			a := New()
			a.Analyze(program)
			bounds := a.Bounds()

			assert.Equal(t, tt.maxIterations, bounds.MaxIterations, "MaxIterations")
			assert.Equal(t, tt.maxLLMCalls, bounds.MaxLLMCalls, "MaxLLMCalls")
			assert.Equal(t, tt.maxAPICalls, bounds.MaxAPICalls, "MaxAPICalls")
		})
	}
}

func TestCycleDetection(t *testing.T) {
	tests := []struct {
		name     string
		graph    map[string][]string
		hasCycle bool
	}{
		{
			name: "no cycle - linear",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {},
			},
			hasCycle: false,
		},
		{
			name: "no cycle - branching",
			graph: map[string][]string{
				"a": {"b", "c"},
				"b": {"d"},
				"c": {"d"},
				"d": {},
			},
			hasCycle: false,
		},
		{
			name: "self loop",
			graph: map[string][]string{
				"a": {"a"},
			},
			hasCycle: true,
		},
		{
			name: "two node cycle",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"a"},
			},
			hasCycle: true,
		},
		{
			name: "three node cycle",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c"},
				"c": {"a"},
			},
			hasCycle: true,
		},
		{
			name: "cycle with dangling nodes",
			graph: map[string][]string{
				"a": {"b"},
				"b": {"c", "d"},
				"c": {"a"},
				"d": {},
			},
			hasCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycles := findCycles(tt.graph)
			if tt.hasCycle {
				assert.NotEmpty(t, cycles, "expected cycle to be detected")
			} else {
				assert.Empty(t, cycles, "expected no cycles")
			}
		})
	}
}

func TestExtractLimit(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedLimit int64
	}{
		{
			name:          "range with literal",
			code:          "for i in range(25):\n    x = i",
			expectedLimit: 25,
		},
		{
			name:          "limit modifier",
			code:          "for item in items with limit(100):\n    process(item)",
			expectedLimit: 100,
		},
		{
			name:          "unknown iterable defaults to 1000",
			code:          "for item in get_items():\n    process(item)",
			expectedLimit: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code)
			p := parser.New(l)
			program := p.ParseProgram()

			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			a := New()
			a.Analyze(program)
			bounds := a.Bounds()

			assert.Equal(t, tt.expectedLimit, bounds.MaxIterations)
		})
	}
}

func TestDebugRange(t *testing.T) {
	code := "for i in range(10):\n    x = i"
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()

	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}

	for _, stmt := range program.Statements {
		t.Logf("Statement type: %T", stmt)
		if forStmt, ok := stmt.(*ast.ForStatement); ok {
			t.Logf("Iterable type: %T", forStmt.Iterable)
			t.Logf("Iterable string: %s", forStmt.Iterable.String())
			if call, ok := forStmt.Iterable.(*ast.CallExpression); ok {
				t.Logf("Function type: %T", call.Function)
				t.Logf("Args count: %d", len(call.Arguments))
			}
		}
	}
}

func TestFormatCycle(t *testing.T) {
	tests := []struct {
		cycle    []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a -> b"},
		{[]string{"a", "b", "c", "a"}, "a -> b -> c -> a"},
	}

	for _, tt := range tests {
		result := formatCycle(tt.cycle)
		assert.Equal(t, tt.expected, result)
	}
}
