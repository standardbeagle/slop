package evaluator

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/standardbeagle/slop/internal/ast"
)

// Checkpoint validation limits — hard caps that a legitimate checkpoint
// written by this runtime will never exceed, used to reject injected state.
const (
	// maxCheckpointIterators caps the number of items in a list iterator to
	// guard against memory exhaustion via a crafted checkpoint.
	maxCheckpointIteratorItems = 1_000_000

	// maxCheckpointCost caps the TotalCost / MaxCost fields to a value that
	// cannot silently reset a budget already consumed by a running script.
	maxCheckpointCost = 1_000_000.0
)

// Deserializer handles deserialization of execution state.
type Deserializer struct {
	scopes        map[string]*Scope
	program       *ast.Program
	builtins      map[string]*BuiltinValue
	services      map[string]Service
	funcDefs      map[string]*ast.DefStatement      // Function definitions by name
	lambdaExprs   map[string]*ast.LambdaExpression  // Lambda expressions by position key
}

// NewDeserializer creates a new deserializer.
func NewDeserializer(program *ast.Program, builtins map[string]*BuiltinValue, services map[string]Service) *Deserializer {
	d := &Deserializer{
		scopes:      make(map[string]*Scope),
		program:     program,
		builtins:    builtins,
		services:    services,
		funcDefs:    make(map[string]*ast.DefStatement),
		lambdaExprs: make(map[string]*ast.LambdaExpression),
	}

	// Index function definitions from AST
	if program != nil {
		d.indexAST(program)
	}

	return d
}

// indexAST walks the AST and indexes function/lambda definitions.
func (d *Deserializer) indexAST(program *ast.Program) {
	for _, stmt := range program.Statements {
		d.indexStatement(stmt)
	}
}

func (d *Deserializer) indexStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.DefStatement:
		d.funcDefs[s.Name.Value] = s
		// Index statements within function body
		if s.Body != nil {
			for _, bodyStmt := range s.Body.Statements {
				d.indexStatement(bodyStmt)
			}
		}
	case *ast.ExpressionStatement:
		d.indexExpression(s.Expression)
	case *ast.AssignStatement:
		d.indexExpression(s.Value)
	case *ast.IfStatement:
		d.indexExpression(s.Condition)
		if s.Consequence != nil {
			for _, bodyStmt := range s.Consequence.Statements {
				d.indexStatement(bodyStmt)
			}
		}
		if s.Alternative != nil {
			// Alternative can be IfStatement (elif) or Block (else)
			d.indexStatement(s.Alternative)
		}
	case *ast.Block:
		for _, bodyStmt := range s.Statements {
			d.indexStatement(bodyStmt)
		}
	case *ast.ForStatement:
		d.indexExpression(s.Iterable)
		if s.Body != nil {
			for _, bodyStmt := range s.Body.Statements {
				d.indexStatement(bodyStmt)
			}
		}
	case *ast.TryStatement:
		if s.Body != nil {
			for _, bodyStmt := range s.Body.Statements {
				d.indexStatement(bodyStmt)
			}
		}
		for _, catch := range s.Catches {
			if catch.Body != nil {
				for _, bodyStmt := range catch.Body.Statements {
					d.indexStatement(bodyStmt)
				}
			}
		}
	case *ast.ReturnStatement:
		if s.Value != nil {
			d.indexExpression(s.Value)
		}
	}
}

