package slop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// CheckpointVersion is the current editable checkpoint format.
const CheckpointVersion = "2.0"

type ExecutionStatus string

const (
	ExecutionCompleted ExecutionStatus = "completed"
	ExecutionPaused    ExecutionStatus = "paused"
	ExecutionFailed    ExecutionStatus = "failed"
)

type PauseReason string

const (
	PauseBreakpoint PauseReason = "breakpoint"
	PauseStatement  PauseReason = "pause"
	PauseError      PauseReason = "error"
)

type ResumeMode string

const (
	ResumeContinue     ResumeMode = "continue"
	ResumeAcceptResult ResumeMode = "accept_result"
	ResumeRetryCall    ResumeMode = "retry_call"
)

type Breakpoint struct {
	Line      int    `json:"line"`
	Condition string `json:"condition,omitempty"`
}

type ExecuteOptions struct {
	Debug       bool
	Breakpoints []Breakpoint
}

type ResumeOptions struct {
	Mode              ResumeMode
	ReplacementResult *SerializedValue
}

type ExecutionResult struct {
	Status     ExecutionStatus
	Result     Value
	Checkpoint *Checkpoint
	Reason     PauseReason
	Failure    *ExecutionFailure
	Emitted    []Value
}

type FailureSource struct {
	Kind   string `json:"kind"`
	NodeID string `json:"node_id,omitempty"`
}

type ExecutionFailure struct {
	Code     string           `json:"code,omitempty"`
	Message  string           `json:"message"`
	Data     *SerializedValue `json:"data,omitempty"`
	Source   FailureSource    `json:"source"`
	Position Position         `json:"position"`
}

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// SerializedValue is the stable wire representation of a SLOP value.
type SerializedValue struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type SerializedProgram struct {
	RootID string           `json:"root_id"`
	Nodes  []SerializedNode `json:"nodes"`
}

type SerializedNode struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Position Position        `json:"position"`
	Fields   json.RawMessage `json:"fields"`
}

type SerializedScope struct {
	ID        string                      `json:"id"`
	ParentID  *string                     `json:"parent_id,omitempty"`
	Variables map[string]*SerializedValue `json:"variables"`
	IsGlobal  bool                        `json:"is_global,omitempty"`
}

type ExecutionState struct {
	Scopes         []SerializedScope  `json:"scopes"`
	CurrentScopeID string             `json:"current_scope_id"`
	Limits         json.RawMessage    `json:"limits"`
	TransactionLog json.RawMessage    `json:"transaction_log,omitempty"`
	Emitted        []*SerializedValue `json:"emitted"`
	ControlFlow    json.RawMessage    `json:"control_flow"`
}

type PendingDestination struct {
	Kind       string `json:"kind"`
	FrameIndex int    `json:"frame_index"`
	Slot       string `json:"slot"`
}

type PendingServiceCall struct {
	Service                string                      `json:"service"`
	Method                 string                      `json:"method"`
	Args                   []*SerializedValue          `json:"args"`
	Kwargs                 map[string]*SerializedValue `json:"kwargs"`
	Failure                *ExecutionFailure           `json:"failure,omitempty"`
	TransactionOperationID *int64                      `json:"transaction_operation_id,omitempty"`
	MayHaveSideEffects     bool                        `json:"may_have_side_effects"`
}

type ContinuationFrame struct {
	Kind        string              `json:"kind"`
	NodeID      string              `json:"node_id"`
	Phase       string              `json:"phase"`
	ScopeID     string              `json:"scope_id"`
	Destination *PendingDestination `json:"destination,omitempty"`
	State       json.RawMessage     `json:"state"`
	Call        *PendingServiceCall `json:"call,omitempty"`
}

