// Package slop provides the public API for the SLOP language runtime.
package slop

import (
	"context"
	"fmt"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/builtin"
	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/standardbeagle/slop/internal/runtime"
)

// Runtime is the main entry point for executing SLOP scripts.
type Runtime struct {
	evaluator        *evaluator.Evaluator
	registry         *builtin.Registry
	mcpManager       *runtime.MCPManager
	llmService       *runtime.LLMService
	resumable        *evaluator.ResumableEvaluator
	checkpointDir    string
	currentScript    string
	currentProgram   *ast.Program
}

// NewRuntime creates a new SLOP runtime with all built-in functions registered.
func NewRuntime() *Runtime {
	eval := evaluator.New()
	registry := builtin.NewRegistry()
	mcpMgr := runtime.NewMCPManager()

	// Register all built-in functions
	eval.Context().RegisterBuiltins(registry)

	// Set up the pipeline function caller
	registry.SetPipelineFuncCaller(&pipelineCaller{eval: eval})

	// Create default LLM service with mock client (for testing)
	llmSvc := runtime.NewLLMService(&runtime.MockLLMClient{})

	// Register LLM service
	eval.Context().RegisterService("llm", llmSvc)

	return &Runtime{
		evaluator:  eval,
		registry:   registry,
		mcpManager: mcpMgr,
		llmService: llmSvc,
	}
}

// NewRuntimeWithContext creates a runtime with a custom context.
func NewRuntimeWithContext(ctx *evaluator.Context) *Runtime {
	eval := evaluator.NewWithContext(ctx)
	registry := builtin.NewRegistry()
	mcpMgr := runtime.NewMCPManager()

	// Register all built-in functions
	ctx.RegisterBuiltins(registry)

	// Set up the pipeline function caller
	registry.SetPipelineFuncCaller(&pipelineCaller{eval: eval})

	// Create default LLM service with mock client (for testing)
	llmSvc := runtime.NewLLMService(&runtime.MockLLMClient{})

	// Register LLM service
	ctx.RegisterService("llm", llmSvc)

	return &Runtime{
		evaluator:  eval,
		registry:   registry,
		mcpManager: mcpMgr,
		llmService: llmSvc,
	}
}

// Config configures the SLOP runtime.
type Config struct {
	MaxIterations int64   // Maximum total loop iterations (0 = unlimited)
	MaxLLMCalls   int64   // Maximum LLM calls (0 = unlimited)
	MaxAPICalls   int64   // Maximum API calls (0 = unlimited)
	MaxDuration   int64   // Maximum execution time in seconds (0 = unlimited)
	MaxCost       float64 // Maximum cost in dollars (0 = unlimited)
	MaxCallDepth  int64   // Maximum nested call depth (0 = DefaultMaxCallDepth)
	CheckpointDir string  // Directory for checkpoint files (empty = checkpoints disabled)
}

// NewRuntimeWithConfig creates a runtime with execution limits.
func NewRuntimeWithConfig(cfg Config) *Runtime {
	limits := &evaluator.ExecutionLimits{
		MaxIterations: cfg.MaxIterations,
		MaxLLMCalls:   cfg.MaxLLMCalls,
		MaxAPICalls:   cfg.MaxAPICalls,
		MaxDuration:   cfg.MaxDuration,
		MaxCost:       cfg.MaxCost,
		MaxCallDepth:  cfg.MaxCallDepth,
	}
	ctx := evaluator.NewContextWithLimits(limits)
	return NewRuntimeWithContext(ctx)
}

// Execute parses and runs a SLOP script, returning the result.
func (r *Runtime) Execute(source string) (evaluator.Value, error) {
	// Parse the source
	program, err := r.Parse(source)
	if err != nil {
		return nil, err
	}

	// Execute the program
	return r.evaluator.Eval(program)
}

// Parse parses SLOP source code into an AST.
func (r *Runtime) Parse(source string) (*ast.Program, error) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, errs
	}
	return program, nil
}

// Eval evaluates an AST node.
func (r *Runtime) Eval(node ast.Node) (evaluator.Value, error) {
	return r.evaluator.Eval(node)
}

// Context returns the runtime's execution context.
func (r *Runtime) Context() *evaluator.Context {
	return r.evaluator.Context()
}

// Emitted returns all values emitted during execution.
func (r *Runtime) Emitted() []evaluator.Value {
	return r.evaluator.Context().Emitted
}

// RegisterService registers an MCP service with the runtime.
func (r *Runtime) RegisterService(name string, service evaluator.Service) {
	r.evaluator.Context().RegisterService(name, service)
}

// RegisterBuiltin registers a custom built-in function.
func (r *Runtime) RegisterBuiltin(name string, fn evaluator.BuiltinFunction) {
	r.evaluator.Context().RegisterBuiltin(name, fn)
}

// Services returns all registered services.
func (r *Runtime) Services() map[string]evaluator.Service {
	return r.evaluator.Context().Services
}

// ConnectMCP connects to an MCP server and registers it as a service.
// The service will be available in scripts as `name.method(args)`.
func (r *Runtime) ConnectMCP(ctx context.Context, config MCPConfig) error {
	mcpConfig := runtime.MCPServiceConfig{
		Name:    config.Name,
		Type:    config.Type,
		Command: config.Command,
		Args:    config.Args,
		Env:     config.Env,
		URL:     config.URL,
		Headers: config.Headers,
	}

	if err := r.mcpManager.Connect(ctx, mcpConfig); err != nil {
		return err
	}

	// Register the service with the evaluator
	if svc, ok := r.mcpManager.GetService(config.Name); ok {
		r.evaluator.Context().RegisterService(config.Name, svc.Service)
	}

	return nil
}

