package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenHandler is a function that handles tokenization for a specific character.
type TokenHandler func(l *Lexer) Token

// Lexer tokenizes SLOP source code.
type Lexer struct {
	input   string
	pos     int  // current position in input
	readPos int  // reading position (after current char)
	ch      rune // current character
	line    int
	column  int

	// Indentation tracking
	indentStack []int  // stack of indentation levels
	pendingToks []Token // tokens to emit before next read
	atLineStart bool   // whether we're at the start of a line
}

// tokenHandlers maps characters to their handler functions.
var tokenHandlers = map[rune]TokenHandler{
	'%': handlePercent,
	'|': handlePipe,
	':': handleColon,
	',': handleComma,
	'(': handleLParen,
	')': handleRParen,
	'[': handleLBrack,
	']': handleRBrack,
	'{': handleLBrace,
	'}': handleRBrace,
	'+': handlePlus,
	'-': handleMinus,
	'*': handleStar,
	'/': handleSlash,
	'.': handleDot,
	'?': handleQuestion,
	'=': handleEqual,
	'!': handleBang,
	'<': handleLess,
	'>': handleGreater,
}

// New creates a new Lexer for the given input.
func New(input string) *Lexer {
	l := &Lexer{
		input:       input,
		line:        1,
		column:      0,
		indentStack: []int{0},
		atLineStart: true,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch, _ = utf8.DecodeRuneInString(l.input[l.readPos:])
	}
	l.pos = l.readPos
	if l.ch != 0 {
		size := utf8.RuneLen(l.ch)
		l.readPos += size
	}
	l.column++
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return ch
}

func (l *Lexer) peekCharN(n int) rune {
	pos := l.readPos
	for i := 0; i < n-1 && pos < len(l.input); i++ {
		_, size := utf8.DecodeRuneInString(l.input[pos:])
		pos += size
	}
	if pos >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[pos:])
	return ch
}

// Token handler functions

