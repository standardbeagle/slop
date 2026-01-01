package parser

import (
	"testing"

	"github.com/anthropics/slop/internal/ast"
	"github.com/anthropics/slop/internal/lexer"
)

func TestParseSourceModule(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectName   string
		expectID     string
		expectUses   map[string]string
		expectProvides []string
		expectBodyLen int
	}{
		{
			name: "simple source module",
			input: `===SOURCE: utils===
def clean(s):
    return s.strip()
`,
			expectName:   "utils",
			expectBodyLen: 1,
		},
		{
			name: "source with id",
			input: `===SOURCE: utils===
id: "mycompany/utils@v1"
---
def clean(s):
    return s.strip()
`,
			expectName:   "utils",
			expectID:     "mycompany/utils@v1",
			expectBodyLen: 1,
		},
		{
			name: "source with uses",
			input: `===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1"}
---
def process(item):
    return utils.clean(item.name)
`,
			expectName:   "processor",
			expectID:     "mycompany/processor@v1",
			expectUses:   map[string]string{"utils": "mycompany/utils@v1"},
			expectBodyLen: 1,
		},
		{
			name: "source with provides",
			input: `===SOURCE: helpers===
id: "lib/helpers@v1"
provides: [func1, func2]
---
def func1():
    pass

def func2():
    pass
`,
			expectName:     "helpers",
			expectID:       "lib/helpers@v1",
			expectProvides: []string{"func1", "func2"},
			expectBodyLen:  2,
		},
		{
			name: "full source module",
			input: `===SOURCE: processor===
id: "mycompany/processor@v1"
uses: {utils: "mycompany/utils@v1", config: "mycompany/config@v1"}
provides: [process, transform]
---
def process(item):
    return utils.clean(item)

def transform(item):
    return item.upper()
`,
			expectName:     "processor",
			expectID:       "mycompany/processor@v1",
			expectUses:     map[string]string{"utils": "mycompany/utils@v1", "config": "mycompany/config@v1"},
			expectProvides: []string{"process", "transform"},
			expectBodyLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Modules) == 0 {
				t.Fatal("expected at least one module, got 0")
			}

			module := program.Modules[0]

			if module.Type != "SOURCE" {
				t.Errorf("expected module type SOURCE, got %s", module.Type)
			}

			if module.Name != tt.expectName {
				t.Errorf("expected name %s, got %s", tt.expectName, module.Name)
			}

			if tt.expectID != "" && module.ID != tt.expectID {
				t.Errorf("expected id %s, got %s", tt.expectID, module.ID)
			}

			if tt.expectUses != nil {
				for k, v := range tt.expectUses {
					if module.Uses[k] != v {
						t.Errorf("expected uses[%s] = %s, got %s", k, v, module.Uses[k])
					}
				}
			}

			if tt.expectProvides != nil {
				if len(module.Provides) != len(tt.expectProvides) {
					t.Errorf("expected %d provides, got %d", len(tt.expectProvides), len(module.Provides))
				}
				for i, p := range tt.expectProvides {
					if i < len(module.Provides) && module.Provides[i] != p {
						t.Errorf("expected provides[%d] = %s, got %s", i, p, module.Provides[i])
					}
				}
			}

			if len(module.Body) != tt.expectBodyLen {
				t.Errorf("expected %d body statements, got %d", tt.expectBodyLen, len(module.Body))
			}
		})
	}
}

func TestParseUseModule(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectName  string
		expectWith  map[string]string
	}{
		{
			name:       "simple use",
			input:      `===USE: mycompany/utils===`,
			expectName: "mycompany/utils",
		},
		{
			name:       "use with version",
			input:      `===USE: mycompany/utils@v1===`,
			expectName: "mycompany/utils@v1",
		},
		{
			name:       "use with path",
			input:      `===USE: github.com/company/mylib@v1.2.3===`,
			expectName: "github.com/company/mylib@v1.2.3",
		},
		{
			name: "use with remapping",
			input: `===USE: mycompany/processor with {utils: my/utils}===`,
			expectName: "mycompany/processor",
			expectWith: map[string]string{"utils": "my/utils"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Modules) == 0 {
				t.Fatal("expected at least one module, got 0")
			}

			module := program.Modules[0]

			if module.Type != "USE" {
				t.Errorf("expected module type USE, got %s", module.Type)
			}

			if module.Name != tt.expectName {
				t.Errorf("expected name %s, got %s", tt.expectName, module.Name)
			}

			if tt.expectWith != nil {
				for k, v := range tt.expectWith {
					if module.WithClauses[k] != v {
						t.Errorf("expected with[%s] = %s, got %s", k, v, module.WithClauses[k])
					}
				}
			}
		})
	}
}

func TestParseMainModule(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectBodyLen int
	}{
		{
			name: "simple main",
			input: `===MAIN===
x = 1 + 2
emit(x)
`,
			expectBodyLen: 2,
		},
		{
			name: "main with function calls",
			input: `===MAIN===
result = processor.process(data)
emit(result)
`,
			expectBodyLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors: %v", p.Errors())
			}

			if len(program.Modules) == 0 {
				t.Fatal("expected at least one module, got 0")
			}

			module := program.Modules[0]

			if module.Type != "MAIN" {
				t.Errorf("expected module type MAIN, got %s", module.Type)
			}

			if len(module.Body) != tt.expectBodyLen {
				t.Errorf("expected %d body statements, got %d", tt.expectBodyLen, len(module.Body))
			}
		})
	}
}

