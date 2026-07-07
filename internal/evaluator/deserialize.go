package evaluator

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/standardbeagle/slop/internal/ast"
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
func DeserializeLimits(sl *SerializedLimits) *ExecutionLimits {
	if sl == nil {
		return &ExecutionLimits{}
	}
	return &ExecutionLimits{
		MaxIterations:  sl.MaxIterations,
		MaxLLMCalls:    sl.MaxLLMCalls,
		MaxAPICalls:    sl.MaxAPICalls,
		MaxDuration:    sl.MaxDuration,
		MaxCost:        sl.MaxCost,
		IterationCount: sl.IterationCount,
		LLMCallCount:   sl.LLMCallCount,
		APICallCount:   sl.APICallCount,
		StartTime:      sl.StartTime,
		TotalCost:      sl.TotalCost,
	}
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

	// Create context
	ctx := &Context{
		Scope:    current,
		Globals:  globals,
		Services: d.services,
		Limits:   DeserializeLimits(sc.Limits),
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
func LoadCheckpoint(data []byte, program *ast.Program, builtins map[string]*BuiltinValue, services map[string]Service) (*Checkpoint, *Context, error) {
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, nil, fmt.Errorf("unmarshaling checkpoint: %w", err)
	}

	// Verify version compatibility
	if checkpoint.Version != CheckpointVersion {
		return nil, nil, fmt.Errorf("checkpoint version mismatch: got %s, expected %s", checkpoint.Version, CheckpointVersion)
	}

	// Deserialize context
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