func handlePercent(l *Lexer) Token {
	tok := Token{Type: PERCENT, Literal: "%", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handlePipe(l *Lexer) Token {
	tok := Token{Type: PIPE, Literal: "|", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleColon(l *Lexer) Token {
	tok := Token{Type: COLON, Literal: ":", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleComma(l *Lexer) Token {
	tok := Token{Type: COMMA, Literal: ",", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleLParen(l *Lexer) Token {
	tok := Token{Type: LPAREN, Literal: "(", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleRParen(l *Lexer) Token {
	tok := Token{Type: RPAREN, Literal: ")", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleLBrack(l *Lexer) Token {
	tok := Token{Type: LBRACK, Literal: "[", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleRBrack(l *Lexer) Token {
	tok := Token{Type: RBRACK, Literal: "]", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleLBrace(l *Lexer) Token {
	tok := Token{Type: LBRACE, Literal: "{", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleRBrace(l *Lexer) Token {
	tok := Token{Type: RBRACE, Literal: "}", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handlePlus(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		tok.Type = PLUSEQ
		tok.Literal = "+="
	} else {
		tok.Type = PLUS
		tok.Literal = "+"
	}
	l.readChar()
	return tok
}

func handleMinus(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '>' {
		l.readChar()
		tok.Type = ARROW
		tok.Literal = "->"
		l.readChar()
	} else if l.peekChar() == '=' {
		l.readChar()
		tok.Type = MINUSEQ
		tok.Literal = "-="
		l.readChar()
	} else {
		tok.Type = MINUS
		tok.Literal = "-"
		l.readChar()
	}
	return tok
}

func handleStar(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '*' {
		l.readChar()
		tok.Type = STARSTAR
		tok.Literal = "**"
		l.readChar()
	} else if l.peekChar() == '=' {
		l.readChar()
		tok.Type = STAREQ
		tok.Literal = "*="
		l.readChar()
	} else {
		tok.Type = STAR
		tok.Literal = "*"
		l.readChar()
	}
	return tok
}

func handleSlash(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		tok.Type = SLASHEQ
		tok.Literal = "/="
	} else {
		tok.Type = SLASH
		tok.Literal = "/"
	}
	l.readChar()
	return tok
}

func handleDot(l *Lexer) Token {
	tok := Token{Type: DOT, Literal: ".", Line: l.line, Column: l.column}
	l.readChar()
	return tok
}

func handleQuestion(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '.' {
		l.readChar()
		tok.Type = OPTDOT
		tok.Literal = "?."
		l.readChar()
	} else if l.peekChar() == '[' {
		l.readChar()
		tok.Type = OPTLBRACK
		tok.Literal = "?["
		l.readChar()
	} else {
		tok.Type = ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
	}
	return tok
}

func handleEqual(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		if l.peekChar() == '=' {
			// Check for module headers like ===SOURCE:
			return l.readModuleHeader()
		}
		tok.Type = EQ
		tok.Literal = "=="
	} else {
		tok.Type = ASSIGN
		tok.Literal = "="
	}
	l.readChar()
	return tok
}

func handleBang(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		tok.Type = NE
		tok.Literal = "!="
		l.readChar()
	} else {
		tok.Type = ILLEGAL
		tok.Literal = string(l.ch)
		l.readChar()
	}
	return tok
}

func handleLess(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		tok.Type = LE
		tok.Literal = "<="
	} else {
		tok.Type = LT
		tok.Literal = "<"
	}
	l.readChar()
	return tok
}

func handleGreater(l *Lexer) Token {
	tok := Token{Line: l.line, Column: l.column}
	if l.peekChar() == '=' {
		l.readChar()
		tok.Type = GE
		tok.Literal = ">="
	} else {
		tok.Type = GT
		tok.Literal = ">"
	}
	l.readChar()
	return tok
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	// Return pending tokens first (from indent/dedent processing)
	if len(l.pendingToks) > 0 {
		tok := l.pendingToks[0]
		l.pendingToks = l.pendingToks[1:]
		return tok
	}

	// Handle indentation at line start
	if l.atLineStart {
		l.atLineStart = false
		if toks := l.processIndentation(); len(toks) > 0 {
			tok := toks[0]
			l.pendingToks = append(l.pendingToks, toks[1:]...)
			return tok
		}
	}

	l.skipWhitespace()

	// Check if there's a registered handler for this character
	if handler, ok := tokenHandlers[l.ch]; ok {
		return handler(l)
	}

	var tok Token
	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case 0:
		// At EOF, emit DEDENT for each remaining indent level
		if len(l.indentStack) > 1 {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			tok.Type = DEDENT
			tok.Literal = ""
			return tok
		}
		tok.Type = EOF
		tok.Literal = ""

	case '\n':
		tok.Type = NEWLINE
		tok.Literal = "\n"
		l.readChar()
		l.line++
		l.column = 0
		l.atLineStart = true

	case '#':
		l.skipComment()
		return l.NextToken()

	case '"', '\'':
		tok.Type = STRING
		tok.Literal = l.readString(l.ch)

	default:
		if isDigit(l.ch) {
			lit, isFloat := l.readNumber()
			if isFloat {
				tok.Type = FLOAT
			} else {
				tok.Type = INT
			}
			tok.Literal = lit
		} else if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
		} else {
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
			l.readChar()
		}
	}

	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) processIndentation() []Token {
	// Skip blank lines and comments
	for {
		indent := 0
		startLine := l.line
		startCol := l.column

		// Count leading spaces
		for l.ch == ' ' {
			indent++
			l.readChar()
		}
		// Handle tabs (count as 4 spaces each)
		for l.ch == '\t' {
			indent += 4
			l.readChar()
		}

		// Skip blank lines
		if l.ch == '\n' {
			l.readChar()
			l.line++
			l.column = 0
			continue
		}

		// Skip comment lines
		if l.ch == '#' {
			l.skipComment()
			if l.ch == '\n' {
				l.readChar()
				l.line++
				l.column = 0
				continue
			}
			break
		}

		// Process indentation change
		currentIndent := l.indentStack[len(l.indentStack)-1]
		var toks []Token

		if indent > currentIndent {
			l.indentStack = append(l.indentStack, indent)
			toks = append(toks, Token{Type: INDENT, Literal: "", Line: startLine, Column: startCol})
		} else if indent < currentIndent {
			for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
				l.indentStack = l.indentStack[:len(l.indentStack)-1]
				toks = append(toks, Token{Type: DEDENT, Literal: "", Line: startLine, Column: startCol})
			}
		}

		return toks
	}
	return nil
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	// If the word so far is a keyword, don't consume hyphens.
	// Keywords (for, if, true, not, etc.) should never start a hyphenated identifier.
	word := l.input[start:l.pos]
	if _, isKeyword := keywords[word]; isKeyword {
		return word
	}
	// Context-aware hyphen: consume '-' followed by a letter as part of the identifier.
	// "dart-query" → single IDENT. "a-1" → IDENT MINUS INT. "a - b" → IDENT MINUS IDENT.
	for l.ch == '-' && isLetter(l.peekChar()) {
		l.readChar() // consume '-'
		for isLetter(l.ch) || isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readNumber() (string, bool) {
	start := l.pos
	isFloat := false

	// Handle underscore-separated numbers like 1_000_000
	for isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume '.'
		for isDigit(l.ch) || l.ch == '_' {
			l.readChar()
		}
	}

	// Check for exponent
	if l.ch == 'e' || l.ch == 'E' {
		isFloat = true
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Remove underscores from the literal
	lit := l.input[start:l.pos]
	lit = strings.ReplaceAll(lit, "_", "")
	return lit, isFloat
}

func (l *Lexer) readString(quote rune) string {
	var result strings.Builder
	l.readChar() // skip opening quote

	for {
		if l.ch == quote {
			l.readChar() // skip closing quote
			break
		}
		if l.ch == 0 || l.ch == '\n' {
			// Unterminated string
			break
		}
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case 'r':
				result.WriteRune('\r')
			case '\\':
				result.WriteRune('\\')
			case '"':
				result.WriteRune('"')
			case '\'':
				result.WriteRune('\'')
			case '{':
				result.WriteRune('{')
			case '}':
				result.WriteRune('}')
			default:
				result.WriteRune('\\')
				result.WriteRune(l.ch)
			}
		} else {
			result.WriteRune(l.ch)
		}
		l.readChar()
	}

	return result.String()
}

func (l *Lexer) readModuleHeader() Token {
	tok := Token{Line: l.line, Column: l.column - 2} // -2 for the == already consumed

	// We've seen "==" and peeked "=", so consume the third "=" and move past it
	l.readChar() // now l.ch is the third '='
	l.readChar() // now l.ch is the first char after "==="

	// Read until we find "===" again or end of line
	var content strings.Builder
	for l.ch != 0 && l.ch != '\n' {
		if l.ch == '=' && l.peekChar() == '=' && l.peekCharN(2) == '=' {
			// End of header
			l.readChar() // =
			l.readChar() // =
			l.readChar() // =
			break
		}
		content.WriteRune(l.ch)
		l.readChar()
	}

	header := strings.TrimSpace(content.String())

	// Parse the header type
	switch {
	case strings.HasPrefix(header, "SOURCE:"):
		tok.Type = SOURCE
		tok.Literal = strings.TrimPrefix(header, "SOURCE:")
		tok.Literal = strings.TrimSpace(tok.Literal)
	case strings.HasPrefix(header, "USE:"):
		tok.Type = USE
		tok.Literal = strings.TrimPrefix(header, "USE:")
		tok.Literal = strings.TrimSpace(tok.Literal)
	case header == "MAIN":
		tok.Type = MAIN
		tok.Literal = "MAIN"
	case header == "EXPORT":
		tok.Type = EXPORT
		tok.Literal = "EXPORT"
	case strings.HasPrefix(header, "INPUT"):
		tok.Type = INPUT
		tok.Literal = strings.TrimPrefix(header, "INPUT")
		tok.Literal = strings.TrimSpace(tok.Literal)
	case strings.HasPrefix(header, "OUTPUT"):
		tok.Type = OUTPUT
		tok.Literal = strings.TrimPrefix(header, "OUTPUT")
		tok.Literal = strings.TrimSpace(tok.Literal)
	default:
		tok.Type = ILLEGAL
		tok.Literal = "===" + header + "==="
	}

	return tok
}

// Tokenize returns all tokens from the input.
func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}
