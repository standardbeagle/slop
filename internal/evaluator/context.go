package evaluator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scope represents a variable scope.
type Scope struct {
	store  map[string]Value
	parent *Scope
	mu     sync.RWMutex
}

// NewScope creates a new scope.
func NewScope() *Scope {
	return &Scope{
		store: make(map[string]Value),
	}
}

// NewEnclosedScope creates a new scope enclosed by the given parent.
func NewEnclosedScope(parent *Scope) *Scope {
	return &Scope{
		store:  make(map[string]Value),
		parent: parent,
	}
}

// Get retrieves a value from the scope or its parents.
func (s *Scope) Get(name string) (Value, bool) {
	s.mu.RLock()
	val, ok := s.store[name]
	s.mu.RUnlock()
	if ok {
		return val, true
	}
	if s.parent != nil {
		return s.parent.Get(name)
	}
	return nil, false
}

// Set sets a value in the current scope.
func (s *Scope) Set(name string, val Value) {
	s.mu.Lock()
	s.store[name] = val
	s.mu.Unlock()
}

// Update updates a value in the scope where it exists (or current scope if new).
func (s *Scope) Update(name string, val Value) {
	// Check if exists in current scope
	s.mu.RLock()
	_, ok := s.store[name]
	s.mu.RUnlock()
	if ok {
		s.mu.Lock()
		s.store[name] = val
		s.mu.Unlock()
		return
	}

	// Check parent scopes
	if s.parent != nil {
		if _, exists := s.parent.Get(name); exists {
			s.parent.Update(name, val)
			return
		}
	}

	// New variable - set in current scope
	s.Set(name, val)
}

// Has checks if a variable exists in this scope or parents.
func (s *Scope) Has(name string) bool {
	_, ok := s.Get(name)
	return ok
}

// evalFrame holds the per-call-frame mutable state for a single evaluation
// context: the active scope and all control-flow signals. Each user-function or
// lambda invocation gets its own evalFrame so concurrent goroutines (e.g. a
// future parallel(N) implementation) cannot interfere with each other's
// shouldReturn/shouldBreak flags or scope pointer.
//
// evalFrame is intentionally a value type: callers copy it or create a new one
// via newEvalFrame/childEvalFrame. The shared, concurrency-safe state (globals,
// services, limits, txlog) lives in Context.
type evalFrame struct {
	// Active variable scope for this call frame.
	Scope     *Scope
	baseScope *Scope

	// Control-flow signals — private to this frame.
	returnValue    Value
	shouldReturn   bool
	shouldBreak    bool
	shouldContinue bool
	shouldStop     bool
	rollback       bool

	// Pause state for checkpointing.
	shouldPause  bool
	pauseMessage string
}

// newEvalFrame creates a root evalFrame with the given scope.
func newEvalFrame(scope *Scope) *evalFrame {
	return &evalFrame{Scope: scope, baseScope: scope}
}

// childEvalFrame creates a fresh evalFrame for a function call. It shares no
// control-flow state with the parent — each call sees a clean slate. The new
// scope is set by the caller after creation.
func childEvalFrame(scope *Scope) *evalFrame {
	return &evalFrame{Scope: scope, baseScope: scope}
}

// Context holds the execution context for the evaluator.
type Context struct {
	// Current scope — kept for backward compatibility with serialize/deserialize
	// and top-level checkpoint restore. During evaluation the active scope lives
	// in the evalFrame; this field reflects the scope at the *start* of top-level
	// evaluation only. It is not read or written by concurrent goroutines once
	// evaluation begins.
	Scope *Scope

	// Global scope for built-ins and services
	Globals *Scope

	// Registered services
	Services map[string]Service

	// Execution limits
	Limits *ExecutionLimits

	// Transaction log for rollback
	TxLog *TransactionLog

	// Emitted values — protected by emitMu when accessed from goroutines.
	Emitted []Value
	emitMu  sync.Mutex

	// Control flow flags — kept for serialize/deserialize and checkpoint restore.
	// During evaluation these are NOT used; evalFrame carries the live state.
	returnValue    Value
	shouldReturn   bool
	shouldBreak    bool
	shouldContinue bool
	shouldStop     bool
	rollback       bool

	// Pause state for checkpointing — same note as above.
	shouldPause  bool
	pauseMessage string
}

