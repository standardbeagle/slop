package scripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/standardbeagle/slop/internal/runtime"
	"github.com/standardbeagle/slop/pkg/slop"
)

// TestHelper provides utilities for script integration tests.
type TestHelper struct {
	t       *testing.T
	runtime *slop.Runtime
}

// NewTestHelper creates a new test helper with a fresh runtime.
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t:       t,
		runtime: slop.NewRuntime(),
	}
}

// SetInput sets an input value for the script.
func (h *TestHelper) SetInput(key string, value evaluator.Value) {
	ctx := h.runtime.Context()
	input, ok := ctx.Scope.Get("input")
	if !ok || input == nil {
		input = evaluator.NewMapValue()
		ctx.Scope.Set("input", input)
	}
	if mapVal, ok := input.(*evaluator.MapValue); ok {
		mapVal.Set(key, value)
	}
}

// SetInputString sets a string input value.
func (h *TestHelper) SetInputString(key, value string) {
	h.SetInput(key, &evaluator.StringValue{Value: value})
}

// SetInputInt sets an integer input value.
func (h *TestHelper) SetInputInt(key string, value int64) {
	h.SetInput(key, &evaluator.IntValue{Value: value})
}

// RunScript runs a SLOP script file and returns emitted values and result.
func (h *TestHelper) RunScript(scriptPath string) ([]evaluator.Value, evaluator.Value, error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, nil, err
	}

	return h.RunSource(string(content))
}

// RunSource runs SLOP source code and returns emitted values and result.
func (h *TestHelper) RunSource(source string) ([]evaluator.Value, evaluator.Value, error) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		h.t.Errorf("parse errors: %v", p.Errors())
		return nil, nil, p.Errors()
	}

	// Ensure input exists
	ctx := h.runtime.Context()
	if _, ok := ctx.Scope.Get("input"); !ok {
		ctx.Scope.Set("input", evaluator.NewMapValue())
	}

	result, err := h.runtime.Eval(program)
	emitted := h.runtime.Emitted()

	return emitted, result, err
}

// Emitted returns all emitted values.
func (h *TestHelper) Emitted() []evaluator.Value {
	return h.runtime.Emitted()
}

// ClearEmitted clears emitted values.
func (h *TestHelper) ClearEmitted() {
	h.runtime.Context().Emitted = []evaluator.Value{}
}

// =========================================================================
// Simple Chat Script Tests
// =========================================================================

