package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/anthropics/slop/internal/evaluator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPService wraps an MCP client session as a SLOP service.
type MCPService struct {
	name      string
	session   *mcp.ClientSession
	transport mcp.Transport
	mu        sync.Mutex
	closed    bool
}

// MCPServiceConfig configures how to connect to an MCP server.
type MCPServiceConfig struct {
	// Name is the service name used in SLOP scripts.
	Name string

	// Type is the transport type: "command", "sse", "streamable"
	Type string

	// For command transport:
	Command string   // Executable path
	Args    []string // Command arguments
	Env     []string // Environment variables

	// For HTTP transports:
	URL     string            // Server URL
	Headers map[string]string // HTTP headers
}

// NewMCPService creates a new MCP service from the given config.
func NewMCPService(ctx context.Context, config MCPServiceConfig) (*MCPService, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "slop",
		Version: "v0.1.0",
	}, nil)

	var transport mcp.Transport

	switch config.Type {
	case "command", "":
		cmd := exec.Command(config.Command, config.Args...)
		if len(config.Env) > 0 {
			cmd.Env = config.Env
		}
		transport = &mcp.CommandTransport{
			Command: cmd,
		}

	case "sse":
		transport = &mcp.SSEClientTransport{
			Endpoint: config.URL,
		}

	case "streamable":
		transport = &mcp.StreamableClientTransport{
			Endpoint: config.URL,
		}

	default:
		return nil, fmt.Errorf("unknown MCP transport type: %s", config.Type)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %s: %w", config.Name, err)
	}

	return &MCPService{
		name:      config.Name,
		session:   session,
		transport: transport,
	}, nil
}

// Name returns the service name.
func (s *MCPService) Name() string {
	return s.name
}

// Call implements evaluator.Service.
func (s *MCPService) Call(method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	return s.CallWithContext(context.Background(), method, args, kwargs)
}

// CallWithContext implements ServiceWithContext.
func (s *MCPService) CallWithContext(ctx context.Context, method string, args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("service %s is closed", s.name)
	}

	// Convert args and kwargs to MCP arguments map
	arguments := make(map[string]any)

	// Add kwargs directly
	for k, v := range kwargs {
		arguments[k] = valueToAny(v)
	}

	// Add positional args as "arg0", "arg1", etc. if they exist
	// (Some MCP tools accept positional args this way)
	for i, arg := range args {
		key := fmt.Sprintf("arg%d", i)
		arguments[key] = valueToAny(arg)
	}

	// If there's a single positional arg and no kwargs, some tools expect it
	// as the direct argument value. We'll also add it without prefix.
	if len(args) == 1 && len(kwargs) == 0 {
		// For simple tools that take a single unnamed argument
		arguments["input"] = valueToAny(args[0])
	}

	params := &mcp.CallToolParams{
		Name:      method,
		Arguments: arguments,
	}

	result, err := s.session.CallTool(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("MCP tool call %s.%s failed: %w", s.name, method, err)
	}

	if result.IsError {
		// Collect error messages from content
		var errMsg string
		for _, content := range result.Content {
			if text, ok := content.(*mcp.TextContent); ok {
				errMsg += text.Text
			}
		}
		if errMsg == "" {
			errMsg = "tool returned error"
		}
		return nil, fmt.Errorf("MCP tool %s.%s error: %s", s.name, method, errMsg)
	}

	// Convert result content to SLOP value
	return contentToValue(result)
}

// Methods returns available tool names from the MCP server.
func (s *MCPService) Methods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	ctx := context.Background()
	tools, err := s.session.ListTools(ctx, nil)
	if err != nil {
		return nil
	}

	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	return names
}

// Close closes the MCP session.
func (s *MCPService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.session.Close()
}

// valueToAny converts a SLOP Value to a Go any type for MCP.
func valueToAny(v evaluator.Value) any {
	switch val := v.(type) {
	case *evaluator.NoneValue:
		return nil
	case *evaluator.BoolValue:
		return val.Value
	case *evaluator.IntValue:
		return val.Value
	case *evaluator.FloatValue:
		return val.Value
	case *evaluator.StringValue:
		return val.Value
	case *evaluator.ListValue:
		items := make([]any, len(val.Elements))
		for i, elem := range val.Elements {
			items[i] = valueToAny(elem)
		}
		return items
	case *evaluator.MapValue:
		m := make(map[string]any)
		for k, v := range val.Pairs {
			m[k] = valueToAny(v)
		}
		return m
	default:
		return v.String()
	}
}

// anyToValue converts a Go any type from MCP to a SLOP Value.
func anyToValue(v any) evaluator.Value {
	if v == nil {
		return evaluator.NONE
	}

	switch val := v.(type) {
	case bool:
		return evaluator.NewBool(val)
	case int:
		return &evaluator.IntValue{Value: int64(val)}
	case int64:
		return &evaluator.IntValue{Value: val}
	case float64:
		return &evaluator.FloatValue{Value: val}
	case string:
		return &evaluator.StringValue{Value: val}
	case []any:
		items := make([]evaluator.Value, len(val))
		for i, elem := range val {
			items[i] = anyToValue(elem)
		}
		return &evaluator.ListValue{Elements: items}
	case map[string]any:
		m := evaluator.NewMapValue()
		for k, v := range val {
			m.Set(k, anyToValue(v))
		}
		return m
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return &evaluator.IntValue{Value: i}
		}
		if f, err := val.Float64(); err == nil {
			return &evaluator.FloatValue{Value: f}
		}
		return &evaluator.StringValue{Value: val.String()}
	default:
		return &evaluator.StringValue{Value: fmt.Sprintf("%v", val)}
	}
}

