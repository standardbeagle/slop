package runtime

import (
	"errors"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// =============================================================================
// TestLLMClient Tests
// =============================================================================

func TestTestLLMClient_BasicUsage(t *testing.T) {
	rt := NewTestRuntime()

	// Set up a conditional response
	rt.LLM.OnPromptContaining("hello").RespondWith(map[string]any{
		"greeting": "Hi there!",
	})

	// Execute SLOP code that calls the LLM
	_, err := rt.Execute(`prompt = "hello world"
response = llm.call(prompt: prompt, schema: {greeting: string})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify the call was made
	if err := rt.LLM.AssertCallCount(1); err != nil {
		t.Error(err)
	}

	// Verify prompt content
	if err := rt.LLM.AssertPromptContains(0, "hello"); err != nil {
		t.Error(err)
	}
}

func TestTestLLMClient_MultipleConditions(t *testing.T) {
	rt := NewTestRuntime()

	// First call: greeting
	rt.LLM.OnCallIndex(0).RespondWith(map[string]any{
		"type": "first",
	})

	// Second call: analysis
	rt.LLM.OnCallIndex(1).RespondWith(map[string]any{
		"type": "second",
	})

	_, err := rt.Execute(`first = llm.call(prompt: "first call", schema: {type: string})
second = llm.call(prompt: "second call", schema: {type: string})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if err := rt.LLM.AssertCallCount(2); err != nil {
		t.Error(err)
	}

	// Verify each call had the expected prompt
	if err := rt.LLM.AssertPromptContains(0, "first call"); err != nil {
		t.Error(err)
	}
	if err := rt.LLM.AssertPromptContains(1, "second call"); err != nil {
		t.Error(err)
	}
}

func TestTestLLMClient_FailureSimulation(t *testing.T) {
	rt := NewTestRuntime()

	// Fail on any prompt containing "fail"
	rt.LLM.OnPromptContaining("fail").RespondWithError(errors.New("simulated LLM failure"))

	_, err := rt.Execute(`response = llm.call(prompt: "please fail", schema: {message: string})`)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestTestLLMClient_SchemaBasedMock(t *testing.T) {
	rt := NewTestRuntime()

	// When schema has "sentiment" field, return sentiment analysis
	rt.LLM.OnSchemaHasField("sentiment").RespondWith(map[string]any{
		"sentiment":  "positive",
		"confidence": 0.95,
	})

	_, err := rt.Execute(`analysis = llm.call(prompt: "Analyze: I love this product!", schema: {sentiment: string, confidence: number})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	call, ok := rt.LLM.GetCall(0)
	if !ok {
		t.Fatal("Expected call to be recorded")
	}
	if call.Request.Schema == nil {
		t.Error("Expected schema in request")
	}
}

func TestTestLLMClient_DynamicHandler(t *testing.T) {
	rt := NewTestRuntime()

	callCount := 0
	rt.LLM.OnPromptMatching(".*").RespondWithHandler(func(req *LLMRequest) (*LLMResponse, error) {
		callCount++
		return &LLMResponse{
			Parsed: map[string]any{
				"call_number": callCount,
				"echo":        req.Prompt,
			},
		}, nil
	})

	_, err := rt.Execute(`r1 = llm.call(prompt: "first", schema: {call_number: number, echo: string})
r2 = llm.call(prompt: "second", schema: {call_number: number, echo: string})
r3 = llm.call(prompt: "third", schema: {call_number: number, echo: string})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls to handler, got %d", callCount)
	}
}

// =============================================================================
// TestService Tests
// =============================================================================

func TestTestService_BasicUsage(t *testing.T) {
	rt := NewTestRuntime()

	// Add a mock service
	search := rt.AddService("search")
	search.SetMethods("query", "lookup")
	search.OnMethod("query").ReturnMap(map[string]any{
		"results": []any{"result1", "result2"},
		"total":   2,
	})

	_, err := rt.Execute(`results = search.query(q: "test query")`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if err := search.AssertCalled("query"); err != nil {
		t.Error(err)
	}
	if err := search.AssertNotCalled("lookup"); err != nil {
		t.Error(err)
	}
}

func TestTestService_CallOrder(t *testing.T) {
	rt := NewTestRuntime()

	db := rt.AddService("db")
	db.SetMethods("begin", "insert", "commit", "rollback")
	db.OnMethod("begin").ReturnString("tx-123")
	db.OnMethod("insert").ReturnMap(map[string]any{"id": 1})
	db.OnMethod("commit").ReturnString("ok")

	_, err := rt.Execute(`tx = db.begin()
record = db.insert(data: {name: "test"})
status = db.commit(tx: tx)`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if err := db.AssertCallOrder("begin", "insert", "commit"); err != nil {
		t.Error(err)
	}
}

func TestTestService_PerMethodFailure(t *testing.T) {
	rt := NewTestRuntime()

	api := rt.AddService("api")
	api.SetMethods("get", "post")
	api.FailureMethods["post"] = errors.New("write operations disabled")
	api.OnMethod("get").ReturnString("data")

	_, err := rt.Execute(`data = api.get(url: "/data")`)
	if err != nil {
		t.Fatalf("GET should succeed: %v", err)
	}

	_, err = rt.Execute(`result = api.post(url: "/data", body: {})`)
	if err == nil {
		t.Error("POST should fail")
	}
}

func TestTestService_DynamicHandler(t *testing.T) {
	rt := NewTestRuntime()

	calculator := rt.AddService("calc")
	calculator.SetMethods("add", "multiply")
	calculator.OnMethod("add").ReturnHandler(func(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
		// Extract a and b from kwargs
		a, _ := kwargs["a"].(*evaluator.IntValue)
		b, _ := kwargs["b"].(*evaluator.IntValue)
		if a == nil || b == nil {
			return &evaluator.IntValue{Value: 0}, nil
		}
		return &evaluator.IntValue{Value: a.Value + b.Value}, nil
	})

	result, err := rt.Execute(`sum = calc.add(a: 5, b: 3)`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check emitted or returned value
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// =============================================================================
// TestRuntime Integration Tests
// =============================================================================

func TestTestRuntime_SetInput(t *testing.T) {
	rt := NewTestRuntime()

	rt.SetInput("name", "Alice")
	rt.SetInput("age", 30)

	rt.LLM.OnPromptContaining("Alice").RespondWith(map[string]any{
		"greeting": "Hello Alice!",
	})

	_, err := rt.Execute(`name = input.name
greeting = llm.call(prompt: "Greet " + name, schema: {greeting: string})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if err := rt.LLM.AssertAnyPromptContains("Alice"); err != nil {
		t.Error(err)
	}
}

func TestTestRuntime_EmitCapture(t *testing.T) {
	rt := NewTestRuntime()

	rt.LLM.OnPromptMatching(".*").RespondWith(map[string]any{
		"message": "test",
	})

	_, err := rt.Execute(`result = llm.call(prompt: "test", schema: {message: string})
emit(result)`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	emitted := rt.Emitted()
	if len(emitted) == 0 {
		t.Error("Expected at least one emitted value")
	}
}

func TestTestRuntime_Reset(t *testing.T) {
	rt := NewTestRuntime()

	// Make some calls
	rt.LLM.OnPromptMatching(".*").RespondWith(map[string]any{"ok": true})

	rt.Execute(`llm.call(prompt: "first", schema: {ok: boolean})`)
	rt.Execute(`llm.call(prompt: "second", schema: {ok: boolean})`)

	if rt.LLM.CallCount() != 2 {
		t.Errorf("Expected 2 calls, got %d", rt.LLM.CallCount())
	}

	// Reset
	rt.Reset()

	if rt.LLM.CallCount() != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", rt.LLM.CallCount())
	}
}

// =============================================================================
// Complex Scenario Tests
// =============================================================================

func TestComplexScenario_ResearchAgent(t *testing.T) {
	rt := NewTestRuntime()

	// Mock search service
	search := rt.AddService("search")
	search.SetMethods("web")
	search.OnMethod("web").ReturnHandler(func(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
		query, _ := kwargs["query"].(*evaluator.StringValue)
		results := evaluator.NewMapValue()
		if query != nil {
			results.Set("query", &evaluator.StringValue{Value: query.Value})
		}
		list := &evaluator.ListValue{Elements: []evaluator.Value{
			&evaluator.StringValue{Value: "Result for query"},
		}}
		results.Set("items", list)
		return results, nil
	})

	// Mock LLM with different responses based on prompt content
	rt.LLM.OnPromptContaining("generate queries").RespondWith(map[string]any{
		"queries": []any{"query1", "query2", "query3"},
	})
	rt.LLM.OnPromptContaining("synthesize").RespondWith(map[string]any{
		"summary": "Research summary based on findings",
		"sources": []any{"source1", "source2"},
	})

	// Simulate a research agent workflow
	_, err := rt.Execute(`queries_response = llm.call(prompt: "generate queries for: AI safety", schema: {queries: list})
results = search.web(query: "AI safety research")
summary = llm.call(prompt: "synthesize the research findings", schema: {summary: string, sources: list})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify the agent made expected calls
	if err := rt.LLM.AssertCallCount(2); err != nil {
		t.Error(err)
	}
	if err := search.AssertCalled("web"); err != nil {
		t.Error(err)
	}
}

func TestComplexScenario_ToolAgent(t *testing.T) {
	rt := NewTestRuntime()

	// Mock file service
	files := rt.AddService("files")
	files.SetMethods("read", "write", "list")
	files.OnMethod("list").ReturnMap(map[string]any{
		"files": []any{"main.go", "test.go", "README.md"},
	})
	files.OnMethod("read").ReturnString("package main\n\nfunc main() {}")

	// Mock code analysis
	rt.LLM.OnPromptContaining("analyze").RespondWith(map[string]any{
		"analysis": "Code is well-structured",
		"issues":   []any{},
		"score":    95,
	})

	_, err := rt.Execute(`file_list = files.list(dir: ".")
content = files.read(path: "main.go")
analysis = llm.call(prompt: "analyze this code: " + content, schema: {analysis: string, issues: list, score: number})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify call sequence
	if err := files.AssertCallOrder("list", "read"); err != nil {
		t.Error(err)
	}

	// Verify LLM was called with code content
	if err := rt.LLM.AssertAnyPromptContains("package main"); err != nil {
		t.Error(err)
	}
}

func TestComplexScenario_ConversationWithMemory(t *testing.T) {
	rt := NewTestRuntime()

	// Simulate a multi-turn conversation
	turnNumber := 0
	rt.LLM.OnPromptMatching(".*").RespondWithHandler(func(req *LLMRequest) (*LLMResponse, error) {
		turnNumber++
		return &LLMResponse{
			Parsed: map[string]any{
				"turn":     turnNumber,
				"response": "Response to turn",
			},
		}, nil
	})

	_, err := rt.Execute(`turn1 = llm.call(prompt: "Hello, who are you?", schema: {turn: number, response: string})
turn2 = llm.call(prompt: "What can you help me with?", schema: {turn: number, response: string})
turn3 = llm.call(prompt: "Tell me about AI safety", schema: {turn: number, response: string})`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all turns were processed
	if err := rt.LLM.AssertCallCount(3); err != nil {
		t.Error(err)
	}

	// Verify conversation flow
	if err := rt.LLM.AssertPromptContains(0, "Hello"); err != nil {
		t.Error(err)
	}
	if err := rt.LLM.AssertPromptContains(1, "help"); err != nil {
		t.Error(err)
	}
	if err := rt.LLM.AssertPromptContains(2, "AI safety"); err != nil {
		t.Error(err)
	}
}

// =============================================================================
// Assertion Helper Tests
// =============================================================================

func TestLLMClient_AssertCallCount(t *testing.T) {
	client := NewTestLLMClient()

	err := client.AssertCallCount(0)
	if err != nil {
		t.Error("Expected no error for 0 calls")
	}

	err = client.AssertCallCount(1)
	if err == nil {
		t.Error("Expected error when asserting 1 call but have 0")
	}
}

func TestService_AssertMethods(t *testing.T) {
	svc := NewTestService("test")
	svc.SetMethods("method1", "method2")

	// No calls yet
	if err := svc.AssertNotCalled("method1"); err != nil {
		t.Error(err)
	}

	// Make a call
	svc.Call("method1", nil, nil)

	if err := svc.AssertCalled("method1"); err != nil {
		t.Error(err)
	}
	if err := svc.AssertNotCalled("method2"); err != nil {
		t.Error(err)
	}
}

func TestService_MethodCallCount(t *testing.T) {
	svc := NewTestService("test")

	svc.Call("m1", nil, nil)
	svc.Call("m1", nil, nil)
	svc.Call("m2", nil, nil)

	if svc.MethodCallCount("m1") != 2 {
		t.Errorf("Expected 2 calls to m1, got %d", svc.MethodCallCount("m1"))
	}
	if svc.MethodCallCount("m2") != 1 {
		t.Errorf("Expected 1 call to m2, got %d", svc.MethodCallCount("m2"))
	}
	if svc.MethodCallCount("m3") != 0 {
		t.Errorf("Expected 0 calls to m3, got %d", svc.MethodCallCount("m3"))
	}
}
