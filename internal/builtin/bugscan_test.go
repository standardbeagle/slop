package builtin

import (
	"testing"

	"github.com/standardbeagle/slop/internal/evaluator"
)

func mkMap(pairs ...interface{}) *evaluator.MapValue {
	m := evaluator.NewMapValue()
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i].(string), &evaluator.IntValue{Value: int64(pairs[i+1].(int))})
	}
	return m
}

func TestBugMapOrderPreserved(t *testing.T) {
	// merge should produce a printable, ordered map
	got, err := builtinMerge([]evaluator.Value{mkMap("a", 1), mkMap("b", 2)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s := got.String(); s == "{}" {
		t.Errorf("merge printed empty: %q (Order lost)", s)
	}
	// copy of a map should also print its contents
	cp, err := builtinCopy([]evaluator.Value{mkMap("x", 9)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s := cp.String(); s == "{}" {
		t.Errorf("copy printed empty: %q (Order lost)", s)
	}
}