func TestSimpleChatIntegration_DefaultInputs(t *testing.T) {
	h := NewTestHelper(t)

	emitted, _, err := h.RunScript("simple_chat.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Should emit at least one value
	if len(emitted) == 0 {
		t.Error("expected at least one emitted value")
	}

	// First emission should be a string (the response)
	if len(emitted) > 0 {
		if _, ok := emitted[0].(*evaluator.StringValue); !ok {
			t.Errorf("expected string emission, got %T", emitted[0])
		}
	}
}

func TestSimpleChatIntegration_CustomMessage(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputString("message", "What is the weather today?")
	h.SetInputString("system", "You are a weather assistant.")

	emitted, _, err := h.RunScript("simple_chat.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Error("expected at least one emitted value")
	}
}

func TestSimpleChatIntegration_EmptyMessage(t *testing.T) {
	h := NewTestHelper(t)

	// Should use default "Hello!" message
	h.SetInputString("message", "")

	emitted, _, err := h.RunScript("simple_chat.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Error("expected emitted value even with empty message")
	}
}

// =========================================================================
// Research Script Tests
// =========================================================================

func TestResearchIntegration_DefaultTopic(t *testing.T) {
	h := NewTestHelper(t)

	emitted, _, err := h.RunScript("research.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Research script should emit summary, key_points, and confidence
	if len(emitted) < 3 {
		t.Errorf("expected at least 3 emissions (summary, key_points, confidence), got %d", len(emitted))
	}
}

func TestResearchIntegration_CustomTopic(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputString("topic", "quantum computing applications")
	h.SetInputInt("max_searches", 3)

	emitted, _, err := h.RunScript("research.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) < 3 {
		t.Errorf("expected at least 3 emissions, got %d", len(emitted))
	}
}

func TestResearchIntegration_MinimalSearches(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputInt("max_searches", 1)

	emitted, _, err := h.RunScript("research.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Should still produce output with minimal searches
	if len(emitted) < 3 {
		t.Errorf("expected at least 3 emissions with minimal searches, got %d", len(emitted))
	}
}

// =========================================================================
// Code Review Script Tests
// =========================================================================

func TestCodeReviewIntegration_DefaultCode(t *testing.T) {
	h := NewTestHelper(t)

	emitted, _, err := h.RunScript("code_review.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Should emit overall_quality, bugs, style_issues, security_issues
	if len(emitted) < 4 {
		t.Errorf("expected at least 4 emissions, got %d", len(emitted))
	}
}

func TestCodeReviewIntegration_PythonCode(t *testing.T) {
	h := NewTestHelper(t)

	pythonCode := `def add(a, b):
    return a + b`
	h.SetInputString("code", pythonCode)
	h.SetInputString("language", "python")

	emitted, _, err := h.RunScript("code_review.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) < 4 {
		t.Errorf("expected at least 4 emissions for code review, got %d", len(emitted))
	}
}

func TestCodeReviewIntegration_GoCode(t *testing.T) {
	h := NewTestHelper(t)

	goCode := `func main() {
    fmt.Println("Hello")
}`
	h.SetInputString("code", goCode)
	h.SetInputString("language", "go")

	emitted, _, err := h.RunScript("code_review.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) < 4 {
		t.Errorf("expected at least 4 emissions for Go code review, got %d", len(emitted))
	}
}

func TestCodeReviewIntegration_JavaScriptCode(t *testing.T) {
	h := NewTestHelper(t)

	jsCode := `function greet(name) { return "Hello, " + name; }`
	h.SetInputString("code", jsCode)
	h.SetInputString("language", "javascript")

	emitted, _, err := h.RunScript("code_review.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) < 4 {
		t.Errorf("expected at least 4 emissions for JavaScript code review, got %d", len(emitted))
	}
}

// =========================================================================
// Tool Agent Script Tests
// =========================================================================

func TestToolAgentIntegration_DefaultTask(t *testing.T) {
	h := NewTestHelper(t)

	emitted, _, err := h.RunScript("tool_agent.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Should emit answer and confidence
	if len(emitted) < 2 {
		t.Errorf("expected at least 2 emissions (answer, confidence), got %d", len(emitted))
	}
}

func TestToolAgentIntegration_CustomTask(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputString("task", "Calculate 2 + 2")

	emitted, _, err := h.RunScript("tool_agent.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) < 2 {
		t.Errorf("expected at least 2 emissions, got %d", len(emitted))
	}
}

func TestToolAgentIntegration_ComplexTask(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputString("task", "Search for recent news about AI and summarize the findings")

	emitted, _, err := h.RunScript("tool_agent.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	// Should still produce output even for complex tasks
	if len(emitted) < 2 {
		t.Errorf("expected at least 2 emissions for complex task, got %d", len(emitted))
	}
}

// =========================================================================
// Test Schema Script Tests
// =========================================================================

func TestSchemaIntegration_Basic(t *testing.T) {
	h := NewTestHelper(t)

	emitted, _, err := h.RunScript("test_schema.slop")
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Error("expected at least one emitted value")
	}

	// The mock returns "mock_string" for string fields
	if len(emitted) > 0 {
		if str, ok := emitted[0].(*evaluator.StringValue); ok {
			if str.Value != "mock_string" {
				t.Errorf("expected 'mock_string', got '%s'", str.Value)
			}
		}
	}
}

// =========================================================================
// Edge Cases and Error Handling Tests
// =========================================================================

func TestScript_NonExistentFile(t *testing.T) {
	h := NewTestHelper(t)

	_, _, err := h.RunScript("nonexistent.slop")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestScript_InvalidSyntax(t *testing.T) {
	// Use a script that will definitely fail parsing
	invalidScript := "def foo( { )"

	l := lexer.New(invalidScript)
	p := parser.New(l)
	p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Error("expected parse errors for invalid syntax")
	}
}

func TestScript_EmptyScript(t *testing.T) {
	h := NewTestHelper(t)

	// Empty script should be valid (just returns nil)
	_, result, err := h.RunSource("")
	if err != nil {
		t.Fatalf("empty script should not error: %v", err)
	}

	// Result should be nil or none for empty script
	if result != nil {
		if _, isNone := result.(*evaluator.NoneValue); !isNone {
			t.Errorf("expected nil/none result for empty script, got %v", result)
		}
	}
}

func TestScript_CommentOnlyScript(t *testing.T) {
	h := NewTestHelper(t)

	_, _, err := h.RunSource(`# This is just a comment
# Another comment`)
	if err != nil {
		t.Fatalf("comment-only script should not error: %v", err)
	}
}

// =========================================================================
// Input/Output Verification Tests
// =========================================================================

func TestInputPropagation(t *testing.T) {
	h := NewTestHelper(t)

	h.SetInputString("test_key", "test_value")

	// Simple script that uses input
	script := `value = input.test_key or "default"
emit(value)`

	emitted, _, err := h.RunSource(script)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Fatal("expected emitted value")
	}

	if str, ok := emitted[0].(*evaluator.StringValue); ok {
		if str.Value != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", str.Value)
		}
	} else {
		t.Errorf("expected string emission, got %T", emitted[0])
	}
}

func TestInputDefaults(t *testing.T) {
	h := NewTestHelper(t)

	// Don't set input, use defaults
	script := `value = input.missing or "default_value"
emit(value)`

	emitted, _, err := h.RunSource(script)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Fatal("expected emitted value")
	}

	if str, ok := emitted[0].(*evaluator.StringValue); ok {
		if str.Value != "default_value" {
			t.Errorf("expected 'default_value', got '%s'", str.Value)
		}
	}
}

func TestMultipleEmissions(t *testing.T) {
	h := NewTestHelper(t)

	script := `emit("first")
emit("second")
emit("third")`

	emitted, _, err := h.RunSource(script)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) != 3 {
		t.Errorf("expected 3 emissions, got %d", len(emitted))
	}

	expected := []string{"first", "second", "third"}
	for i, e := range emitted {
		if str, ok := e.(*evaluator.StringValue); ok {
			if str.Value != expected[i] {
				t.Errorf("emission %d: expected '%s', got '%s'", i, expected[i], str.Value)
			}
		}
	}
}

// =========================================================================
// LLM Integration Tests
// =========================================================================

func TestLLMCallWithSchema(t *testing.T) {
	h := NewTestHelper(t)

	script := `result = llm.call(
    prompt: "Test prompt",
    schema: {name: string, age: number}
)
emit(result.name)
emit(result.age)`

	emitted, _, err := h.RunSource(script)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) != 2 {
		t.Errorf("expected 2 emissions, got %d", len(emitted))
	}

	// Mock returns "mock_string" for string, 3.14 for number
	if len(emitted) >= 1 {
		if str, ok := emitted[0].(*evaluator.StringValue); ok {
			if str.Value != "mock_string" {
				t.Errorf("expected 'mock_string' for name, got '%s'", str.Value)
			}
		}
	}
}