// DefaultMaxCallDepth bounds recursion when no explicit MaxCallDepth is set.
// SLOP's analyzer rejects recursion statically, but that check is advisory and
// not enforced at runtime, so an untrusted script (or one fed past the analyzer)
// can still recurse unbounded and exhaust the Go stack — an unrecoverable crash
// of the host. This default keeps the guard always-on while staying well below
// the depth at which the goroutine stack overflows.
const DefaultMaxCallDepth = 5000

// ExecutionLimits defines limits on script execution.
type ExecutionLimits struct {
	MaxIterations  int64 // Maximum total loop iterations
	MaxLLMCalls    int64 // Maximum LLM calls
	MaxAPICalls    int64 // Maximum API calls
	MaxDuration    int64 // Maximum execution time in seconds
	MaxCost        float64 // Maximum cost in dollars
	MaxCallDepth   int64 // Maximum nested user function/lambda call depth (0 = DefaultMaxCallDepth)

	// Counters
	IterationCount int64
	LLMCallCount   int64
	APICallCount   int64
	StartTime      int64
	TotalCost      float64
	CallDepth      int64 // Current nested call depth
}

// TransactionLog records operations for potential rollback.
type TransactionLog struct {
	Operations []Operation
	nextID     int64
	mu         sync.Mutex
}

// Operation represents a logged operation.
type Operation struct {
	ID         int64             // Unique operation ID
	Timestamp  int64             // Unix timestamp
	Type       string            // "call", "write", "delete", "update", etc.
	Service    string            // Service name
	Method     string            // Method name
	Args       []Value           // Arguments
	Kwargs     map[string]Value  // Keyword arguments
	Result     Value             // Result
	Error      error             // Error if any
	Reversible bool              // Whether this can be undone
	UndoMethod string            // Method to call for undo
	UndoData   map[string]Value  // Data needed to undo
}

// NewTransactionLog creates a new transaction log.
func NewTransactionLog() *TransactionLog {
	return &TransactionLog{
		Operations: []Operation{},
	}
}

// Log adds an operation to the log.
func (t *TransactionLog) Log(op Operation) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nextID++
	op.ID = t.nextID
	op.Timestamp = unixNow()
	t.Operations = append(t.Operations, op)
	return op.ID
}

// LogCall creates and logs a service call operation.
func (t *TransactionLog) LogCall(service, method string, args []Value, kwargs map[string]Value, result Value, err error) int64 {
	return t.Log(Operation{
		Type:    "call",
		Service: service,
		Method:  method,
		Args:    args,
		Kwargs:  kwargs,
		Result:  result,
		Error:   err,
	})
}

// GetOperations returns a copy of all operations.
func (t *TransactionLog) GetOperations() []Operation {
	t.mu.Lock()
	defer t.mu.Unlock()
	ops := make([]Operation, len(t.Operations))
	copy(ops, t.Operations)
	return ops
}

// GetReversibleOperations returns operations that can be rolled back.
func (t *TransactionLog) GetReversibleOperations() []Operation {
	t.mu.Lock()
	defer t.mu.Unlock()
	var reversible []Operation
	for _, op := range t.Operations {
		if op.Reversible && op.Error == nil {
			reversible = append(reversible, op)
		}
	}
	return reversible
}

// Clear clears the transaction log.
func (t *TransactionLog) Clear() {
	t.mu.Lock()
	t.Operations = []Operation{}
	t.mu.Unlock()
}

// Size returns the number of logged operations.
func (t *TransactionLog) Size() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Operations)
}

// unixNow returns current unix timestamp in nanoseconds.
func unixNow() int64 {
	return 0 // Simplified - could use time.Now().UnixNano()
}

// Rollback represents a rollback handler.
type Rollback struct {
	log      *TransactionLog
	services map[string]Service
}

// NewRollback creates a rollback handler.
func NewRollback(log *TransactionLog, services map[string]Service) *Rollback {
	return &Rollback{log: log, services: services}
}

