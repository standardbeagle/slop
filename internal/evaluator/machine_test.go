package evaluator

import (
	"context"
	"errors"
	"testing"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
)

type contextService struct{ called bool }

func (s *contextService) Call(string, []Value, map[string]Value) (Value, error) { panic("legacy call") }
func (s *contextService) CallContext(ctx context.Context, _ string, _ []Value, _ map[string]Value) (Value, error) {
	s.called = true
	return &IntValue{Value: 5}, ctx.Err()
}

type countingService struct {
	calls  int
	result Value
	err    error
}

func (s *countingService) Call(_ string, _ []Value, _ map[string]Value) (Value, error) {
	s.calls++
	return s.result, s.err
}

func parseMachineProgram(t *testing.T, source string) *ast.Program {
	t.Helper()
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse: %v", errs)
	}
	return program
}

func newTestMachine(t *testing.T, source string, svc Service) *Machine {
	t.Helper()
	program := parseMachineProgram(t, source)
	ctx := NewContext()
	if svc != nil {
		ctx.RegisterService("api", svc)
	}
	return NewMachine(program, ctx)
}

func TestMachineNestedExpressionAndAssignment(t *testing.T) {
	m := newTestMachine(t, "x = 1 + 2 * 3\nx + 4", nil)
	r := m.Run()
	if r.Status != MachineCompleted {
		t.Fatalf("status=%s err=%v", r.Status, r.Err)
	}
	if got := r.Value.(*IntValue).Value; got != 11 {
		t.Fatalf("got %d", got)
	}
}

func TestMachineServiceFailureAcceptResultDoesNotRetry(t *testing.T) {
	svc := &countingService{err: errors.New("down")}
	m := newTestMachine(t, "x = api.fetch(1 + 2)\nx + 4", svc)
	r := m.Run()
	if r.Status != MachineFailed || r.Checkpoint == nil {
		t.Fatalf("result=%+v", r)
	}
	if svc.calls != 1 {
		t.Fatalf("calls=%d", svc.calls)
	}
	r = m.Resume(r.Checkpoint, MachineResumeOptions{Mode: MachineAcceptResult, Replacement: &IntValue{Value: 6}})
	if r.Status != MachineCompleted || r.Value.(*IntValue).Value != 10 {
		t.Fatalf("result=%+v", r)
	}
	if svc.calls != 1 {
		t.Fatalf("accept retried: calls=%d", svc.calls)
	}
}

func TestMachineServiceFailureRetryExactlyOnce(t *testing.T) {
	svc := &countingService{err: errors.New("down")}
	m := newTestMachine(t, "api.fetch(1)", svc)
	r := m.Run()
	if svc.calls != 1 {
		t.Fatalf("calls=%d", svc.calls)
	}
	svc.err, svc.result = nil, &IntValue{Value: 9}
	r = m.Resume(r.Checkpoint, MachineResumeOptions{Mode: MachineRetryCall})
	if r.Status != MachineCompleted || r.Value.(*IntValue).Value != 9 {
		t.Fatalf("result=%+v", r)
	}
	if svc.calls != 2 {
		t.Fatalf("retry calls=%d", svc.calls)
	}
}

func TestMachineCollectionsControlFlowEmitAndReturn(t *testing.T) {
	m := newTestMachine(t, "emit([1, 2 + 3][1])\nif not false:\n    return 7\n9", nil)
	r := m.Run()
	if r.Status != MachineCompleted {
		t.Fatalf("%+v", r)
	}
	if got := m.ctx.Emitted[0].(*IntValue).Value; got != 5 {
		t.Fatalf("emit=%d", got)
	}
	if got := r.Value.(*IntValue).Value; got != 7 {
		t.Fatalf("result=%d", got)
	}
}

func TestMachinePauseAndBreakpointContinueOnce(t *testing.T) {
	m := newTestMachine(t, "x = 1\npause\nx + 2", nil)
	r := m.Run()
	if r.Status != MachinePaused {
		t.Fatalf("%+v", r)
	}
	r = m.Resume(r.Checkpoint, MachineResumeOptions{Mode: MachineContinue})
	if r.Status != MachineCompleted || r.Value.(*IntValue).Value != 3 {
		t.Fatalf("%+v", r)
	}
	m = newTestMachine(t, "x = 1\nx + 2", nil)
	m.SetBreakpoint(2)
	r = m.Run()
	if r.Status != MachinePaused {
		t.Fatalf("%+v", r)
	}
	r = m.Resume(r.Checkpoint, MachineResumeOptions{Mode: MachineContinue})
	if r.Status != MachineCompleted {
		t.Fatalf("%+v", r)
	}
}

func TestMachineUsesContextService(t *testing.T) {
	s := &contextService{}
	m := newTestMachine(t, "api.fetch()", s)
	r := m.RunContext(context.Background())
	if r.Status != MachineCompleted || !s.called {
		t.Fatalf("%+v called=%v", r, s.called)
	}
}

func TestMachineDifferentialCollectionsAndNamedEmit(t *testing.T) {
	for _, src := range []string{"[1, 2 + 3][1]", "{a: 1, b: 2}", "{1, 2, 2}"} {
		legacy := New()
		p := parseMachineProgram(t, src)
		want, err := legacy.Eval(p)
		if err != nil {
			t.Fatal(err)
		}
		m := NewMachine(p, NewContext())
		got := m.Run()
		equal := got.Value.String() == want.String()
		if gotSet, ok := got.Value.(*SetValue); ok {
			wantSet, wantOK := want.(*SetValue)
			equal = wantOK && len(gotSet.Elements) == len(wantSet.Elements)
			for key := range gotSet.Elements {
				_, present := wantSet.Elements[key]
				equal = equal && present
			}
		}
		if got.Status != MachineCompleted || !equal {
			t.Fatalf("%s: got %v want %v", src, got.Value, want)
		}
	}
	m := newTestMachine(t, "emit(a: 1, b: 2)", nil)
	r := m.Run()
	if r.Status != MachineCompleted || len(m.ctx.Emitted) != 1 {
		t.Fatalf("%+v", r)
	}
}
