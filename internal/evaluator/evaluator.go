package evaluator

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/limits"
)

// Evaluator is a tree-walking interpreter for SLOP.
//
// Goroutine safety: the shared Context (globals, services, limits, txlog) is
// safe for concurrent reads. Emitted values are protected by emitMu. Each
// Evaluator instance owns exactly one evalFrame which holds the mutable
// per-call state (active Scope + control-flow flags). User-function and lambda
// calls create a fresh child Evaluator with its own frame, so parallel
// goroutines never share control-flow state or scope pointers.
type Evaluator struct {
	ctx   *Context
	frame *evalFrame
}

// New creates a new Evaluator with a fresh context.
func New() *Evaluator {
	ctx := NewContext()
	return &Evaluator{
		ctx:   ctx,
		frame: newEvalFrame(ctx.Scope),
	}
}

// NewWithContext creates an Evaluator with the given context.
func NewWithContext(ctx *Context) *Evaluator {
	return &Evaluator{
		ctx:   ctx,
		frame: newEvalFrame(ctx.Scope),
	}
}

// Context returns the evaluator's context.
func (e *Evaluator) Context() *Context {
	return e.ctx
}

// childEval creates a child evaluator for a function call. The child shares the
// same Context (globals, services, limits) but gets its own evalFrame with the
// supplied scope, so control-flow signals are isolated per invocation.
func (e *Evaluator) childEval(scope *Scope) *Evaluator {
	return &Evaluator{
		ctx:   e.ctx,
		frame: childEvalFrame(scope),
	}
}

// withContext temporarily replaces this evaluator's context and frame, invokes
// fn, then restores the originals. Used by module loading (BuildScopes) which
// needs to evaluate module bodies in a separate context.
// Returns the saved (ctx, frame) so the caller can observe post-eval state if needed.
func (e *Evaluator) withContext(ctx *Context, fn func()) {
	origCtx := e.ctx
	origFrame := e.frame
	e.ctx = ctx
	e.frame = newEvalFrame(ctx.Scope)
	fn()
	e.ctx = origCtx
	e.frame = origFrame
}

// Eval evaluates an AST node and returns the result.
// This is the main entry point for the interpreter. It dispatches to the
// appropriate evaluation method based on the node type.
//
// After evaluation completes, the frame's control-flow state is synced back
// to Context so that callers inspecting ctx.ShouldPause(), ctx.ShouldStop(),
// etc. see the correct state — this is required for checkpointing and for
// external callers that query the context after a top-level Eval call.
func (e *Evaluator) Eval(node ast.Node) (Value, error) {
	if node == nil {
		return nil, nil
	}
	result, err := evalNode(e, node)
	// Sync per-frame control-flow state to Context so external observers
	// (checkpoint, tests, runtime) see consistent state.
	e.syncFrameToCtx()
	return result, err
}

// syncFrameToCtx copies the current frame's control-flow flags into the shared
// Context. This is called after every top-level Eval so that code that inspects
// ctx.ShouldPause() / ctx.ShouldStop() etc. sees the live state.
func (e *Evaluator) syncFrameToCtx() {
	e.ctx.shouldPause = e.frame.shouldPause
	e.ctx.pauseMessage = e.frame.pauseMessage
	e.ctx.shouldStop = e.frame.shouldStop
	e.ctx.rollback = e.frame.rollback
	e.ctx.shouldReturn = e.frame.shouldReturn
	e.ctx.returnValue = e.frame.returnValue
	e.ctx.shouldBreak = e.frame.shouldBreak
	e.ctx.shouldContinue = e.frame.shouldContinue
	// Keep ctx.Scope in sync for checkpoint serialization.
	e.ctx.Scope = e.frame.Scope
}

// FrameShouldReturn reports whether the active frame has a pending return.
func (e *Evaluator) FrameShouldReturn() bool { return e.frame.shouldReturn }

// FrameGetReturn consumes the pending return value from the active frame,
// clearing both the frame and ctx state atomically.
func (e *Evaluator) FrameGetReturn() (Value, bool) {
	val, ok := e.frame.GetReturn()
	if ok {
		// Also clear the ctx mirror so observers don't see stale state.
		e.ctx.shouldReturn = false
		e.ctx.returnValue = nil
	}
	return val, ok
}

// FrameShouldStop reports whether the active frame has a stop signal.
func (e *Evaluator) FrameShouldStop() bool { return e.frame.shouldStop }

// FrameShouldPause reports whether the active frame has a pause signal.
func (e *Evaluator) FrameShouldPause() bool { return e.frame.shouldPause }

// FrameClearPause clears the pause signal on both frame and ctx.
func (e *Evaluator) FrameClearPause() {
	e.frame.ClearPause()
	e.ctx.ClearPause()
}