// Execute performs rollback of reversible operations in reverse order.
func (r *Rollback) Execute() []error {
	ops := r.log.GetReversibleOperations()
	var errors []error

	// Process in reverse order
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if err := r.rollbackOperation(op); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// rollbackOperation attempts to undo a single operation.
func (r *Rollback) rollbackOperation(op Operation) error {
	svc, ok := r.services[op.Service]
	if !ok {
		return fmt.Errorf("service not found for rollback: %s", op.Service)
	}

	// Check if service supports reversible operations
	if reversible, ok := svc.(ReversibleService); ok {
		return reversible.Undo(op.Method, op.UndoData)
	}

	// Try calling undo method if specified
	if op.UndoMethod != "" {
		var undoArgs []Value
		if op.UndoData != nil {
			if args, ok := op.UndoData["args"]; ok {
				if list, ok := args.(*ListValue); ok {
					undoArgs = list.Elements
				}
			}
		}
		_, err := svc.Call(op.UndoMethod, undoArgs, nil)
		return err
	}

	return fmt.Errorf("no undo method for operation: %s.%s", op.Service, op.Method)
}

// ReversibleService is implemented by services that support rollback.
type ReversibleService interface {
	Service
	// Undo reverses an operation given the method name and undo data.
	Undo(method string, undoData map[string]Value) error
	// IsReversible returns whether a method call can be undone.
	IsReversible(method string) bool
}

// ---- evalFrame control-flow accessors ----

func (f *evalFrame) SetReturn(value Value) {
	f.returnValue = value
	f.shouldReturn = true
}

func (f *evalFrame) GetReturn() (Value, bool) {
	if f.shouldReturn {
		val := f.returnValue
		f.returnValue = nil
		f.shouldReturn = false
		return val, true
	}
	return nil, false
}

func (f *evalFrame) ShouldReturn() bool   { return f.shouldReturn }
func (f *evalFrame) SetBreak()            { f.shouldBreak = true }
func (f *evalFrame) ClearBreak()          { f.shouldBreak = false }
func (f *evalFrame) ShouldBreak() bool    { return f.shouldBreak }
func (f *evalFrame) SetContinue()         { f.shouldContinue = true }
func (f *evalFrame) ClearContinue()       { f.shouldContinue = false }
func (f *evalFrame) ShouldContinue() bool { return f.shouldContinue }

func (f *evalFrame) SetStop(rollback bool) {
	f.shouldStop = true
	f.rollback = rollback
}

func (f *evalFrame) ShouldStop() bool    { return f.shouldStop }
func (f *evalFrame) NeedsRollback() bool { return f.rollback }

func (f *evalFrame) SetPause(message string) {
	f.shouldPause = true
	f.pauseMessage = message
}

func (f *evalFrame) ClearPause() {
	f.shouldPause = false
	f.pauseMessage = ""
}

func (f *evalFrame) ShouldPause() bool    { return f.shouldPause }
func (f *evalFrame) GetPauseMessage() string { return f.pauseMessage }

func (f *evalFrame) ShouldInterrupt() bool {
	return f.shouldStop || f.shouldReturn || f.shouldBreak || f.shouldContinue || f.shouldPause
}

// PushScope creates a new child scope and makes it current in this frame.
func (f *evalFrame) PushScope() {
	f.Scope = NewEnclosedScope(f.Scope)
}

// SetLocal updates an existing binding in this call frame or creates one in
// the active scope, without crossing into a closure or global parent scope.
func (f *evalFrame) SetLocal(name string, val Value) {
	for scope := f.Scope; scope != nil; scope = scope.parent {
		scope.mu.RLock()
		_, ok := scope.store[name]
		scope.mu.RUnlock()
		if ok {
			scope.Set(name, val)
			return
		}
		if scope == f.baseScope {
			break
		}
	}
	f.Scope.Set(name, val)
}

// PopScope returns to the parent scope in this frame.
func (f *evalFrame) PopScope() {
	if f.Scope.parent != nil {
		f.Scope = f.Scope.parent
	}
}

// NewContext creates a new execution context.
func NewContext() *Context {
	globals := NewScope()
	return &Context{
		Scope:    NewEnclosedScope(globals),
		Globals:  globals,
		Services: make(map[string]Service),
		Limits:   &ExecutionLimits{},
		TxLog:    NewTransactionLog(),
		Emitted:  []Value{},
	}
}

// NewContextWithLimits creates a new context with execution limits.
func NewContextWithLimits(limits *ExecutionLimits) *Context {
	ctx := NewContext()
	ctx.Limits = limits
	return ctx
}

// PushScope creates a new child scope and makes it current.
func (c *Context) PushScope() {
	c.Scope = NewEnclosedScope(c.Scope)
}

// PopScope returns to the parent scope.
func (c *Context) PopScope() {
	if c.Scope.parent != nil {
		c.Scope = c.Scope.parent
	}
}

// RegisterService registers an MCP service.
func (c *Context) RegisterService(name string, service Service) {
	c.Services[name] = service
	c.Globals.Set(name, &ServiceValue{Name: name, Service: service})
}

// GetService retrieves a registered service.
func (c *Context) GetService(name string) (Service, bool) {
	svc, ok := c.Services[name]
	return svc, ok
}

// RegisterBuiltin registers a built-in function.
func (c *Context) RegisterBuiltin(name string, fn BuiltinFunction) {
	c.Globals.Set(name, &BuiltinValue{Name: name, Fn: fn})
}

// Emit records an emitted value. Safe for concurrent goroutines.
func (c *Context) Emit(value Value) {
	c.emitMu.Lock()
	c.Emitted = append(c.Emitted, value)
	c.emitMu.Unlock()
}

// SetReturn sets the return value and flags return.
func (c *Context) SetReturn(value Value) {
	c.returnValue = value
	c.shouldReturn = true
}

// GetReturn gets the return value and clears the flag.
func (c *Context) GetReturn() (Value, bool) {
	if c.shouldReturn {
		val := c.returnValue
		c.returnValue = nil
		c.shouldReturn = false
		return val, true
	}
	return nil, false
}

// ShouldReturn checks if we should return from current function.
func (c *Context) ShouldReturn() bool {
	return c.shouldReturn
}

// SetBreak sets the break flag.
func (c *Context) SetBreak() {
	c.shouldBreak = true
}

// ClearBreak clears the break flag.
func (c *Context) ClearBreak() {
	c.shouldBreak = false
}

// ShouldBreak checks if we should break from current loop.
func (c *Context) ShouldBreak() bool {
	return c.shouldBreak
}

// SetContinue sets the continue flag.
func (c *Context) SetContinue() {
	c.shouldContinue = true
}

// ClearContinue clears the continue flag.
func (c *Context) ClearContinue() {
	c.shouldContinue = false
}

// ShouldContinue checks if we should continue to next iteration.
func (c *Context) ShouldContinue() bool {
	return c.shouldContinue
}

// SetStop sets the stop flag.
func (c *Context) SetStop(rollback bool) {
	c.shouldStop = true
	c.rollback = rollback
}

// ShouldStop checks if execution should stop.
func (c *Context) ShouldStop() bool {
	return c.shouldStop
}

// NeedsRollback checks if a rollback is needed.
func (c *Context) NeedsRollback() bool {
	return c.rollback
}

// SetPause sets the pause flag with an optional message.
func (c *Context) SetPause(message string) {
	c.shouldPause = true
	c.pauseMessage = message
}

// ClearPause clears the pause flag.
func (c *Context) ClearPause() {
	c.shouldPause = false
	c.pauseMessage = ""
}

// ShouldPause checks if execution should pause for checkpointing.
func (c *Context) ShouldPause() bool {
	return c.shouldPause
}

// GetPauseMessage returns the pause message (checkpoint name).
func (c *Context) GetPauseMessage() string {
	return c.pauseMessage
}

// IncrementIterations increments the iteration counter and checks limits.
func (c *Context) IncrementIterations() error {
	c.Limits.IterationCount++
	if c.Limits.MaxIterations > 0 && c.Limits.IterationCount > c.Limits.MaxIterations {
		return fmt.Errorf("iteration limit exceeded (%d)", c.Limits.MaxIterations)
	}
	return nil
}

// IncrementLLMCalls increments the LLM call counter and checks limits.
func (c *Context) IncrementLLMCalls() error {
	c.Limits.LLMCallCount++
	if c.Limits.MaxLLMCalls > 0 && c.Limits.LLMCallCount > c.Limits.MaxLLMCalls {
		return fmt.Errorf("LLM call limit exceeded (%d)", c.Limits.MaxLLMCalls)
	}
	return nil
}

// IncrementAPICalls increments the API call counter and checks limits.
func (c *Context) IncrementAPICalls() error {
	c.Limits.APICallCount++
	if c.Limits.MaxAPICalls > 0 && c.Limits.APICallCount > c.Limits.MaxAPICalls {
		return fmt.Errorf("API call limit exceeded (%d)", c.Limits.MaxAPICalls)
	}
	return nil
}

// EnterCall records entry into a user function or lambda body and enforces the
// recursion-depth cap. It must be paired with ExitCall (via defer) so the depth
// counter unwinds even when the body errors. The cap defaults to
// DefaultMaxCallDepth when MaxCallDepth is unset, so the guard protects the host
// against stack-exhaustion DoS regardless of caller configuration. The counter
// is decremented before returning the error so a recovered/caught overflow does
// not leave the depth permanently inflated.
func (c *Context) EnterCall() error {
	max := c.Limits.MaxCallDepth
	if max <= 0 {
		max = DefaultMaxCallDepth
	}
	c.Limits.CallDepth++
	if c.Limits.CallDepth > max {
		c.Limits.CallDepth--
		return fmt.Errorf("call depth limit exceeded (%d): possible infinite recursion", max)
	}
	return nil
}

// ExitCall records return from a user function or lambda body. It is the inverse
// of EnterCall and must run on every exit path (defer) to keep CallDepth honest.
func (c *Context) ExitCall() {
	if c.Limits.CallDepth > 0 {
		c.Limits.CallDepth--
	}
}

// AddCost adds to the cost counter and checks limits.
func (c *Context) AddCost(cost float64) error {
	c.Limits.TotalCost += cost
	if c.Limits.MaxCost > 0 && c.Limits.TotalCost > c.Limits.MaxCost {
		return fmt.Errorf("cost limit exceeded (%.2f)", c.Limits.MaxCost)
	}
	return nil
}

// CheckDuration enforces the wall-clock execution limit. The clock starts on
// the first call (or from a restored StartTime after checkpoint resume), so it
// is independent of the execution entry point. Returns an error once the
// elapsed time exceeds MaxDuration seconds.
func (c *Context) CheckDuration() error {
	if c.Limits.MaxDuration <= 0 {
		return nil
	}
	now := time.Now().Unix()
	if c.Limits.StartTime == 0 {
		c.Limits.StartTime = now
		return nil
	}
	if now-c.Limits.StartTime > c.Limits.MaxDuration {
		return fmt.Errorf("duration limit exceeded (%ds)", c.Limits.MaxDuration)
	}
	return nil
}

// CallContext derives a context for a single service call, bounded by the
// remaining execution time. A hung LLM/MCP call must not block its goroutine
// forever: the returned context cancels when the wall-clock budget would be
// exceeded, so the call unwinds with context.DeadlineExceeded instead of
// hanging. When MaxDuration is unset (<= 0) the parent context is returned
// unbounded.
//
// The clock anchor (StartTime) is shared with CheckDuration, so a call started
// late in a long-running script gets only the time left, and a restored
// StartTime after checkpoint resume keeps the deadline stable. The caller is
// responsible for invoking the returned CancelFunc to release resources.
func (c *Context) CallContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if c.Limits == nil || c.Limits.MaxDuration <= 0 {
		return context.WithCancel(parent)
	}
	start := c.Limits.StartTime
	if start == 0 {
		// Clock not yet started; anchor it now so the deadline and CheckDuration
		// agree on the same origin.
		start = time.Now().Unix()
		c.Limits.StartTime = start
	}
	deadline := time.Unix(start+c.Limits.MaxDuration, 0)
	return context.WithDeadline(parent, deadline)
}

// ShouldInterrupt checks if execution should be interrupted.
func (c *Context) ShouldInterrupt() bool {
	return c.shouldStop || c.shouldReturn || c.shouldBreak || c.shouldContinue || c.shouldPause
}