func TestLLMCallWithListSchema(t *testing.T) {
	h := NewTestHelper(t)

	script := `result = llm.call(
    prompt: "Test prompt",
    schema: {items: list}
)
emit(result.items)`

	emitted, _, err := h.RunSource(script)
	if err != nil {
		t.Fatalf("script execution failed: %v", err)
	}

	if len(emitted) == 0 {
		t.Error("expected at least one emission")
	}

	// The mock may return different list representations
	// Just verify we got some value back
	if len(emitted) > 0 && emitted[0] == nil {
		t.Error("expected non-nil emission for list schema")
	}
}

// =========================================================================
// All Scripts Parse Test (comprehensive)
// =========================================================================

func TestAllScriptsExecute(t *testing.T) {
	scriptsDir := "."
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		t.Fatalf("failed to read scripts directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".slop" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			h := NewTestHelper(t)

			scriptPath := filepath.Join(scriptsDir, entry.Name())
			_, _, err := h.RunScript(scriptPath)
			if err != nil {
				t.Errorf("script execution failed: %v", err)
			}
		})
	}
}

// =========================================================================
// Regression Tests
// =========================================================================

func TestNoTripleQuotedStrings(t *testing.T) {
	// Ensure we don't accidentally use triple-quoted strings in scripts
	scriptsDir := "."
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		t.Fatalf("failed to read scripts directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".slop" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(scriptsDir, entry.Name()))
		if err != nil {
			t.Fatalf("failed to read %s: %v", entry.Name(), err)
		}

		if strings.Contains(string(content), `"""`) {
			t.Errorf("%s contains triple-quoted strings which are not supported", entry.Name())
		}
	}
}

