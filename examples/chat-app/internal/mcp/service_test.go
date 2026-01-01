package mcp

import (
	"testing"

	"github.com/anthropics/slop/internal/evaluator"
)

func TestNewService(t *testing.T) {
	client := NewClient()
	service := NewService(client, "test-service")

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.Name() != "test-service" {
		t.Errorf("expected name 'test-service', got '%s'", service.Name())
	}

	if service.client != client {
		t.Error("client not stored correctly")
	}
}

func TestValueToGoInt(t *testing.T) {
	val := &evaluator.IntValue{Value: 42}
	result := valueToGo(val)

	if result != int64(42) {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestValueToGoFloat(t *testing.T) {
	val := &evaluator.FloatValue{Value: 3.14}
	result := valueToGo(val)

	if result != 3.14 {
		t.Errorf("expected 3.14, got %v", result)
	}
}

func TestValueToGoString(t *testing.T) {
	val := &evaluator.StringValue{Value: "hello"}
	result := valueToGo(val)

	if result != "hello" {
		t.Errorf("expected 'hello', got %v", result)
	}
}

func TestValueToGoBool(t *testing.T) {
	val := &evaluator.BoolValue{Value: true}
	result := valueToGo(val)

	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestValueToGoNone(t *testing.T) {
	val := &evaluator.NoneValue{}
	result := valueToGo(val)

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestValueToGoList(t *testing.T) {
	val := &evaluator.ListValue{
		Elements: []evaluator.Value{
			&evaluator.IntValue{Value: 1},
			&evaluator.IntValue{Value: 2},
			&evaluator.IntValue{Value: 3},
		},
	}
	result := valueToGo(val)

	list, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", result)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 elements, got %d", len(list))
	}

	if list[0] != int64(1) {
		t.Errorf("expected first element 1, got %v", list[0])
	}
}

func TestValueToGoMap(t *testing.T) {
	val := &evaluator.MapValue{
		Pairs: map[string]evaluator.Value{
			"name":  &evaluator.StringValue{Value: "test"},
			"value": &evaluator.IntValue{Value: 42},
		},
	}
	result := valueToGo(val)

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}

	if m["name"] != "test" {
		t.Errorf("expected name 'test', got %v", m["name"])
	}

	if m["value"] != int64(42) {
		t.Errorf("expected value 42, got %v", m["value"])
	}
}

func TestValueToGoNestedStructure(t *testing.T) {
	val := &evaluator.MapValue{
		Pairs: map[string]evaluator.Value{
			"items": &evaluator.ListValue{
				Elements: []evaluator.Value{
					&evaluator.MapValue{
						Pairs: map[string]evaluator.Value{
							"id":   &evaluator.IntValue{Value: 1},
							"name": &evaluator.StringValue{Value: "item1"},
						},
					},
				},
			},
		},
	}
	result := valueToGo(val)

	m := result.(map[string]interface{})
	items := m["items"].([]interface{})
	item := items[0].(map[string]interface{})

	if item["id"] != int64(1) {
		t.Errorf("expected id 1, got %v", item["id"])
	}

	if item["name"] != "item1" {
		t.Errorf("expected name 'item1', got %v", item["name"])
	}
}

func TestResultToValueSuccess(t *testing.T) {
	result := &ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Hello, World!"},
		},
		IsError: false,
	}

	value := resultToValue(result)

	strVal, ok := value.(*evaluator.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", value)
	}

	if strVal.Value != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", strVal.Value)
	}
}

func TestResultToValueError(t *testing.T) {
	result := &ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Something went wrong"},
		},
		IsError: true,
	}

	value := resultToValue(result)

	mapVal, ok := value.(*evaluator.MapValue)
	if !ok {
		t.Fatalf("expected MapValue, got %T", value)
	}

	errorVal := mapVal.Pairs["error"].(*evaluator.BoolValue)
	if !errorVal.Value {
		t.Error("expected error to be true")
	}

	contentVal := mapVal.Pairs["content"].(*evaluator.StringValue)
	if contentVal.Value != "Something went wrong" {
		t.Errorf("expected error content, got '%s'", contentVal.Value)
	}
}

func TestResultToValueEmptyContent(t *testing.T) {
	result := &ToolResult{
		Content: []ContentBlock{},
		IsError: false,
	}

	value := resultToValue(result)

	if value != evaluator.NONE {
		t.Errorf("expected NONE, got %v", value)
	}
}

func TestResultToValueNoTextContent(t *testing.T) {
	result := &ToolResult{
		Content: []ContentBlock{
			{Type: "image", Text: ""},
		},
		IsError: false,
	}

	value := resultToValue(result)

	if value != evaluator.NONE {
		t.Errorf("expected NONE, got %v", value)
	}
}

func TestGetTextContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []ContentBlock
		expected string
	}{
		{
			name: "single text block",
			content: []ContentBlock{
				{Type: "text", Text: "Hello"},
			},
			expected: "Hello",
		},
		{
			name: "multiple blocks, first text",
			content: []ContentBlock{
				{Type: "text", Text: "First"},
				{Type: "image", Text: ""},
			},
			expected: "First",
		},
		{
			name: "multiple blocks, non-text first",
			content: []ContentBlock{
				{Type: "image", Text: ""},
				{Type: "text", Text: "Second"},
			},
			expected: "Second",
		},
		{
			name:     "empty content",
			content:  []ContentBlock{},
			expected: "",
		},
		{
			name: "no text blocks",
			content: []ContentBlock{
				{Type: "image", Text: ""},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTextContent(tt.content)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestServiceFindToolNotFound(t *testing.T) {
	client := NewClient()
	service := NewService(client, "test")

	tool := service.findTool("nonexistent")
	if tool != nil {
		t.Errorf("expected nil for nonexistent tool, got %v", tool)
	}
}

func TestValueToGoUnknownType(t *testing.T) {
	// Test that unknown types are converted to string representation
	val := &evaluator.FunctionValue{Name: "testFunc"}
	result := valueToGo(val)

	// Should convert to string representation
	str, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}

	if str == "" {
		t.Error("expected non-empty string representation")
	}
}
