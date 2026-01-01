package chat

import (
	"testing"

	"github.com/anthropics/slop/examples/chat-app/internal/config"
	"github.com/anthropics/slop/internal/evaluator"
)

// TestInputVariableEdgeCases tests edge cases with input variable handling
func TestInputVariableEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		inputData   map[string]evaluator.Value
		shouldError bool
		checkResult func(t *testing.T, val evaluator.Value)
	}{
		{
			name:   "input with missing field returns default",
			script: `result = input.missing_field or "default_value"`,
			inputData: map[string]evaluator.Value{
				"existing": &evaluator.StringValue{Value: "exists"},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				str, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if str.Value != "default_value" {
					t.Errorf("expected 'default_value', got '%s'", str.Value)
				}
			},
		},
		{
			name:   "input with existing field uses value",
			script: `result = input.message or "default"`,
			inputData: map[string]evaluator.Value{
				"message": &evaluator.StringValue{Value: "hello"},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				str, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if str.Value != "hello" {
					t.Errorf("expected 'hello', got '%s'", str.Value)
				}
			},
		},
		{
			name:   "input with empty string uses empty string (not default)",
			script: `result = input.value or "default"`,
			inputData: map[string]evaluator.Value{
				"value": &evaluator.StringValue{Value: ""},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				// Empty string is falsy, so 'or' should return default
				str, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if str.Value != "default" {
					t.Errorf("expected 'default' (empty string is falsy), got '%s'", str.Value)
				}
			},
		},
		{
			name:   "input with zero integer uses zero (falsy)",
			script: `result = input.count or 10`,
			inputData: map[string]evaluator.Value{
				"count": &evaluator.IntValue{Value: 0},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				// Zero is falsy, so 'or' should return 10
				intVal, ok := val.(*evaluator.IntValue)
				if !ok {
					t.Errorf("expected IntValue, got %T", val)
					return
				}
				if intVal.Value != 10 {
					t.Errorf("expected 10 (zero is falsy), got %d", intVal.Value)
				}
			},
		},
		{
			name:   "input with non-zero integer uses value",
			script: `result = input.count or 10`,
			inputData: map[string]evaluator.Value{
				"count": &evaluator.IntValue{Value: 5},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				intVal, ok := val.(*evaluator.IntValue)
				if !ok {
					t.Errorf("expected IntValue, got %T", val)
					return
				}
				if intVal.Value != 5 {
					t.Errorf("expected 5, got %d", intVal.Value)
				}
			},
		},
		{
			name:   "input with false boolean uses default (falsy)",
			script: `result = input.flag or true`,
			inputData: map[string]evaluator.Value{
				"flag": evaluator.FALSE,
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				// False is falsy, so 'or' should return true
				boolVal, ok := val.(*evaluator.BoolValue)
				if !ok {
					t.Errorf("expected BoolValue, got %T", val)
					return
				}
				if !boolVal.Value {
					t.Errorf("expected true (false is falsy)")
				}
			},
		},
		{
			name:   "input with true boolean uses value",
			script: `result = input.flag or false`,
			inputData: map[string]evaluator.Value{
				"flag": evaluator.TRUE,
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				boolVal, ok := val.(*evaluator.BoolValue)
				if !ok {
					t.Errorf("expected BoolValue, got %T", val)
					return
				}
				if !boolVal.Value {
					t.Errorf("expected true, got false")
				}
			},
		},
		{
			name:   "input with list field accessed",
			script: `result = len(input.tags)`,
			inputData: map[string]evaluator.Value{
				"tags": &evaluator.ListValue{Elements: []evaluator.Value{
					&evaluator.IntValue{Value: 1},
					&evaluator.IntValue{Value: 2},
					&evaluator.IntValue{Value: 3},
				}},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				intVal, ok := val.(*evaluator.IntValue)
				if !ok {
					t.Errorf("expected IntValue, got %T", val)
					return
				}
				if intVal.Value != 3 {
					t.Errorf("expected 3, got %d", intVal.Value)
				}
			},
		},
		{
			name:   "input with complex nested data",
			script: `result = input.user.profile.name or "Anonymous"`,
			inputData: map[string]evaluator.Value{
				"user": &evaluator.MapValue{
					Pairs: map[string]evaluator.Value{
						"profile": &evaluator.MapValue{
							Pairs: map[string]evaluator.Value{
								"name": &evaluator.StringValue{Value: "Alice"},
							},
						},
					},
				},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				str, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if str.Value != "Alice" {
					t.Errorf("expected 'Alice', got '%s'", str.Value)
				}
			},
		},
		{
			name:   "input with multiple fields accessed",
			script: `name = input.name or "Unknown"
age = input.age or 0
result = name + " is " + str(age) + " years old"`,
			inputData: map[string]evaluator.Value{
				"name": &evaluator.StringValue{Value: "Bob"},
				"age":  &evaluator.IntValue{Value: 25},
			},
			shouldError: false,
			checkResult: func(t *testing.T, val evaluator.Value) {
				str, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if str.Value != "Bob is 25 years old" {
					t.Errorf("expected 'Bob is 25 years old', got '%s'", str.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey: "test-key",
				Model:  "claude-sonnet",
			}
			app, err := New(cfg, false)
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}
			defer app.Close()

			// Create input map value
			inputMap := &evaluator.MapValue{Pairs: tt.inputData}

			// Execute script with input
			result, err := app.executeScriptWithInput(tt.script, inputMap)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, but execution succeeded")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// executeScriptWithInput is a helper to execute a script with custom input
func (a *App) executeScriptWithInput(script string, input *evaluator.MapValue) (evaluator.Value, error) {
	// Parse the script
	program, err := a.runtime.Parse(script)
	if err != nil {
		return nil, err
	}

	// Execute with input
	ctx := a.runtime.Context()
	ctx.PushScope()
	defer ctx.PopScope()

	// Set input variable
	ctx.Scope.Set("input", input)

	// Evaluate the parsed program
	return a.runtime.Eval(program)
}

// TestInputTypeConversions tests edge cases with type conversions on input values
func TestInputTypeConversions(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		inputValue  evaluator.Value
		shouldError bool
		checkResult func(t *testing.T, val evaluator.Value)
	}{
		{
			name:       "convert input string to int",
			script:     `result = int(input.value)`,
			inputValue: &evaluator.StringValue{Value: "42"},
			checkResult: func(t *testing.T, val evaluator.Value) {
				intVal, ok := val.(*evaluator.IntValue)
				if !ok {
					t.Errorf("expected IntValue, got %T", val)
					return
				}
				if intVal.Value != 42 {
					t.Errorf("expected 42, got %d", intVal.Value)
				}
			},
		},
		{
			name:       "convert input int to string",
			script:     `result = str(input.value)`,
			inputValue: &evaluator.IntValue{Value: 123},
			checkResult: func(t *testing.T, val evaluator.Value) {
				strVal, ok := val.(*evaluator.StringValue)
				if !ok {
					t.Errorf("expected StringValue, got %T", val)
					return
				}
				if strVal.Value != "123" {
					t.Errorf("expected '123', got '%s'", strVal.Value)
				}
			},
		},
		{
			name:       "convert input list to set",
			script:     `result = set(input.value)`,
			inputValue: &evaluator.ListValue{Elements: []evaluator.Value{
				&evaluator.StringValue{Value: "a"},
				&evaluator.StringValue{Value: "b"},
				&evaluator.StringValue{Value: "a"}, // duplicate
			}},
			checkResult: func(t *testing.T, val evaluator.Value) {
				setVal, ok := val.(*evaluator.SetValue)
				if !ok {
					t.Errorf("expected SetValue, got %T", val)
					return
				}
				// Set should have 2 unique elements
				if len(setVal.Elements) != 2 {
					t.Errorf("expected set with 2 elements, got %d", len(setVal.Elements))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey: "test-key",
				Model:  "claude-sonnet",
			}
			app, err := New(cfg, false)
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}
			defer app.Close()

			// Create input with the test value
			inputMap := &evaluator.MapValue{
				Pairs: map[string]evaluator.Value{
					"value": tt.inputValue,
				},
			}

			result, err := app.executeScriptWithInput(tt.script, inputMap)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, but execution succeeded")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestInputValidation tests validation of input data
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		inputData   map[string]evaluator.Value
		shouldError bool
	}{
		{
			name:   "input validation - type checking with is_string",
			script: `result = is_string(input.name)`,
			inputData: map[string]evaluator.Value{
				"name": &evaluator.StringValue{Value: "Alice"},
			},
			shouldError: false,
		},
		{
			name:   "input validation - type checking with is_int",
			script: `result = is_int(input.age)`,
			inputData: map[string]evaluator.Value{
				"age": &evaluator.IntValue{Value: 25},
			},
			shouldError: false,
		},
		{
			name:   "input validation - using or for required fields",
			script: `result = input.required_field or "default value provided"`,
			inputData: map[string]evaluator.Value{
				"required_field": &evaluator.StringValue{Value: "present"},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				APIKey: "test-key",
				Model:  "claude-sonnet",
			}
			app, err := New(cfg, false)
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}
			defer app.Close()

			inputMap := &evaluator.MapValue{Pairs: tt.inputData}
			_, err = app.executeScriptWithInput(tt.script, inputMap)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error, but execution succeeded")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
