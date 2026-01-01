package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextToken_Operators(t *testing.T) {
	input := `+ - * / % ** == != < > <= >= = += -= *= /= | -> . ?. ?[`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{PLUS, "+"},
		{MINUS, "-"},
		{STAR, "*"},
		{SLASH, "/"},
		{PERCENT, "%"},
		{STARSTAR, "**"},
		{EQ, "=="},
		{NE, "!="},
		{LT, "<"},
		{GT, ">"},
		{LE, "<="},
		{GE, ">="},
		{ASSIGN, "="},
		{PLUSEQ, "+="},
		{MINUSEQ, "-="},
		{STAREQ, "*="},
		{SLASHEQ, "/="},
		{PIPE, "|"},
		{ARROW, "->"},
		{DOT, "."},
		{OPTDOT, "?."},
		{OPTLBRACK, "?["},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		assert.Equal(t, tt.expectedType, tok.Type, "test[%d] - tokentype wrong", i)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_Delimiters(t *testing.T) {
	input := `( ) [ ] { } : ,`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{LPAREN, "("},
		{RPAREN, ")"},
		{LBRACK, "["},
		{RBRACK, "]"},
		{LBRACE, "{"},
		{RBRACE, "}"},
		{COLON, ":"},
		{COMMA, ","},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		assert.Equal(t, tt.expectedType, tok.Type, "test[%d] - tokentype wrong", i)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_Keywords(t *testing.T) {
	input := `if elif else for in with match def return emit stop and or not true false none range limit rate parallel timeout try catch break continue`

	tests := []struct {
		expectedType    TokenType
		expectedLiteral string
	}{
		{IF, "if"},
		{ELIF, "elif"},
		{ELSE, "else"},
		{FOR, "for"},
		{IN, "in"},
		{WITH, "with"},
		{MATCH, "match"},
		{DEF, "def"},
		{RETURN, "return"},
		{EMIT, "emit"},
		{STOP, "stop"},
		{AND, "and"},
		{OR, "or"},
		{NOT, "not"},
		{TRUE, "true"},
		{FALSE, "false"},
		{NONE, "none"},
		{RANGE, "range"},
		{LIMIT, "limit"},
		{RATE, "rate"},
		{PARALLEL, "parallel"},
		{TIMEOUT, "timeout"},
		{TRY, "try"},
		{CATCH, "catch"},
		{BREAK, "break"},
		{CONTINUE, "continue"},
		{EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		assert.Equal(t, tt.expectedType, tok.Type, "test[%d] - tokentype wrong for %s", i, tt.expectedLiteral)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_Identifiers(t *testing.T) {
	input := `foo bar_baz _private camelCase PascalCase var123`

	tests := []struct {
		expectedLiteral string
	}{
		{"foo"},
		{"bar_baz"},
		{"_private"},
		{"camelCase"},
		{"PascalCase"},
		{"var123"},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		assert.Equal(t, IDENT, tok.Type, "test[%d] - expected IDENT", i)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_Numbers(t *testing.T) {
	tests := []struct {
		input           string
		expectedType    TokenType
		expectedLiteral string
	}{
		{"42", INT, "42"},
		{"0", INT, "0"},
		{"123456", INT, "123456"},
		{"1_000_000", INT, "1000000"},
		{"3.14", FLOAT, "3.14"},
		{"0.5", FLOAT, "0.5"},
		{"1e10", FLOAT, "1e10"},
		{"2.5e-3", FLOAT, "2.5e-3"},
		{"1_000.50", FLOAT, "1000.50"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		assert.Equal(t, tt.expectedType, tok.Type, "input %q - tokentype wrong", tt.input)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "input %q - literal wrong", tt.input)
	}
}

func TestNextToken_Strings(t *testing.T) {
	tests := []struct {
		input           string
		expectedLiteral string
	}{
		{`"hello"`, "hello"},
		{`'world'`, "world"},
		{`"hello world"`, "hello world"},
		{`"line1\nline2"`, "line1\nline2"},
		{`"tab\there"`, "tab\there"},
		{`"quote\"here"`, "quote\"here"},
		{`"backslash\\"`, "backslash\\"},
		{`"brace\{here\}"`, "brace{here}"},
		{`""`, ""},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		assert.Equal(t, STRING, tok.Type, "input %q - expected STRING", tt.input)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "input %q - literal wrong", tt.input)
	}
}

func TestNextToken_Comments(t *testing.T) {
	input := `x = 1  # this is a comment
y = 2`

	l := New(input)

	// x
	tok := l.NextToken()
	assert.Equal(t, IDENT, tok.Type)
	assert.Equal(t, "x", tok.Literal)

	// =
	tok = l.NextToken()
	assert.Equal(t, ASSIGN, tok.Type)

	// 1
	tok = l.NextToken()
	assert.Equal(t, INT, tok.Type)

	// NEWLINE
	tok = l.NextToken()
	assert.Equal(t, NEWLINE, tok.Type)

	// y
	tok = l.NextToken()
	assert.Equal(t, IDENT, tok.Type)
	assert.Equal(t, "y", tok.Literal)
}

func TestNextToken_Indentation(t *testing.T) {
	input := `if x:
    y = 1
    z = 2
w = 3`

	l := New(input)

	expected := []TokenType{
		IF, IDENT, COLON, NEWLINE,
		INDENT, IDENT, ASSIGN, INT, NEWLINE,
		IDENT, ASSIGN, INT, NEWLINE,
		DEDENT, IDENT, ASSIGN, INT,
		EOF,
	}

	for i, expectedType := range expected {
		tok := l.NextToken()
		assert.Equal(t, expectedType, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, expectedType)
	}
}

func TestNextToken_NestedIndentation(t *testing.T) {
	input := `if x:
    if y:
        z = 1
    w = 2
v = 3`

	l := New(input)

	expected := []TokenType{
		IF, IDENT, COLON, NEWLINE,
		INDENT, IF, IDENT, COLON, NEWLINE,
		INDENT, IDENT, ASSIGN, INT, NEWLINE,
		DEDENT, IDENT, ASSIGN, INT, NEWLINE,
		DEDENT, IDENT, ASSIGN, INT,
		EOF,
	}

	for i, expectedType := range expected {
		tok := l.NextToken()
		assert.Equal(t, expectedType, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, expectedType)
	}
}

func TestNextToken_ModuleHeaders(t *testing.T) {
	tests := []struct {
		input           string
		expectedType    TokenType
		expectedLiteral string
	}{
		{"===SOURCE: mymodule===", SOURCE, "mymodule"},
		{"===USE: lib/utils===", USE, "lib/utils"},
		{"===MAIN===", MAIN, "MAIN"},
		{"===EXPORT===", EXPORT, "EXPORT"},
		{"===INPUT===", INPUT, ""},
		{"===OUTPUT===", OUTPUT, ""},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()
		assert.Equal(t, tt.expectedType, tok.Type, "input %q - tokentype wrong", tt.input)
		assert.Equal(t, tt.expectedLiteral, tok.Literal, "input %q - literal wrong", tt.input)
	}
}

func TestNextToken_SimpleScript(t *testing.T) {
	input := `x = 42
for i in range(10):
    print(i)
emit(x)`

	l := New(input)
	tokens := l.Tokenize()

	require.NotEmpty(t, tokens)
	assert.Equal(t, EOF, tokens[len(tokens)-1].Type)

	// Check we got the expected keywords
	tokenTypes := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		tokenTypes[i] = tok.Type
	}

	assert.Contains(t, tokenTypes, FOR)
	assert.Contains(t, tokenTypes, IN)
	assert.Contains(t, tokenTypes, RANGE)
	assert.Contains(t, tokenTypes, INDENT)
	assert.Contains(t, tokenTypes, DEDENT)
	assert.Contains(t, tokenTypes, EMIT)
}

func TestNextToken_ForWithModifiers(t *testing.T) {
	input := `for item in items with limit(100), rate(10/s):
    process(item)`

	l := New(input)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{FOR, "for"},
		{IDENT, "item"},
		{IN, "in"},
		{IDENT, "items"},
		{WITH, "with"},
		{LIMIT, "limit"},
		{LPAREN, "("},
		{INT, "100"},
		{RPAREN, ")"},
		{COMMA, ","},
		{RATE, "rate"},
		{LPAREN, "("},
		{INT, "10"},
		{SLASH, "/"},
		{IDENT, "s"},
		{RPAREN, ")"},
		{COLON, ":"},
		{NEWLINE, "\n"},
		{INDENT, ""},
		{IDENT, "process"},
		{LPAREN, "("},
		{IDENT, "item"},
		{RPAREN, ")"},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		assert.Equal(t, exp.typ, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, exp.typ)
		if exp.lit != "" {
			assert.Equal(t, exp.lit, tok.Literal, "test[%d] - literal wrong", i)
		}
	}
}

func TestNextToken_LambdaExpression(t *testing.T) {
	input := `items.map(x -> x * 2)`

	l := New(input)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{IDENT, "items"},
		{DOT, "."},
		{IDENT, "map"},
		{LPAREN, "("},
		{IDENT, "x"},
		{ARROW, "->"},
		{IDENT, "x"},
		{STAR, "*"},
		{INT, "2"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		assert.Equal(t, exp.typ, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, exp.typ)
		assert.Equal(t, exp.lit, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_MatchExpression(t *testing.T) {
	input := `match status:
    200 -> "ok"
    404 -> "not found"
    _ -> "error"`

	l := New(input)
	tokens := l.Tokenize()

	// Find the match keyword and arrows
	var foundMatch, foundArrows int
	for _, tok := range tokens {
		if tok.Type == MATCH {
			foundMatch++
		}
		if tok.Type == ARROW {
			foundArrows++
		}
	}

	assert.Equal(t, 1, foundMatch, "should have one match keyword")
	assert.Equal(t, 3, foundArrows, "should have three arrows")
}

func TestNextToken_OptionalChaining(t *testing.T) {
	input := `data?.user?.profile?.name`

	l := New(input)

	expected := []struct {
		typ TokenType
		lit string
	}{
		{IDENT, "data"},
		{OPTDOT, "?."},
		{IDENT, "user"},
		{OPTDOT, "?."},
		{IDENT, "profile"},
		{OPTDOT, "?."},
		{IDENT, "name"},
		{EOF, ""},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		assert.Equal(t, exp.typ, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, exp.typ)
		assert.Equal(t, exp.lit, tok.Literal, "test[%d] - literal wrong", i)
	}
}

func TestNextToken_PipelineExpression(t *testing.T) {
	input := `items | filter(x -> x > 0) | map(x -> x * 2)`

	l := New(input)

	var pipeCount int
	for {
		tok := l.NextToken()
		if tok.Type == EOF {
			break
		}
		if tok.Type == PIPE {
			pipeCount++
		}
	}

	assert.Equal(t, 2, pipeCount, "should have two pipe operators")
}

func TestNextToken_LLMCall(t *testing.T) {
	input := `result = llm.call(
    prompt: "Hello {name}",
    schema: {answer: string}
)`

	l := New(input)

	expected := []TokenType{
		IDENT, ASSIGN, IDENT, DOT, IDENT, LPAREN, NEWLINE,
		INDENT, IDENT, COLON, STRING, COMMA, NEWLINE,
		IDENT, COLON, LBRACE, IDENT, COLON, IDENT, RBRACE, NEWLINE,
		DEDENT, RPAREN,
		EOF,
	}

	for i, expectedType := range expected {
		tok := l.NextToken()
		assert.Equal(t, expectedType, tok.Type, "test[%d] - got %s, expected %s", i, tok.Type, expectedType)
	}
}

func TestNextToken_BlankLines(t *testing.T) {
	input := `x = 1

y = 2


z = 3`

	l := New(input)

	// Should handle blank lines gracefully
	tokens := l.Tokenize()

	var identCount int
	for _, tok := range tokens {
		if tok.Type == IDENT {
			identCount++
		}
	}

	assert.Equal(t, 3, identCount, "should have 3 identifiers (x, y, z)")
}

func TestNextToken_Position(t *testing.T) {
	input := `x = 1
y = 2`

	l := New(input)

	// x
	tok := l.NextToken()
	assert.Equal(t, 1, tok.Line)
	assert.Equal(t, 1, tok.Column)

	// =
	tok = l.NextToken()
	assert.Equal(t, 1, tok.Line)

	// 1
	tok = l.NextToken()
	assert.Equal(t, 1, tok.Line)

	// NEWLINE
	tok = l.NextToken()
	assert.Equal(t, NEWLINE, tok.Type)

	// y (should be on line 2)
	tok = l.NextToken()
	assert.Equal(t, IDENT, tok.Type)
	assert.Equal(t, "y", tok.Literal)
	assert.Equal(t, 2, tok.Line)
}

func TestTokenize(t *testing.T) {
	input := `def hello(name):
    return "Hello, " + name`

	l := New(input)
	tokens := l.Tokenize()

	require.NotEmpty(t, tokens)
	assert.Equal(t, DEF, tokens[0].Type)
	assert.Equal(t, EOF, tokens[len(tokens)-1].Type)
}

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		ident    string
		expected TokenType
	}{
		{"if", IF},
		{"else", ELSE},
		{"for", FOR},
		{"def", DEF},
		{"true", TRUE},
		{"false", FALSE},
		{"none", NONE},
		{"foo", IDENT},
		{"myVar", IDENT},
		{"_private", IDENT},
	}

	for _, tt := range tests {
		result := LookupIdent(tt.ident)
		assert.Equal(t, tt.expected, result, "LookupIdent(%q) wrong", tt.ident)
	}
}

func TestTokenType_String(t *testing.T) {
	assert.Equal(t, "if", IF.String())
	assert.Equal(t, "+", PLUS.String())
	assert.Equal(t, "IDENT", IDENT.String())
	assert.Equal(t, "EOF", EOF.String())
}