func TestScriptPathResolution(t *testing.T) {
	// Test that scripts can be found with various path formats
	testCases := []struct {
		name string
		path string
	}{
		{"with extension", "simple_chat.slop"},
		{"without extension", "simple_chat"},
		{"full path", "./simple_chat.slop"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewTestHelper(t)

			// Try to find and parse the script
			pathsToTry := []string{tc.path}
			if !strings.HasSuffix(tc.path, ".slop") {
				pathsToTry = append(pathsToTry, tc.path+".slop")
			}

			var content []byte
			var err error
			for _, p := range pathsToTry {
				content, err = os.ReadFile(p)
				if err == nil {
					break
				}
			}

			if err != nil {
				t.Skipf("could not find script: %v", pathsToTry)
			}

			_, _, err = h.RunSource(string(content))
			if err != nil {
				t.Errorf("failed to run script with path %s: %v", tc.path, err)
			}
		})
	}
}

// =========================================================================
// Mock LLM Integration Tests - Using TestLLMClient
// =========================================================================

func TestMockLLM_BasicIntegration(t *testing.T) {
	// Create runtime with mock LLM
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	testLLM.OnPromptContaining("hello").RespondWith(map[string]any{
		"response": "Hello! How can I help you?",
	})

	rt.SetLLMClient(testLLM)

	// Execute script
	_, err := rt.Execute(`result = llm.call(
    prompt: "User says: hello world",
    schema: {response: string}
)
emit(result.response)`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify call was made
	if err := testLLM.AssertCallCount(1); err != nil {
		t.Error(err)
	}

	// Verify prompt content
	if err := testLLM.AssertPromptContains(0, "hello"); err != nil {
		t.Error(err)
	}

	// Verify response
	emitted := rt.Emitted()
	if len(emitted) == 0 {
		t.Fatal("Expected emitted response")
	}
	if !strings.Contains(emitted[0].String(), "Hello") {
		t.Errorf("Unexpected response: %s", emitted[0])
	}
}

func TestMockLLM_MultiTurnConversation(t *testing.T) {
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	callNum := 0
	testLLM.OnPromptMatching(".*").RespondWithHandler(func(req *runtime.LLMRequest) (*runtime.LLMResponse, error) {
		callNum++
		responses := []string{
			"Hello! I'm an AI assistant.",
			"The weather is sunny with 72°F.",
			"You're welcome! Have a great day!",
		}
		idx := callNum - 1
		if idx >= len(responses) {
			idx = len(responses) - 1
		}
		return &runtime.LLMResponse{
			Parsed: map[string]any{"response": responses[idx]},
		}, nil
	})

	rt.SetLLMClient(testLLM)

	_, err := rt.Execute(`r1 = llm.call(prompt: "Hi!", schema: {response: string})
emit(r1.response)
r2 = llm.call(prompt: "Weather?", schema: {response: string})
emit(r2.response)
r3 = llm.call(prompt: "Thanks!", schema: {response: string})
emit(r3.response)`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if err := testLLM.AssertCallCount(3); err != nil {
		t.Error(err)
	}

	emitted := rt.Emitted()
	if len(emitted) != 3 {
		t.Fatalf("Expected 3 emissions, got %d", len(emitted))
	}
}

func TestMockLLM_WithTestService(t *testing.T) {
	rt := slop.NewRuntime()

	// Set up mock LLM
	testLLM := runtime.NewTestLLMClient()
	testLLM.OnPromptContaining("analyze").RespondWith(map[string]any{
		"sentiment": "positive",
		"keywords":  []any{"happy", "excited", "great"},
	})
	rt.SetLLMClient(testLLM)

	// Set up mock service
	dataSvc := runtime.NewTestService("data")
	dataSvc.SetMethods("fetch")
	dataSvc.OnMethod("fetch").ReturnMap(map[string]any{
		"text": "I'm so happy and excited about this great news!",
	})
	rt.RegisterService("data", dataSvc)

	// Run workflow
	_, err := rt.Execute(`content = data.fetch(id: "123")
analysis = llm.call(
    prompt: "analyze this text: " + content.text,
    schema: {sentiment: string, keywords: list}
)
emit(analysis.sentiment)`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify service was called
	if err := dataSvc.AssertCalled("fetch"); err != nil {
		t.Error(err)
	}

	// Verify LLM received the fetched content
	if err := testLLM.AssertAnyPromptContains("happy"); err != nil {
		t.Error(err)
	}

	emitted := rt.Emitted()
	if len(emitted) == 0 {
		t.Fatal("Expected emission")
	}
	if !strings.Contains(emitted[0].String(), "positive") {
		t.Errorf("Expected positive sentiment, got: %s", emitted[0])
	}
}

func TestMockLLM_TokenTracking(t *testing.T) {
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	testLLM.OnPromptMatching(".*").RespondWith(map[string]any{"ok": true})
	rt.SetLLMClient(testLLM)

	// Execute multiple calls
	_, err := rt.Execute(`r1 = llm.call(prompt: "First prompt with content", schema: {ok: boolean})
r2 = llm.call(prompt: "Second prompt with more content", schema: {ok: boolean})
r3 = llm.call(prompt: "Third prompt with additional content", schema: {ok: boolean})`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	input, output := testLLM.TotalTokens()
	if input == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if output == 0 {
		t.Error("Expected non-zero output tokens")
	}

	// Verify cost estimation
	cost := testLLM.TokenCost(3.0, 15.0)
	if cost <= 0 {
		t.Error("Expected positive cost estimate")
	}
}

func TestMockLLM_ErrorSimulation(t *testing.T) {
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	testLLM.OnPromptContaining("fail").RespondWithError(&testError{msg: "API rate limit exceeded"})
	testLLM.OnPromptMatching(".*").RespondWith(map[string]any{"ok": true})

	rt.SetLLMClient(testLLM)

	// First call should succeed
	_, err := rt.Execute(`r1 = llm.call(prompt: "Normal request", schema: {ok: boolean})`)
	if err != nil {
		t.Fatalf("First call should succeed: %v", err)
	}

	// Second call should fail
	_, err = rt.Execute(`r2 = llm.call(prompt: "Please fail this request", schema: {ok: boolean})`)
	if err == nil {
		t.Error("Expected error for 'fail' prompt")
	}
}

func TestMockLLM_CallIndexMatching(t *testing.T) {
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	testLLM.OnCallIndex(0).RespondWith(map[string]any{"turn": "first"})
	testLLM.OnCallIndex(1).RespondWith(map[string]any{"turn": "second"})
	testLLM.OnCallIndex(2).RespondWith(map[string]any{"turn": "third"})

	rt.SetLLMClient(testLLM)

	_, err := rt.Execute(`t1 = llm.call(prompt: "a", schema: {turn: string})
emit(t1.turn)
t2 = llm.call(prompt: "b", schema: {turn: string})
emit(t2.turn)
t3 = llm.call(prompt: "c", schema: {turn: string})
emit(t3.turn)`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	emitted := rt.Emitted()
	if len(emitted) != 3 {
		t.Fatalf("Expected 3 emissions, got %d", len(emitted))
	}

	expected := []string{"first", "second", "third"}
	for i, em := range emitted {
		if !strings.Contains(em.String(), expected[i]) {
			t.Errorf("Turn %d: expected '%s', got '%s'", i+1, expected[i], em.String())
		}
	}
}

func TestMockLLM_SchemaFieldMatching(t *testing.T) {
	rt := slop.NewRuntime()

	testLLM := runtime.NewTestLLMClient()
	testLLM.OnSchemaHasField("sentiment").RespondWith(map[string]any{
		"sentiment":  "positive",
		"confidence": 0.95,
	})
	testLLM.OnSchemaHasField("summary").RespondWith(map[string]any{
		"summary": "This is a summary of the content.",
	})

	rt.SetLLMClient(testLLM)

	_, err := rt.Execute(`analysis = llm.call(
    prompt: "Analyze this",
    schema: {sentiment: string, confidence: number}
)
emit(analysis.sentiment)

summary = llm.call(
    prompt: "Summarize this",
    schema: {summary: string}
)
emit(summary.summary)`)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	emitted := rt.Emitted()
	if len(emitted) != 2 {
		t.Fatalf("Expected 2 emissions, got %d", len(emitted))
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
