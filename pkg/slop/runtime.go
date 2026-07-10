// Package slop provides the public API for the SLOP language runtime.
package slop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/builtin"
	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/standardbeagle/slop/internal/runtime"
)

// Runtime is the main entry point for executing SLOP scripts.
type Runtime struct {
	evaluator      *evaluator.Evaluator
	registry       *builtin.Registry
	mcpManager     *runtime.MCPManager
	llmService     *runtime.LLMService
	resumable      *evaluator.ResumableEvaluator
	checkpointDir  string
	currentScript  string
	currentProgram *ast.Program
	resumableMu    sync.Mutex
}

// NewRuntime creates a new SLOP runtime with all built-in functions registered.
func NewRuntime() *Runtime {
	eval := evaluator.New()
	registry := builtin.NewRegistry()
	mcpMgr := runtime.NewMCPManager()

	// Register all built-in functions
	eval.Context().RegisterBuiltins(registry)

	// Set up the pipeline function caller
	builtin.SetPipelineFuncCaller(&pipelineCaller{eval: eval})

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
	builtin.SetPipelineFuncCaller(&pipelineCaller{eval: eval})

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

// ExecuteResumable executes source using the explicit-stack evaluator.
func (r *Runtime) ExecuteResumable(ctx context.Context, source string, options ExecuteOptions) (ExecutionResult, error) {
	r.resumableMu.Lock()
	defer r.resumableMu.Unlock()
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	program, err := r.Parse(source)
	if err != nil {
		return ExecutionResult{}, err
	}
	machine := evaluator.NewMachine(program, r.evaluator.Context())
	for _, breakpoint := range options.Breakpoints {
		if breakpoint.Condition == "" {
			machine.SetBreakpoint(breakpoint.Line)
			continue
		}
		condition, err := r.parseBreakpointCondition(breakpoint.Condition)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("breakpoint line %d condition: %w", breakpoint.Line, err)
		}
		machine.SetConditionalBreakpoint(breakpoint.Line, condition)
	}
	result := machine.RunContext(ctx)
	return r.publicMachineResult(source, program, machine, result, options.Breakpoints)
}

// SupportsResumable reports whether source uses only nodes supported by the
// explicit-stack evaluator. It parses and inspects source without executing it.
func (r *Runtime) SupportsResumable(source string) error {
	program, err := r.Parse(source)
	if err != nil {
		return err
	}
	return evaluator.ValidateMachineProgram(program)
}

// Resume continues a checkpoint produced by ExecuteResumable on this Runtime.
func (r *Runtime) Resume(ctx context.Context, checkpoint *Checkpoint, options ResumeOptions) (result ExecutionResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = ExecutionResult{}
			err = fmt.Errorf("invalid resumable checkpoint: %v", recovered)
		}
	}()
	r.resumableMu.Lock()
	defer r.resumableMu.Unlock()
	if err := ctx.Err(); err != nil {
		return ExecutionResult{}, err
	}
	if err := ValidateResumeOptions(checkpoint, options); err != nil {
		return ExecutionResult{}, err
	}
	program, err := r.Parse(checkpoint.Script)
	if err != nil {
		return ExecutionResult{}, err
	}
	snapshot, err := evaluator.UnmarshalMachineSnapshot(checkpoint.MachineSnapshot)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("decoding machine snapshot: %w", err)
	}
	machine, machineCheckpoint, err := evaluator.RestoreMachine(program, snapshot, map[string]*evaluator.BuiltinValue{}, r.evaluator.Context().Services)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("restoring machine snapshot: %w", err)
	}
	for _, breakpoint := range checkpoint.Breakpoints {
		if breakpoint.Condition != "" {
			condition, parseErr := r.parseBreakpointCondition(breakpoint.Condition)
			if parseErr != nil {
				return ExecutionResult{}, parseErr
			}
			machine.SetConditionalBreakpoint(breakpoint.Line, condition)
		}
	}
	internal := evaluator.MachineResumeOptions{}
	switch options.Mode {
	case ResumeContinue:
		internal.Mode = evaluator.MachineContinue
	case ResumeAcceptResult:
		value, err := deserializePublicValue(options.ReplacementResult)
		if err != nil {
			return ExecutionResult{}, err
		}
		internal.Mode, internal.Replacement = evaluator.MachineAcceptResult, value
	case ResumeRetryCall:
		if len(checkpoint.Continuation) == 0 || checkpoint.Continuation[len(checkpoint.Continuation)-1].Call == nil {
			return ExecutionResult{}, &CheckpointValidationError{Path: "$.continuation", Code: "active_call", Message: "retry_call requires the active frame to be a failed service call"}
		}
		call := checkpoint.Continuation[len(checkpoint.Continuation)-1].Call
		args := make([]evaluator.Value, len(call.Args))
		for i, serialized := range call.Args {
			value, err := deserializePublicValue(serialized)
			if err != nil {
				return ExecutionResult{}, err
			}
			args[i] = value
		}
		kwargs := make(map[string]evaluator.Value, len(call.Kwargs))
		for key, serialized := range call.Kwargs {
			value, err := deserializePublicValue(serialized)
			if err != nil {
				return ExecutionResult{}, err
			}
			kwargs[key] = value
		}
		if err := machineCheckpoint.EditFailedCall(call.Service, call.Method, args, kwargs); err != nil {
			return ExecutionResult{}, err
		}
		internal.Mode = evaluator.MachineRetryCall
	}
	return r.publicMachineResult(checkpoint.Script, program, machine, machine.ResumeContext(ctx, machineCheckpoint, internal), checkpoint.Breakpoints)
}

