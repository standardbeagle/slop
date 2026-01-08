package evaluator

import (
	"strings"
	"testing"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
)

func TestModuleResolverLoadModules(t *testing.T) {
	input := `===SOURCE: utils===
id: "myapp/utils@v1"
uses: {}
---
def helper():
    return "hello"

===SOURCE: processor===
id: "myapp/processor@v1"
uses: {utils: "myapp/utils@v1"}
---
def process():
    return utils.helper()

===MAIN===
result = processor.process()
emit(result)
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	resolver := NewModuleResolver()
	err := resolver.LoadModules(program.Modules)
	if err != nil {
		t.Fatalf("LoadModules error: %v", err)
	}

	// Check that modules were loaded
	if len(resolver.sourcesByID) != 2 {
		t.Errorf("expected 2 sources by ID, got %d", len(resolver.sourcesByID))
	}

	if resolver.mainModule == nil {
		t.Error("expected MAIN module to be loaded")
	}

	// Check specific modules
	if resolver.sourcesByID["myapp/utils@v1"] == nil {
		t.Error("expected myapp/utils@v1 to be loaded")
	}

	if resolver.sourcesByID["myapp/processor@v1"] == nil {
		t.Error("expected myapp/processor@v1 to be loaded")
	}
}

func TestModuleResolverValidate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid dependencies",
			input: `===SOURCE: utils===
id: "lib/utils@v1"
uses: {}
---
def clean():
    pass

===SOURCE: processor===
id: "lib/processor@v1"
uses: {utils: "lib/utils@v1"}
---
def process():
    pass

===USE: lib/utils@v1===
===USE: lib/processor@v1===

===MAIN===
processor.process()
`,
			expectError: false,
		},
		{
			name: "missing dependency",
			input: `===SOURCE: processor===
id: "lib/processor@v1"
uses: {utils: "lib/utils@v1"}
---
def process():
    pass

===MAIN===
processor.process()
`,
			expectError: true,
			errorMsg:    "lib/utils@v1",
		},
		{
			name: "multiple MAIN modules",
			input: `===MAIN===
x = 1

===MAIN===
y = 2
`,
			expectError: true,
			errorMsg:    "multiple MAIN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			resolver := NewModuleResolver()
			err := resolver.LoadModules(program.Modules)

			// Check for load errors (like multiple MAIN)
			if err != nil {
				if tt.expectError && strings.Contains(err.Error(), tt.errorMsg) {
					return // Expected error found
				}
				t.Fatalf("LoadModules error: %v", err)
			}

			// Validate dependencies
			errors := resolver.Validate()

			if tt.expectError {
				if len(errors) == 0 {
					t.Error("expected validation error, got none")
				} else {
					found := false
					for _, e := range errors {
						if strings.Contains(e.Error(), tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error containing %q, got %v", tt.errorMsg, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("expected no validation errors, got %v", errors)
				}
			}
		})
	}
}

func TestModuleResolverCycleDetection(t *testing.T) {
	// Create modules with circular dependency manually
	modA := &ast.Module{
		Type: "SOURCE",
		Name: "a",
		ID:   "lib/a@v1",
		Uses: map[string]string{"b": "lib/b@v1"},
		Body: []ast.Statement{},
	}

	modB := &ast.Module{
		Type: "SOURCE",
		Name: "b",
		ID:   "lib/b@v1",
		Uses: map[string]string{"a": "lib/a@v1"},
		Body: []ast.Statement{},
	}

	resolver := NewModuleResolver()
	err := resolver.LoadModules([]*ast.Module{modA, modB})
	if err != nil {
		t.Fatalf("LoadModules error: %v", err)
	}

	errors := resolver.Validate()

	// Should detect cycle
	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "circular") {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected circular dependency error, got none")
	}
}

func TestModuleResolverResolveModule(t *testing.T) {
	modByID := &ast.Module{
		Type: "SOURCE",
		Name: "utils",
		ID:   "mycompany/utils@v1",
		Uses: map[string]string{},
		Body: []ast.Statement{},
	}

	modByName := &ast.Module{
		Type: "SOURCE",
		Name: "helpers",
		Uses: map[string]string{},
		Body: []ast.Statement{},
	}

	resolver := NewModuleResolver()
	if err := resolver.LoadModules([]*ast.Module{modByID, modByName}); err != nil {
		t.Fatalf("failed to load modules: %v", err)
	}

	tests := []struct {
		query    string
		expected string
	}{
		{"mycompany/utils@v1", "utils"},
		{"utils", "utils"},
		{"helpers", "helpers"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := resolver.ResolveModule(tt.query)
			if tt.expected == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("expected module %s, got nil", tt.expected)
				} else if result.Name != tt.expected {
					t.Errorf("expected module name %s, got %s", tt.expected, result.Name)
				}
			}
		})
	}
}

func TestGetShortName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"utils", "utils"},
		{"mycompany/utils", "utils"},
		{"mycompany/utils@v1", "utils"},
		{"github.com/company/lib@v1.2.3", "lib"},
		{"lib/a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getShortName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestModuleValue(t *testing.T) {
	scope := NewScope()
	scope.Set("foo", &IntValue{Value: 42})
	scope.Set("bar", &StringValue{Value: "hello"})

	mod := &ModuleValue{Name: "test", Scope: scope}

	// Test type and string
	if mod.Type() != "module" {
		t.Errorf("expected type 'module', got %s", mod.Type())
	}

	if !strings.Contains(mod.String(), "test") {
		t.Errorf("expected string to contain 'test', got %s", mod.String())
	}

	// Test Get
	val, ok := mod.Get("foo")
	if !ok {
		t.Error("expected to find 'foo'")
	}
	if intVal, ok := val.(*IntValue); !ok || intVal.Value != 42 {
		t.Errorf("expected IntValue(42), got %v", val)
	}

	// Test non-existent
	_, ok = mod.Get("nonexistent")
	if ok {
		t.Error("expected not to find 'nonexistent'")
	}
}

func TestModuleResolverBuildScopes(t *testing.T) {
	input := `===SOURCE: utils===
id: "lib/utils@v1"
uses: {}
---
def helper():
    return 42

===MAIN===
x = 1
`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	resolver := NewModuleResolver()
	err := resolver.LoadModules(program.Modules)
	if err != nil {
		t.Fatalf("LoadModules error: %v", err)
	}

	// Create evaluator
	e := New()

	// Build scopes
	err = resolver.BuildScopes(e)
	if err != nil {
		t.Fatalf("BuildScopes error: %v", err)
	}

	// Check that utils scope was created
	utilsScope := resolver.scopes["lib/utils@v1"]
	if utilsScope == nil {
		t.Fatal("expected utils scope to be created")
	}

	// Check that helper function exists in scope
	helperVal, ok := utilsScope.Get("helper")
	if !ok {
		t.Fatal("expected helper function to be in utils scope")
	}

	if _, ok := helperVal.(*FunctionValue); !ok {
		t.Errorf("expected FunctionValue, got %T", helperVal)
	}
}

func TestModuleIntegration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectEmit  string
		expectError bool
	}{
		{
			name: "simple module call",
			input: `===SOURCE: utils===
id: "lib/utils@v1"
uses: {}
---
def double(n):
    return n * 2

===USE: lib/utils@v1===

===MAIN===
result = utils.double(21)
emit(result)
`,
			expectEmit: "42",
		},
		{
			name: "chained module dependencies",
			input: `===SOURCE: math===
id: "lib/math@v1"
uses: {}
---
def add(a, b):
    return a + b

===SOURCE: calc===
id: "lib/calc@v1"
uses: {math: "lib/math@v1"}
---
def sum3(a, b, c):
    temp = math.add(a, b)
    return math.add(temp, c)

===USE: lib/calc@v1===

===MAIN===
result = calc.sum3(10, 20, 12)
emit(result)
`,
			expectEmit: "42",
		},
		{
			name: "multiple modules",
			input: `===SOURCE: strings===
id: "lib/strings@v1"
uses: {}
---
def greet(name):
    return "Hello, " + name

===SOURCE: numbers===
id: "lib/numbers@v1"
uses: {}
---
def square(n):
    return n * n

===USE: lib/strings@v1===
===USE: lib/numbers@v1===

===MAIN===
msg = strings.greet("World")
num = numbers.square(6)
emit(msg)
`,
			expectEmit: "Hello, World",
		},
		{
			name: "module with internal state",
			input: `===SOURCE: counter===
id: "lib/counter@v1"
uses: {}
---
count = 0

def increment():
    count = count + 1
    return count

===USE: lib/counter@v1===

===MAIN===
r1 = counter.increment()
emit(r1)
`,
			expectEmit: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			e := New()
			_, err := e.Eval(program)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got none")
				}
			} else {
				if err != nil {
					t.Fatalf("evaluation error: %v", err)
				}

				// Check emitted value
				emissions := e.Context().Emitted
				if len(emissions) == 0 {
					t.Fatal("expected emission, got none")
				}

				emitStr := emissions[0].String()
				if emitStr != tt.expectEmit {
					t.Errorf("expected emit %q, got %q", tt.expectEmit, emitStr)
				}
			}
		})
	}
}
