// Package lexer provides tokenization for the SLOP language.
package lexer

import "fmt"

// TokenType represents the type of a token.
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	NEWLINE
	INDENT
	DEDENT

	// Literals
	IDENT  // identifier
	INT    // integer literal
	FLOAT  // float literal
	STRING // string literal

	// Operators
	PLUS      // +
	MINUS     // -
	STAR      // *
	SLASH     // /
	PERCENT   // %
	STARSTAR  // **
	EQ        // ==
	NE        // !=
	LT        // <
	GT        // >
	LE        // <=
	GE        // >=
	ASSIGN    // =
	PLUSEQ    // +=
	MINUSEQ   // -=
	STAREQ    // *=
	SLASHEQ   // /=
	PIPE      // |
	ARROW     // ->
	DOT       // .
	OPTDOT    // ?.
	OPTLBRACK // ?[
	COLON     // :
	COMMA     // ,

	// Delimiters
	LPAREN   // (
	RPAREN   // )
	LBRACK   // [
	RBRACK   // ]
	LBRACE   // {
	RBRACE   // }

	// Keywords
	IF
	ELIF
	ELSE
	FOR
	IN
	WITH
	MATCH
	DEF
	RETURN
	EMIT
	STOP
	AND
	OR
	NOT
	TRUE
	FALSE
	NONE
	RANGE
	LIMIT
	RATE
	PARALLEL
	TIMEOUT
	TRY
	CATCH
	BREAK
	CONTINUE

	// Module keywords
	SOURCE // ===SOURCE:
	USE    // ===USE:
	MAIN   // ===MAIN===
	EXPORT // ===EXPORT===
	INPUT  // ===INPUT===
	OUTPUT // ===OUTPUT===
)

var tokenNames = map[TokenType]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	NEWLINE: "NEWLINE",
	INDENT:  "INDENT",
	DEDENT:  "DEDENT",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	STRING: "STRING",

	PLUS:      "+",
	MINUS:     "-",
	STAR:      "*",
	SLASH:     "/",
	PERCENT:   "%",
	STARSTAR:  "**",
	EQ:        "==",
	NE:        "!=",
	LT:        "<",
	GT:        ">",
	LE:        "<=",
	GE:        ">=",
	ASSIGN:    "=",
	PLUSEQ:    "+=",
	MINUSEQ:   "-=",
	STAREQ:    "*=",
	SLASHEQ:   "/=",
	PIPE:      "|",
	ARROW:     "->",
	DOT:       ".",
	OPTDOT:    "?.",
	OPTLBRACK: "?[",
	COLON:     ":",
	COMMA:     ",",

	LPAREN: "(",
	RPAREN: ")",
	LBRACK: "[",
	RBRACK: "]",
	LBRACE: "{",
	RBRACE: "}",

	IF:       "if",
	ELIF:     "elif",
	ELSE:     "else",
	FOR:      "for",
	IN:       "in",
	WITH:     "with",
	MATCH:    "match",
	DEF:      "def",
	RETURN:   "return",
	EMIT:     "emit",
	STOP:     "stop",
	AND:      "and",
	OR:       "or",
	NOT:      "not",
	TRUE:     "true",
	FALSE:    "false",
	NONE:     "none",
	RANGE:    "range",
	LIMIT:    "limit",
	RATE:     "rate",
	PARALLEL: "parallel",
	TIMEOUT:  "timeout",
	TRY:      "try",
	CATCH:    "catch",
	BREAK:    "break",
	CONTINUE: "continue",

	SOURCE: "SOURCE",
	USE:    "USE",
	MAIN:   "MAIN",
	EXPORT: "EXPORT",
	INPUT:  "INPUT",
	OUTPUT: "OUTPUT",
}

var keywords = map[string]TokenType{
	"if":       IF,
	"elif":     ELIF,
	"else":     ELSE,
	"for":      FOR,
	"in":       IN,
	"with":     WITH,
	"match":    MATCH,
	"def":      DEF,
	"return":   RETURN,
	"emit":     EMIT,
	"stop":     STOP,
	"and":      AND,
	"or":       OR,
	"not":      NOT,
	"true":     TRUE,
	"false":    FALSE,
	"none":     NONE,
	"range":    RANGE,
	"limit":    LIMIT,
	"rate":     RATE,
	"parallel": PARALLEL,
	"timeout":  TIMEOUT,
	"try":      TRY,
	"catch":    CATCH,
	"break":    BREAK,
	"continue": CONTINUE,
}

// LookupIdent returns the token type for an identifier.
// If the identifier is a keyword, it returns the keyword token type.
// Otherwise, it returns IDENT.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// Token represents a lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return fmt.Sprintf("Token{%s, %q, %d:%d}", t.Type, t.Literal, t.Line, t.Column)
}

// Position represents a position in the source code.
type Position struct {
	Line   int
	Column int
	Offset int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}