// DisconnectMCP disconnects an MCP service.
func (r *Runtime) DisconnectMCP(name string) error {
	return r.mcpManager.Disconnect(name)
}

// Close closes all MCP connections and releases resources.
func (r *Runtime) Close() error {
	return r.mcpManager.CloseAll()
}

// SetCheckpointDir sets the directory for checkpoint files.
// This enables checkpoint support for the runtime.
func (r *Runtime) SetCheckpointDir(dir string) {
	r.checkpointDir = dir

	// Create a resumable evaluator using the existing evaluator's context
	// This ensures proper scope chain connectivity to globals and preserves services/builtins
	r.resumable = evaluator.NewResumableEvaluatorWithEvaluator(r.evaluator, dir)
}

// ExecuteWithCheckpoints parses and runs a SLOP script with checkpoint support.
// If a pause statement is encountered, it saves a checkpoint and returns.
// Returns: result value, checkpoint path (empty if completed), error
func (r *Runtime) ExecuteWithCheckpoints(source string) (evaluator.Value, string, error) {
	if r.checkpointDir == "" {
		// No checkpoint dir - fall back to regular execution
		result, err := r.Execute(source)
		return result, "", err
	}

	// Ensure resumable evaluator is set up
	if r.resumable == nil {
		r.SetCheckpointDir(r.checkpointDir)
	}

	// Parse the source
	program, err := r.Parse(source)
	if err != nil {
		return nil, "", err
	}

	// Store for potential resume
	r.currentScript = source
	r.currentProgram = program
	r.resumable.SetProgram(program, source)

	// Execute with checkpoint support
	return r.resumable.EvalWithCheckpoints(program)
}

// ResumeFromCheckpoint resumes execution from a checkpoint file.
// Returns: result value, new checkpoint path (empty if completed), error
func (r *Runtime) ResumeFromCheckpoint(checkpointPath string) (evaluator.Value, string, error) {
	if r.checkpointDir == "" {
		return nil, "", fmt.Errorf("checkpoint directory not configured")
	}

	// Ensure resumable evaluator is set up
	if r.resumable == nil {
		r.SetCheckpointDir(r.checkpointDir)
	}

	// Get builtins and services for restoration
	// Note: Builtins stored in checkpoint will be restored if they exist in globals
	builtins := make(map[string]*evaluator.BuiltinValue)
	services := r.evaluator.Context().Services

	// Resume execution
	return r.resumable.ResumeFromCheckpoint(checkpointPath, builtins, services)
}

// ListCheckpoints returns all checkpoints in the checkpoint directory.
func (r *Runtime) ListCheckpoints() ([]evaluator.CheckpointInfo, error) {
	if r.checkpointDir == "" {
		return nil, fmt.Errorf("checkpoint directory not configured")
	}

	if r.resumable == nil {
		r.SetCheckpointDir(r.checkpointDir)
	}

	return r.resumable.GetCheckpointManager().ListCheckpoints()
}

// IsPaused returns true if the runtime is paused at a checkpoint.
func (r *Runtime) IsPaused() bool {
	if r.resumable == nil {
		return false
	}
	return r.resumable.Context().ShouldPause()
}

// GetPauseMessage returns the pause message if paused.
func (r *Runtime) GetPauseMessage() string {
	if r.resumable == nil {
		return ""
	}
	return r.resumable.Context().GetPauseMessage()
}

// SetCurrentProgram sets the current program and script for checkpoint validation.
// This is used when resuming from a checkpoint.
func (r *Runtime) SetCurrentProgram(program *ast.Program, script string) {
	r.currentProgram = program
	r.currentScript = script
	if r.resumable != nil {
		r.resumable.SetProgram(program, script)
	}
}

// SetLLMClient sets a custom LLM client for the runtime.
// This replaces the default mock client.
func (r *Runtime) SetLLMClient(client LLMClient) {
	r.llmService = runtime.NewLLMService(client)
	r.evaluator.Context().RegisterService("llm", r.llmService)
}

// LLMClient is the interface for LLM providers.
// Implement this interface to connect SLOP to your LLM backend.
type LLMClient = runtime.LLMClient

// LLMRequest represents a request to the LLM.
type LLMRequest = runtime.LLMRequest

// LLMResponse represents a response from the LLM.
type LLMResponse = runtime.LLMResponse

// Schema represents a JSON schema for structured output.
type Schema = runtime.Schema

// MCPConfig configures how to connect to an MCP server.
type MCPConfig struct {
	// Name is the service name used in SLOP scripts.
	Name string

	// Type is the transport type: "command", "sse", "streamable"
	// Default is "command" for subprocess execution.
	Type string

	// For command transport:
	Command string   // Executable path
	Args    []string // Command arguments
	Env     []string // Environment variables

	// For HTTP transports:
	URL     string            // Server URL/endpoint
	Headers map[string]string // HTTP headers
}

// pipelineCaller implements the builtin.PipelineFuncCaller interface.
type pipelineCaller struct {
	eval *evaluator.Evaluator
}

// CallFunction calls a callable value with the given arguments. It delegates
// to the evaluator's InvokeFunction so all calls go through the proper
// per-frame scope isolation machinery.
func (p *pipelineCaller) CallFunction(fn evaluator.Value, args []evaluator.Value) (evaluator.Value, error) {
	if err := p.eval.Context().IncrementIterations(); err != nil {
		return nil, err
	}
	return p.eval.InvokeFunction(fn, args)
}
