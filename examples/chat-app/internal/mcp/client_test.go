package mcp

import (
	"encoding/json"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.pending == nil {
		t.Error("pending map not initialized")
	}
}

func TestToolJSON(t *testing.T) {
	tool := Tool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input parameter",
				},
			},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("failed to marshal tool: %v", err)
	}

	var parsed Tool
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal tool: %v", err)
	}

	if parsed.Name != "test-tool" {
		t.Errorf("expected name 'test-tool', got '%s'", parsed.Name)
	}

	if parsed.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", parsed.Description)
	}
}

func TestRequestJSON(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  nil,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	expected := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if string(data) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(data))
	}
}

func TestRequestWithParamsJSON(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "test-tool",
			"arguments": map[string]interface{}{
				"input": "test input",
			},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%v'", parsed["jsonrpc"])
	}

	if parsed["method"] != "tools/call" {
		t.Errorf("expected method 'tools/call', got '%v'", parsed["method"])
	}

	params := parsed["params"].(map[string]interface{})
	if params["name"] != "test-tool" {
		t.Errorf("expected params.name 'test-tool', got '%v'", params["name"])
	}
}

func TestResponseJSON(t *testing.T) {
	responseJSON := `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"test","description":"Test tool"}]}}`

	var resp Response
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got '%s'", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("expected result, got nil")
	}
}

func TestResponseWithErrorJSON(t *testing.T) {
	responseJSON := `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`

	var resp Response
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", resp.Error.Code)
	}

	if resp.Error.Message != "Invalid Request" {
		t.Errorf("expected error message 'Invalid Request', got '%s'", resp.Error.Message)
	}
}

func TestToolResultJSON(t *testing.T) {
	resultJSON := `{
		"content": [
			{"type": "text", "text": "Hello, World!"}
		],
		"isError": false
	}`

	var result ToolResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to unmarshal tool result: %v", err)
	}

	if result.IsError {
		t.Error("expected isError to be false")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got '%s'", result.Content[0].Type)
	}

	if result.Content[0].Text != "Hello, World!" {
		t.Errorf("expected content text 'Hello, World!', got '%s'", result.Content[0].Text)
	}
}

func TestToolResultErrorJSON(t *testing.T) {
	resultJSON := `{
		"content": [
			{"type": "text", "text": "Error: something went wrong"}
		],
		"isError": true
	}`

	var result ToolResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to unmarshal tool result: %v", err)
	}

	if !result.IsError {
		t.Error("expected isError to be true")
	}

	if result.Content[0].Text != "Error: something went wrong" {
		t.Errorf("expected error text, got '%s'", result.Content[0].Text)
	}
}

func TestContentBlockJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected ContentBlock
	}{
		{
			name:     "text content",
			json:     `{"type": "text", "text": "test content"}`,
			expected: ContentBlock{Type: "text", Text: "test content"},
		},
		{
			name:     "empty text",
			json:     `{"type": "text", "text": ""}`,
			expected: ContentBlock{Type: "text", Text: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var block ContentBlock
			if err := json.Unmarshal([]byte(tt.json), &block); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if block.Type != tt.expected.Type {
				t.Errorf("expected type '%s', got '%s'", tt.expected.Type, block.Type)
			}

			if block.Text != tt.expected.Text {
				t.Errorf("expected text '%s', got '%s'", tt.expected.Text, block.Text)
			}
		})
	}
}

func TestClientToolsEmpty(t *testing.T) {
	client := NewClient()
	tools := client.Tools()

	if tools != nil && len(tools) != 0 {
		t.Errorf("expected empty tools, got %d", len(tools))
	}
}

func TestRPCErrorJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		code    int
		message string
	}{
		{
			name:    "parse error",
			json:    `{"code": -32700, "message": "Parse error"}`,
			code:    -32700,
			message: "Parse error",
		},
		{
			name:    "invalid request",
			json:    `{"code": -32600, "message": "Invalid Request"}`,
			code:    -32600,
			message: "Invalid Request",
		},
		{
			name:    "method not found",
			json:    `{"code": -32601, "message": "Method not found"}`,
			code:    -32601,
			message: "Method not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err RPCError
			if e := json.Unmarshal([]byte(tt.json), &err); e != nil {
				t.Fatalf("failed to unmarshal: %v", e)
			}

			if err.Code != tt.code {
				t.Errorf("expected code %d, got %d", tt.code, err.Code)
			}

			if err.Message != tt.message {
				t.Errorf("expected message '%s', got '%s'", tt.message, err.Message)
			}
		})
	}
}

func TestClientMsgIDIncrement(t *testing.T) {
	client := NewClient()

	id1 := client.msgID.Add(1)
	id2 := client.msgID.Add(1)
	id3 := client.msgID.Add(1)

	if id1 != 1 {
		t.Errorf("expected first id 1, got %d", id1)
	}

	if id2 != 2 {
		t.Errorf("expected second id 2, got %d", id2)
	}

	if id3 != 3 {
		t.Errorf("expected third id 3, got %d", id3)
	}
}