type CheckpointDiagnostic struct {
	CreatedAt string `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	Message   string `json:"message,omitempty"`
}

type Checkpoint struct {
	Version         string                `json:"version"`
	Script          string                `json:"script"`
	ScriptHash      string                `json:"script_hash"`
	Program         SerializedProgram     `json:"program"`
	Position        Position              `json:"position"`
	State           ExecutionState        `json:"state"`
	Continuation    []ContinuationFrame   `json:"continuation"`
	Breakpoints     []Breakpoint          `json:"breakpoints"`
	Reason          PauseReason           `json:"reason"`
	Failure         *ExecutionFailure     `json:"failure,omitempty"`
	Diagnostic      *CheckpointDiagnostic `json:"diagnostic,omitempty"`
	MachineSnapshot json.RawMessage       `json:"machine_snapshot"`
}

type CheckpointValidationError struct {
	Path    string
	Code    string
	Message string
}

func (e *CheckpointValidationError) Error() string {
	return fmt.Sprintf("invalid checkpoint at %s (%s): %s", e.Path, e.Code, e.Message)
}

type CheckpointIncompatibilityError struct {
	Version string
	Reason  string
}

func (e *CheckpointIncompatibilityError) Error() string {
	return fmt.Sprintf("checkpoint version %q is incompatible: %s", e.Version, e.Reason)
}

func invalid(path, code, message string) error {
	return &CheckpointValidationError{Path: path, Code: code, Message: message}
}

var nodeKinds = setOf(
	"program", "module", "block", "expression_statement", "assign", "if", "for", "match",
	"match_arm", "def", "parameter", "return", "emit", "stop", "try", "catch", "break",
	"continue", "pause", "identifier", "integer", "float", "string", "boolean", "none", "list",
	"map", "set", "prefix", "infix", "call", "index", "slice", "member", "lambda", "pipeline",
	"ternary", "match_expression", "list_comprehension", "map_comprehension", "schema", "schema_field",
	"enum", "type", "range",
)

var frameKinds = setOf(
	"program", "block", "expression_statement", "assign", "if", "for", "match", "function_call",
	"lambda_call", "return", "emit", "try", "call", "service_call", "index", "slice", "member",
	"list", "map", "set", "prefix", "infix", "pipeline", "ternary", "list_comprehension",
	"map_comprehension", "range", "schema", "enum",
)

var valueKinds = setOf("none", "bool", "int", "float", "string", "list", "map", "set", "function", "lambda", "builtin", "service", "error", "iterator")

func setOf(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, value := range values {
		m[value] = struct{}{}
	}
	return m
}

// MarshalCheckpoint validates and deterministically marshals a v2 checkpoint.
func MarshalCheckpoint(checkpoint *Checkpoint) ([]byte, error) {
	if err := ValidateCheckpoint(checkpoint); err != nil {
		return nil, err
	}
	return json.Marshal(checkpoint)
}

// UnmarshalCheckpoint strictly decodes and validates a checkpoint.
func UnmarshalCheckpoint(data []byte) (*Checkpoint, error) {
	var header struct {
		Version string `json:"version"`
	}
	headerDecoder := json.NewDecoder(bytes.NewReader(data))
	if err := headerDecoder.Decode(&header); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint header: %w", err)
	}
	if header.Version != CheckpointVersion {
		return nil, &CheckpointIncompatibilityError{Version: header.Version, Reason: "only v2 checkpoints can be resumed by this API"}
	}
	var checkpoint Checkpoint
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshaling checkpoint: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, invalid("$", "trailing_data", "multiple JSON values")
		}
		return nil, fmt.Errorf("unmarshaling trailing checkpoint data: %w", err)
	}
	if err := ValidateCheckpoint(&checkpoint); err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

func ValidateCheckpoint(c *Checkpoint) error {
	if c == nil {
		return invalid("$", "required", "checkpoint is nil")
	}
	if c.Version != CheckpointVersion {
		return &CheckpointIncompatibilityError{Version: c.Version, Reason: "expected " + CheckpointVersion}
	}
	if c.Script == "" {
		return invalid("$.script", "required", "script is empty")
	}
	if len(c.ScriptHash) != 64 || strings.Trim(c.ScriptHash, "0123456789abcdef") != "" {
		return invalid("$.script_hash", "format", "must be a lowercase SHA-256 hash")
	}
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(c.Script)))
	if c.ScriptHash != expectedHash {
		return invalid("$.script_hash", "mismatch", "does not match script")
	}
	if c.Position.Line < 1 || c.Position.Column < 1 {
		return invalid("$.position", "range", "line and column must be positive")
	}
	if !validObject(c.MachineSnapshot) {
		return invalid("$.machine_snapshot", "type", "must be a JSON object")
	}
	if c.Program.RootID == "" {
		return invalid("$.program.root_id", "required", "root node is missing")
	}
	nodes := make(map[string]struct{}, len(c.Program.Nodes))
	for i, node := range c.Program.Nodes {
		path := fmt.Sprintf("$.program.nodes[%d]", i)
		if node.ID == "" {
			return invalid(path+".id", "required", "node ID is empty")
		}
		if _, exists := nodes[node.ID]; exists {
			return invalid(path+".id", "duplicate", "node ID is not unique")
		}
		nodes[node.ID] = struct{}{}
		if _, ok := nodeKinds[node.Kind]; !ok {
			return invalid(path+".kind", "unknown_kind", node.Kind)
		}
		if !validObject(node.Fields) {
			return invalid(path+".fields", "type", "must be a JSON object")
		}
	}
	if _, ok := nodes[c.Program.RootID]; !ok {
		return invalid("$.program.root_id", "dangling_reference", "root node does not exist")
	}

	scopes := make(map[string]*string, len(c.State.Scopes))
	for i := range c.State.Scopes {
		s := &c.State.Scopes[i]
		path := fmt.Sprintf("$.state.scopes[%d]", i)
		if s.ID == "" {
			return invalid(path+".id", "required", "scope ID is empty")
		}
		if _, exists := scopes[s.ID]; exists {
			return invalid(path+".id", "duplicate", "scope ID is not unique")
		}
		scopes[s.ID] = s.ParentID
		for name, value := range s.Variables {
			if err := validateValue(value, path+".variables."+name); err != nil {
				return err
			}
		}
	}
	if _, ok := scopes[c.State.CurrentScopeID]; !ok {
		return invalid("$.state.current_scope_id", "dangling_reference", "current scope does not exist")
	}
	for id, parent := range scopes {
		if parent != nil {
			if _, ok := scopes[*parent]; !ok {
				return invalid("$.state.scopes", "dangling_reference", "parent of "+id+" does not exist")
			}
		}
	}
	for id := range scopes {
		seen := map[string]bool{}
		for p := id; p != ""; {
			if seen[p] {
				return invalid("$.state.scopes", "cycle", "scope parent cycle at "+p)
			}
			seen[p] = true
			parent := scopes[p]
			if parent == nil {
				break
			}
			p = *parent
		}
	}
	if !validObject(c.State.Limits) {
		return invalid("$.state.limits", "type", "must be a JSON object")
	}
	if !validObject(c.State.ControlFlow) {
		return invalid("$.state.control_flow", "type", "must be a JSON object")
	}
	for i, value := range c.State.Emitted {
		if err := validateValue(value, fmt.Sprintf("$.state.emitted[%d]", i)); err != nil {
			return err
		}
	}
	for i, b := range c.Breakpoints {
		if b.Line < 1 {
			return invalid(fmt.Sprintf("$.breakpoints[%d].line", i), "range", "must be positive")
		}
	}
	if c.Reason != PauseBreakpoint && c.Reason != PauseStatement && c.Reason != PauseError {
		return invalid("$.reason", "enum", string(c.Reason))
	}
	if c.Reason == PauseError && c.Failure == nil {
		return invalid("$.failure", "required", "error checkpoints require failure metadata")
	}
	if c.Reason != PauseError && c.Failure != nil {
		return invalid("$.failure", "incompatible", "failure metadata is only valid for error checkpoints")
	}
	for i, frame := range c.Continuation {
		path := fmt.Sprintf("$.continuation[%d]", i)
		if _, ok := frameKinds[frame.Kind]; !ok {
			return invalid(path+".kind", "unknown_kind", frame.Kind)
		}
		if _, ok := nodes[frame.NodeID]; !ok {
			return invalid(path+".node_id", "dangling_reference", "node does not exist")
		}
		if _, ok := scopes[frame.ScopeID]; !ok {
			return invalid(path+".scope_id", "dangling_reference", "scope does not exist")
		}
		if frame.Phase == "" {
			return invalid(path+".phase", "required", "phase is empty")
		}
		if !validObject(frame.State) {
			return invalid(path+".state", "type", "must be a JSON object")
		}
		if frame.Destination != nil {
			d := frame.Destination
			if d.Kind == "" || d.Slot == "" {
				return invalid(path+".destination", "required", "kind and slot are required")
			}
			if d.FrameIndex < 0 || d.FrameIndex >= len(c.Continuation) {
				return invalid(path+".destination.frame_index", "dangling_reference", "frame does not exist")
			}
		}
		if frame.Kind == "service_call" && frame.Call == nil {
			return invalid(path+".call", "required", "service-call frame requires call state")
		}
		if frame.Kind != "service_call" && frame.Call != nil {
			return invalid(path+".call", "incompatible", "call state requires service_call frame")
		}
		if frame.Call != nil {
			if frame.Call.Service == "" || frame.Call.Method == "" {
				return invalid(path+".call", "required", "service and method are required")
			}
			for j, value := range frame.Call.Args {
				if err := validateValue(value, fmt.Sprintf("%s.call.args[%d]", path, j)); err != nil {
					return err
				}
			}
			for name, value := range frame.Call.Kwargs {
				if err := validateValue(value, path+".call.kwargs."+name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validObject(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil && object != nil
}

func validateValue(value *SerializedValue, path string) error {
	if value == nil {
		return invalid(path, "required", "value is null")
	}
	if _, ok := valueKinds[value.Type]; !ok {
		return invalid(path+".type", "unknown_kind", value.Type)
	}
	if len(value.Data) > 0 && !json.Valid(value.Data) {
		return invalid(path+".data", "json", "invalid JSON")
	}
	return nil
}

// ValidateResumeOptions verifies mode compatibility before execution mutates state.
func ValidateResumeOptions(checkpoint *Checkpoint, options ResumeOptions) error {
	if err := ValidateCheckpoint(checkpoint); err != nil {
		return err
	}
	hasFailedCall := false
	if len(checkpoint.Continuation) > 0 {
		frame := checkpoint.Continuation[len(checkpoint.Continuation)-1]
		hasFailedCall = frame.Kind == "service_call" && frame.Call != nil && frame.Call.Failure != nil
	}
	switch options.Mode {
	case ResumeContinue:
		if checkpoint.Reason == PauseError && hasFailedCall {
			return invalid("$.mode", "incompatible", "failed service calls require accept_result or retry_call")
		}
	case ResumeAcceptResult:
		if !hasFailedCall {
			return invalid("$.mode", "incompatible", "accept_result requires a failed service call")
		}
		if options.ReplacementResult == nil {
			return invalid("$.replacement_result", "required", "accept_result requires a replacement")
		}
		return validateValue(options.ReplacementResult, "$.replacement_result")
	case ResumeRetryCall:
		if !hasFailedCall {
			return invalid("$.mode", "incompatible", "retry_call requires a failed service call")
		}
	default:
		return invalid("$.mode", "enum", string(options.Mode))
	}
	return nil
}

// ContextService is implemented by services that support cancellation. Legacy Service remains supported.
type ContextService interface {
	CallContext(context.Context, string, []Value, map[string]Value) (Value, error)
}

// SortedNodeIDs is useful to diagnostics without exposing map iteration order.
func (c *Checkpoint) SortedNodeIDs() []string {
	if c == nil {
		return nil
	}
	ids := make([]string, len(c.Program.Nodes))
	for i := range c.Program.Nodes {
		ids[i] = c.Program.Nodes[i].ID
	}
	sort.Strings(ids)
	return ids
}
