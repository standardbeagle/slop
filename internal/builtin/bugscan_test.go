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

func lst(ns ...int) *evaluator.ListValue {
	items := make([]evaluator.Value, len(ns))
	for i, n := range ns {
		items[i] = &evaluator.IntValue{Value: int64(n)}
	}
	return &evaluator.ListValue{Elements: items}
}

// Pipeline dispatch prepends the piped list as args[0]: `xs | take(2)` -> take(xs, 2).
func TestBugPipelineArgOrder(t *testing.T) {
	xs := lst(1, 2, 3, 4, 5)
	two := &evaluator.IntValue{Value: 2}
	checks := []struct {
		name string
		fn   func([]evaluator.Value, map[string]evaluator.Value) (evaluator.Value, error)
		want string
	}{
		{"take", builtinTake, "[1, 2]"},
		{"drop", builtinDrop, "[3, 4, 5]"},
		{"chunk", builtinChunk, "[[1, 2], [3, 4], [5]]"},
		{"window", builtinWindow, "[[1, 2], [2, 3], [3, 4], [4, 5]]"},
	}
	for _, c := range checks {
		// pipeline order: list first, n second
		got, err := c.fn([]evaluator.Value{xs, two}, nil)
		if err != nil {
			t.Errorf("%s(list, n) err: %v", c.name, err)
			continue
		}
		if got.String() != c.want {
			t.Errorf("%s(list, n) = %s, want %s", c.name, got.String(), c.want)
		}
		// direct order still works: n first, list second
		got2, err := c.fn([]evaluator.Value{two, xs}, nil)
		if err != nil {
			t.Errorf("%s(n, list) err: %v", c.name, err)
			continue
		}
		if got2.String() != c.want {
			t.Errorf("%s(n, list) = %s, want %s", c.name, got2.String(), c.want)
		}
	}
}
