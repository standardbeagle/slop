package chat

import (
	"testing"

	"github.com/standardbeagle/slop/examples/chat-app/internal/config"
	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
)

// TestSchemaEdgeCases tests edge cases in schema validation and parsing
func TestSchemaEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		shouldParse bool
		expectError bool
	}{
		{
			name: "empty schema",
			script: `
result = llm.call(prompt: "test", schema: {})
emit(result)
`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "nested schema two levels",
			script: `result = llm.call(prompt: "test", schema: {user: {name: string, age: number}})
emit(result)`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "schema with all basic types",
			script: `result = llm.call(prompt: "test", schema: {str_field: string, num_field: number, int_field: integer, bool_field: boolean, list_field: list, obj_field: object})
emit(result)`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "schema with list type using builtin",
			script: `
result = llm.call(
    prompt: "test",
    schema: {queries: list, count: number}
)
emit(result)
`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "schema with mixed nested structures",
			script: `result = llm.call(prompt: "test", schema: {metadata: {version: string, timestamp: number}, items: list, flags: {active: boolean, verified: boolean}})
emit(result)`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "schema using array builtin",
			script: `
result = llm.call(
    prompt: "test",
    schema: {items: array, count: integer}
)
emit(result)
`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "single field schema",
			script: `
result = llm.call(
    prompt: "test",
    schema: {response: string}
)
emit(result)
`,
			shouldParse: true,
			expectError: false,
		},
		{
			name: "schema with many fields",
			script: `result = llm.call(prompt: "test", schema: {f1: string, f2: string, f3: string, f4: string, f5: string, f6: number, f7: number, f8: boolean, f9: boolean, f10: list})
emit(result)`,
			shouldParse: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parsing
			l := lexer.New(tt.script)
			p := parser.New(l)
			program := p.ParseProgram()

			if tt.shouldParse {
				if len(p.Errors()) > 0 {
					t.Errorf("expected script to parse, but got errors:")
					for _, err := range p.Errors() {
						t.Errorf("  %s", err)
					}
					return
				}
			} else {
				if len(p.Errors()) == 0 {
					t.Errorf("expected parsing to fail, but it succeeded")
					return
				}
				// Parsing failed as expected
				return
			}

			// Test evaluation (with mock LLM)
			cfg := &config.Config{
				APIKey: "test-key",
				Model:  "claude-sonnet",
			}
			app, err := New(cfg, false)
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}
			defer app.Close()

			// Evaluate the program
			ctx := app.runtime.Context()
			ctx.PushScope()
			defer ctx.PopScope()

			_, err = app.runtime.Eval(program)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected evaluation to fail, but it succeeded")
				}
			} else {
				if err != nil {
					t.Errorf("expected evaluation to succeed, but got error: %v", err)
				}
			}
		})
	}
}

// TestSchemaTypeIdentifierEdgeCases tests edge cases with type identifiers
func TestSchemaTypeIdentifierEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		typeRef     string
		shouldWork  bool
	}{
		{"string type", "string", true},
		{"number type", "number", true},
		{"integer type", "integer", true},
		{"boolean type", "boolean", true},
		{"list type", "list", true},
		{"array type", "array", true},
		{"object type", "object", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := `result = llm.call(prompt: "test", schema: {field: ` + tt.typeRef + `})`

			l := lexer.New(script)
			p := parser.New(l)
			_ = p.ParseProgram()

			if tt.shouldWork {
				if len(p.Errors()) > 0 {
					t.Errorf("expected script to parse, but got errors:")
					for _, err := range p.Errors() {
						t.Errorf("  %s", err)
					}
				}
			}
		})
	}
}

