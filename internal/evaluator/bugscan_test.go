package evaluator

import (
	"testing"

	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
)

func TestBugListEquality(t *testing.T) {
	cases := map[string]bool{
		"[1, 2, 3] == [1, 2, 3]":     true,
		"[1, 2] == [1, 2, 3]":        false,
		"[1, [2, 3]] == [1, [2, 3]]": true,
		`{"a": 1} == {"a": 1}`:       true,
		`2 in [1, 2, 3]`:             true,
		`[2, 3] in [[1, 2], [2, 3]]`: true,
	}
	for src, want := range cases {
		got := testEval(t, src)
		b, ok := got.(*BoolValue)
		if !ok {
			t.Fatalf("%s: not bool, got %T", src, got)
		}
		if b.Value != want {
			t.Errorf("%s = %v, want %v", src, b.Value, want)
		}
	}
}

func TestBugArithmetic(t *testing.T) {
	cases := []struct {
		src  string
		want string
		typ  string
	}{
		{`5.5 % 2.0`, "1.5", "float"},
		{`2 ** -3`, "0.125", "float"},
		{`2.0 ** -2`, "0.25", "float"},
		{`3 * "x"`, "xxx", "string"},
		{`2 ** 10`, "1024", "int"},
		{`10 % 3`, "1", "int"},
	}
	for _, c := range cases {
		got := testEval(t, c.src)
		if got.String() != c.want || got.Type() != c.typ {
			t.Errorf("%s = %s (%s), want %s (%s)", c.src, got.String(), got.Type(), c.want, c.typ)
		}
	}
}

func TestBugSlice(t *testing.T) {
	cases := map[string]string{
		`[10,11,12,13,14][2::-1]`: "[12, 11, 10]",
		`[10,11,12,13,14][:2:-1]`: "[14, 13]",
		`[10,11,12,13,14][::-1]`:  "[14, 13, 12, 11, 10]",
		`[10,11,12,13,14][1:4]`:   "[11, 12, 13]",
		`[10,11,12,13,14][-2:]`:   "[13, 14]",
		`[10,11,12,13,14][::2]`:   "[10, 12, 14]",
		`"hello"[::-1]`:           "olleh",
		`"hello"[1:4]`:            "ell",
		`"hello"[4:1:-1]`:         "oll",
	}
	for src, want := range cases {
		got := testEval(t, src).String()
		if got != want {
			t.Errorf("%s = %q, want %q", src, got, want)
		}
	}
}

func TestBugNotIn(t *testing.T) {
	cases := map[string]bool{
		`3 not in [1, 2, 4]`: true,
		`2 not in [1, 2, 4]`: false,
		`"x" not in "text"`:  false,
		`"z" not in "text"`:  true,
	}
	for src, want := range cases {
		got := testEval(t, src)
		b, ok := got.(*BoolValue)
		if !ok {
			t.Fatalf("%s: not bool: %T", src, got)
		}
		if b.Value != want {
			t.Errorf("%s = %v, want %v", src, b.Value, want)
		}
	}
}

func TestBugSetTypeCollision(t *testing.T) {
	// string "1" must not be found in a set of the int 1
	if testEval(t, `"1" in {1}`).(*BoolValue).Value {
		t.Error(`"1" in {1} should be false`)
	}
	if !testEval(t, `1 in {1}`).(*BoolValue).Value {
		t.Error(`1 in {1} should be true`)
	}
}

func TestBugIterationCounterOffByOne(t *testing.T) {
	src := "total = 0\nfor x in [1, 2, 3]:\n    total = total + x\ntotal\n"
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse: %v", p.Errors())
	}
	// Exactly 3 iterations should fit within a limit of 3.
	ctx := NewContextWithLimits(&ExecutionLimits{MaxIterations: 3})
	e := NewWithContext(ctx)
	res, err := e.Eval(prog)
	if err != nil {
		t.Fatalf("loop of 3 items exceeded MaxIterations=3: %v", err)
	}
	if res.String() != "6" {
		t.Errorf("total = %s, want 6", res.String())
	}
	if ctx.Limits.IterationCount != 3 {
		t.Errorf("IterationCount = %d, want 3", ctx.Limits.IterationCount)
	}
}

func TestBugNotInPrecedence(t *testing.T) {
	// "not in" must bind like "in": arithmetic on the left applies first.
	cases := map[string]bool{
		`1 + 1 not in [2]`:    false, // (1+1) not in [2] -> 2 not in [2] -> false
		`1 + 1 not in [3, 4]`: true,  // 2 not in [3,4] -> true
		`1 + 1 in [2]`:        true,  // parity check with `in`
	}
	for src, want := range cases {
		got := testEval(t, src).(*BoolValue).Value
		if got != want {
			t.Errorf("%s = %v, want %v", src, got, want)
		}
	}
}
