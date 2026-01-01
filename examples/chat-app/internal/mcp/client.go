package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client is an MCP client that communicates with MCP servers.
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	stderr  io.ReadCloser
	tools   []Tool
	mu      sync.Mutex
	msgID   atomic.Int64
	pending map[int64]chan *Response
}

// Tool represents an MCP tool.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Request is an MCP JSON-RPC request.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response is an MCP JSON-RPC response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolResult is the result of a tool call.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a content block in a tool result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// NewClient creates a new MCP client.
func NewClient() *Client {
	return &Client{
		pending: make(map[int64]chan *Response),
	}
}

// Connect connects to an MCP server via stdio.
func (c *Client) Connect(command string, args []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cmd = exec.Command(command, args...)

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	c.stdout = bufio.NewReader(stdout)

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start reading responses
	go c.readResponses()

	// Initialize connection
	if err := c.initialize(); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Fetch available tools
	if err := c.fetchTools(); err != nil {
		c.Close()
		return fmt.Errorf("failed to fetch tools: %w", err)
	}

	return nil
}

// Close closes the MCP client.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}

// Tools returns the available tools.
func (c *Client) Tools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tools
}

// CallTool calls an MCP tool.
func (c *Client) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.call("tools/call", params)
	if err != nil {
		return nil, err
	}

	var result ToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &result, nil
}

func (c *Client) initialize() error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "slop-chat",
			"version": "1.0.0",
		},
	}

	_, err := c.call("initialize", params)
	if err != nil {
		return err
	}

	// Send initialized notification
	return c.notify("notifications/initialized", nil)
}

func (c *Client) fetchTools() error {
	resp, err := c.call("tools/list", nil)
	if err != nil {
		return err
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse tools: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return nil
}

func (c *Client) call(method string, params interface{}) (*Response, error) {
	id := c.msgID.Add(1)

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Create response channel
	respChan := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[id] = respChan
	c.mu.Unlock()

	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	_, err = fmt.Fprintf(c.stdin, "%s\n", data)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}

	// Wait for response
	resp := <-respChan

	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return resp, nil
}

func (c *Client) notify(method string, params interface{}) error {
	req := Request{
		JSONRPC: "2.0",
		ID:      0, // Notifications don't have an ID
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	c.mu.Lock()
	_, err = fmt.Fprintf(c.stdin, "%s\n", data)
	c.mu.Unlock()

	return err
}

func (c *Client) readResponses() {
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return
		}

		var resp Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}

		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
		}
		c.mu.Unlock()
	}
}