// contentToValue converts MCP CallToolResult to a SLOP Value.
func contentToValue(result *mcp.CallToolResult) (evaluator.Value, error) {
	// If there's structured content, prefer that
	if result.StructuredContent != nil {
		return anyToValue(result.StructuredContent), nil
	}

	// Otherwise, process content array
	if len(result.Content) == 0 {
		return evaluator.NONE, nil
	}

	// If single content item, return it directly
	if len(result.Content) == 1 {
		return contentItemToValue(result.Content[0])
	}

	// Multiple content items, return as list
	items := make([]evaluator.Value, 0, len(result.Content))
	for _, content := range result.Content {
		val, err := contentItemToValue(content)
		if err != nil {
			return nil, err
		}
		items = append(items, val)
	}
	return &evaluator.ListValue{Elements: items}, nil
}

// contentItemToValue converts a single MCP Content item to a SLOP Value.
func contentItemToValue(content mcp.Content) (evaluator.Value, error) {
	switch c := content.(type) {
	case *mcp.TextContent:
		// Try to parse as JSON
		var parsed any
		if err := json.Unmarshal([]byte(c.Text), &parsed); err == nil {
			return anyToValue(parsed), nil
		}
		// Otherwise return as string
		return &evaluator.StringValue{Value: c.Text}, nil

	case *mcp.ImageContent:
		// Return image info as a map
		m := evaluator.NewMapValue()
		m.Set("type", &evaluator.StringValue{Value: "image"})
		m.Set("mimeType", &evaluator.StringValue{Value: c.MIMEType})
		// Data is []byte, encode as base64 string
		m.Set("data", &evaluator.StringValue{Value: base64.StdEncoding.EncodeToString(c.Data)})
		return m, nil

	case *mcp.AudioContent:
		// Return audio info as a map
		m := evaluator.NewMapValue()
		m.Set("type", &evaluator.StringValue{Value: "audio"})
		m.Set("mimeType", &evaluator.StringValue{Value: c.MIMEType})
		m.Set("data", &evaluator.StringValue{Value: base64.StdEncoding.EncodeToString(c.Data)})
		return m, nil

	case *mcp.EmbeddedResource:
		// Return embedded resource info as a map
		m := evaluator.NewMapValue()
		m.Set("type", &evaluator.StringValue{Value: "resource"})
		if c.Resource != nil {
			m.Set("uri", &evaluator.StringValue{Value: c.Resource.URI})
			if c.Resource.Text != "" {
				m.Set("text", &evaluator.StringValue{Value: c.Resource.Text})
			}
			if c.Resource.MIMEType != "" {
				m.Set("mimeType", &evaluator.StringValue{Value: c.Resource.MIMEType})
			}
			if len(c.Resource.Blob) > 0 {
				m.Set("blob", &evaluator.StringValue{Value: base64.StdEncoding.EncodeToString(c.Resource.Blob)})
			}
		}
		return m, nil

	case *mcp.ResourceLink:
		// Return resource link as a map
		m := evaluator.NewMapValue()
		m.Set("type", &evaluator.StringValue{Value: "resourceLink"})
		m.Set("uri", &evaluator.StringValue{Value: c.URI})
		if c.Name != "" {
			m.Set("name", &evaluator.StringValue{Value: c.Name})
		}
		return m, nil

	default:
		// Unknown content type, try to convert via interface
		return &evaluator.StringValue{Value: fmt.Sprintf("%v", content)}, nil
	}
}

// MCPManager manages multiple MCP service connections.
type MCPManager struct {
	registry *ServiceRegistry
	mu       sync.Mutex
}

// NewMCPManager creates a new MCP manager with a service registry.
func NewMCPManager() *MCPManager {
	return &MCPManager{
		registry: NewServiceRegistry(),
	}
}

// Registry returns the underlying service registry.
func (m *MCPManager) Registry() *ServiceRegistry {
	return m.registry
}

// Connect connects to an MCP server and registers it as a service.
func (m *MCPManager) Connect(ctx context.Context, config MCPServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, err := NewMCPService(ctx, config)
	if err != nil {
		return err
	}

	return m.registry.Register(svc)
}

// Disconnect closes and unregisters an MCP service.
func (m *MCPManager) Disconnect(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, ok := m.registry.Get(name)
	if !ok {
		return fmt.Errorf("service not found: %s", name)
	}

	if err := svc.Close(); err != nil {
		return err
	}

	m.registry.Unregister(name)
	return nil
}

// CloseAll closes all MCP connections.
func (m *MCPManager) CloseAll() error {
	return m.registry.CloseAll()
}

// GetService retrieves a service by name for use in the evaluator.
func (m *MCPManager) GetService(name string) (*evaluator.ServiceValue, bool) {
	return m.registry.CreateServiceValue(name)
}