func TestParseMultipleModules(t *testing.T) {
	input := `===SOURCE: utils===
id: "myapp/utils@v1"
uses: {}
---
def helper():
    pass

===SOURCE: processor===
id: "myapp/processor@v1"
uses: {utils: "myapp/utils@v1"}
---
def process():
    return utils.helper()

===MAIN===
processor.process()
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if len(program.Modules) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(program.Modules))
	}

	// Check first module (utils)
	utils := program.Modules[0]
	if utils.Type != "SOURCE" || utils.Name != "utils" {
		t.Errorf("expected SOURCE:utils, got %s:%s", utils.Type, utils.Name)
	}
	if utils.ID != "myapp/utils@v1" {
		t.Errorf("expected id myapp/utils@v1, got %s", utils.ID)
	}
	if len(utils.Body) != 1 {
		t.Errorf("expected 1 body statement in utils, got %d", len(utils.Body))
	}

	// Check second module (processor)
	processor := program.Modules[1]
	if processor.Type != "SOURCE" || processor.Name != "processor" {
		t.Errorf("expected SOURCE:processor, got %s:%s", processor.Type, processor.Name)
	}
	if processor.Uses["utils"] != "myapp/utils@v1" {
		t.Errorf("expected uses[utils] = myapp/utils@v1, got %s", processor.Uses["utils"])
	}
	if len(processor.Body) != 1 {
		t.Errorf("expected 1 body statement in processor, got %d", len(processor.Body))
	}

	// Check third module (main)
	main := program.Modules[2]
	if main.Type != "MAIN" {
		t.Errorf("expected MAIN, got %s", main.Type)
	}
	if len(main.Body) != 1 {
		t.Errorf("expected 1 body statement in main, got %d", len(main.Body))
	}
}

func TestParseModuleWithUseBlocks(t *testing.T) {
	input := `===USE: lib/strings===
===USE: lib/numbers===
===USE: mycompany/processor with {utils: other/utils}===

===MAIN===
clean_name = strings.clean(name)
rounded = numbers.round(value, 2)
result = processor.process(clean_name)
emit(result)
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if len(program.Modules) != 4 {
		t.Fatalf("expected 4 modules, got %d", len(program.Modules))
	}

	// Check USE modules
	for i := 0; i < 3; i++ {
		if program.Modules[i].Type != "USE" {
			t.Errorf("expected module %d to be USE, got %s", i, program.Modules[i].Type)
		}
	}

	// Check third USE has with clause
	if program.Modules[2].WithClauses["utils"] != "other/utils" {
		t.Errorf("expected with[utils] = other/utils, got %s", program.Modules[2].WithClauses["utils"])
	}

	// Check MAIN module
	if program.Modules[3].Type != "MAIN" {
		t.Errorf("expected MAIN, got %s", program.Modules[3].Type)
	}
	if len(program.Modules[3].Body) != 4 {
		t.Errorf("expected 4 body statements, got %d", len(program.Modules[3].Body))
	}
}

func TestParseModuleDefStatements(t *testing.T) {
	input := `===SOURCE: utils===
id: "lib/utils@v1"
---
def clean(s):
    return s.strip().lower()

def normalize(s):
    parts = s.split()
    return " ".join(parts)
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if len(program.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(program.Modules))
	}

	module := program.Modules[0]
	if len(module.Body) != 2 {
		t.Fatalf("expected 2 body statements, got %d", len(module.Body))
	}

	// Check both are DefStatements
	for i, stmt := range module.Body {
		def, ok := stmt.(*ast.DefStatement)
		if !ok {
			t.Errorf("expected DefStatement at %d, got %T", i, stmt)
			continue
		}

		if i == 0 && def.Name.Value != "clean" {
			t.Errorf("expected function name 'clean', got %s", def.Name.Value)
		}
		if i == 1 && def.Name.Value != "normalize" {
			t.Errorf("expected function name 'normalize', got %s", def.Name.Value)
		}
	}
}

func TestParseMixedModulesAndStatements(t *testing.T) {
	// Test that regular statements without module headers work
	input := `x = 1
y = 2
emit(x + y)
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	// No modules expected
	if len(program.Modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(program.Modules))
	}

	// 3 statements expected
	if len(program.Statements) != 3 {
		t.Errorf("expected 3 statements, got %d", len(program.Statements))
	}
}

func TestModuleWithForLoop(t *testing.T) {
	input := `===SOURCE: processor===
---
def process_all(items):
    results = []
    for item in items with limit(100):
        results.append(item.upper())
    return results
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	if len(program.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(program.Modules))
	}

	module := program.Modules[0]
	if len(module.Body) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(module.Body))
	}

	def, ok := module.Body[0].(*ast.DefStatement)
	if !ok {
		t.Fatalf("expected DefStatement, got %T", module.Body[0])
	}

	if def.Name.Value != "process_all" {
		t.Errorf("expected function name 'process_all', got %s", def.Name.Value)
	}
}