func (d *Deserializer) indexExpression(expr ast.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.LambdaExpression:
		key := fmt.Sprintf("%d:%d", e.Token.Line, e.Token.Column)
		d.lambdaExprs[key] = e
		d.indexExpression(e.Body)
	case *ast.CallExpression:
		d.indexExpression(e.Function)
		for _, arg := range e.Arguments {
			d.indexExpression(arg)
		}
	case *ast.IndexExpression:
		d.indexExpression(e.Left)
		d.indexExpression(e.Index)
	case *ast.MemberExpression:
		d.indexExpression(e.Object)
	case *ast.InfixExpression:
		d.indexExpression(e.Left)
		d.indexExpression(e.Right)
	case *ast.PrefixExpression:
		d.indexExpression(e.Right)
	case *ast.TernaryExpression:
		d.indexExpression(e.Condition)
		d.indexExpression(e.Consequence)
		d.indexExpression(e.Alternative)
	case *ast.ListLiteral:
		for _, elem := range e.Elements {
			d.indexExpression(elem)
		}
	case *ast.MapLiteral:
		// MapLiteral.Pairs is map[Expression]Expression
		for k, v := range e.Pairs {
			d.indexExpression(k)
			d.indexExpression(v)
		}
	}
}

// DeserializeValue converts a SerializedValue back to a Value.
func (d *Deserializer) DeserializeValue(sv *SerializedValue) (Value, error) {
	if sv == nil {
		return NONE, nil
	}

	switch sv.Type {
	case "none":
		return NONE, nil

	case "bool":
		var b bool
		if err := json.Unmarshal(sv.Data, &b); err != nil {
			return nil, fmt.Errorf("deserializing bool: %w", err)
		}
		return NewBool(b), nil

	case "int":
		var s string
		if err := json.Unmarshal(sv.Data, &s); err != nil {
			return nil, fmt.Errorf("deserializing int string: %w", err)
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing int: %w", err)
		}
		return &IntValue{Value: i}, nil

	case "float":
		var f float64
		if err := json.Unmarshal(sv.Data, &f); err != nil {
			return nil, fmt.Errorf("deserializing float: %w", err)
		}
		return &FloatValue{Value: f}, nil

	case "string":
		var s string
		if err := json.Unmarshal(sv.Data, &s); err != nil {
			return nil, fmt.Errorf("deserializing string: %w", err)
		}
		return &StringValue{Value: s}, nil

	case "list":
		return d.deserializeList(sv.Data)

	case "map":
		return d.deserializeMap(sv.Data)

	case "set":
		return d.deserializeSet(sv.Data)

	case "function":
		return d.deserializeFunction(sv.Data)

	case "lambda":
		return d.deserializeLambda(sv.Data)

	case "builtin":
		var name string
		if err := json.Unmarshal(sv.Data, &name); err != nil {
			return nil, fmt.Errorf("deserializing builtin name: %w", err)
		}
		if b, ok := d.builtins[name]; ok {
			return b, nil
		}
		return nil, fmt.Errorf("unknown builtin: %s", name)

	case "service":
		var name string
		if err := json.Unmarshal(sv.Data, &name); err != nil {
			return nil, fmt.Errorf("deserializing service name: %w", err)
		}
		if svc, ok := d.services[name]; ok {
			return &ServiceValue{Name: name, Service: svc}, nil
		}
		// Service not available - create placeholder
		return &ServiceValue{Name: name, Service: nil}, nil

	case "error":
		return d.deserializeError(sv.Data)

	case "iterator":
		return d.deserializeIterator(sv.Data)

	default:
		return nil, fmt.Errorf("unknown value type: %s", sv.Type)
	}
}

func (d *Deserializer) deserializeList(data json.RawMessage) (*ListValue, error) {
	var elements []*SerializedValue
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("deserializing list elements: %w", err)
	}

	list := &ListValue{Elements: make([]Value, len(elements))}
	for i, elem := range elements {
		val, err := d.DeserializeValue(elem)
		if err != nil {
			return nil, fmt.Errorf("deserializing list element %d: %w", i, err)
		}
		list.Elements[i] = val
	}
	return list, nil
}

func (d *Deserializer) deserializeMap(data json.RawMessage) (*MapValue, error) {
	var sm SerializedMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("deserializing map: %w", err)
	}

	m := NewMapValue()
	// Restore in order
	for _, key := range sm.Order {
		sv, ok := sm.Pairs[key]
		if !ok {
			continue
		}
		val, err := d.DeserializeValue(sv)
		if err != nil {
			return nil, fmt.Errorf("deserializing map value for %q: %w", key, err)
		}
		m.Set(key, val)
	}
	return m, nil
}