// evalNode dispatches to specific evaluation functions based on node type.
// Extracted from Eval to separate nil checking from type dispatching.
func evalNode(e *Evaluator, node ast.Node) (Value, error) {
	switch node := node.(type) {
	// Program
	case *ast.Program:
		return e.evalProgram(node)

	// Control flow statements
	case *ast.Block:
		return e.evalBlock(node)
	case *ast.ReturnStatement:
		return e.evalReturnStatement(node)
	case *ast.EmitStatement:
		return e.evalEmitStatement(node)
	case *ast.StopStatement:
		return e.evalStopStatement(node)
	case *ast.TryStatement:
		return e.evalTryStatement(node)
	case *ast.BreakStatement:
		e.frame.SetBreak()
		return NONE, nil
	case *ast.ContinueStatement:
		e.frame.SetContinue()
		return NONE, nil
	case *ast.PauseStatement:
		return e.evalPauseStatement(node)

	// Conditional and loop statements
	case *ast.IfStatement:
		return e.evalIfStatement(node)
	case *ast.ForStatement:
		return e.evalForStatement(node)
	case *ast.MatchStatement:
		return e.evalMatchStatement(node)

	// Definition and assignment
	case *ast.DefStatement:
		return e.evalDefStatement(node)
	case *ast.AssignStatement:
		return e.evalAssignment(node)

	// Expression statements (just evaluate the expression)
	case *ast.ExpressionStatement:
		return e.Eval(node.Expression)

	// Literals
	case *ast.IntegerLiteral:
		return &IntValue{Value: node.Value}, nil
	case *ast.FloatLiteral:
		return &FloatValue{Value: node.Value}, nil
	case *ast.StringLiteral:
		return &StringValue{Value: node.Value}, nil
	case *ast.BooleanLiteral:
		return NewBool(node.Value), nil
	case *ast.NoneLiteral:
		return NONE, nil

	// Collection literals
	case *ast.ListLiteral:
		return e.evalListLiteral(node)
	case *ast.MapLiteral:
		return e.evalMapLiteral(node)
	case *ast.SetLiteral:
		return e.evalSetLiteral(node)

	// Operators and expressions
	case *ast.Identifier:
		return e.evalIdentifier(node)
	case *ast.PrefixExpression:
		return e.evalPrefixExpression(node)
	case *ast.InfixExpression:
		return e.evalInfixExpression(node)
	case *ast.TernaryExpression:
		return e.evalTernaryExpression(node)

	// Index and slice operations
	case *ast.IndexExpression:
		return e.evalIndexExpression(node)
	case *ast.SliceExpression:
		return e.evalSliceExpression(node)

	// Member and property access
	case *ast.MemberExpression:
		return e.evalMemberExpression(node)

	// Call and invocation
	case *ast.CallExpression:
		return e.evalCallExpression(node)
	case *ast.PipelineExpression:
		return e.evalPipelineExpression(node)

	// Lambda and functional
	case *ast.LambdaExpression:
		return e.evalLambdaExpression(node)

	// Pattern matching and iteration
	case *ast.MatchExpression:
		return e.evalMatchExpression(node)
	case *ast.RangeExpression:
		return e.evalRangeExpression(node)
	case *ast.ListComprehension:
		return e.evalListComprehension(node)
	case *ast.MapComprehension:
		return e.evalMapComprehension(node)

	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

func (e *Evaluator) evalProgram(program *ast.Program) (Value, error) {
	// If program has modules, handle them
	if len(program.Modules) > 0 {
		return e.evalProgramWithModules(program)
	}

	// Regular program without modules
	var result Value = NONE

	for _, stmt := range program.Statements {
		val, err := e.Eval(stmt)
		if err != nil {
			return nil, err
		}
		result = val

		// Check for control flow
		if e.frame.ShouldReturn() {
			result, _ = e.frame.GetReturn()
			break
		}
		if e.frame.ShouldStop() || e.frame.ShouldPause() {
			break
		}
	}

	return result, nil
}

func (e *Evaluator) evalProgramWithModules(program *ast.Program) (Value, error) {
	// Create module resolver
	resolver := NewModuleResolver()

	// Load all modules
	if err := resolver.LoadModules(program.Modules); err != nil {
		return nil, err
	}

	// Validate dependencies
	if errors := resolver.Validate(); len(errors) > 0 {
		// Return first error for simplicity
		return nil, errors[0]
	}

	// Build scopes for all SOURCE modules
	if err := resolver.BuildScopes(e); err != nil {
		return nil, err
	}

	// Build main scope with USE modules wired in
	mainScope, err := resolver.BuildMainScope()
	if err != nil {
		return nil, err
	}

	// Merge main scope into evaluator's scope
	for name, val := range mainScope.store {
		e.frame.Scope.Set(name, val)
	}

	// Execute MAIN module body if present
	mainModule := resolver.GetMainModule()
	if mainModule != nil {
		for _, stmt := range mainModule.Body {
			result, err := e.Eval(stmt)
			if err != nil {
				return nil, err
			}

			// Check for control flow
			if e.frame.ShouldReturn() {
				result, _ = e.frame.GetReturn()
				return result, nil
			}
			if e.frame.ShouldStop() || e.frame.ShouldPause() {
				return result, nil
			}
		}
	}

	return NONE, nil
}

func (e *Evaluator) evalBlock(block *ast.Block) (Value, error) {
	var result Value = NONE

	for _, stmt := range block.Statements {
		val, err := e.Eval(stmt)
		if err != nil {
			return nil, err
		}
		result = val

		// Check for control flow
		if e.frame.ShouldInterrupt() {
			break
		}
	}

	return result, nil
}

func (e *Evaluator) evalIdentifier(node *ast.Identifier) (Value, error) {
	val, ok := e.frame.Scope.Get(node.Value)
	if ok {
		return val, nil
	}

	// Check globals
	val, ok = e.ctx.Globals.Get(node.Value)
	if ok {
		return val, nil
	}

	// Fallback: if hyphenated and all parts are in scope, treat as subtraction
	if strings.Contains(node.Value, "-") {
		return e.evalHyphenatedFallback(node.Value)
	}

	return nil, fmt.Errorf("undefined variable: %s", node.Value)
}

func (e *Evaluator) evalHyphenatedFallback(name string) (Value, error) {
	parts := strings.Split(name, "-")
	values := make([]Value, len(parts))
	for i, part := range parts {
		val, ok := e.frame.Scope.Get(part)
		if !ok {
			val, ok = e.ctx.Globals.Get(part)
		}
		if !ok {
			return nil, fmt.Errorf("undefined variable: %s", name)
		}
		values[i] = val
	}
	// Left-to-right subtraction: a-b-c → (a - b) - c
	result := values[0]
	for i := 1; i < len(values); i++ {
		var err error
		result, err = e.binaryOp("-", result, values[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (e *Evaluator) evalAssignment(node *ast.AssignStatement) (Value, error) {
	val, err := e.Eval(node.Value)
	if err != nil {
		return nil, err
	}

	// Handle compound assignment
	if node.Operator != "=" {
		for _, target := range node.Targets {
			oldVal, err := e.Eval(target)
			if err != nil {
				return nil, err
			}

			switch node.Operator {
			case "+=":
				val, err = e.binaryOp("+", oldVal, val)
			case "-=":
				val, err = e.binaryOp("-", oldVal, val)
			case "*=":
				val, err = e.binaryOp("*", oldVal, val)
			case "/=":
				val, err = e.binaryOp("/", oldVal, val)
			default:
				return nil, fmt.Errorf("unknown assignment operator: %s", node.Operator)
			}
			if err != nil {
				return nil, err
			}
		}
	}

	// Assign to targets
	for _, target := range node.Targets {
		if err := e.assignToTarget(target, val, node.Operator != "="); err != nil {
			return nil, err
		}
	}

	return val, nil
}

func (e *Evaluator) assignToTarget(target ast.Expression, val Value, update bool) error {
	switch t := target.(type) {
	case *ast.Identifier:
		if update {
			e.frame.Scope.Update(t.Value, val)
		} else {
			e.frame.SetLocal(t.Value, val)
		}
		return nil

	case *ast.IndexExpression:
		obj, err := e.Eval(t.Left)
		if err != nil {
			return err
		}
		idx, err := e.Eval(t.Index)
		if err != nil {
			return err
		}

		switch o := obj.(type) {
		case *ListValue:
			i, ok := ToInt(idx)
			if !ok {
				return fmt.Errorf("list index must be integer, got %s", idx.Type())
			}
			if i < 0 || i >= int64(len(o.Elements)) {
				return fmt.Errorf("list index out of range: %d", i)
			}
			o.Elements[i] = val
			return nil

		case *MapValue:
			key, ok := idx.(*StringValue)
			if !ok {
				key = &StringValue{Value: idx.String()}
			}
			o.Set(key.Value, val)
			return nil

		default:
			return fmt.Errorf("cannot index %s", obj.Type())
		}

	case *ast.MemberExpression:
		obj, err := e.Eval(t.Object)
		if err != nil {
			return err
		}

		if m, ok := obj.(*MapValue); ok {
			m.Set(t.Property.Value, val)
			return nil
		}

		return fmt.Errorf("cannot set property on %s", obj.Type())

	default:
		return fmt.Errorf("invalid assignment target: %T", target)
	}
}

func (e *Evaluator) evalListLiteral(node *ast.ListLiteral) (Value, error) {
	elements := make([]Value, len(node.Elements))
	for i, elem := range node.Elements {
		val, err := e.Eval(elem)
		if err != nil {
			return nil, err
		}
		elements[i] = val
	}
	return &ListValue{Elements: elements}, nil
}

func (e *Evaluator) evalMapLiteral(node *ast.MapLiteral) (Value, error) {
	m := NewMapValue()
	for _, key := range node.Order {
		// If key is an identifier, use it as a string key (like {a: 1} means {"a": 1})
		var keyStr string
		if ident, ok := key.(*ast.Identifier); ok {
			keyStr = ident.Value
		} else {
			keyVal, err := e.Eval(key)
			if err != nil {
				return nil, err
			}
			keyStr = keyVal.String()
			if sv, ok := keyVal.(*StringValue); ok {
				keyStr = sv.Value
			}
		}

		valExpr := node.Pairs[key]
		val, err := e.Eval(valExpr)
		if err != nil {
			return nil, err
		}

		m.Set(keyStr, val)
	}
	return m, nil
}

func (e *Evaluator) evalSetLiteral(node *ast.SetLiteral) (Value, error) {
	s := NewSetValue()
	for _, elem := range node.Elements {
		val, err := e.Eval(elem)
		if err != nil {
			return nil, err
		}
		s.Add(val)
	}
	return s, nil
}

func (e *Evaluator) evalPrefixExpression(node *ast.PrefixExpression) (Value, error) {
	right, err := e.Eval(node.Right)
	if err != nil {
		return nil, err
	}

	switch node.Operator {
	case "-":
		switch r := right.(type) {
		case *IntValue:
			return &IntValue{Value: -r.Value}, nil
		case *FloatValue:
			return &FloatValue{Value: -r.Value}, nil
		default:
			return nil, fmt.Errorf("cannot negate %s", right.Type())
		}

	case "not", "!":
		return NewBool(!right.IsTruthy()), nil

	default:
		return nil, fmt.Errorf("unknown prefix operator: %s", node.Operator)
	}
}

func (e *Evaluator) evalInfixExpression(node *ast.InfixExpression) (Value, error) {
	// Short-circuit evaluation for boolean operators
	if node.Operator == "and" {
		left, err := e.Eval(node.Left)
		if err != nil {
			return nil, err
		}
		if !left.IsTruthy() {
			return left, nil
		}
		return e.Eval(node.Right)
	}

	if node.Operator == "or" {
		left, err := e.Eval(node.Left)
		if err != nil {
			return nil, err
		}
		if left.IsTruthy() {
			return left, nil
		}
		return e.Eval(node.Right)
	}

	left, err := e.Eval(node.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.Eval(node.Right)
	if err != nil {
		return nil, err
	}

	return e.binaryOp(node.Operator, left, right)
}

func (e *Evaluator) binaryOp(op string, left, right Value) (Value, error) {
	// Comparison operators
	switch op {
	case "==":
		return NewBool(Equal(left, right)), nil
	case "!=":
		return NewBool(!Equal(left, right)), nil
	case "<", ">", "<=", ">=":
		cmp, err := Compare(left, right)
		if err != nil {
			return nil, err
		}
		switch op {
		case "<":
			return NewBool(cmp < 0), nil
		case ">":
			return NewBool(cmp > 0), nil
		case "<=":
			return NewBool(cmp <= 0), nil
		case ">=":
			return NewBool(cmp >= 0), nil
		}
	case "in":
		return e.evalInOperator(left, right)
	}

	// Numeric operators
	if IsNumber(left) && IsNumber(right) {
		return e.numericOp(op, left, right)
	}

	// String concatenation
	if op == "+" {
		if ls, ok := left.(*StringValue); ok {
			if rs, ok := right.(*StringValue); ok {
				return &StringValue{Value: ls.Value + rs.Value}, nil
			}
		}
		// String + anything
		if ls, ok := left.(*StringValue); ok {
			return &StringValue{Value: ls.Value + right.String()}, nil
		}
	}

	// String repetition
	if op == "*" {
		if ls, ok := left.(*StringValue); ok {
			if ri, ok := right.(*IntValue); ok {
				result := ""
				for i := int64(0); i < ri.Value; i++ {
					result += ls.Value
				}
				return &StringValue{Value: result}, nil
			}
		}
	}

	// List concatenation
	if op == "+" {
		if ll, ok := left.(*ListValue); ok {
			if rl, ok := right.(*ListValue); ok {
				elements := make([]Value, len(ll.Elements)+len(rl.Elements))
				copy(elements, ll.Elements)
				copy(elements[len(ll.Elements):], rl.Elements)
				return &ListValue{Elements: elements}, nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported operation: %s %s %s", left.Type(), op, right.Type())
}

func (e *Evaluator) numericOp(op string, left, right Value) (Value, error) {
	// If either is float, use float arithmetic
	if IsFloat(left) || IsFloat(right) {
		lf, _ := ToFloat(left)
		rf, _ := ToFloat(right)

		switch op {
		case "+":
			return &FloatValue{Value: lf + rf}, nil
		case "-":
			return &FloatValue{Value: lf - rf}, nil
		case "*":
			return &FloatValue{Value: lf * rf}, nil
		case "/":
			if rf == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return &FloatValue{Value: lf / rf}, nil
		case "%":
			if rf == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return &FloatValue{Value: math.Mod(lf, rf)}, nil
		case "**":
			return &FloatValue{Value: math.Pow(lf, rf)}, nil
		}
	}

	// Integer arithmetic
	li, _ := ToInt(left)
	ri, _ := ToInt(right)

	switch op {
	case "+":
		return &IntValue{Value: li + ri}, nil
	case "-":
		return &IntValue{Value: li - ri}, nil
	case "*":
		return &IntValue{Value: li * ri}, nil
	case "/":
		if ri == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return &IntValue{Value: li / ri}, nil
	case "%":
		if ri == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return &IntValue{Value: li % ri}, nil
	case "**":
		result := int64(1)
		for i := int64(0); i < ri; i++ {
			result *= li
		}
		return &IntValue{Value: result}, nil
	}

	return nil, fmt.Errorf("unknown operator: %s", op)
}

func (e *Evaluator) evalInOperator(left, right Value) (Value, error) {
	switch r := right.(type) {
	case *ListValue:
		for _, elem := range r.Elements {
			if Equal(left, elem) {
				return TRUE, nil
			}
		}
		return FALSE, nil

	case *MapValue:
		key := left.String()
		if sv, ok := left.(*StringValue); ok {
			key = sv.Value
		}
		_, ok := r.Get(key)
		return NewBool(ok), nil

	case *SetValue:
		return NewBool(r.Has(left)), nil

	case *StringValue:
		if ls, ok := left.(*StringValue); ok {
			for i := 0; i <= len(r.Value)-len(ls.Value); i++ {
				if r.Value[i:i+len(ls.Value)] == ls.Value {
					return TRUE, nil
				}
			}
		}
		return FALSE, nil

	default:
		return nil, fmt.Errorf("cannot use 'in' with %s", right.Type())
	}
}

func (e *Evaluator) evalCallExpression(node *ast.CallExpression) (Value, error) {
	// Evaluate the function
	fn, err := e.Eval(node.Function)
	if err != nil {
		return nil, err
	}

	// Evaluate arguments
	args := make([]Value, len(node.Arguments))
	for i, arg := range node.Arguments {
		val, err := e.Eval(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Evaluate keyword arguments
	kwargs := make(map[string]Value)
	for name, expr := range node.Kwargs {
		val, err := e.Eval(expr)
		if err != nil {
			return nil, err
		}
		kwargs[name] = val
	}

	return e.callFunction(fn, args, kwargs)
}

// InvokeFunction calls a callable Value with the given args. It is the public
// entry point for external callers (e.g. pipeline builtins) that hold a Value
// and need to invoke it with the evaluator's full call machinery — including
// per-frame scope isolation and control-flow safety.
func (e *Evaluator) InvokeFunction(fn Value, args []Value) (Value, error) {
	return e.callFunction(fn, args, nil)
}

func (e *Evaluator) callFunction(fn Value, args []Value, kwargs map[string]Value) (Value, error) {
	switch f := fn.(type) {
	case *FunctionValue:
		return e.callUserFunction(f, args, kwargs)

	case *LambdaValue:
		return e.callLambda(f, args)

	case *BuiltinValue:
		return f.Fn(args, kwargs)

	case *BoundMethodValue:
		// Call the service method and log the call
		return e.callServiceMethod(f, args, kwargs)

	default:
		return nil, fmt.Errorf("not callable: %s", fn.Type())
	}
}

// callServiceMethod calls a service method and logs it to the transaction log.
func (e *Evaluator) callServiceMethod(bound *BoundMethodValue, args []Value, kwargs map[string]Value) (Value, error) {
	// Enforce resource limits before performing any work. Failing fast here is
	// what makes MaxLLMCalls/MaxAPICalls/MaxDuration effective against untrusted
	// scripts: the call (and its cost) never happens once a cap is reached.
	if err := e.ctx.CheckDuration(); err != nil {
		return nil, err
	}
	if isLLMService(bound.Service, bound.ServiceName) {
		if err := e.ctx.IncrementLLMCalls(); err != nil {
			return nil, err
		}
	} else {
		if err := e.ctx.IncrementAPICalls(); err != nil {
			return nil, err
		}
	}

	// Call the service. Prefer the context-aware path so a hung LLM/MCP call
	// cancels when the wall-clock budget is exhausted instead of blocking its
	// goroutine forever; the deadline is rooted in the execution duration limit.
	// Services without context support fall back to the plain Call.
	var result Value
	var err error
	if ctxSvc, ok := bound.Service.(ContextService); ok {
		callCtx, cancel := e.ctx.CallContext(context.Background())
		result, err = ctxSvc.CallWithContext(callCtx, bound.Method, args, kwargs)
		cancel()
	} else {
		result, err = bound.Service.Call(bound.Method, args, kwargs)
	}

	// Accumulate cost for services that report it, enforcing MaxCost. We do this
	// after the call because cost is only known once the call completes. A
	// failed call still counts toward the call budget but contributes no cost.
	if err == nil {
		if reporter, ok := bound.Service.(CostReporter); ok {
			if cerr := e.ctx.AddCost(reporter.LastCallCost()); cerr != nil {
				return nil, cerr
			}
		}
	}

	// Log the call to the transaction log
	var logErr error
	if err != nil {
		logErr = err
	}

	// Check if the operation is reversible
	reversible := false
	if rev, ok := bound.Service.(ReversibleService); ok {
		reversible = rev.IsReversible(bound.Method)
	}

	e.ctx.TxLog.Log(Operation{
		Type:       "call",
		Service:    bound.ServiceName,
		Method:     bound.Method,
		Args:       args,
		Kwargs:     kwargs,
		Result:     result,
		Error:      logErr,
		Reversible: reversible,
	})

	return result, err
}

// isLLMService reports whether a service's calls count against the LLM budget.
// A service may declare itself via the LLMServiceMarker interface; otherwise we
// fall back to the conventional "llm" registration name.
func isLLMService(svc Service, name string) bool {
	if marker, ok := svc.(LLMServiceMarker); ok {
		return marker.IsLLMService()
	}
	return name == "llm"
}

func (e *Evaluator) callUserFunction(fn *FunctionValue, args []Value, kwargs map[string]Value) (Value, error) {
	// Bound recursion before entering the body: an unbounded recursive call would
	// exhaust the Go stack and crash the host unrecoverably. The depth unwinds on
	// every exit path via the deferred ExitCall.
	if err := e.ctx.EnterCall(); err != nil {
		return nil, err
	}
	defer e.ctx.ExitCall()

	// Create new scope for function execution, enclosed by the function's
	// definition environment (closure semantics).
	fnScope := NewEnclosedScope(fn.Env)

	// Bind parameters
	for i, param := range fn.Parameters {
		var val Value

		// Check kwargs first
		if kwVal, ok := kwargs[param.Name.Value]; ok {
			val = kwVal
		} else if i < len(args) {
			val = args[i]
		} else if param.Default != nil {
			// Evaluate default value in the *caller's* frame.
			var err error
			val, err = e.Eval(param.Default)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("missing argument: %s", param.Name.Value)
		}

		fnScope.Set(param.Name.Value, val)
	}

	// Execute function body in an isolated child evaluator. The child owns its
	// own evalFrame (fresh control-flow flags + fnScope) so a concurrent caller
	// cannot observe or corrupt this call's return/break/stop state.
	child := e.childEval(fnScope)
	result, err := child.evalBlock(fn.Body)
	if err != nil {
		return nil, err
	}

	// Harvest return value from the child frame — it is never visible to the
	// caller's frame, keeping control-flow signals local.
	if retVal, ok := child.frame.GetReturn(); ok {
		return retVal, nil
	}

	// If the function's body triggered stop/pause, bubble the signal up to
	// the caller's frame so top-level evaluation can react.
	if child.frame.ShouldStop() {
		e.frame.SetStop(child.frame.NeedsRollback())
	}
	if child.frame.ShouldPause() {
		e.frame.SetPause(child.frame.GetPauseMessage())
	}

	return result, nil
}

func (e *Evaluator) callLambda(fn *LambdaValue, args []Value) (Value, error) {
	if len(args) != len(fn.Parameters) {
		return nil, fmt.Errorf("lambda expects %d arguments, got %d", len(fn.Parameters), len(args))
	}

	// Bound recursion before entering the body; see callUserFunction.
	if err := e.ctx.EnterCall(); err != nil {
		return nil, err
	}
	defer e.ctx.ExitCall()

	// Create new scope enclosed by the lambda's capture environment.
	fnScope := NewEnclosedScope(fn.Env)

	// Bind parameters
	for i, param := range fn.Parameters {
		fnScope.Set(param.Value, args[i])
	}

	// Execute body in an isolated child evaluator (own frame + fnScope).
	child := e.childEval(fnScope)
	result, err := child.Eval(fn.Body)
	if err != nil {
		return nil, err
	}

	// Bubble stop/pause signals from lambda body to caller frame.
	if child.frame.ShouldStop() {
		e.frame.SetStop(child.frame.NeedsRollback())
	}
	if child.frame.ShouldPause() {
		e.frame.SetPause(child.frame.GetPauseMessage())
	}

	return result, nil
}

func (e *Evaluator) evalIndexExpression(node *ast.IndexExpression) (Value, error) {
	left, err := e.Eval(node.Left)
	if err != nil {
		return nil, err
	}

	index, err := e.Eval(node.Index)
	if err != nil {
		return nil, err
	}

	// Handle optional access
	if node.Optional && IsNone(left) {
		return NONE, nil
	}

	switch l := left.(type) {
	case *ListValue:
		i, ok := ToInt(index)
		if !ok {
			return nil, fmt.Errorf("list index must be integer, got %s", index.Type())
		}
		// Handle negative indices
		if i < 0 {
			i = int64(len(l.Elements)) + i
		}
		if i < 0 || i >= int64(len(l.Elements)) {
			if node.Optional {
				return NONE, nil
			}
			return nil, fmt.Errorf("list index out of range: %d", i)
		}
		return l.Elements[i], nil

	case *MapValue:
		key := index.String()
		if sv, ok := index.(*StringValue); ok {
			key = sv.Value
		}
		val, ok := l.Get(key)
		if !ok {
			if node.Optional {
				return NONE, nil
			}
			return NONE, nil // Maps return none for missing keys
		}
		return val, nil

	case *StringValue:
		i, ok := ToInt(index)
		if !ok {
			return nil, fmt.Errorf("string index must be integer, got %s", index.Type())
		}
		if i < 0 {
			i = int64(len(l.Value)) + i
		}
		if i < 0 || i >= int64(len(l.Value)) {
			if node.Optional {
				return NONE, nil
			}
			return nil, fmt.Errorf("string index out of range: %d", i)
		}
		return &StringValue{Value: string(l.Value[i])}, nil

	default:
		return nil, fmt.Errorf("cannot index %s", left.Type())
	}
}

func (e *Evaluator) evalSliceExpression(node *ast.SliceExpression) (Value, error) {
	left, err := e.Eval(node.Left)
	if err != nil {
		return nil, err
	}

	var start, end, step int64
	var length int64

	switch l := left.(type) {
	case *ListValue:
		length = int64(len(l.Elements))
	case *StringValue:
		length = int64(len(l.Value))
	default:
		return nil, fmt.Errorf("cannot slice %s", left.Type())
	}

	end = length

	if node.Start != nil {
		startVal, err := e.Eval(node.Start)
		if err != nil {
			return nil, err
		}
		start, _ = ToInt(startVal)
		if start < 0 {
			start = length + start
		}
	}

	if node.End != nil {
		endVal, err := e.Eval(node.End)
		if err != nil {
			return nil, err
		}
		end, _ = ToInt(endVal)
		if end < 0 {
			end = length + end
		}
	}

	if node.Step != nil {
		stepVal, err := e.Eval(node.Step)
		if err != nil {
			return nil, err
		}
		step, _ = ToInt(stepVal)
		if step == 0 {
			return nil, fmt.Errorf("slice step cannot be zero")
		}
	} else {
		step = 1
	}

	// Clamp values
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	if start > end {
		start = end
	}

	switch l := left.(type) {
	case *ListValue:
		elements := []Value{}
		if step > 0 {
			for i := start; i < end; i += step {
				elements = append(elements, l.Elements[i])
			}
		} else {
			for i := end - 1; i >= start; i += step {
				elements = append(elements, l.Elements[i])
			}
		}
		return &ListValue{Elements: elements}, nil

	case *StringValue:
		result := ""
		if step > 0 {
			for i := start; i < end; i += step {
				result += string(l.Value[i])
			}
		} else {
			for i := end - 1; i >= start; i += step {
				result += string(l.Value[i])
			}
		}
		return &StringValue{Value: result}, nil
	}

	return nil, fmt.Errorf("cannot slice %s", left.Type())
}

func (e *Evaluator) evalMemberExpression(node *ast.MemberExpression) (Value, error) {
	obj, err := e.Eval(node.Object)
	if err != nil {
		return nil, err
	}

	// Handle optional access
	if node.Optional && IsNone(obj) {
		return NONE, nil
	}

	propName := node.Property.Value

	// Check for method on value types
	if method := e.getMethod(obj, propName); method != nil {
		return method, nil
	}

	// Handle map property access
	if m, ok := obj.(*MapValue); ok {
		val, ok := m.Get(propName)
		if !ok {
			if node.Optional {
				return NONE, nil
			}
			return NONE, nil
		}
		return val, nil
	}

	// Handle service access
	if svc, ok := obj.(*ServiceValue); ok {
		return &BoundMethodValue{
			ServiceName: svc.Name,
			Service:     svc.Service,
			Method:      propName,
		}, nil
	}

	// Handle module access
	if mod, ok := obj.(*ModuleValue); ok {
		val, ok := mod.Get(propName)
		if !ok {
			if node.Optional {
				return NONE, nil
			}
			return nil, fmt.Errorf("module '%s' has no member '%s'", mod.Name, propName)
		}
		return val, nil
	}

	return nil, fmt.Errorf("cannot access property '%s' on %s", propName, obj.Type())
}

// BoundMethodValue represents a method bound to a service.
type BoundMethodValue struct {
	ServiceName string
	Service     Service
	Method      string
}

func (b *BoundMethodValue) Type() string { return "bound_method" }
func (b *BoundMethodValue) String() string {
	return fmt.Sprintf("<method %s.%s>", b.ServiceName, b.Method)
}
func (b *BoundMethodValue) IsTruthy() bool { return true }

// getMethod is defined in builtins.go

func (e *Evaluator) evalLambdaExpression(node *ast.LambdaExpression) (Value, error) {
	return &LambdaValue{
		Parameters:  node.Parameters,
		Body:        node.Body,
		Env:         e.frame.Scope,
		TokenLine:   node.Token.Line,
		TokenColumn: node.Token.Column,
	}, nil
}

func (e *Evaluator) evalPipelineExpression(node *ast.PipelineExpression) (Value, error) {
	// Evaluate the left side
	left, err := e.Eval(node.Left)
	if err != nil {
		return nil, err
	}

	// The right side should be a function call
	// We need to call it with the left value as an argument
	switch r := node.Right.(type) {
	case *ast.CallExpression:
		// Evaluate the function
		fn, err := e.Eval(r.Function)
		if err != nil {
			return nil, err
		}

		// Prepend left value to arguments
		args := make([]Value, len(r.Arguments)+1)
		args[0] = left
		for i, arg := range r.Arguments {
			val, err := e.Eval(arg)
			if err != nil {
				return nil, err
			}
			args[i+1] = val
		}

		// Evaluate kwargs
		kwargs := make(map[string]Value)
		for name, expr := range r.Kwargs {
			val, err := e.Eval(expr)
			if err != nil {
				return nil, err
			}
			kwargs[name] = val
		}

		return e.callFunction(fn, args, kwargs)

	case *ast.Identifier:
		// Simple function reference - call with left as sole argument
		fn, err := e.Eval(r)
		if err != nil {
			return nil, err
		}
		return e.callFunction(fn, []Value{left}, nil)

	default:
		return nil, fmt.Errorf("invalid pipeline right-hand side: %T", node.Right)
	}
}

func (e *Evaluator) evalTernaryExpression(node *ast.TernaryExpression) (Value, error) {
	condition, err := e.Eval(node.Condition)
	if err != nil {
		return nil, err
	}

	if condition.IsTruthy() {
		return e.Eval(node.Consequence)
	}
	return e.Eval(node.Alternative)
}

func (e *Evaluator) evalIfStatement(node *ast.IfStatement) (Value, error) {
	condition, err := e.Eval(node.Condition)
	if err != nil {
		return nil, err
	}

	if condition.IsTruthy() {
		return e.evalBlock(node.Consequence)
	}

	if node.Alternative != nil {
		return e.Eval(node.Alternative)
	}

	return NONE, nil
}

func (e *Evaluator) evalForStatement(node *ast.ForStatement) (Value, error) {
	iterable, err := e.Eval(node.Iterable)
	if err != nil {
		return nil, err
	}

	// Get iterator
	iter, err := e.makeIterator(iterable)
	if err != nil {
		return nil, err
	}

	// Parse modifiers
	loopOpts := limits.LoopOptions{}
	for _, mod := range node.Modifiers {
		modVal, err := e.Eval(mod.Value)
		if err != nil {
			return nil, err
		}
		switch mod.Type {
		case "limit":
			loopOpts.Limit, _ = ToInt(modVal)
		case "rate":
			// Rate can be a float (ops/sec) or string like "10/s"
			switch v := modVal.(type) {
			case *FloatValue:
				loopOpts.Rate = v.Value
			case *IntValue:
				loopOpts.Rate = float64(v.Value)
			case *StringValue:
				rate, err := limits.ParseRate(v.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid rate: %v", err)
				}
				loopOpts.Rate = rate
			}
		case "timeout":
			// Timeout can be a duration string or integer seconds
			switch v := modVal.(type) {
			case *StringValue:
				d, err := limits.ParseDuration(v.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid timeout: %v", err)
				}
				loopOpts.Timeout = d
			case *IntValue:
				loopOpts.Timeout = time.Duration(v.Value) * time.Second
			case *FloatValue:
				loopOpts.Timeout = time.Duration(v.Value * float64(time.Second))
			}
		case "parallel":
			return nil, fmt.Errorf("parallel loop execution is not supported")
		}
	}

	// Create loop controller for rate limiting and timeout
	loopCtx := context.Background()
	lc := limits.NewLoopController(loopCtx, loopOpts)
	defer lc.Done()

	// Create scope for loop
	e.frame.PushScope()
	defer e.frame.PopScope()

	var result Value = NONE
	index := int64(0)

	for {
		// Apply loop controller (rate limiting, limit check, timeout)
		if err := lc.BeforeIteration(); err != nil {
			if err == limits.ErrLimitExceeded {
				// Limit exceeded is a normal termination condition
				break
			}
			if err == context.DeadlineExceeded {
				// Timeout is also normal termination
				break
			}
			return nil, err
		}

		// Get next value
		val, ok := iter.Next()
		if !ok {
			break
		}

		// Account only for values actually processed, so an exhausted iterator
		// does not consume one extra global iteration.
		if err := e.ctx.IncrementIterations(); err != nil {
			return nil, err
		}

		// Check wall-clock limit so compute-bound loops also honor MaxDuration.
		if err := e.ctx.CheckDuration(); err != nil {
			return nil, err
		}

		// Give closures a stable binding for this iteration rather than
		// reusing the loop scope's variables on the next iteration.
		e.frame.PushScope()
		if node.Index != nil {
			e.frame.Scope.Set(node.Index.Value, &IntValue{Value: index})
		}
		e.frame.Scope.Set(node.Variable.Value, val)

		// Execute body
		result, err = e.evalBlock(node.Body)
		e.frame.PopScope()
		if err != nil {
			return nil, err
		}

		// Handle control flow
		if e.frame.ShouldBreak() {
			e.frame.ClearBreak()
			break
		}
		if e.frame.ShouldContinue() {
			e.frame.ClearContinue()
		}
		if e.frame.ShouldReturn() || e.frame.ShouldStop() || e.frame.ShouldPause() {
			break
		}

		index++
	}

	return result, nil
}

func (e *Evaluator) makeIterator(val Value) (*IteratorValue, error) {
	switch v := val.(type) {
	case *ListValue:
		return &IteratorValue{
			Type_:   "list",
			Items:   v.Elements,
			Current: 0,
		}, nil

	case *StringValue:
		items := make([]Value, len(v.Value))
		for i, ch := range v.Value {
			items[i] = &StringValue{Value: string(ch)}
		}
		return &IteratorValue{
			Type_:   "string",
			Items:   items,
			Current: 0,
		}, nil

	case *IteratorValue:
		return v, nil

	case *MapValue:
		items := make([]Value, 0, len(v.Pairs))
		for _, key := range v.Order {
			items = append(items, &StringValue{Value: key})
		}
		return &IteratorValue{
			Type_:   "map",
			Items:   items,
			Current: 0,
		}, nil

	case *SetValue:
		items := make([]Value, 0, len(v.Elements))
		for _, elem := range v.Elements {
			items = append(items, elem)
		}
		return &IteratorValue{
			Type_:   "set",
			Items:   items,
			Current: 0,
		}, nil

	default:
		return nil, fmt.Errorf("cannot iterate over %s", val.Type())
	}
}

func (e *Evaluator) evalMatchStatement(node *ast.MatchStatement) (Value, error) {
	subject, err := e.Eval(node.Subject)
	if err != nil {
		return nil, err
	}

	for _, arm := range node.Arms {
		matches, err := e.matchPattern(arm.Pattern, subject)
		if err != nil {
			return nil, err
		}

		if matches {
			// Check guard if present
			if arm.Guard != nil {
				guard, err := e.Eval(arm.Guard)
				if err != nil {
					return nil, err
				}
				if !guard.IsTruthy() {
					continue
				}
			}

			// Execute body
			result, err := e.Eval(arm.Body)
			if err != nil {
				return nil, err
			}

			// Check for control flow keywords
			if ident, ok := arm.Body.(*ast.Identifier); ok {
				switch ident.Value {
				case "continue":
					e.frame.SetContinue()
				case "break":
					e.frame.SetBreak()
				}
			}

			return result, nil
		}
	}

	return NONE, nil
}

func (e *Evaluator) evalMatchExpression(node *ast.MatchExpression) (Value, error) {
	subject, err := e.Eval(node.Subject)
	if err != nil {
		return nil, err
	}

	for _, arm := range node.Arms {
		matches, err := e.matchPattern(arm.Pattern, subject)
		if err != nil {
			return nil, err
		}

		if matches {
			if arm.Guard != nil {
				guard, err := e.Eval(arm.Guard)
				if err != nil {
					return nil, err
				}
				if !guard.IsTruthy() {
					continue
				}
			}

			return e.Eval(arm.Body)
		}
	}

	return NONE, nil
}

func (e *Evaluator) matchPattern(pattern ast.Expression, subject Value) (bool, error) {
	switch p := pattern.(type) {
	case *ast.Identifier:
		// Wildcard pattern
		if p.Value == "_" {
			return true, nil
		}
		// Named pattern - bind value
		e.frame.Scope.Set(p.Value, subject)
		return true, nil

	case *ast.IntegerLiteral:
		if sv, ok := subject.(*IntValue); ok {
			return sv.Value == p.Value, nil
		}
		return false, nil

	case *ast.FloatLiteral:
		if sv, ok := subject.(*FloatValue); ok {
			return sv.Value == p.Value, nil
		}
		return false, nil

	case *ast.StringLiteral:
		if sv, ok := subject.(*StringValue); ok {
			return sv.Value == p.Value, nil
		}
		return false, nil

	case *ast.BooleanLiteral:
		if sv, ok := subject.(*BoolValue); ok {
			return sv.Value == p.Value, nil
		}
		return false, nil

	case *ast.NoneLiteral:
		return IsNone(subject), nil

	default:
		// Evaluate pattern and compare
		patternVal, err := e.Eval(pattern)
		if err != nil {
			return false, err
		}
		return Equal(patternVal, subject), nil
	}
}

func (e *Evaluator) evalDefStatement(node *ast.DefStatement) (Value, error) {
	fn := &FunctionValue{
		Name:       node.Name.Value,
		Parameters: node.Parameters,
		Body:       node.Body,
		Env:        e.frame.Scope,
	}

	e.frame.Scope.Set(node.Name.Value, fn)
	return fn, nil
}

func (e *Evaluator) evalReturnStatement(node *ast.ReturnStatement) (Value, error) {
	var val Value = NONE
	if node.Value != nil {
		var err error
		val, err = e.Eval(node.Value)
		if err != nil {
			return nil, err
		}
	}

	e.frame.SetReturn(val)
	return val, nil
}

func (e *Evaluator) evalEmitStatement(node *ast.EmitStatement) (Value, error) {
	// Handle positional values
	for _, expr := range node.Values {
		val, err := e.Eval(expr)
		if err != nil {
			return nil, err
		}
		e.ctx.Emit(val)
	}

	// Handle named values - emit as a map
	if len(node.Named) > 0 {
		m := NewMapValue()
		for name, expr := range node.Named {
			val, err := e.Eval(expr)
			if err != nil {
				return nil, err
			}
			m.Set(name, val)
		}
		e.ctx.Emit(m)
	}

	return NONE, nil
}

func (e *Evaluator) evalStopStatement(node *ast.StopStatement) (Value, error) {
	e.frame.SetStop(node.Rollback)
	return NONE, nil
}

func (e *Evaluator) evalPauseStatement(node *ast.PauseStatement) (Value, error) {
	var message string
	if node.Message != nil {
		msgVal, err := e.Eval(node.Message)
		if err != nil {
			return nil, err
		}
		// Use the String() method from the Value interface
		message = msgVal.String()
	}
	e.frame.SetPause(message)
	return NONE, nil
}

func (e *Evaluator) evalTryStatement(node *ast.TryStatement) (Value, error) {
	result, err := e.evalBlock(node.Body)
	if err != nil {
		// Try to match catch clauses
		for _, catch := range node.Catches {
			// TODO: Implement error type matching
			if catch.Variable != nil {
				e.frame.Scope.Set(catch.Variable.Value, &SlopError{Message: err.Error()})
			}
			// Execute catch block - if it succeeds, return its result
			if catchResult, catchErr := e.evalBlock(catch.Body); catchErr == nil {
				return catchResult, nil
			}
		}
		return nil, err
	}
	return result, nil
}

func (e *Evaluator) evalRangeExpression(node *ast.RangeExpression) (Value, error) {
	var start, end, step int64

	if node.Start != nil {
		startVal, err := e.Eval(node.Start)
		if err != nil {
			return nil, err
		}
		start, _ = ToInt(startVal)
	}

	endVal, err := e.Eval(node.End)
	if err != nil {
		return nil, err
	}
	end, _ = ToInt(endVal)

	if node.Step != nil {
		stepVal, err := e.Eval(node.Step)
		if err != nil {
			return nil, err
		}
		step, _ = ToInt(stepVal)
		if step == 0 {
			return nil, fmt.Errorf("range step cannot be zero")
		}
	} else {
		step = 1
	}

	return &IteratorValue{
		Type_:   "range",
		Current: int(start),
		End:     int(end),
		Step:    int(step),
	}, nil
}

func (e *Evaluator) evalListComprehension(node *ast.ListComprehension) (Value, error) {
	iterable, err := e.Eval(node.Iterable)
	if err != nil {
		return nil, err
	}

	iter, err := e.makeIterator(iterable)
	if err != nil {
		return nil, err
	}

	e.frame.PushScope()
	defer e.frame.PopScope()

	elements := []Value{}
	index := int64(0)

	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err := e.ctx.IncrementIterations(); err != nil {
			return nil, err
		}

		if node.Index != nil {
			e.frame.Scope.Set(node.Index.Value, &IntValue{Value: index})
		}
		e.frame.Scope.Set(node.Variable.Value, val)

		// Check filter
		if node.Filter != nil {
			filterVal, err := e.Eval(node.Filter)
			if err != nil {
				return nil, err
			}
			if !filterVal.IsTruthy() {
				index++
				continue
			}
		}

		// Evaluate element expression
		elem, err := e.Eval(node.Element)
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		index++
	}

	return &ListValue{Elements: elements}, nil
}

func (e *Evaluator) evalMapComprehension(node *ast.MapComprehension) (Value, error) {
	iterable, err := e.Eval(node.Iterable)
	if err != nil {
		return nil, err
	}

	iter, err := e.makeIterator(iterable)
	if err != nil {
		return nil, err
	}

	e.frame.PushScope()
	defer e.frame.PopScope()

	result := NewMapValue()

	for {
		val, ok := iter.Next()
		if !ok {
			break
		}
		if err := e.ctx.IncrementIterations(); err != nil {
			return nil, err
		}

		// For map comprehension, we expect pairs
		// Bind variables based on iteration type
		switch v := val.(type) {
		case *ListValue:
			if len(v.Elements) >= 2 {
				e.frame.Scope.Set(node.KeyVar.Value, v.Elements[0])
				e.frame.Scope.Set(node.ValueVar.Value, v.Elements[1])
			}
		default:
			e.frame.Scope.Set(node.KeyVar.Value, val)
			e.frame.Scope.Set(node.ValueVar.Value, val)
		}

		// Check filter
		if node.Filter != nil {
			filterVal, err := e.Eval(node.Filter)
			if err != nil {
				return nil, err
			}
			if !filterVal.IsTruthy() {
				continue
			}
		}

		// Evaluate key and value expressions
		keyVal, err := e.Eval(node.Key)
		if err != nil {
			return nil, err
		}
		valVal, err := e.Eval(node.Value)
		if err != nil {
			return nil, err
		}

		key := keyVal.String()
		if sv, ok := keyVal.(*StringValue); ok {
			key = sv.Value
		}
		result.Set(key, valVal)
	}

	return result, nil
}
