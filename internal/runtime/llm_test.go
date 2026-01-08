package runtime

import (
	"context"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
)

func TestValueToSchema(t *testing.T) {
	tests := []struct {
		name     string
		input    evaluator.Value
		wantType string
		wantErr  bool
	}{
		{
			name:     "string type",
			input:    &evaluator.StringValue{Value: "string"},
			wantType: "string",
		},
		{
			name:     "int type",
			input:    &evaluator.StringValue{Value: "int"},
			wantType: "integer",
		},
		{
			name:     "float type",
			input:    &evaluator.StringValue{Value: "float"},
			wantType: "number",
		},
		{
			name:     "bool type",
			input:    &evaluator.StringValue{Value: "bool"},
			wantType: "boolean",
		},
		{
			name:     "simple object",
			input:    &evaluator.MapValue{Pairs: map[string]evaluator.Value{}},
			wantType: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := valueToSchema(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("valueToSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && schema.Type != tt.wantType {
				t.Errorf("valueToSchema() type = %v, want %v", schema.Type, tt.wantType)
			}
		})
	}
}

func TestParseTypeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantEnum []string
		wantErr  bool
	}{
		{
			name:     "simple string",
			input:    "string",
			wantType: "string",
		},
		{
			name:     "simple int",
			input:    "int",
			wantType: "integer",
		},
		{
			name:     "list of strings",
			input:    "list(string)",
			wantType: "array",
		},
		{
			name:     "enum",
			input:    "enum(a, b, c)",
			wantType: "string",
			wantEnum: []string{"a", "b", "c"},
		},
		{
			name:     "optional string",
			input:    "string?",
			wantType: "string",
		},
		{
			name:     "constrained int",
			input:    "int(min: 1, max: 5)",
			wantType: "integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := parseTypeString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTypeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if schema.Type != tt.wantType {
				t.Errorf("parseTypeString() type = %v, want %v", schema.Type, tt.wantType)
			}
			if tt.wantEnum != nil {
				if len(schema.Enum) != len(tt.wantEnum) {
					t.Errorf("parseTypeString() enum length = %d, want %d", len(schema.Enum), len(tt.wantEnum))
				}
				for i, e := range tt.wantEnum {
					if schema.Enum[i] != e {
						t.Errorf("parseTypeString() enum[%d] = %v, want %v", i, schema.Enum[i], e)
					}
				}
			}
		})
	}
}

