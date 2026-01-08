package chat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/standardbeagle/slop/examples/chat-app/internal/config"
	"github.com/standardbeagle/slop/internal/evaluator"
)

// TestRunScriptWithInput tests providing input context to scripts
func TestRunScriptWithInput(t *testing.T) {
	// Create a simple test script
	tmpDir := t.TempDir()
	testScript := filepath.Join(tmpDir, "test_input.slop")
	scriptContent := `# Test script that uses input
message = input.message or "default"
emit(message)
`
	if err := os.WriteFile(testScript, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	cfg := config.DefaultConfig()
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// RunScript now provides an empty input map by default
	err = app.RunScript(testScript)
	if err != nil {
		t.Errorf("script should work with input: %v", err)
	}
}

// TestInputVariableProvided documents that 'input' is now provided
func TestInputVariableProvided(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "with input reference",
			script: `x = input.value or 5`,
		},
		{
			name:   "without input reference",
			script: `x = 5`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			scriptPath := filepath.Join(tmpDir, "test.slop")
			if err := os.WriteFile(scriptPath, []byte(tt.script), 0644); err != nil {
				t.Fatalf("failed to create test script: %v", err)
			}

			cfg := config.DefaultConfig()
			app, _ := New(cfg, false)
			defer app.Close()

			err := app.RunScript(scriptPath)
			if err != nil {
				t.Errorf("script should work: %v", err)
			}
		})
	}
}

// TestScriptContextProvision tests what context is provided to scripts
func TestScriptContextProvision(t *testing.T) {
	tmpDir := t.TempDir()
	testScript := filepath.Join(tmpDir, "test_context.slop")

	// Test that 'history' and 'model' are available
	scriptContent := `# Test what context is available
# We expect 'history' and 'model' to be defined
result = model
emit(result)
`
	if err := os.WriteFile(testScript, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Model = "test-model"
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	err = app.RunScript(testScript)
	if err != nil {
		t.Errorf("script should work with 'model' context: %v", err)
	}
}

// TestRunScriptWithInputContext tests that input context is provided
func TestRunScriptWithInputContext(t *testing.T) {
	tmpDir := t.TempDir()
	testScript := filepath.Join(tmpDir, "test_input.slop")
	scriptContent := `message = input.message or "default"
emit(message)
`
	if err := os.WriteFile(testScript, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to create test script: %v", err)
	}

	cfg := config.DefaultConfig()
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// RunScript now provides input context
	err = app.RunScript(testScript)
	if err != nil {
		t.Errorf("script should work with input context: %v", err)
	}
}

// TestCreateInputMapValue tests creating an input map for scripts
func TestCreateInputMapValue(t *testing.T) {
	// Helper to create input map values for scripts
	inputMap := evaluator.NewMapValue()
	inputMap.Set("message", &evaluator.StringValue{Value: "test message"})
	inputMap.Set("count", &evaluator.IntValue{Value: 42})

	// Verify structure
	msg, ok := inputMap.Get("message")
	if !ok {
		t.Error("failed to get message from input map")
	}

	strMsg, ok := msg.(*evaluator.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", msg)
	}

	if strMsg.Value != "test message" {
		t.Errorf("expected 'test message', got '%s'", strMsg.Value)
	}

	count, ok := inputMap.Get("count")
	if !ok {
		t.Error("failed to get count from input map")
	}

	intCount, ok := count.(*evaluator.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", count)
	}

	if intCount.Value != 42 {
		t.Errorf("expected 42, got %d", intCount.Value)
	}
}
