package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// LLMClient is the interface for LLM providers.
type LLMClient interface {
	// Complete sends a prompt to the LLM and returns the response.
	Complete(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
}

// LLMRequest represents a request to the LLM.
type LLMRequest struct {
	Prompt      string
	System      string
	Model       string
	MaxTokens   int
	Temperature float64
	Schema      *Schema // For structured output
}

// LLMResponse represents a response from the LLM.
type LLMResponse struct {
	Content string
	Parsed  any // Parsed JSON if schema was provided
}

// Schema represents a JSON schema for structured output.
type Schema struct {
	Type       string             `json:"type"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Enum       []string           `json:"enum,omitempty"`
	Required   []string           `json:"required,omitempty"`
	// Additional constraints
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`
	Format  string   `json:"format,omitempty"`
}

// LLMService implements evaluator.Service for LLM calls.
type LLMService struct {
	client LLMClient
}

// NewLLMService creates a new LLM service with the given client.
func NewLLMService(client LLMClient) *LLMService {
	return &LLMService{client: client}
}

// Call implements evaluator.Service.
func (s *LLMService) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	return s.CallWithContext(context.Background(), method, args, kwargs)
}

// CallWithContext implements ServiceWithContext.
func (s *LLMService) CallWithContext(ctx context.Context, method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if method != "call" {
		return nil, fmt.Errorf("llm service only supports 'call' method, got: %s", method)
	}

	// Extract required parameters
	prompt, err := getStringKwarg(kwargs, "prompt")
	if err != nil {
		return nil, fmt.Errorf("llm.call requires 'prompt' parameter: %w", err)
	}

	schemaVal, ok := kwargs["schema"]
	if !ok {
		return nil, fmt.Errorf("llm.call requires 'schema' parameter")
	}

	schema, err := valueToSchema(schemaVal)
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Extract optional parameters
	request := &LLMRequest{
		Prompt:      prompt,
		Schema:      schema,
		Model:       "claude-sonnet",
		MaxTokens:   4096,
		Temperature: 0.0,
	}

	if model, err := getStringKwarg(kwargs, "model"); err == nil {
		request.Model = model
	}
	if system, err := getStringKwarg(kwargs, "system"); err == nil {
		request.System = system
	}
	if maxTokens, err := getIntKwarg(kwargs, "max_tokens"); err == nil {
		request.MaxTokens = int(maxTokens)
	}
	if temp, err := getFloatKwarg(kwargs, "temperature"); err == nil {
		request.Temperature = temp
	}

	// Make the LLM call
	response, err := s.client.Complete(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("llm.call failed: %w", err)
	}

	// Convert the parsed response to a SLOP value
	if response.Parsed != nil {
		return anyToValue(response.Parsed), nil
	}

	// Fall back to returning the raw content as a string
	return &evaluator.StringValue{Value: response.Content}, nil
}

// Name returns the service name.
func (s *LLMService) Name() string {
	return "llm"
}

// Methods returns available methods.
func (s *LLMService) Methods() []string {
	return []string{"call"}
}

// Close is a no-op for the LLM service.
func (s *LLMService) Close() error {
	return nil
}

// Helper functions for extracting kwargs

func getStringKwarg(kwargs map[string]evaluator.Value, name string) (string, error) {
	val, ok := kwargs[name]
	if !ok {
		return "", fmt.Errorf("missing parameter: %s", name)
	}
	if sv, ok := val.(*evaluator.StringValue); ok {
		return sv.Value, nil
	}
	return "", fmt.Errorf("parameter %s must be string, got %s", name, val.Type())
}

func getIntKwarg(kwargs map[string]evaluator.Value, name string) (int64, error) {
	val, ok := kwargs[name]
	if !ok {
		return 0, fmt.Errorf("missing parameter: %s", name)
	}
	if iv, ok := val.(*evaluator.IntValue); ok {
		return iv.Value, nil
	}
	return 0, fmt.Errorf("parameter %s must be int, got %s", name, val.Type())
}

func getFloatKwarg(kwargs map[string]evaluator.Value, name string) (float64, error) {
	val, ok := kwargs[name]
	if !ok {
		return 0, fmt.Errorf("missing parameter: %s", name)
	}
	switch v := val.(type) {
	case *evaluator.FloatValue:
		return v.Value, nil
	case *evaluator.IntValue:
		return float64(v.Value), nil
	}
	return 0, fmt.Errorf("parameter %s must be float, got %s", name, val.Type())
}