func TestMapToSchema(t *testing.T) {
	// Test object schema with properties
	input := &evaluator.MapValue{
		Pairs: map[string]evaluator.Value{
			"name":   &evaluator.StringValue{Value: "string"},
			"age":    &evaluator.StringValue{Value: "int"},
			"active": &evaluator.StringValue{Value: "bool"},
		},
	}

	schema, err := mapToSchema(input)
	if err != nil {
		t.Fatalf("mapToSchema() error = %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("schema.Type = %v, want object", schema.Type)
	}

	if len(schema.Properties) != 3 {
		t.Errorf("schema.Properties length = %d, want 3", len(schema.Properties))
	}

	// Check property types
	if schema.Properties["name"].Type != "string" {
		t.Errorf("name type = %v, want string", schema.Properties["name"].Type)
	}
	if schema.Properties["age"].Type != "integer" {
		t.Errorf("age type = %v, want integer", schema.Properties["age"].Type)
	}
	if schema.Properties["active"].Type != "boolean" {
		t.Errorf("active type = %v, want boolean", schema.Properties["active"].Type)
	}

	// All non-optional properties should be required
	if len(schema.Required) != 3 {
		t.Errorf("schema.Required length = %d, want 3", len(schema.Required))
	}
}

func TestNestedSchema(t *testing.T) {
	// Test nested object schema
	inner := &evaluator.MapValue{
		Pairs: map[string]evaluator.Value{
			"street": &evaluator.StringValue{Value: "string"},
			"city":   &evaluator.StringValue{Value: "string"},
		},
	}

	outer := &evaluator.MapValue{
		Pairs: map[string]evaluator.Value{
			"name":    &evaluator.StringValue{Value: "string"},
			"address": inner,
		},
	}

	schema, err := mapToSchema(outer)
	if err != nil {
		t.Fatalf("mapToSchema() error = %v", err)
	}

	if schema.Properties["address"].Type != "object" {
		t.Errorf("address type = %v, want object", schema.Properties["address"].Type)
	}

	addrProps := schema.Properties["address"].Properties
	if addrProps["street"].Type != "string" {
		t.Errorf("address.street type = %v, want string", addrProps["street"].Type)
	}
}

func TestListSchema(t *testing.T) {
	// Test list schema with inner type
	input := &evaluator.ListValue{
		Elements: []evaluator.Value{
			&evaluator.StringValue{Value: "string"},
		},
	}

	schema, err := propertyToSchema(input)
	if err != nil {
		t.Fatalf("propertyToSchema() error = %v", err)
	}

	if schema.Type != "array" {
		t.Errorf("schema.Type = %v, want array", schema.Type)
	}

	if schema.Items == nil {
		t.Fatal("schema.Items is nil")
	}

	if schema.Items.Type != "string" {
		t.Errorf("schema.Items.Type = %v, want string", schema.Items.Type)
	}
}

func TestValidateAgainstSchema(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		schema  *Schema
		wantErr bool
	}{
		{
			name:    "valid string",
			value:   "hello",
			schema:  &Schema{Type: "string"},
			wantErr: false,
		},
		{
			name:    "invalid string (number)",
			value:   42,
			schema:  &Schema{Type: "string"},
			wantErr: true,
		},
		{
			name:    "valid enum",
			value:   "active",
			schema:  &Schema{Type: "string", Enum: []string{"active", "pending", "done"}},
			wantErr: false,
		},
		{
			name:    "invalid enum",
			value:   "invalid",
			schema:  &Schema{Type: "string", Enum: []string{"active", "pending", "done"}},
			wantErr: true,
		},
		{
			name:    "valid integer",
			value:   42,
			schema:  &Schema{Type: "integer"},
			wantErr: false,
		},
		{
			name:    "valid float as integer",
			value:   42.0,
			schema:  &Schema{Type: "integer"},
			wantErr: false,
		},
		{
			name:    "valid number",
			value:   3.14,
			schema:  &Schema{Type: "number"},
			wantErr: false,
		},
		{
			name:    "valid boolean",
			value:   true,
			schema:  &Schema{Type: "boolean"},
			wantErr: false,
		},
		{
			name:    "valid array",
			value:   []any{"a", "b", "c"},
			schema:  &Schema{Type: "array", Items: &Schema{Type: "string"}},
			wantErr: false,
		},
		{
			name:    "invalid array item",
			value:   []any{"a", 42, "c"},
			schema:  &Schema{Type: "array", Items: &Schema{Type: "string"}},
			wantErr: true,
		},
		{
			name:  "valid object",
			value: map[string]any{"name": "John", "age": 30},
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": {Type: "string"},
					"age":  {Type: "integer"},
				},
				Required: []string{"name"},
			},
			wantErr: false,
		},
		{
			name:  "missing required field",
			value: map[string]any{"age": 30},
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"name": {Type: "string"},
					"age":  {Type: "integer"},
				},
				Required: []string{"name"},
			},
			wantErr: true,
		},
		{
			name:    "integer with minimum",
			value:   5,
			schema:  &Schema{Type: "integer", Minimum: ptr(1.0)},
			wantErr: false,
		},
		{
			name:    "integer below minimum",
			value:   0,
			schema:  &Schema{Type: "integer", Minimum: ptr(1.0)},
			wantErr: true,
		},
		{
			name:    "integer with maximum",
			value:   5,
			schema:  &Schema{Type: "integer", Maximum: ptr(10.0)},
			wantErr: false,
		},
		{
			name:    "integer above maximum",
			value:   15,
			schema:  &Schema{Type: "integer", Maximum: ptr(10.0)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgainstSchema(tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgainstSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		format  string
		wantErr bool
	}{
		{
			name:    "valid email",
			value:   "test@example.com",
			format:  "email",
			wantErr: false,
		},
		{
			name:    "invalid email",
			value:   "not-an-email",
			format:  "email",
			wantErr: true,
		},
		{
			name:    "valid url",
			value:   "https://example.com",
			format:  "url",
			wantErr: false,
		},
		{
			name:    "invalid url",
			value:   "not-a-url",
			format:  "url",
			wantErr: true,
		},
		{
			name:    "valid uuid",
			value:   "550e8400-e29b-41d4-a716-446655440000",
			format:  "uuid",
			wantErr: false,
		},
		{
			name:    "invalid uuid",
			value:   "not-a-uuid",
			format:  "uuid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFormat(tt.value, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractJSONFromResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "bare json object",
			content: `{"name": "John", "age": 30}`,
			want:    `{"name": "John", "age": 30}`,
			wantErr: false,
		},
		{
			name:    "bare json array",
			content: `["a", "b", "c"]`,
			want:    `["a", "b", "c"]`,
			wantErr: false,
		},
		{
			name: "json in code block",
			content: "```json\n{\"name\": \"John\"}\n```",
			want:    `{"name": "John"}`,
			wantErr: false,
		},
		{
			name: "json in generic code block",
			content: "```\n{\"name\": \"John\"}\n```",
			want:    `{"name": "John"}`,
			wantErr: false,
		},
		{
			name:    "no json",
			content: "This is just text without JSON",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSONFromResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractJSONFromResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("ExtractJSONFromResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLLMResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		schema  *Schema
		wantErr bool
	}{
		{
			name:    "valid response",
			content: `{"action": "search", "query": "test"}`,
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"action": {Type: "string", Enum: []string{"search", "done"}},
					"query":  {Type: "string"},
				},
				Required: []string{"action", "query"},
			},
			wantErr: false,
		},
		{
			name:    "invalid enum value",
			content: `{"action": "invalid", "query": "test"}`,
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"action": {Type: "string", Enum: []string{"search", "done"}},
					"query":  {Type: "string"},
				},
				Required: []string{"action", "query"},
			},
			wantErr: true,
		},
		{
			name:    "missing required field",
			content: `{"action": "search"}`,
			schema: &Schema{
				Type: "object",
				Properties: map[string]*Schema{
					"action": {Type: "string"},
					"query":  {Type: "string"},
				},
				Required: []string{"action", "query"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLLMResponse(tt.content, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLLMResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockLLMClient(t *testing.T) {
	// Test that mock client generates appropriate responses based on schema
	client := &MockLLMClient{}

	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"action": {Type: "string", Enum: []string{"search", "done"}},
			"query":  {Type: "string"},
			"count":  {Type: "integer"},
		},
	}

	request := &LLMRequest{
		Prompt: "Test prompt",
		Schema: schema,
	}

	response, err := client.Complete(context.Background(), request)
	if err != nil {
		t.Fatalf("MockLLMClient.Complete() error = %v", err)
	}

	if response.Parsed == nil {
		t.Fatal("MockLLMClient.Complete() returned nil Parsed")
	}

	// Verify the mock response matches the schema
	parsed, ok := response.Parsed.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", response.Parsed)
	}

	// Check that enum returns first value
	if parsed["action"] != "search" {
		t.Errorf("action = %v, want search", parsed["action"])
	}

	// Check that integer returns 42
	if parsed["count"] != 42 {
		t.Errorf("count = %v, want 42", parsed["count"])
	}
}

func TestLLMService(t *testing.T) {
	// Test LLMService.Call with mock client
	mockClient := &MockLLMClient{
		Response: func(req *LLMRequest) *LLMResponse {
			return &LLMResponse{
				Parsed: map[string]any{
					"answer":     "The answer is 42",
					"confidence": 0.95,
				},
			}
		},
	}

	service := NewLLMService(mockClient)

	// Test basic call
	kwargs := map[string]evaluator.Value{
		"prompt": &evaluator.StringValue{Value: "What is the meaning of life?"},
		"schema": &evaluator.MapValue{
			Pairs: map[string]evaluator.Value{
				"answer":     &evaluator.StringValue{Value: "string"},
				"confidence": &evaluator.StringValue{Value: "float"},
			},
		},
	}

	result, err := service.CallWithContext(context.Background(), "call", nil, kwargs)
	if err != nil {
		t.Fatalf("LLMService.Call() error = %v", err)
	}

	mapResult, ok := result.(*evaluator.MapValue)
	if !ok {
		t.Fatalf("Expected MapValue, got %T", result)
	}

	answer, ok := mapResult.Pairs["answer"].(*evaluator.StringValue)
	if !ok || answer.Value != "The answer is 42" {
		t.Errorf("answer = %v, want 'The answer is 42'", mapResult.Pairs["answer"])
	}
}

func TestLLMServiceMissingParams(t *testing.T) {
	service := NewLLMService(&MockLLMClient{})

	// Test missing prompt
	kwargs := map[string]evaluator.Value{
		"schema": &evaluator.MapValue{Pairs: map[string]evaluator.Value{}},
	}

	_, err := service.Call("call", nil, kwargs)
	if err == nil {
		t.Error("Expected error for missing prompt")
	}

	// Test missing schema
	kwargs = map[string]evaluator.Value{
		"prompt": &evaluator.StringValue{Value: "test"},
	}

	_, err = service.Call("call", nil, kwargs)
	if err == nil {
		t.Error("Expected error for missing schema")
	}
}

func TestLLMServiceInvalidMethod(t *testing.T) {
	service := NewLLMService(&MockLLMClient{})

	kwargs := map[string]evaluator.Value{
		"prompt": &evaluator.StringValue{Value: "test"},
		"schema": &evaluator.MapValue{Pairs: map[string]evaluator.Value{}},
	}

	_, err := service.Call("invalid_method", nil, kwargs)
	if err == nil {
		t.Error("Expected error for invalid method")
	}
}