func (r *Runtime) publicMachineResult(source string, program *ast.Program, machine *evaluator.Machine, result evaluator.MachineResult, breakpoints []Breakpoint) (ExecutionResult, error) {
	if result.Status == evaluator.MachineCompleted {
		return ExecutionResult{Status: ExecutionCompleted, Result: result.Value, Emitted: machine.Emitted()}, nil
	}
	cp, err := makePublicMachineCheckpoint(source, program, machine, result.Checkpoint, breakpoints)
	if err != nil {
		if result.Err != nil {
			return ExecutionResult{Status: ExecutionFailed, Failure: executionFailure(result.Err), Emitted: machine.Emitted()}, fmt.Errorf("%w (checkpoint creation failed: %v)", result.Err, err)
		}
		return ExecutionResult{}, err
	}
	if result.Status == evaluator.MachinePaused {
		return ExecutionResult{Status: ExecutionPaused, Checkpoint: cp, Reason: cp.Reason, Emitted: machine.Emitted()}, nil
	}
	return ExecutionResult{Status: ExecutionFailed, Checkpoint: cp, Reason: PauseError, Failure: cp.Failure, Emitted: machine.Emitted()}, result.Err
}

func makePublicMachineCheckpoint(source string, program *ast.Program, machine *evaluator.Machine, internal *evaluator.MachineCheckpoint, breakpoints []Breakpoint) (*Checkpoint, error) {
	reason := PauseReason(internal.PauseReason())
	var failure *ExecutionFailure
	var call *PendingServiceCall
	if !internal.IsPause() {
		reason, failure = PauseError, executionFailure(internal.Failure())
		state, ok := internal.FailedCall()
		if ok {
			args := make([]*SerializedValue, len(state.Args))
			for i, value := range state.Args {
				var err error
				args[i], err = serializePublicValue(value)
				if err != nil {
					return nil, err
				}
			}
			kwargs := make(map[string]*SerializedValue, len(state.Kwargs))
			for key, value := range state.Kwargs {
				serialized, err := serializePublicValue(value)
				if err != nil {
					return nil, err
				}
				kwargs[key] = serialized
			}
			call = &PendingServiceCall{Service: state.Service, Method: state.Method, Args: args, Kwargs: kwargs, Failure: failure, MayHaveSideEffects: true}
		}
	}
	hash := sha256.Sum256([]byte(source))
	position := internal.Position()
	if position.Line < 1 {
		position = ast.Position{Line: 1, Column: 1}
	}
	if position.Column < 1 {
		position.Column = 1
	}
	if failure != nil {
		failure.Position = Position{Line: position.Line, Column: position.Column}
	}
	kind := "program"
	if call != nil {
		kind = "service_call"
	}
	snapshot, err := machine.Snapshot(program)
	if err != nil {
		return nil, fmt.Errorf("snapshotting machine: %w", err)
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("encoding machine snapshot: %w", err)
	}
	return &Checkpoint{Version: CheckpointVersion, Script: source, ScriptHash: fmt.Sprintf("%x", hash), Program: SerializedProgram{RootID: "program", Nodes: []SerializedNode{{ID: "program", Kind: "program", Position: Position{Line: position.Line, Column: position.Column}, Fields: json.RawMessage(`{}`)}}}, Position: Position{Line: position.Line, Column: position.Column}, State: ExecutionState{Scopes: []SerializedScope{{ID: "global", Variables: map[string]*SerializedValue{}, IsGlobal: true}}, CurrentScopeID: "global", Limits: json.RawMessage(`{}`), Emitted: []*SerializedValue{}, ControlFlow: json.RawMessage(`{}`)}, Continuation: []ContinuationFrame{{Kind: kind, NodeID: "program", Phase: map[bool]string{true: "failed", false: "paused"}[call != nil], ScopeID: "global", State: json.RawMessage(`{}`), Call: call}}, Breakpoints: append([]Breakpoint(nil), breakpoints...), Reason: reason, Failure: failure, MachineSnapshot: snapshotJSON}, nil
}