// TestSchemaWithInvalidTypes tests that invalid type references fail appropriately
func TestSchemaWithInvalidTypes(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{
			name: "undefined type reference",
			script: `result = llm.call(prompt: "test", schema: {field: undefined_type})`,
		},
		{
			name: "numeric literal as type",
			script: `result = llm.call(prompt: "test", schema: {field: 123})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.script)
			p := parser.New(l)
			program := p.ParseProgram()

			// These should parse (syntax is valid)
			if len(p.Errors()) > 0 {
				// Parsing error is acceptable for some edge cases
				return
			}

			// But should fail at evaluation
			cfg := &config.Config{
				APIKey: "test-key",
				Model:  "claude-sonnet",
			}
			app, err := New(cfg, false)
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}
			defer app.Close()

			ctx := app.runtime.Context()
			ctx.PushScope()
			defer ctx.PopScope()

			_, err = app.runtime.Eval(program)
			if err == nil {
				t.Errorf("expected evaluation to fail with invalid type, but it succeeded")
			}
		})
	}
}

// TestSchemaFieldNames tests edge cases with field naming
func TestSchemaFieldNames(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		shouldParse bool
	}{
		{"simple name", "response", true},
		{"underscore name", "user_name", true},
		{"camelCase", "userName", true},
		{"single char", "x", true},
		{"with numbers", "field123", true},
		{"all caps", "API_KEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := `result = llm.call(prompt: "test", schema: {` + tt.fieldName + `: string})`

			l := lexer.New(script)
			p := parser.New(l)
			_ = p.ParseProgram()

			if tt.shouldParse {
				if len(p.Errors()) > 0 {
					t.Errorf("expected script to parse, but got errors:")
					for _, err := range p.Errors() {
						t.Errorf("  %s", err)
					}
				}
			} else {
				if len(p.Errors()) == 0 {
					t.Errorf("expected parsing to fail, but it succeeded")
				}
			}
		})
	}
}

// TestSchemaResponseValidation tests that mock responses match schema structure
func TestSchemaResponseValidation(t *testing.T) {
	cfg := &config.Config{
		APIKey: "test-key",
		Model:  "claude-sonnet",
	}
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	tests := []struct {
		name         string
		script       string
		checkResult  func(t *testing.T, val evaluator.Value)
	}{
		{
			name: "string field returns string",
			script: `result = llm.call(prompt: "test", schema: {response: string})`,
			checkResult: func(t *testing.T, val evaluator.Value) {
				mapVal, ok := val.(*evaluator.MapValue)
				if !ok {
					t.Errorf("expected MapValue, got %T", val)
					return
				}
				if response, ok := mapVal.Pairs["response"]; ok {
					if _, ok := response.(*evaluator.StringValue); !ok {
						t.Errorf("expected response to be StringValue, got %T", response)
					}
				} else {
					t.Errorf("expected 'response' field in result")
				}
			},
		},
		{
			name: "number field returns number",
			script: `result = llm.call(prompt: "test", schema: {score: number})`,
			checkResult: func(t *testing.T, val evaluator.Value) {
				mapVal, ok := val.(*evaluator.MapValue)
				if !ok {
					t.Errorf("expected MapValue, got %T", val)
					return
				}
				if score, ok := mapVal.Pairs["score"]; ok {
					if _, ok := score.(*evaluator.FloatValue); !ok {
						t.Errorf("expected score to be FloatValue, got %T", score)
					}
				} else {
					t.Errorf("expected 'score' field in result")
				}
			},
		},
		{
			name: "list field returns list",
			script: `result = llm.call(prompt: "test", schema: {items: list})`,
			checkResult: func(t *testing.T, val evaluator.Value) {
				mapVal, ok := val.(*evaluator.MapValue)
				if !ok {
					t.Errorf("expected MapValue, got %T", val)
					return
				}
				if items, ok := mapVal.Pairs["items"]; ok {
					if _, ok := items.(*evaluator.ListValue); !ok {
						t.Errorf("expected items to be ListValue, got %T", items)
					}
				} else {
					t.Errorf("expected 'items' field in result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.script)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parsing failed: %v", p.Errors())
			}

			ctx := app.runtime.Context()
			ctx.PushScope()
			defer ctx.PopScope()

			result, err := app.runtime.Eval(program)
			if err != nil {
				t.Fatalf("evaluation failed: %v", err)
			}

			tt.checkResult(t, result)
		})
	}
}