func (d *Deserializer) deserializeSet(data json.RawMessage) (*SetValue, error) {
	var elements []*SerializedValue
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("deserializing set elements: %w", err)
	}

	set := NewSetValue()
	for _, elem := range elements {
		val, err := d.DeserializeValue(elem)
		if err != nil {
			return nil, fmt.Errorf("deserializing set element: %w", err)
		}
		set.Add(val)
	}
	return set, nil
}

func (d *Deserializer) deserializeFunction(data json.RawMessage) (*FunctionValue, error) {
	var ref FunctionRef
	if err := json.Unmarshal(data, &ref); err != nil {
		return nil, fmt.Errorf("deserializing function ref: %w", err)
	}

	// Look up function definition in AST
	defStmt, ok := d.funcDefs[ref.Name]
	if !ok {
		return nil, fmt.Errorf("function definition not found: %s", ref.Name)
	}

	f := &FunctionValue{
		Name:       ref.Name,
		Parameters: defStmt.Parameters,
		Body:       defStmt.Body,
	}

	// Restore closure scope reference
	if ref.ClosureScopeID != nil {
		if scope, ok := d.scopes[*ref.ClosureScopeID]; ok {
			f.Env = scope
		}
	}

	return f, nil
}

func (d *Deserializer) deserializeLambda(data json.RawMessage) (*LambdaValue, error) {
	var ref LambdaRef
	if err := json.Unmarshal(data, &ref); err != nil {
		return nil, fmt.Errorf("deserializing lambda ref: %w", err)
	}

	// Try to look up lambda in AST by position
	var lambdaExpr *ast.LambdaExpression
	if ref.Position != nil {
		key := fmt.Sprintf("%d:%d", ref.Position.Line, ref.Position.Column)
		lambdaExpr = d.lambdaExprs[key]
	}

	l := &LambdaValue{}

	if lambdaExpr != nil {
		l.Parameters = lambdaExpr.Parameters
		l.Body = lambdaExpr.Body
	} else {
		// Create placeholder parameters from names
		for _, name := range ref.ParameterNames {
			l.Parameters = append(l.Parameters, &ast.Identifier{Value: name})
		}
	}

	// Restore closure scope reference
	if ref.ClosureScopeID != nil {
		if scope, ok := d.scopes[*ref.ClosureScopeID]; ok {
			l.Env = scope
		}
	}

	return l, nil
}

func (d *Deserializer) deserializeError(data json.RawMessage) (*SlopError, error) {
	var se SerializedError
	if err := json.Unmarshal(data, &se); err != nil {
		return nil, fmt.Errorf("deserializing error: %w", err)
	}

	e := &SlopError{Message: se.Message}
	if se.Data != nil {
		val, err := d.DeserializeValue(se.Data)
		if err != nil {
			return nil, fmt.Errorf("deserializing error data: %w", err)
		}
		e.Data = val
	}
	return e, nil
}

