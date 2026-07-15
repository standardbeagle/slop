package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/standardbeagle/slop/internal/evaluator"
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
	URL                 string            // Server URL
	Headers             map[string]string // HTTP headers
	AllowPrivateNetwork bool              // Explicitly allow local/private HTTP destinations
}

type mcpLookupFunc func(context.Context, string) ([]netip.Addr, error)
type mcpDialFunc func(context.Context, string, string) (net.Conn, error)

// specialPurposeMCPNetworks mirrors the IANA IPv4 and IPv6 special-purpose
// address registries. MCP HTTP transports may only dial ordinary public
// unicast addresses unless private/loopback access is explicitly enabled.
var specialPurposeMCPNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"),
	netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("::ffff:0:0/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fec0::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

var awsIPv6MetadataAddr = netip.MustParseAddr("fd00:ec2::254")

const mcpDialFallbackDelay = 250 * time.Millisecond

// NewMCPService creates a new MCP service from the given config.
func NewMCPService(ctx context.Context, config MCPServiceConfig) (*MCPService, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "slop",
		Version: "v0.1.0",
	}, nil)

	var transport mcp.Transport

	switch config.Type {
	case "command", "":
		transport = &mcp.CommandTransport{
			Command: newMCPCommand(config),
		}

	case "sse":
		httpClient, err := newMCPHTTPClient(ctx, config.URL, config.AllowPrivateNetwork)
		if err != nil {
			return nil, err
		}
		transport = &mcp.SSEClientTransport{
			Endpoint:   config.URL,
			HTTPClient: httpClient,
		}

	case "streamable":
		httpClient, err := newMCPHTTPClient(ctx, config.URL, config.AllowPrivateNetwork)
		if err != nil {
			return nil, err
		}
		transport = &mcp.StreamableClientTransport{
			Endpoint:   config.URL,
			HTTPClient: httpClient,
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

func newMCPCommand(config MCPServiceConfig) *exec.Cmd {
	cmd := exec.Command(config.Command, config.Args...)
	cmd.Env = append([]string{}, config.Env...)
	return cmd
}

func newMCPHTTPClient(ctx context.Context, rawURL string, allowPrivate bool) (*http.Client, error) {
	lookup := func(ctx context.Context, host string) ([]netip.Addr, error) {
		return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	}
	dialer := &net.Dialer{}
	return newMCPHTTPClientWithNetwork(ctx, rawURL, allowPrivate, lookup, dialer.DialContext)
}

func newMCPHTTPClientWithLookup(ctx context.Context, rawURL string, allowPrivate bool, lookup mcpLookupFunc) (*http.Client, error) {
	dialer := &net.Dialer{}
	return newMCPHTTPClientWithNetwork(ctx, rawURL, allowPrivate, lookup, dialer.DialContext)
}

func newMCPHTTPClientWithNetwork(ctx context.Context, rawURL string, allowPrivate bool, lookup mcpLookupFunc, dial mcpDialFunc) (*http.Client, error) {
	return newMCPHTTPClientWithNetworkDelay(ctx, rawURL, allowPrivate, lookup, dial, mcpDialFallbackDelay)
}

func newMCPHTTPClientWithNetworkDelay(ctx context.Context, rawURL string, allowPrivate bool, lookup mcpLookupFunc, dial mcpDialFunc, fallbackDelay time.Duration) (*http.Client, error) {
	if _, err := validateMCPURL(ctx, rawURL, allowPrivate, lookup); err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("unsafe MCP URL: invalid dial address: %w", err)
		}
		addrs, err := lookupMCPHost(ctx, host, allowPrivate, lookup)
		if err != nil {
			return nil, err
		}
		conn, err := dialMCPAddresses(ctx, network, port, addrs, dial, fallbackDelay)
		if err != nil {
			return nil, fmt.Errorf("failed to dial validated MCP host %q: %w", host, err)
		}
		return conn, nil
	}

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			_, err := validateMCPURL(req.Context(), req.URL.String(), allowPrivate, lookup)
			return err
		},
	}, nil
}

type mcpDialResult struct {
	conn net.Conn
	err  error
}

func dialMCPAddresses(ctx context.Context, network, port string, addrs []netip.Addr, dial mcpDialFunc, fallbackDelay time.Duration) (net.Conn, error) {
	dialCtx, cancel := context.WithCancel(ctx)
	results := make(chan mcpDialResult, len(addrs))
	for i, addr := range addrs {
		delay := time.Duration(i) * fallbackDelay
		go func() {
			if delay > 0 {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-dialCtx.Done():
					results <- mcpDialResult{err: dialCtx.Err()}
					return
				}
			}
			conn, err := dial(dialCtx, network, net.JoinHostPort(addr.String(), port))
			results <- mcpDialResult{conn: conn, err: err}
		}()
	}

	var dialErrors []error
	for received := 0; received < len(addrs); received++ {
		select {
		case result := <-results:
			if result.err == nil {
				cancel()
				go closeMCPDialResults(results, len(addrs)-received-1)
				return result.conn, nil
			}
			if result.conn != nil {
				result.conn.Close()
			}
			dialErrors = append(dialErrors, result.err)
		case <-ctx.Done():
			cancel()
			go closeMCPDialResults(results, len(addrs)-received)
			return nil, ctx.Err()
		}
	}
	cancel()
	return nil, errors.Join(dialErrors...)
}

func closeMCPDialResults(results <-chan mcpDialResult, remaining int) {
	for range remaining {
		if result := <-results; result.conn != nil {
			result.conn.Close()
		}
	}
}

func validateMCPURL(ctx context.Context, rawURL string, allowPrivate bool, lookup mcpLookupFunc) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("unsafe MCP URL: invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsafe MCP URL: scheme must be http or https")
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("unsafe MCP URL: host is required")
	}
	if u.User != nil {
		return nil, fmt.Errorf("unsafe MCP URL: embedded credentials are not allowed")
	}
	if _, err := lookupMCPHost(ctx, u.Hostname(), allowPrivate, lookup); err != nil {
		return nil, err
	}
	return u, nil
}

func lookupMCPHost(ctx context.Context, host string, allowPrivate bool, lookup mcpLookupFunc) ([]netip.Addr, error) {
	var addrs []netip.Addr
	if literal, err := netip.ParseAddr(strings.TrimSuffix(host, ".")); err == nil {
		addrs = []netip.Addr{literal}
	} else {
		var lookupErr error
		addrs, lookupErr = lookup(ctx, host)
		if lookupErr != nil {
			return nil, fmt.Errorf("unsafe MCP URL: cannot resolve host %q: %w", host, lookupErr)
		}
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("unsafe MCP URL: host %q resolved to no addresses", host)
	}
	for _, addr := range addrs {
		if metadataMCPAddr(addr) || (unsafeMCPAddr(addr) && !(allowPrivate && localMCPAddr(addr))) {
			return nil, fmt.Errorf("unsafe MCP URL: host %q resolves to blocked address %s", host, addr)
		}
	}
	return addrs, nil
}

func metadataMCPAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr == netip.MustParseAddr("169.254.169.254") || addr == awsIPv6MetadataAddr
}

func localMCPAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return addr.IsPrivate() || addr.IsLoopback()
}

func unsafeMCPAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	for _, prefix := range specialPurposeMCPNetworks {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
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