func (r *Runtime) parseBreakpointCondition(source string) (ast.Expression, error) {
	program, err := r.Parse(source)
	if err != nil {
		return nil, err
	}
	if len(program.Statements) != 1 {
		return nil, fmt.Errorf("condition must contain one expression")
	}
	statement, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		return nil, fmt.Errorf("condition must be an expression")
	}
	return statement.Expression, nil
}

func executionFailure(err error) *ExecutionFailure {
	message := "execution failed"
	if err != nil {
		message = err.Error()
	}
	kind := "evaluation"
	if strings.Contains(message, "breakpoint condition") {
		kind = "breakpoint_condition"
	} else if strings.Contains(message, "service") || strings.Contains(message, "down") {
		kind = "service_call"
	}
	return &ExecutionFailure{Code: "evaluation_error", Message: message, Source: FailureSource{Kind: kind, NodeID: "program"}, Position: Position{Line: 1, Column: 1}}
}

func serializePublicValue(value evaluator.Value) (*SerializedValue, error) {
	internal, err := evaluator.NewSerializer(nil).SerializeValue(value)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(internal)
	if err != nil {
		return nil, err
	}
	var public SerializedValue
	if err := json.Unmarshal(data, &public); err != nil {
		return nil, err
	}
	return &public, nil
}

func deserializePublicValue(value *SerializedValue) (evaluator.Value, error) {
	if err := validateValue(value, "$.replacement_result"); err != nil {
		return nil, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var internal evaluator.SerializedValue
	if err := json.Unmarshal(data, &internal); err != nil {
		return nil, err
	}
	return evaluator.NewDeserializer(nil, nil, nil).DeserializeValue(&internal)
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

// CallFunction calls a user-defined function with the given arguments.
func (p *pipelineCaller) CallFunction(fn evaluator.Value, args []evaluator.Value) (evaluator.Value, error) {
	switch f := fn.(type) {
	case *evaluator.FunctionValue:
		return p.callUserFunction(f, args)
	case *evaluator.LambdaValue:
		return p.callLambda(f, args)
	case *evaluator.BuiltinValue:
		return f.Fn(args, nil)
	default:
		return nil, nil
	}
}

func (p *pipelineCaller) callUserFunction(fn *evaluator.FunctionValue, args []evaluator.Value) (evaluator.Value, error) {
	// Create new scope for function execution
	fnScope := evaluator.NewEnclosedScope(fn.Env)

	// Bind parameters
	for i, param := range fn.Parameters {
		var val evaluator.Value
		if i < len(args) {
			val = args[i]
		} else if param.Default != nil {
			var err error
			val, err = p.eval.Eval(param.Default)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil // Missing argument
		}
		fnScope.Set(param.Name.Value, val)
	}

	// Execute function body
	ctx := p.eval.Context()
	oldScope := ctx.Scope
	ctx.Scope = fnScope
	defer func() {
		ctx.Scope = oldScope
	}()

	result, err := p.eval.Eval(fn.Body)
	if err != nil {
		return nil, err
	}

	// Check for return value
	if retVal, ok := ctx.GetReturn(); ok {
		return retVal, nil
	}

	return result, nil
}

func (p *pipelineCaller) callLambda(fn *evaluator.LambdaValue, args []evaluator.Value) (evaluator.Value, error) {
	// Create new scope
	fnScope := evaluator.NewEnclosedScope(fn.Env)

	// Bind parameters
	for i, param := range fn.Parameters {
		if i < len(args) {
			fnScope.Set(param.Value, args[i])
		}
	}

	// Execute body
	ctx := p.eval.Context()
	oldScope := ctx.Scope
	ctx.Scope = fnScope
	defer func() {
		ctx.Scope = oldScope
	}()

	return p.eval.Eval(fn.Body)
}