func (d *Deserializer) deserializeIterator(data json.RawMessage) (*IteratorValue, error) {
	var si SerializedIterator
	if err := json.Unmarshal(data, &si); err != nil {
		return nil, fmt.Errorf("deserializing iterator: %w", err)
	}

	// Validate iterator type.
	if si.IterType != "range" && si.IterType != "list" {
		return nil, fmt.Errorf("invalid iterator type: %q", si.IterType)
	}

	// Bounds-check range iterator fields to prevent injected giant / negative
	// step values that would cause the evaluator to loop for an unbounded time
	// or produce nonsense results.
	if si.IterType == "range" {
		// Step = 0 produces an infinite loop; negative step with Current<End
		// (or vice versa) is also pathological.
		if si.Step == 0 {
			return nil, fmt.Errorf("invalid range iterator: step must not be zero")
		}
		// Current must be on the correct side of End for the step direction.
		if si.Step > 0 && si.Current > si.End {
			return nil, fmt.Errorf("invalid range iterator: current (%d) > end (%d) with positive step", si.Current, si.End)
		}
		if si.Step < 0 && si.Current < si.End {
			return nil, fmt.Errorf("invalid range iterator: current (%d) < end (%d) with negative step", si.Current, si.End)
		}
		// Guard against absurdly large iteration counts that would block the host.
		// Use int64 arithmetic; guard Step == math.MinInt32 before negating to
		// avoid overflow (SerializedIterator uses int, which is at least 32 bits).
		if si.Step != 0 {
			absStep := int64(si.Step)
			if absStep < 0 {
				absStep = -absStep
			}
			if absStep == 0 {
				// Overflow guard: MinInt negated wraps back to MinInt on 64-bit;
				// treat as an invalid step.
				return nil, fmt.Errorf("invalid range iterator: step value overflows")
			}
			count := int64(0)
			if si.Step > 0 {
				count = (int64(si.End) - int64(si.Current) + absStep - 1) / absStep
			} else {
				count = (int64(si.Current) - int64(si.End) + absStep - 1) / absStep
			}
			if count > int64(maxCheckpointIteratorItems) {
				return nil, fmt.Errorf("invalid range iterator: iteration count %d exceeds limit %d", count, maxCheckpointIteratorItems)
			}
		}
	}

	// Bounds-check list iterator.
	if si.IterType == "list" {
		if si.Current < 0 {
			return nil, fmt.Errorf("invalid list iterator: current (%d) must not be negative", si.Current)
		}
		if len(si.Items) > maxCheckpointIteratorItems {
			return nil, fmt.Errorf("invalid list iterator: item count %d exceeds limit %d", len(si.Items), maxCheckpointIteratorItems)
		}
		if si.Current > len(si.Items) {
			return nil, fmt.Errorf("invalid list iterator: current (%d) exceeds item count (%d)", si.Current, len(si.Items))
		}
	}

	iter := &IteratorValue{
		Type_:   si.IterType,
		Current: si.Current,
		End:     si.End,
		Step:    si.Step,
	}

	if si.IterType == "list" && len(si.Items) > 0 {
		iter.Items = make([]Value, len(si.Items))
		for i, item := range si.Items {
			val, err := d.DeserializeValue(item)
			if err != nil {
				return nil, fmt.Errorf("deserializing iterator item %d: %w", i, err)
			}
			iter.Items[i] = val
		}
	}

	return iter, nil
}

// DeserializeScope converts a SerializedScope back to a Scope.
func (d *Deserializer) DeserializeScope(ss *SerializedScope) (*Scope, error) {
	// Check if already deserialized
	if scope, ok := d.scopes[ss.ID]; ok {
		return scope, nil
	}

	scope := NewScope()
	d.scopes[ss.ID] = scope

	// Deserialize variables
	for name, sv := range ss.Variables {
		val, err := d.DeserializeValue(sv)
		if err != nil {
			return nil, fmt.Errorf("deserializing variable %q: %w", name, err)
		}
		scope.Set(name, val)
	}

	return scope, nil
}

// DeserializeScopeChain reconstructs the scope chain.
func (d *Deserializer) DeserializeScopeChain(scopes []*SerializedScope, currentID string) (*Scope, *Scope, error) {
	// First pass: create all scopes
	for _, ss := range scopes {
		if _, err := d.DeserializeScope(ss); err != nil {
			return nil, nil, err
		}
	}

	// Second pass: link parent relationships
	var globals *Scope
	for _, ss := range scopes {
		scope := d.scopes[ss.ID]
		if ss.ParentID != nil {
			if parent, ok := d.scopes[*ss.ParentID]; ok {
				scope.parent = parent
			}
		}
		if ss.IsGlobal {
			globals = scope
		}
	}

	current, ok := d.scopes[currentID]
	if !ok {
		return nil, nil, fmt.Errorf("current scope not found: %s", currentID)
	}

	return current, globals, nil
}

