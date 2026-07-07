package slop

import "testing"

func execStr(t *testing.T, src string) string {
	t.Helper()
	rt := NewRuntime()
	v, err := rt.Execute(src)
	if err != nil {
		t.Fatalf("execute %q: %v", src, err)
	}
	return v.String()
}

func TestBugSortedKeyFn(t *testing.T) {
	cases := map[string]string{
		`sorted(["bbb", "a", "cc"], x -> len(x))`:      "[a, cc, bbb]",
		`sorted([3, 1, 2])`:                            "[1, 2, 3]",
		`sorted([3, 1, 2], reverse: true)`:             "[3, 2, 1]",
		`sorted(["bbb", "a", "cc"], key: x -> len(x))`: "[a, cc, bbb]",
	}
	for src, want := range cases {
		if got := execStr(t, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}

func TestBugPipelineChain(t *testing.T) {
	cases := map[string]string{
		`[1, 2, 3, 4, 5] | take(2)`:                            "[1, 2]",
		`[1, 2, 3, 4, 5] | drop(2)`:                            "[3, 4, 5]",
		`[1, 2, 3, 4, 5] | chunk(2)`:                           "[[1, 2], [3, 4], [5]]",
		`[1, 2, 3, 4] | filter(x -> x > 2) | map(x -> x * 10)`: "[30, 40]",
	}
	for src, want := range cases {
		if got := execStr(t, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}

func TestBugSetSemantics(t *testing.T) {
	cases := map[string]string{
		`len({1, "1"})`:     "2", // distinct types, not collapsed
		`len({1, 1.0})`:     "1", // numeric equals collapse (matches ==)
		`len({1, 2, 2, 3})`: "3",
	}
	for src, want := range cases {
		if got := execStr(t, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}
