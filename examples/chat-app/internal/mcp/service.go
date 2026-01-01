package mcp

import (
	"fmt"

	"github.com/anthropics/slop/internal/evaluator"
)

// Service adapts an MCP client to the SLOP Service interface.
type Service struct {
	client *Client
	name   string
}

// NewService creates a new MCP service adapter.
func NewService(client *Client, name string) *Service {
	return &Service{
		client: client,
		name:   name,
	}
}

// Name returns the service name.
func (s *Service) Name() string {
	return s.name
}

// Call calls an MCP tool.
func (s *Service) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	// Convert SLOP values to Go values for MCP
	mcpArgs := make(map[string]interface{})

	// Convert positional args to named args based on tool schema
	tool := s.findTool(method)
	if tool != nil && tool.InputSchema != nil {
		if props, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
			i := 0
			for name := range props {
				if i < len(args) {
					mcpArgs[name] = valueToGo(args[i])
					i++
				}
			}
		}
	}

	// Add keyword args
	for k, v := range kwargs {
		mcpArgs[k] = valueToGo(v)
	}

	// Call the tool
	result, err := s.client.CallTool(method, mcpArgs)
	if err != nil {
		return nil, err
	}

	// Convert result back to SLOP value
	return resultToValue(result), nil
}

func (s *Service) findTool(name string) *Tool {
	for _, t := range s.client.Tools() {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

func valueToGo(v evaluator.Value) interface{} {
	switch val := v.(type) {
	case *evaluator.IntValue:
		return val.Value
	case *evaluator.FloatValue:
		return val.Value
	case *evaluator.StringValue:
		return val.Value
	case *evaluator.BoolValue:
		return val.Value
	case *evaluator.ListValue:
		result := make([]interface{}, len(val.Elements))
		for i, item := range val.Elements {
			result[i] = valueToGo(item)
		}
		return result
	case *evaluator.MapValue:
		result := make(map[string]interface{})
		for k, v := range val.Pairs {
			result[k] = valueToGo(v)
		}
		return result
	case *evaluator.NoneValue:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

func resultToValue(result *ToolResult) evaluator.Value {
	if result.IsError {
		// Return error as a map
		return &evaluator.MapValue{
			Pairs: map[string]evaluator.Value{
				"error": &evaluator.BoolValue{Value: true},
				"content": &evaluator.StringValue{
					Value: getTextContent(result.Content),
				},
			},
		}
	}

	// Return content as string for simple cases
	text := getTextContent(result.Content)
	if text != "" {
		return &evaluator.StringValue{Value: text}
	}

	return evaluator.NONE
}

func getTextContent(content []ContentBlock) string {
	for _, block := range content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return ""
}
