package chat

import (
	"testing"

	"github.com/anthropics/slop/examples/chat-app/internal/config"
	"github.com/anthropics/slop/internal/evaluator"
)

func TestNewApp(t *testing.T) {
	cfg := config.DefaultConfig()
	app, err := New(cfg, false)

	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app == nil {
		t.Fatal("New returned nil app")
	}

	if app.cfg != cfg {
		t.Error("config not stored correctly")
	}

	if app.verbose {
		t.Error("verbose should be false")
	}

	if app.runtime == nil {
		t.Error("runtime not initialized")
	}

	if app.history == nil {
		t.Error("history not initialized")
	}
}

func TestNewAppVerbose(t *testing.T) {
	cfg := config.DefaultConfig()
	app, err := New(cfg, true)

	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if !app.verbose {
		t.Error("verbose should be true")
	}
}

func TestAppClose(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	// Close with no MCP client should not error
	err := app.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestListToolsNoMCP(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	tools := app.ListTools()
	if tools != nil {
		t.Errorf("expected nil tools without MCP, got %v", tools)
	}
}

func TestConnectMCPUnknownServer(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	err := app.ConnectMCP("nonexistent")
	if err == nil {
		t.Error("expected error for unknown MCP server")
	}
}

func TestHistoryToValue(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	// Empty history
	val := app.historyToValue()
	listVal, ok := val.(*evaluator.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", val)
	}
	if len(listVal.Elements) != 0 {
		t.Errorf("expected empty list, got %d elements", len(listVal.Elements))
	}

	// Add some history
	app.history = []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	val = app.historyToValue()
	listVal = val.(*evaluator.ListValue)

	if len(listVal.Elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(listVal.Elements))
	}

	// Check first message
	firstMsg := listVal.Elements[0].(*evaluator.MapValue)
	role := firstMsg.Pairs["role"].(*evaluator.StringValue)
	content := firstMsg.Pairs["content"].(*evaluator.StringValue)

	if role.Value != "user" {
		t.Errorf("expected role 'user', got '%s'", role.Value)
	}
	if content.Value != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", content.Value)
	}
}

func TestFormatHistory(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	// Empty history
	formatted := app.formatHistory()
	if formatted != "No previous conversation." {
		t.Errorf("expected 'No previous conversation.', got '%s'", formatted)
	}

	// With history
	app.history = []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi!"},
	}

	formatted = app.formatHistory()
	if formatted == "No previous conversation." {
		t.Error("expected formatted history, got empty message")
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "test message",
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", msg.Role)
	}

	if msg.Content != "test message" {
		t.Errorf("expected content 'test message', got '%s'", msg.Content)
	}
}

func TestToolInfo(t *testing.T) {
	info := ToolInfo{
		Name:        "test-tool",
		Description: "A test tool",
	}

	if info.Name != "test-tool" {
		t.Errorf("expected name 'test-tool', got '%s'", info.Name)
	}

	if info.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", info.Description)
	}
}

func TestAppWithCustomConfig(t *testing.T) {
	cfg := &config.Config{
		Model:       "claude-opus",
		MaxTokens:   8192,
		Temperature: 0.5,
		ScriptsDir:  "custom-scripts",
		MCPServers:  make(map[string]config.MCPServerConfig),
	}

	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app.cfg.Model != "claude-opus" {
		t.Errorf("expected model 'claude-opus', got '%s'", app.cfg.Model)
	}

	if app.cfg.ScriptsDir != "custom-scripts" {
		t.Errorf("expected scripts_dir 'custom-scripts', got '%s'", app.cfg.ScriptsDir)
	}
}

func TestHistoryClear(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	// Add history
	app.history = []Message{
		{Role: "user", Content: "Hello"},
	}

	if len(app.history) != 1 {
		t.Error("expected 1 message in history")
	}

	// Clear history
	app.history = []Message{}

	if len(app.history) != 0 {
		t.Error("expected empty history after clear")
	}
}

func TestHistoryAppend(t *testing.T) {
	cfg := config.DefaultConfig()
	app, _ := New(cfg, false)

	app.history = append(app.history, Message{Role: "user", Content: "First"})
	app.history = append(app.history, Message{Role: "assistant", Content: "Response"})
	app.history = append(app.history, Message{Role: "user", Content: "Second"})

	if len(app.history) != 3 {
		t.Errorf("expected 3 messages, got %d", len(app.history))
	}

	if app.history[0].Content != "First" {
		t.Error("first message wrong")
	}

	if app.history[1].Role != "assistant" {
		t.Error("second message role wrong")
	}

	if app.history[2].Content != "Second" {
		t.Error("third message wrong")
	}
}
