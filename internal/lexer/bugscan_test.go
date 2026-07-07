package lexer

import "testing"

func TestBugUnterminatedString(t *testing.T) {
	for _, src := range []string{`"hello`, `x = "hello`, `'abc`} {
		l := New(src)
		sawIllegal := false
		for {
			tok := l.NextToken()
			if tok.Type == ILLEGAL {
				sawIllegal = true
			}
			if tok.Type == EOF {
				break
			}
		}
		if !sawIllegal {
			t.Errorf("unterminated string %q produced no ILLEGAL token", src)
		}
	}
	// A properly terminated string must NOT be illegal.
	l := New(`"hello"`)
	tok := l.NextToken()
	if tok.Type != STRING || tok.Literal != "hello" {
		t.Errorf(`"hello" => %v %q, want STRING "hello"`, tok.Type, tok.Literal)
	}
}
