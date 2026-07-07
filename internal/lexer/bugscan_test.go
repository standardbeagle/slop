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

func tokTypes(src string) []TokenType {
	l := New(src)
	var out []TokenType
	for {
		t := l.NextToken()
		out = append(out, t.Type)
		if t.Type == EOF {
			return out
		}
	}
}

func TestBugCRLFBlankLine(t *testing.T) {
	lf := "for x in y:\n    a\n\n    b\n"
	crlf := "for x in y:\r\n    a\r\n\r\n    b\r\n"
	a, b := tokTypes(lf), tokTypes(crlf)
	if len(a) != len(b) {
		t.Fatalf("token count differs LF=%d CRLF=%d\nLF=%v\nCRLF=%v", len(a), len(b), a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("token %d differs: LF=%v CRLF=%v", i, a[i], b[i])
		}
	}
}
