package evaluator

import "testing"

func TestBugListEquality(t *testing.T) {
	cases := map[string]bool{
		"[1, 2, 3] == [1, 2, 3]":       true,
		"[1, 2] == [1, 2, 3]":          false,
		"[1, [2, 3]] == [1, [2, 3]]":   true,
		`{"a": 1} == {"a": 1}`:         true,
		`2 in [1, 2, 3]`:               true,
		`[2, 3] in [[1, 2], [2, 3]]`:   true,
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