// DeserializeLimits converts SerializedLimits back to ExecutionLimits.
// It rejects negative counters and counter values that exceed their
// corresponding maximums (which would effectively disable a limit that was
// already partially consumed), as well as out-of-range cost fields.
func DeserializeLimits(sl *SerializedLimits) (*ExecutionLimits, error) {
	if sl == nil {
		return &ExecutionLimits{}, nil
	}

	// Reject negative counter values — a legitimate checkpoint never produces
	// these, and injecting them resets budget consumption to below zero.
	if sl.IterationCount < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative iteration_count (%d)", sl.IterationCount)
	}
	if sl.LLMCallCount < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative llm_call_count (%d)", sl.LLMCallCount)
	}
	if sl.APICallCount < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative api_call_count (%d)", sl.APICallCount)
	}
	if sl.CallDepth < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative call_depth (%d)", sl.CallDepth)
	}
	if sl.TotalCost < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative total_cost (%v)", sl.TotalCost)
	}

	// Reject non-finite cost fields.
	if math.IsNaN(sl.TotalCost) || math.IsInf(sl.TotalCost, 0) {
		return nil, fmt.Errorf("invalid checkpoint: non-finite total_cost (%v)", sl.TotalCost)
	}
	if math.IsNaN(sl.MaxCost) || math.IsInf(sl.MaxCost, 0) {
		return nil, fmt.Errorf("invalid checkpoint: non-finite max_cost (%v)", sl.MaxCost)
	}

	// Reject implausibly large cost values.
	if sl.TotalCost > maxCheckpointCost {
		return nil, fmt.Errorf("invalid checkpoint: total_cost (%v) exceeds limit (%v)", sl.TotalCost, maxCheckpointCost)
	}
	if sl.MaxCost < 0 {
		return nil, fmt.Errorf("invalid checkpoint: negative max_cost (%v)", sl.MaxCost)
	}

	// If a maximum is set, the counter must not exceed it — resuming a script
	// that reports 1 000 000 of 10 iterations consumed would bypass the limit
	// on the very first operation after resume.
	if sl.MaxIterations > 0 && sl.IterationCount > sl.MaxIterations {
		return nil, fmt.Errorf("invalid checkpoint: iteration_count (%d) exceeds max_iterations (%d)", sl.IterationCount, sl.MaxIterations)
	}
	if sl.MaxLLMCalls > 0 && sl.LLMCallCount > sl.MaxLLMCalls {
		return nil, fmt.Errorf("invalid checkpoint: llm_call_count (%d) exceeds max_llm_calls (%d)", sl.LLMCallCount, sl.MaxLLMCalls)
	}
	if sl.MaxAPICalls > 0 && sl.APICallCount > sl.MaxAPICalls {
		return nil, fmt.Errorf("invalid checkpoint: api_call_count (%d) exceeds max_api_calls (%d)", sl.APICallCount, sl.MaxAPICalls)
	}
	if sl.MaxCost > 0 && sl.TotalCost > sl.MaxCost {
		return nil, fmt.Errorf("invalid checkpoint: total_cost (%v) exceeds max_cost (%v)", sl.TotalCost, sl.MaxCost)
	}
	// Call depth must not exceed the effective maximum.
	effectiveMaxCallDepth := sl.MaxCallDepth
	if effectiveMaxCallDepth <= 0 {
		effectiveMaxCallDepth = DefaultMaxCallDepth
	}
	if sl.CallDepth > effectiveMaxCallDepth {
		return nil, fmt.Errorf("invalid checkpoint: call_depth (%d) exceeds max_call_depth (%d)", sl.CallDepth, effectiveMaxCallDepth)
	}

	return &ExecutionLimits{
		MaxIterations:  sl.MaxIterations,
		MaxLLMCalls:    sl.MaxLLMCalls,
		MaxAPICalls:    sl.MaxAPICalls,
		MaxDuration:    sl.MaxDuration,
		MaxCost:        sl.MaxCost,
		MaxCallDepth:   sl.MaxCallDepth,
		IterationCount: sl.IterationCount,
		LLMCallCount:   sl.LLMCallCount,
		APICallCount:   sl.APICallCount,
		StartTime:      sl.StartTime,
		TotalCost:      sl.TotalCost,
		CallDepth:      sl.CallDepth,
	}, nil
}