// valueToSchema converts a SLOP Value to a Schema.
// Supports both JSON Schema format and SLOP's simplified schema syntax.
func valueToSchema(val evaluator.Value) (*Schema, error) {
	switch v := val.(type) {
	case *evaluator.MapValue:
		return mapToSchema(v)
	case *evaluator.StringValue:
		// Simple type like "string", "int", "float", "bool"
		return &Schema{Type: normalizeType(v.Value)}, nil
	default:
		return nil, fmt.Errorf("schema must be a map or string, got %s", val.Type())
	}
}

func mapToSchema(m *evaluator.MapValue) (*Schema, error) {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for key, val := range m.Pairs {
		propSchema, err := propertyToSchema(val)
		if err != nil {
			return nil, fmt.Errorf("invalid schema for property %s: %w", key, err)
		}
		schema.Properties[key] = propSchema

		// By default, all properties are required unless marked optional with ?
		if !strings.HasSuffix(key, "?") {
			schema.Required = append(schema.Required, key)
		}
	}

	return schema, nil
}

func propertyToSchema(val evaluator.Value) (*Schema, error) {
	switch v := val.(type) {
	case *evaluator.StringValue:
		return parseTypeString(v.Value)
	case *evaluator.MapValue:
		return mapToSchema(v)
	case *evaluator.ListValue:
		// list(type) - extract inner type
		if len(v.Elements) == 1 {
			itemSchema, err := propertyToSchema(v.Elements[0])
			if err != nil {
				return nil, err
			}
			return &Schema{Type: "array", Items: itemSchema}, nil
		}
		return &Schema{Type: "array"}, nil
	case *evaluator.BuiltinValue:
		// Handle builtin type identifiers like list, dict, set
		// These can be used as type names in schemas
		return parseTypeString(v.Name)
	default:
		return nil, fmt.Errorf("unsupported schema type: %s", val.Type())
	}
}

// parseTypeString parses SLOP type syntax like "string", "int", "list(string)", "enum(a, b, c)"
func parseTypeString(s string) (*Schema, error) {
	s = strings.TrimSpace(s)

	// Check for list(type)
	if strings.HasPrefix(s, "list(") && strings.HasSuffix(s, ")") {
		inner := s[5 : len(s)-1]
		itemSchema, err := parseTypeString(inner)
		if err != nil {
			return nil, err
		}
		return &Schema{Type: "array", Items: itemSchema}, nil
	}

	// Check for enum(a, b, c)
	if strings.HasPrefix(s, "enum(") && strings.HasSuffix(s, ")") {
		inner := s[5 : len(s)-1]
		values := strings.Split(inner, ",")
		for i := range values {
			values[i] = strings.TrimSpace(values[i])
		}
		return &Schema{Type: "string", Enum: values}, nil
	}

	// Check for constrained types like int(min: 1, max: 5)
	if idx := strings.Index(s, "("); idx != -1 {
		baseType := s[:idx]
		// TODO: Parse constraints
		return &Schema{Type: normalizeType(baseType)}, nil
	}

	// Handle optional types (ending with ?)
	optional := strings.HasSuffix(s, "?")
	if optional {
		s = s[:len(s)-1]
	}

	return &Schema{Type: normalizeType(s)}, nil
}

// normalizeType converts SLOP types to JSON Schema types.
func normalizeType(t string) string {
	switch strings.ToLower(t) {
	case "string", "str":
		return "string"
	case "int", "integer":
		return "integer"
	case "float", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "list", "array":
		return "array"
	case "map", "object":
		return "object"
	default:
		return t
	}
}

// MockLLMClient is a simple mock client for testing.
type MockLLMClient struct {
	Response func(request *LLMRequest) *LLMResponse
}

// Complete implements LLMClient for testing.
func (c *MockLLMClient) Complete(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if c.Response != nil {
		return c.Response(request), nil
	}

	// Default: generate a response based on the schema
	response := generateMockResponse(request.Schema)
	return &LLMResponse{
		Parsed: response,
	}, nil
}

// generateMockResponse creates a mock response matching the schema.
func generateMockResponse(schema *Schema) any {
	if schema == nil {
		return nil
	}

	switch schema.Type {
	case "string":
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		return "mock_string"
	case "integer":
		return 42
	case "number":
		return 3.14
	case "boolean":
		return true
	case "array":
		if schema.Items != nil {
			return []any{generateMockResponse(schema.Items)}
		}
		return []any{}
	case "object":
		if schema.Properties == nil {
			return map[string]any{}
		}
		result := make(map[string]any)
		for key, propSchema := range schema.Properties {
			result[key] = generateMockResponse(propSchema)
		}
		return result
	default:
		return nil
	}
}

