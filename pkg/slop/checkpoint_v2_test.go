package slop

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
)

type resumableTestService struct {
	calls int
	value Value
	err   error
}

func (s *resumableTestService) Call(_ string, _ []Value, _ map[string]Value) (Value, error) {
	s.calls++
	return s.value, s.err
}

func validV2Checkpoint() *Checkpoint {
	cp := &Checkpoint{
		Version:    CheckpointVersion,
		Script:     "pause\n",
		ScriptHash: "",
		Program: SerializedProgram{
			RootID: "n0",
			Nodes:  []SerializedNode{{ID: "n0", Kind: "program", Position: Position{Line: 1, Column: 1}, Fields: json.RawMessage(`{}`)}},
		},
		Position: Position{Line: 1, Column: 1},
		State: ExecutionState{
			Scopes:         []SerializedScope{{ID: "s0", Variables: map[string]*SerializedValue{}}},
			CurrentScopeID: "s0",
			Limits:         json.RawMessage(`{}`),
			Emitted:        []*SerializedValue{},
			ControlFlow:    json.RawMessage(`{}`),
		},
		Continuation:    []ContinuationFrame{{Kind: "program", NodeID: "n0", Phase: "next", ScopeID: "s0", State: json.RawMessage(`{}`)}},
		Breakpoints:     []Breakpoint{},
		Reason:          PauseStatement,
		MachineSnapshot: json.RawMessage(`{}`),
	}
	cp.ScriptHash = fmt.Sprintf("%x", sha256.Sum256([]byte(cp.Script)))
	return cp
}