// DeserializeTransactionLog converts SerializedTxLog back to TransactionLog.
func (d *Deserializer) DeserializeTransactionLog(st *SerializedTxLog) (*TransactionLog, error) {
	if st == nil {
		return NewTransactionLog(), nil
	}

	log := &TransactionLog{
		nextID:     st.NextID,
		Operations: make([]Operation, len(st.Operations)),
	}

	for i, sop := range st.Operations {
		op := Operation{
			ID:         sop.ID,
			Timestamp:  sop.Timestamp,
			Type:       sop.Type,
			Service:    sop.Service,
			Method:     sop.Method,
			Reversible: sop.Reversible,
			UndoMethod: sop.UndoMethod,
		}

		// Deserialize args
		if len(sop.Args) > 0 {
			op.Args = make([]Value, len(sop.Args))
			for j, arg := range sop.Args {
				val, err := d.DeserializeValue(arg)
				if err != nil {
					return nil, fmt.Errorf("deserializing operation %d arg %d: %w", i, j, err)
				}
				op.Args[j] = val
			}
		}

		// Deserialize kwargs
		if len(sop.Kwargs) > 0 {
			op.Kwargs = make(map[string]Value)
			for k, v := range sop.Kwargs {
				val, err := d.DeserializeValue(v)
				if err != nil {
					return nil, fmt.Errorf("deserializing operation %d kwarg %q: %w", i, k, err)
				}
				op.Kwargs[k] = val
			}
		}

		// Deserialize result
		if sop.Result != nil {
			val, err := d.DeserializeValue(sop.Result)
			if err != nil {
				return nil, fmt.Errorf("deserializing operation %d result: %w", i, err)
			}
			op.Result = val
		}

		// Deserialize error
		if sop.Error != nil {
			op.Error = fmt.Errorf("%s", *sop.Error)
		}

		// Deserialize undo data
		if len(sop.UndoData) > 0 {
			op.UndoData = make(map[string]Value)
			for k, v := range sop.UndoData {
				val, err := d.DeserializeValue(v)
				if err != nil {
					return nil, fmt.Errorf("deserializing operation %d undo data %q: %w", i, k, err)
				}
				op.UndoData[k] = val
			}
		}

		log.Operations[i] = op
	}

	return log, nil
}

