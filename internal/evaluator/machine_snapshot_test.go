package evaluator

import (
	"encoding/json"
	"errors"
	"github.com/standardbeagle/slop/internal/ast"
	"testing"
)

func TestMachineSnapshotJSONRestoreAcceptResult(t *testing.T) {
	s1 := &countingService{err: errors.New("down")}
	m := newTestMachine(t, "x = api.fetch(1 + 2)\nx + 4", s1)
	r := m.Run()
	program := parseMachineProgram(t, "x = api.fetch(1 + 2)\nx + 4")
	snap, err := m.Snapshot(program)
	if err == nil {
		t.Fatal("expected different AST rejection")
	}
	program = m.stack[0].node.(*ast.Program)
	snap, err = m.Snapshot(program)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var decoded MachineSnapshot
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	s2 := &countingService{err: errors.New("must not call")}
	fresh := parseMachineProgram(t, "x = api.fetch(1 + 2)\nx + 4")
	restored, cp, err := RestoreMachine(fresh, &decoded, nil, map[string]Service{"api": s2})
	if err != nil {
		t.Fatal(err)
	}
	r = restored.Resume(cp, MachineResumeOptions{Mode: MachineAcceptResult, Replacement: &IntValue{Value: 6}})
	if r.Status != MachineCompleted || r.Value.(*IntValue).Value != 10 || s2.calls != 0 {
		t.Fatalf("result=%+v calls=%d", r, s2.calls)
	}
}

func TestMachineSnapshotJSONRestoreRetryOnce(t *testing.T) {
	s := &countingService{err: errors.New("down")}
	m := newTestMachine(t, "api.fetch(1)", s)
	m.Run()
	program := m.stack[0].node.(*ast.Program)
	snap, err := m.Snapshot(program)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(snap)
	var decoded MachineSnapshot
	json.Unmarshal(data, &decoded)
	freshSvc := &countingService{result: &IntValue{Value: 9}}
	restored, cp, err := RestoreMachine(parseMachineProgram(t, "api.fetch(1)"), &decoded, nil, map[string]Service{"api": freshSvc})
	if err != nil {
		t.Fatal(err)
	}
	r := restored.Resume(cp, MachineResumeOptions{Mode: MachineRetryCall})
	if r.Status != MachineCompleted || freshSvc.calls != 1 {
		t.Fatalf("result=%+v calls=%d", r, freshSvc.calls)
	}
}
