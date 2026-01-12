package evaluator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/standardbeagle/slop/internal/ast"
)

// CheckpointVersion is the current checkpoint format version.
const CheckpointVersion = "1.0"

// Checkpoint represents a serializable execution state.
type Checkpoint struct {
	Version           string             `json:"version"`
	Script            string             `json:"script"`
	ScriptHash        string             `json:"script_hash"`
	ScriptPath        string             `json:"script_path,omitempty"`
	Position          Position           `json:"position"`
	Context           *SerializedContext `json:"context"`
	ServiceRefs       []ServiceRef       `json:"service_refs,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	CheckpointName    string             `json:"checkpoint_name,omitempty"`
	CheckpointMessage string             `json:"checkpoint_message,omitempty"`
}

// Position represents a location in source code.
type Position struct {
	Line           int `json:"line"`
	Column         int `json:"column"`
	StatementIndex int `json:"statement_index,omitempty"`
}

// SerializedContext represents the execution context in serializable form.
type SerializedContext struct {
	Scopes         []*SerializedScope  `json:"scopes"`
	CurrentScopeID string              `json:"current_scope_id"`
	Limits         *SerializedLimits   `json:"limits"`
	TxLog          *SerializedTxLog    `json:"tx_log,omitempty"`
	Emitted        []*SerializedValue  `json:"emitted"`
	ControlFlow    *ControlFlowState   `json:"control_flow"`
}

// SerializedScope represents a scope in serializable form.
type SerializedScope struct {
	ID        string                      `json:"id"`
	ParentID  *string                     `json:"parent_id,omitempty"`
	Variables map[string]*SerializedValue `json:"variables"`
	IsGlobal  bool                        `json:"is_global,omitempty"`
}

// SerializedValue represents any Value in serializable form.
type SerializedValue struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// SerializedMap preserves insertion order for maps.
type SerializedMap struct {
	Pairs map[string]*SerializedValue `json:"pairs"`
	Order []string                    `json:"order"`
}

// FunctionRef references a function definition.
type FunctionRef struct {
	Name           string         `json:"name"`
	Position       *Position      `json:"position,omitempty"`
	ClosureScopeID *string        `json:"closure_scope_id,omitempty"`
	Parameters     []ParameterDef `json:"parameters,omitempty"`
}

// ParameterDef describes a function parameter.
type ParameterDef struct {
	Name       string `json:"name"`
	HasDefault bool   `json:"has_default,omitempty"`
	IsVariadic bool   `json:"is_variadic,omitempty"`
	IsKwargs   bool   `json:"is_kwargs,omitempty"`
}

// LambdaRef references a lambda definition.
type LambdaRef struct {
	Position       *Position `json:"position,omitempty"`
	ClosureScopeID *string   `json:"closure_scope_id,omitempty"`
	ParameterNames []string  `json:"parameter_names,omitempty"`
}

// SerializedError represents an error value.
type SerializedError struct {
	Message string           `json:"message"`
	Data    *SerializedValue `json:"data,omitempty"`
}

// SerializedIterator represents an iterator.
type SerializedIterator struct {
	IterType string             `json:"iter_type"`
	Current  int                `json:"current"`
	End      int                `json:"end,omitempty"`
	Step     int                `json:"step,omitempty"`
	Items    []*SerializedValue `json:"items,omitempty"`
}

// SerializedLimits represents execution limits.
type SerializedLimits struct {
	MaxIterations  int64   `json:"max_iterations,omitempty"`
	MaxLLMCalls    int64   `json:"max_llm_calls,omitempty"`
	MaxAPICalls    int64   `json:"max_api_calls,omitempty"`
	MaxDuration    int64   `json:"max_duration,omitempty"`
	MaxCost        float64 `json:"max_cost,omitempty"`
	IterationCount int64   `json:"iteration_count"`
	LLMCallCount   int64   `json:"llm_call_count"`
	APICallCount   int64   `json:"api_call_count"`
	StartTime      int64   `json:"start_time"`
	TotalCost      float64 `json:"total_cost"`
}

// SerializedTxLog represents the transaction log.
type SerializedTxLog struct {
	Operations []*SerializedOperation `json:"operations"`
	NextID     int64                  `json:"next_id"`
}

// SerializedOperation represents a single operation in the transaction log.
type SerializedOperation struct {
	ID         int64                       `json:"id"`
	Timestamp  int64                       `json:"timestamp"`
	Type       string                      `json:"type"`
	Service    string                      `json:"service"`
	Method     string                      `json:"method"`
	Args       []*SerializedValue          `json:"args,omitempty"`
	Kwargs     map[string]*SerializedValue `json:"kwargs,omitempty"`
	Result     *SerializedValue            `json:"result,omitempty"`
	Error      *string                     `json:"error,omitempty"`
	Reversible bool                        `json:"reversible,omitempty"`
	UndoMethod string                      `json:"undo_method,omitempty"`
	UndoData   map[string]*SerializedValue `json:"undo_data,omitempty"`
}

// ControlFlowState represents control flow flags.
type ControlFlowState struct {
	ReturnValue    *SerializedValue `json:"return_value,omitempty"`
	ShouldReturn   bool             `json:"should_return,omitempty"`
	ShouldBreak    bool             `json:"should_break,omitempty"`
	ShouldContinue bool             `json:"should_continue,omitempty"`
	ShouldStop     bool             `json:"should_stop,omitempty"`
	Rollback       bool             `json:"rollback,omitempty"`
}

// ServiceRef describes how to reconnect to a service.
type ServiceRef struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	EnvKeys   []string          `json:"env_keys,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

// Serializer handles serialization of execution state.
type Serializer struct {
	scopeIDs   map[*Scope]string
	nextID     int
	program    *ast.Program // Original AST for function/lambda position lookup
}

// NewSerializer creates a new serializer.
func NewSerializer(program *ast.Program) *Serializer {
	return &Serializer{
		scopeIDs: make(map[*Scope]string),
		program:  program,
	}
}

// SerializeValue converts a Value to SerializedValue.
func (s *Serializer) SerializeValue(v Value) (*SerializedValue, error) {
	if v == nil {
		return &SerializedValue{Type: "none", Data: json.RawMessage("null")}, nil
	}

	switch val := v.(type) {
	case *NoneValue:
		return &SerializedValue{Type: "none", Data: json.RawMessage("null")}, nil

	case *BoolValue:
		data, _ := json.Marshal(val.Value)
		return &SerializedValue{Type: "bool", Data: data}, nil

	case *IntValue:
		// Use string to preserve int64 precision
		data, _ := json.Marshal(strconv.FormatInt(val.Value, 10))
		return &SerializedValue{Type: "int", Data: data}, nil

	case *FloatValue:
		data, _ := json.Marshal(val.Value)
		return &SerializedValue{Type: "float", Data: data}, nil

	case *StringValue:
		data, _ := json.Marshal(val.Value)
		return &SerializedValue{Type: "string", Data: data}, nil

	case *ListValue:
		return s.serializeList(val)

	case *MapValue:
		return s.serializeMap(val)

	case *SetValue:
		return s.serializeSet(val)

	case *FunctionValue:
		return s.serializeFunction(val)

	case *LambdaValue:
		return s.serializeLambda(val)

	case *BuiltinValue:
		data, _ := json.Marshal(val.Name)
		return &SerializedValue{Type: "builtin", Data: data}, nil

	case *ServiceValue:
		data, _ := json.Marshal(val.Name)
		return &SerializedValue{Type: "service", Data: data}, nil

	case *SlopError:
		return s.serializeError(val)

	case *IteratorValue:
		return s.serializeIterator(val)

	default:
		return nil, fmt.Errorf("unsupported value type: %T", v)
	}
}

func (s *Serializer) serializeList(l *ListValue) (*SerializedValue, error) {
	elements := make([]*SerializedValue, len(l.Elements))
	for i, elem := range l.Elements {
		ser, err := s.SerializeValue(elem)
		if err != nil {
			return nil, fmt.Errorf("serializing list element %d: %w", i, err)
		}
		elements[i] = ser
	}
	data, _ := json.Marshal(elements)
	return &SerializedValue{Type: "list", Data: data}, nil
}

func (s *Serializer) serializeMap(m *MapValue) (*SerializedValue, error) {
	pairs := make(map[string]*SerializedValue, len(m.Pairs))
	for key, val := range m.Pairs {
		ser, err := s.SerializeValue(val)
		if err != nil {
			return nil, fmt.Errorf("serializing map value for key %q: %w", key, err)
		}
		pairs[key] = ser
	}
	sm := &SerializedMap{
		Pairs: pairs,
		Order: m.Order,
	}
	data, _ := json.Marshal(sm)
	return &SerializedValue{Type: "map", Data: data}, nil
}

func (s *Serializer) serializeSet(set *SetValue) (*SerializedValue, error) {
	elements := make([]*SerializedValue, 0, len(set.Elements))
	for _, val := range set.Elements {
		ser, err := s.SerializeValue(val)
		if err != nil {
			return nil, fmt.Errorf("serializing set element: %w", err)
		}
		elements = append(elements, ser)
	}
	data, _ := json.Marshal(elements)
	return &SerializedValue{Type: "set", Data: data}, nil
}

func (s *Serializer) serializeFunction(f *FunctionValue) (*SerializedValue, error) {
	ref := FunctionRef{
		Name: f.Name,
	}

	// Extract position from AST if available
	if f.Body != nil && f.Body.Pos.Line > 0 {
		ref.Position = &Position{
			Line:   f.Body.Pos.Line,
			Column: f.Body.Pos.Column,
		}
	}

	// Reference closure scope
	if f.Env != nil {
		scopeID := s.getScopeID(f.Env)
		ref.ClosureScopeID = &scopeID
	}

	// Serialize parameter definitions
	for _, p := range f.Parameters {
		param := ParameterDef{Name: p.Name.Value}
		if p.Default != nil {
			param.HasDefault = true
		}
		// Note: SLOP doesn't have *args/**kwargs syntax in parameters yet
		ref.Parameters = append(ref.Parameters, param)
	}

	data, _ := json.Marshal(ref)
	return &SerializedValue{Type: "function", Data: data}, nil
}

func (s *Serializer) serializeLambda(l *LambdaValue) (*SerializedValue, error) {
	ref := LambdaRef{}

	// Reference closure scope
	if l.Env != nil {
		scopeID := s.getScopeID(l.Env)
		ref.ClosureScopeID = &scopeID
	}

	// Extract parameter names
	for _, p := range l.Parameters {
		ref.ParameterNames = append(ref.ParameterNames, p.Value)
	}

	data, _ := json.Marshal(ref)
	return &SerializedValue{Type: "lambda", Data: data}, nil
}

func (s *Serializer) serializeError(e *SlopError) (*SerializedValue, error) {
	se := SerializedError{Message: e.Message}
	if e.Data != nil {
		data, err := s.SerializeValue(e.Data)
		if err != nil {
			return nil, fmt.Errorf("serializing error data: %w", err)
		}
		se.Data = data
	}
	data, _ := json.Marshal(se)
	return &SerializedValue{Type: "error", Data: data}, nil
}

func (s *Serializer) serializeIterator(i *IteratorValue) (*SerializedValue, error) {
	si := SerializedIterator{
		Current: i.Current,
		End:     i.End,
		Step:    i.Step,
	}

	if i.Items != nil {
		si.IterType = "list"
		for _, item := range i.Items {
			ser, err := s.SerializeValue(item)
			if err != nil {
				return nil, fmt.Errorf("serializing iterator item: %w", err)
			}
			si.Items = append(si.Items, ser)
		}
	} else {
		si.IterType = "range"
	}

	data, _ := json.Marshal(si)
	return &SerializedValue{Type: "iterator", Data: data}, nil
}

// getScopeID returns or generates a unique ID for a scope.
func (s *Serializer) getScopeID(scope *Scope) string {
	if id, ok := s.scopeIDs[scope]; ok {
		return id
	}
	s.nextID++
	id := fmt.Sprintf("scope_%d", s.nextID)
	s.scopeIDs[scope] = id
	return id
}

// SerializeScope converts a Scope to SerializedScope.
func (s *Serializer) SerializeScope(scope *Scope, isGlobal bool) (*SerializedScope, error) {
	scope.mu.RLock()
	defer scope.mu.RUnlock()

	ss := &SerializedScope{
		ID:        s.getScopeID(scope),
		Variables: make(map[string]*SerializedValue),
		IsGlobal:  isGlobal,
	}

	if scope.parent != nil {
		parentID := s.getScopeID(scope.parent)
		ss.ParentID = &parentID
	}

	for name, val := range scope.store {
		// Skip builtins and services in serialization (they're restored on load)
		if _, ok := val.(*BuiltinValue); ok {
			continue
		}
		if _, ok := val.(*ServiceValue); ok {
			continue
		}

		ser, err := s.SerializeValue(val)
		if err != nil {
			return nil, fmt.Errorf("serializing variable %q: %w", name, err)
		}
		ss.Variables[name] = ser
	}

	return ss, nil
}

// SerializeScopeChain serializes the entire scope chain.
func (s *Serializer) SerializeScopeChain(current *Scope, globals *Scope) ([]*SerializedScope, string, error) {
	var scopes []*SerializedScope
	visited := make(map[*Scope]bool)

	// Collect all scopes from current to root
	var chain []*Scope
	for scope := current; scope != nil; scope = scope.parent {
		if visited[scope] {
			break // Avoid cycles
		}
		visited[scope] = true
		chain = append(chain, scope)
	}

	// Serialize in reverse order (root first)
	for i := len(chain) - 1; i >= 0; i-- {
		scope := chain[i]
		isGlobal := scope == globals
		ss, err := s.SerializeScope(scope, isGlobal)
		if err != nil {
			return nil, "", err
		}
		scopes = append(scopes, ss)
	}

	currentID := s.getScopeID(current)
	return scopes, currentID, nil
}

// SerializeLimits converts ExecutionLimits to SerializedLimits.
func SerializeLimits(l *ExecutionLimits) *SerializedLimits {
	if l == nil {
		return &SerializedLimits{}
	}
	return &SerializedLimits{
		MaxIterations:  l.MaxIterations,
		MaxLLMCalls:    l.MaxLLMCalls,
		MaxAPICalls:    l.MaxAPICalls,
		MaxDuration:    l.MaxDuration,
		MaxCost:        l.MaxCost,
		IterationCount: l.IterationCount,
		LLMCallCount:   l.LLMCallCount,
		APICallCount:   l.APICallCount,
		StartTime:      l.StartTime,
		TotalCost:      l.TotalCost,
	}
}

// SerializeTransactionLog converts TransactionLog to SerializedTxLog.
func (s *Serializer) SerializeTransactionLog(log *TransactionLog) (*SerializedTxLog, error) {
	if log == nil {
		return nil, nil
	}

	log.mu.Lock()
	defer log.mu.Unlock()

	st := &SerializedTxLog{
		NextID:     log.nextID,
		Operations: make([]*SerializedOperation, len(log.Operations)),
	}

	for i, op := range log.Operations {
		sop := &SerializedOperation{
			ID:         op.ID,
			Timestamp:  op.Timestamp,
			Type:       op.Type,
			Service:    op.Service,
			Method:     op.Method,
			Reversible: op.Reversible,
			UndoMethod: op.UndoMethod,
		}

		// Serialize args
		if len(op.Args) > 0 {
			sop.Args = make([]*SerializedValue, len(op.Args))
			for j, arg := range op.Args {
				ser, err := s.SerializeValue(arg)
				if err != nil {
					return nil, fmt.Errorf("serializing operation %d arg %d: %w", i, j, err)
				}
				sop.Args[j] = ser
			}
		}

		// Serialize kwargs
		if len(op.Kwargs) > 0 {
			sop.Kwargs = make(map[string]*SerializedValue)
			for k, v := range op.Kwargs {
				ser, err := s.SerializeValue(v)
				if err != nil {
					return nil, fmt.Errorf("serializing operation %d kwarg %q: %w", i, k, err)
				}
				sop.Kwargs[k] = ser
			}
		}

		// Serialize result
		if op.Result != nil {
			ser, err := s.SerializeValue(op.Result)
			if err != nil {
				return nil, fmt.Errorf("serializing operation %d result: %w", i, err)
			}
			sop.Result = ser
		}

		// Serialize error
		if op.Error != nil {
			errStr := op.Error.Error()
			sop.Error = &errStr
		}

		// Serialize undo data
		if len(op.UndoData) > 0 {
			sop.UndoData = make(map[string]*SerializedValue)
			for k, v := range op.UndoData {
				ser, err := s.SerializeValue(v)
				if err != nil {
					return nil, fmt.Errorf("serializing operation %d undo data %q: %w", i, k, err)
				}
				sop.UndoData[k] = ser
			}
		}

		st.Operations[i] = sop
	}

	return st, nil
}

// SerializeControlFlow converts control flow state.
func (s *Serializer) SerializeControlFlow(ctx *Context) (*ControlFlowState, error) {
	cf := &ControlFlowState{
		ShouldReturn:   ctx.shouldReturn,
		ShouldBreak:    ctx.shouldBreak,
		ShouldContinue: ctx.shouldContinue,
		ShouldStop:     ctx.shouldStop,
		Rollback:       ctx.rollback,
	}

	if ctx.returnValue != nil {
		rv, err := s.SerializeValue(ctx.returnValue)
		if err != nil {
			return nil, fmt.Errorf("serializing return value: %w", err)
		}
		cf.ReturnValue = rv
	}

	return cf, nil
}

// SerializeContext converts a Context to SerializedContext.
func (s *Serializer) SerializeContext(ctx *Context) (*SerializedContext, error) {
	scopes, currentScopeID, err := s.SerializeScopeChain(ctx.Scope, ctx.Globals)
	if err != nil {
		return nil, fmt.Errorf("serializing scope chain: %w", err)
	}

	txLog, err := s.SerializeTransactionLog(ctx.TxLog)
	if err != nil {
		return nil, fmt.Errorf("serializing transaction log: %w", err)
	}

	controlFlow, err := s.SerializeControlFlow(ctx)
	if err != nil {
		return nil, fmt.Errorf("serializing control flow: %w", err)
	}

	// Serialize emitted values
	emitted := make([]*SerializedValue, len(ctx.Emitted))
	for i, v := range ctx.Emitted {
		ser, err := s.SerializeValue(v)
		if err != nil {
			return nil, fmt.Errorf("serializing emitted value %d: %w", i, err)
		}
		emitted[i] = ser
	}

	return &SerializedContext{
		Scopes:         scopes,
		CurrentScopeID: currentScopeID,
		Limits:         SerializeLimits(ctx.Limits),
		TxLog:          txLog,
		Emitted:        emitted,
		ControlFlow:    controlFlow,
	}, nil
}

// CreateCheckpoint creates a full checkpoint from the current execution state.
func (s *Serializer) CreateCheckpoint(ctx *Context, script string, pos Position, message string) (*Checkpoint, error) {
	serCtx, err := s.SerializeContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("serializing context: %w", err)
	}

	// Generate script hash
	hash := sha256.Sum256([]byte(script))
	hashStr := fmt.Sprintf("%x", hash)

	return &Checkpoint{
		Version:           CheckpointVersion,
		Script:            script,
		ScriptHash:        hashStr,
		Position:          pos,
		Context:           serCtx,
		CreatedAt:         time.Now(),
		CheckpointMessage: message,
	}, nil
}

// HashScript returns the SHA256 hash of a script.
func HashScript(script string) string {
	hash := sha256.Sum256([]byte(script))
	return fmt.Sprintf("%x", hash)
}