// ValidateAgainstSchema validates a JSON value against a schema.
// Type-specific validation helpers

func validateString(value any, schema *Schema) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}
	if len(schema.Enum) > 0 {
		if err := validateEnum(s, schema.Enum); err != nil {
			return err
		}
	}
	if schema.Format != "" {
		if err := validateFormat(s, schema.Format); err != nil {
			return err
		}
	}
	return nil
}

func validateInteger(value any, schema *Schema) error {
	var n int64
	switch v := value.(type) {
	case int:
		n = int64(v)
	case int64:
		n = v
	case float64:
		n = int64(v)
	default:
		return fmt.Errorf("expected integer, got %T", value)
	}
	return validateNumericConstraints(float64(n), schema)
}

func validateNumber(value any, schema *Schema) error {
	var n float64
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	default:
		return fmt.Errorf("expected number, got %T", value)
	}
	return validateNumericConstraints(n, schema)
}

func validateBoolean(value any, schema *Schema) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("expected boolean, got %T", value)
	}
	return nil
}

func validateArray(value any, schema *Schema) error {
	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", value)
	}
	if schema.Items != nil {
		for i, item := range arr {
			if err := ValidateAgainstSchema(item, schema.Items); err != nil {
				return fmt.Errorf("array item %d: %w", i, err)
			}
		}
	}
	return nil
}

func validateObject(value any, schema *Schema) error {
	obj, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("expected object, got %T", value)
	}
	// Check required fields
	for _, req := range schema.Required {
		if _, ok := obj[req]; !ok {
			return fmt.Errorf("missing required field: %s", req)
		}
	}
	// Validate properties
	for key, propSchema := range schema.Properties {
		if val, ok := obj[key]; ok {
			if err := ValidateAgainstSchema(val, propSchema); err != nil {
				return fmt.Errorf("property %s: %w", key, err)
			}
		}
	}
	return nil
}

// Constraint validation helpers

func validateEnum(value string, enum []string) error {
	for _, e := range enum {
		if value == e {
			return nil
		}
	}
	return fmt.Errorf("value %q not in enum %v", value, enum)
}

func validateNumericConstraints(n float64, schema *Schema) error {
	if schema.Minimum != nil && n < *schema.Minimum {
		return fmt.Errorf("value %v less than minimum %v", n, *schema.Minimum)
	}
	if schema.Maximum != nil && n > *schema.Maximum {
		return fmt.Errorf("value %v greater than maximum %v", n, *schema.Maximum)
	}
	return nil
}

func ValidateAgainstSchema(value any, schema *Schema) error {
	if schema == nil {
		return nil
	}

	switch schema.Type {
	case "string":
		return validateString(value, schema)
	case "integer":
		return validateInteger(value, schema)
	case "number":
		return validateNumber(value, schema)
	case "boolean":
		return validateBoolean(value, schema)
	case "array":
		return validateArray(value, schema)
	case "object":
		return validateObject(value, schema)
	default:
		return nil
	}
}

func validateFormat(s, format string) error {
	switch format {
	case "email":
		if !regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`).MatchString(s) {
			return fmt.Errorf("invalid email format: %s", s)
		}
	case "url":
		if !regexp.MustCompile(`^https?://`).MatchString(s) {
			return fmt.Errorf("invalid URL format: %s", s)
		}
	case "uuid":
		if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(strings.ToLower(s)) {
			return fmt.Errorf("invalid UUID format: %s", s)
		}
	}
	return nil
}

// ExtractJSONFromResponse extracts JSON from an LLM response that may contain
// markdown code blocks or other formatting.
func ExtractJSONFromResponse(content string) (string, error) {
	// Try to find JSON in markdown code block
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end]), nil
		}
	}

	// Try to find JSON in generic code block
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + 3
		// Skip to next line if there's a language identifier
		if nl := strings.Index(content[start:], "\n"); nl != -1 {
			start += nl + 1
		}
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end]), nil
		}
	}

	// Try to find bare JSON object or array
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
		return content, nil
	}

	return "", fmt.Errorf("no JSON found in response")
}

// ParseLLMResponse parses the LLM response content according to the schema.
func ParseLLMResponse(content string, schema *Schema) (any, error) {
	jsonStr, err := ExtractJSONFromResponse(content)
	if err != nil {
		return nil, err
	}

	var result any
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if err := ValidateAgainstSchema(result, schema); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return result, nil
}