func TestCheckpointV2RoundTripDeterministic(t *testing.T) {
	cp := validV2Checkpoint()
	first, err := MarshalCheckpoint(cp)
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalCheckpoint(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarshalCheckpoint(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("marshal is not deterministic:\n%s\n%s", first, second)
	}
	if !reflect.DeepEqual(cp, got) {
		t.Fatalf("round trip differs:\nwant %#v\ngot  %#v", cp, got)
	}
}

func TestUnmarshalCheckpointStrict(t *testing.T) {
	data, err := MarshalCheckpoint(validV2Checkpoint())
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"reason":"pause"`, `"reason":"pause","unexpected":true`, 1))
	_, err = UnmarshalCheckpoint(data)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestUnmarshalCheckpointRejectsTrailingValue(t *testing.T) {
	data, err := MarshalCheckpoint(validV2Checkpoint())
	if err != nil {
		t.Fatal(err)
	}
	_, err = UnmarshalCheckpoint(append(data, []byte(` {}`)...))
	var validation *CheckpointValidationError
	if !errors.As(err, &validation) || validation.Code != "trailing_data" {
		t.Fatalf("expected trailing_data error, got %v", err)
	}
}

func TestUnmarshalCheckpointV1IsInspectableIncompatibility(t *testing.T) {
	_, err := UnmarshalCheckpoint([]byte(`{"version":"1.0","script":"pause"}`))
	var incompatibility *CheckpointIncompatibilityError
	if !errors.As(err, &incompatibility) || incompatibility.Version != "1.0" {
		t.Fatalf("expected typed incompatibility, got %v", err)
	}
}

func TestValidateCheckpointReferencesAndKinds(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Checkpoint)
		path, code string
	}{
		{"unknown node", func(c *Checkpoint) { c.Program.Nodes[0].Kind = "mystery" }, "$.program.nodes[0].kind", "unknown_kind"},
		{"missing root", func(c *Checkpoint) { c.Program.RootID = "missing" }, "$.program.root_id", "dangling_reference"},
		{"missing current scope", func(c *Checkpoint) { c.State.CurrentScopeID = "missing" }, "$.state.current_scope_id", "dangling_reference"},
		{"frame node", func(c *Checkpoint) { c.Continuation[0].NodeID = "missing" }, "$.continuation[0].node_id", "dangling_reference"},
		{"frame scope", func(c *Checkpoint) { c.Continuation[0].ScopeID = "missing" }, "$.continuation[0].scope_id", "dangling_reference"},
		{"bad destination", func(c *Checkpoint) {
			c.Continuation[0].Destination = &PendingDestination{Kind: "value", FrameIndex: 9, Slot: "result"}
		}, "$.continuation[0].destination.frame_index", "dangling_reference"},
		{"failure on pause", func(c *Checkpoint) { c.Failure = &ExecutionFailure{Message: "x"} }, "$.failure", "incompatible"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := validV2Checkpoint()
			tt.mutate(cp)
			err := ValidateCheckpoint(cp)
			var validation *CheckpointValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("expected validation error, got %v", err)
			}
			if validation.Path != tt.path || validation.Code != tt.code {
				t.Fatalf("got path=%q code=%q", validation.Path, validation.Code)
			}
		})
	}
}

func TestValidateCheckpointRejectsScopeCycle(t *testing.T) {
	cp := validV2Checkpoint()
	parent := "s1"
	parentBack := "s0"
	cp.State.Scopes[0].ParentID = &parent
	cp.State.Scopes = append(cp.State.Scopes, SerializedScope{ID: "s1", ParentID: &parentBack, Variables: map[string]*SerializedValue{}})
	err := ValidateCheckpoint(cp)
	var validation *CheckpointValidationError
	if !errors.As(err, &validation) || validation.Code != "cycle" {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestValidateResumeOptions(t *testing.T) {
	cp := validV2Checkpoint()
	if err := ValidateResumeOptions(cp, ResumeOptions{Mode: ResumeContinue}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateResumeOptions(cp, ResumeOptions{Mode: ResumeRetryCall}); err == nil {
		t.Fatal("retry_call unexpectedly accepted pause checkpoint")
	}

	cp.Reason = PauseError
	cp.Failure = &ExecutionFailure{Message: "failed", Source: FailureSource{Kind: "service_call"}, Position: Position{Line: 1, Column: 1}}
	cp.Continuation[0] = ContinuationFrame{Kind: "service_call", NodeID: "n0", Phase: "failed", ScopeID: "s0", State: json.RawMessage(`{}`), Call: &PendingServiceCall{Service: "api", Method: "get", Args: []*SerializedValue{}, Kwargs: map[string]*SerializedValue{}, Failure: cp.Failure}}
	if err := ValidateResumeOptions(cp, ResumeOptions{Mode: ResumeAcceptResult}); err == nil {
		t.Fatal("accept_result accepted missing replacement")
	}
	replacement := &SerializedValue{Type: "int", Data: json.RawMessage(`"42"`)}
	if err := ValidateResumeOptions(cp, ResumeOptions{Mode: ResumeAcceptResult, ReplacementResult: replacement}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateResumeOptions(cp, ResumeOptions{Mode: ResumeRetryCall}); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeResumableAcceptResultDoesNotRetry(t *testing.T) {
	runtime := NewRuntime()
	service := &resumableTestService{err: errors.New("down")}
	runtime.RegisterExternalService("api", service)
	result, err := runtime.ExecuteResumable(context.Background(), "x = api.fetch(1 + 2)\nx + 4", ExecuteOptions{})
	if err == nil || result.Status != ExecutionFailed || result.Checkpoint == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	replacement := &SerializedValue{Type: "int", Data: json.RawMessage(`"6"`)}
	data, marshalErr := MarshalCheckpoint(result.Checkpoint)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	checkpoint, unmarshalErr := UnmarshalCheckpoint(data)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	fresh := NewRuntime()
	fresh.RegisterExternalService("api", service)
	result, err = fresh.Resume(context.Background(), checkpoint, ResumeOptions{Mode: ResumeAcceptResult, ReplacementResult: replacement})
	if err != nil || result.Status != ExecutionCompleted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if got := result.Result.(*IntValue).Value; got != 10 {
		t.Fatalf("result=%d", got)
	}
	if service.calls != 1 {
		t.Fatalf("accept_result calls=%d", service.calls)
	}
}

func TestRuntimeResumableRetryCallExactlyOnce(t *testing.T) {
	runtime := NewRuntime()
	service := &resumableTestService{err: errors.New("down")}
	runtime.RegisterExternalService("api", service)
	result, _ := runtime.ExecuteResumable(context.Background(), "api.fetch(1)", ExecuteOptions{})
	service.err, service.value = nil, NewIntValue(9)
	data, marshalErr := MarshalCheckpoint(result.Checkpoint)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	checkpoint, unmarshalErr := UnmarshalCheckpoint(data)
	if unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	fresh := NewRuntime()
	fresh.RegisterExternalService("api", service)
	result, err := fresh.Resume(context.Background(), checkpoint, ResumeOptions{Mode: ResumeRetryCall})
	if err != nil || result.Status != ExecutionCompleted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if service.calls != 2 {
		t.Fatalf("retry calls=%d", service.calls)
	}
}

func TestRuntimeConditionalBreakpointUsesCurrentScope(t *testing.T) {
	runtime := NewRuntime()
	result, err := runtime.ExecuteResumable(context.Background(), "x = 1\ny = 2\ny", ExecuteOptions{Breakpoints: []Breakpoint{{Line: 2, Condition: "x == 1"}}})
	if err != nil || result.Status != ExecutionPaused || result.Reason != PauseBreakpoint || result.Checkpoint.Position.Line != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, err := MarshalCheckpoint(result.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := UnmarshalCheckpoint(data)
	if err != nil {
		t.Fatal(err)
	}
	result, err = NewRuntime().Resume(context.Background(), cp, ResumeOptions{Mode: ResumeContinue})
	if err != nil || result.Status != ExecutionCompleted || result.Result.(*IntValue).Value != 2 {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	result, err = NewRuntime().ExecuteResumable(context.Background(), "x = 1\ny = 2\ny", ExecuteOptions{Breakpoints: []Breakpoint{{Line: 2, Condition: "x == 9"}}})
	if err != nil || result.Status != ExecutionCompleted {
		t.Fatalf("false condition result=%+v err=%v", result, err)
	}
}

func TestRuntimeConditionalBreakpointErrorCheckpoint(t *testing.T) {
	result, err := NewRuntime().ExecuteResumable(context.Background(), "x = 1\nx", ExecuteOptions{Breakpoints: []Breakpoint{{Line: 2, Condition: "missing"}}})
	if err == nil || result.Checkpoint == nil || result.Reason != PauseError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if result.Failure.Source.Kind != "breakpoint_condition" || result.Checkpoint.Position.Line != 2 {
		t.Fatalf("failure=%+v position=%+v", result.Failure, result.Checkpoint.Position)
	}
}

func TestPublicValueCodecCompositeValues(t *testing.T) {
	m := NewMapValue()
	m.Set("items", NewListValue([]Value{NewIntValue(1), NewStringValue("x")}))
	set := evaluator.NewSetValue()
	set.Add(NewStringValue("a"))
	m.Set("set", set)
	m.Set("error", &evaluator.SlopError{Message: "bad", Data: NewBoolValue(true)})
	serialized, err := serializePublicValue(m)
	if err != nil {
		t.Fatal(err)
	}
	value, err := deserializePublicValue(serialized)
	if err != nil {
		t.Fatal(err)
	}
	restored := value.(*MapValue)
	if restored.Pairs["items"].(*ListValue).Elements[0].(*IntValue).Value != 1 || restored.Pairs["set"].(*evaluator.SetValue).Elements["a"].String() != "a" || restored.Pairs["error"].(*evaluator.SlopError).Message != "bad" {
		t.Fatalf("roundtrip differs: %v", restored)
	}
}

func TestExecuteResumableMatchesExecute(t *testing.T) {
	source := "x = [1, 2 + 3]\nx[1] + (4 if true else 9)"
	want, err := NewRuntime().Execute(source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewRuntime().ExecuteResumable(context.Background(), source, ExecuteOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ExecutionCompleted || got.Result.String() != want.String() || got.Result.Type() != want.Type() {
		t.Fatalf("ordinary=%s/%s resumable=%+v", want.Type(), want.String(), got)
	}
}

func TestRuntimeSupportsResumable(t *testing.T) {
	runtime := NewRuntime()
	if err := runtime.SupportsResumable("x = api.fetch(1 + 2)\nx + 4"); err != nil {
		t.Fatalf("supported program rejected: %v", err)
	}
	for name, source := range map[string]string{
		"function": "def add(x):\n    return x + 1\nadd(1)",
		"loop":     "for x in [1, 2]:\n    emit(x)",
	} {
		t.Run(name, func(t *testing.T) {
			if err := runtime.SupportsResumable(source); err == nil || !strings.Contains(err.Error(), "unsupported resumable node") {
				t.Fatalf("expected unsupported error, got %v", err)
			}
		})
	}
}

func TestExecutionResultEmittedSurvivesFreshResume(t *testing.T) {
	service := &resumableTestService{err: errors.New("down")}
	runtime := NewRuntime()
	runtime.RegisterExternalService("api", service)
	result, runErr := runtime.ExecuteResumable(context.Background(), "emit(1)\napi.fetch()", ExecuteOptions{})
	if result.Checkpoint == nil {
		t.Fatalf("missing checkpoint: result=%+v err=%v", result, runErr)
	}
	if len(result.Emitted) != 1 || result.Emitted[0].(*IntValue).Value != 1 {
		t.Fatalf("failure emitted=%v", result.Emitted)
	}
	data, err := MarshalCheckpoint(result.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := UnmarshalCheckpoint(data)
	if err != nil {
		t.Fatal(err)
	}
	fresh := NewRuntime()
	fresh.RegisterExternalService("api", service)
	replacement := &SerializedValue{Type: "int", Data: json.RawMessage(`"2"`)}
	result, err = fresh.Resume(context.Background(), cp, ResumeOptions{Mode: ResumeAcceptResult, ReplacementResult: replacement})
	if err != nil || len(result.Emitted) != 1 || result.Emitted[0].(*IntValue).Value != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGenericErrorCheckpointCanBePatchedAndContinuedFresh(t *testing.T) {
	result, err := NewRuntime().ExecuteResumable(context.Background(), "missing + 1", ExecuteOptions{})
	if err == nil || result.Checkpoint == nil {
		t.Fatalf("expected resumable failure, result=%+v err=%v", result, err)
	}
	var snapshot map[string]any
	if err := json.Unmarshal(result.Checkpoint.MachineSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	contextState := snapshot["context"].(map[string]any)
	scopes := contextState["scopes"].([]any)
	for _, raw := range scopes {
		scope := raw.(map[string]any)
		if global, _ := scope["is_global"].(bool); global {
			continue
		}
		variables := scope["variables"].(map[string]any)
		variables["missing"] = map[string]any{"type": "int", "data": "2"}
	}
	result.Checkpoint.MachineSnapshot, err = json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	data, err := MarshalCheckpoint(result.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := UnmarshalCheckpoint(data)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := NewRuntime().Resume(context.Background(), cp, ResumeOptions{Mode: ResumeContinue})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != ExecutionCompleted || resumed.Result.String() != "3" {
		t.Fatalf("resumed=%+v", resumed)
	}
}

type panicService struct{}

func (*panicService) Call(string, []Value, map[string]Value) (Value, error) { panic("boom") }

func TestServicePanicBecomesResumableFailure(t *testing.T) {
	runtime := NewRuntime()
	runtime.RegisterExternalService("api", &panicService{})
	result, err := runtime.ExecuteResumable(context.Background(), "api.fetch()", ExecuteOptions{})
	if err == nil || result.Status != ExecutionFailed || result.Checkpoint == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !strings.Contains(err.Error(), "service call panicked") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResumeRejectsMalformedCallFrameBeforeExecution(t *testing.T) {
	service := &resumableTestService{err: errors.New("down")}
	runtime := NewRuntime()
	runtime.RegisterExternalService("api", service)
	result, _ := runtime.ExecuteResumable(context.Background(), "api.fetch()", ExecuteOptions{})
	var snapshot map[string]any
	if err := json.Unmarshal(result.Checkpoint.MachineSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	frames := snapshot["frames"].([]any)
	top := frames[len(frames)-1].(map[string]any)
	top["phase"], top["index"], top["kw_names"] = float64(3), float64(1), []any{}
	data, _ := json.Marshal(snapshot)
	result.Checkpoint.MachineSnapshot = data
	encoded, err := MarshalCheckpoint(result.Checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	cp, err := UnmarshalCheckpoint(encoded)
	if err != nil {
		t.Fatal(err)
	}
	fresh := NewRuntime()
	fresh.RegisterExternalService("api", service)
	_, err = fresh.Resume(context.Background(), cp, ResumeOptions{Mode: ResumeRetryCall})
	if err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatalf("expected frame index validation error, got %v", err)
	}
}