// DeserializeContext reconstructs a Context from SerializedContext.
func (d *Deserializer) DeserializeContext(sc *SerializedContext) (*Context, error) {
	// Deserialize scope chain
	current, globals, err := d.DeserializeScopeChain(sc.Scopes, sc.CurrentScopeID)
	if err != nil {
		return nil, fmt.Errorf("deserializing scope chain: %w", err)
	}

	// Deserialize transaction log
	txLog, err := d.DeserializeTransactionLog(sc.TxLog)
	if err != nil {
		return nil, fmt.Errorf("deserializing transaction log: %w", err)
	}

	// Deserialize emitted values
	emitted := make([]Value, len(sc.Emitted))
	for i, sv := range sc.Emitted {
		val, err := d.DeserializeValue(sv)
		if err != nil {
			return nil, fmt.Errorf("deserializing emitted value %d: %w", i, err)
		}
		emitted[i] = val
	}

	// Deserialize execution limits with bounds validation.
	limits, err := DeserializeLimits(sc.Limits)
	if err != nil {
		return nil, fmt.Errorf("deserializing limits: %w", err)
	}

	// Create context
	ctx := &Context{
		Scope:    current,
		Globals:  globals,
		Services: d.services,
		Limits:   limits,
		TxLog:    txLog,
		Emitted:  emitted,
	}

	// Restore control flow state
	if sc.ControlFlow != nil {
		ctx.shouldReturn = sc.ControlFlow.ShouldReturn
		ctx.shouldBreak = sc.ControlFlow.ShouldBreak
		ctx.shouldContinue = sc.ControlFlow.ShouldContinue
		ctx.shouldStop = sc.ControlFlow.ShouldStop
		ctx.rollback = sc.ControlFlow.Rollback

		if sc.ControlFlow.ReturnValue != nil {
			rv, err := d.DeserializeValue(sc.ControlFlow.ReturnValue)
			if err != nil {
				return nil, fmt.Errorf("deserializing return value: %w", err)
			}
			ctx.returnValue = rv
		}
	}

	// Register builtins in globals
	for name, builtin := range d.builtins {
		ctx.Globals.Set(name, builtin)
	}

	// Register services in globals
	for name, svc := range d.services {
		ctx.Globals.Set(name, &ServiceValue{Name: name, Service: svc})
	}

	return ctx, nil
}

// LoadCheckpoint loads and deserializes a checkpoint.
// It performs structural validation before restoring any mutable state so that
// a tampered or malformed checkpoint file cannot inject out-of-bounds values
// into the execution context.
func LoadCheckpoint(data []byte, program *ast.Program, builtins map[string]*BuiltinValue, services map[string]Service) (*Checkpoint, *Context, error) {
	// Reject empty input before attempting any parse.
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("checkpoint data is empty")
	}

	// Validate that the top-level structure is a JSON object and extract the
	// required fields without first unmarshaling into the full Checkpoint type
	// (which would silently ignore unknown keys and accept out-of-range values).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("checkpoint is not a JSON object: %w", err)
	}

	// Require the mandatory top-level fields defined in the schema.
	for _, required := range []string{"version", "script_hash", "position", "context", "created_at"} {
		if _, ok := raw[required]; !ok {
			return nil, nil, fmt.Errorf("checkpoint missing required field: %q", required)
		}
	}

	// Validate version before doing any further work.
	var version string
	if err := json.Unmarshal(raw["version"], &version); err != nil {
		return nil, nil, fmt.Errorf("checkpoint version is not a string: %w", err)
	}
	if version != CheckpointVersion {
		return nil, nil, fmt.Errorf("checkpoint version mismatch: got %s, expected %s", version, CheckpointVersion)
	}

	// Validate script_hash format: must be a 64-char lowercase hex string.
	var scriptHash string
	if err := json.Unmarshal(raw["script_hash"], &scriptHash); err != nil {
		return nil, nil, fmt.Errorf("checkpoint script_hash is not a string: %w", err)
	}
	if len(scriptHash) != 64 {
		return nil, nil, fmt.Errorf("checkpoint script_hash has wrong length: got %d, want 64", len(scriptHash))
	}
	for _, c := range scriptHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return nil, nil, fmt.Errorf("checkpoint script_hash contains non-hex character: %q", c)
		}
	}

	// Now unmarshal the full structure — post-structural-validation, so all
	// further field-level validation (limits bounds, iterator bounds) will
	// execute inside DeserializeContext / DeserializeLimits.
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling checkpoint: %w", err)
	}

	// Deserialize context (includes limits + iterator bounds validation).
	deserializer := NewDeserializer(program, builtins, services)
	ctx, err := deserializer.DeserializeContext(checkpoint.Context)
	if err != nil {
		return nil, nil, fmt.Errorf("deserializing context: %w", err)
	}

	return &checkpoint, ctx, nil
}

// SaveCheckpoint serializes and saves a checkpoint to bytes.
func SaveCheckpoint(checkpoint *Checkpoint) ([]byte, error) {
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling checkpoint: %w", err)
	}
	return data, nil
}
