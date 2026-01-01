package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anthropics/slop/internal/ast"
	"github.com/anthropics/slop/internal/lexer"
)

// Precedence levels for Pratt parsing.
const (
	_ int = iota
	LOWEST
	TERNARY     // if else
	PIPELINE    // |
	OR          // or
	AND         // and
	NOT         // not
	COMPARISON  // == != < > <= >=
	MEMBERSHIP  // in, not in
	ADDITION    // + -
	MULTIPLY    // * / %
	POWER       // **
	PREFIX      // -x, not x
	CALL        // fn()
	MEMBER      // .x, [x], ?.x, ?[x]
	LAMBDA      // x -> expr
)

var precedences = map[lexer.TokenType]int{
	lexer.IF:       TERNARY,
	lexer.PIPE:     PIPELINE,
	lexer.OR:       OR,
	lexer.AND:      AND,
	lexer.NOT:      NOT,
	lexer.EQ:       COMPARISON,
	lexer.NE:       COMPARISON,
	lexer.LT:       COMPARISON,
	lexer.GT:       COMPARISON,
	lexer.LE:       COMPARISON,
	lexer.GE:       COMPARISON,
	lexer.IN:       MEMBERSHIP,
	lexer.PLUS:     ADDITION,
	lexer.MINUS:    ADDITION,
	lexer.STAR:     MULTIPLY,
	lexer.SLASH:    MULTIPLY,
	lexer.PERCENT:  MULTIPLY,
	lexer.STARSTAR: POWER,
	lexer.LPAREN:   CALL,
	lexer.LBRACK:   MEMBER,
	lexer.DOT:      MEMBER,
	lexer.OPTDOT:   MEMBER,
	lexer.OPTLBRACK: MEMBER,
	lexer.ARROW:    LAMBDA,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

// Parser parses SLOP source code into an AST.
type Parser struct {
	l      *lexer.Lexer
	errors Errors

	curToken  lexer.Token
	peekToken lexer.Token

	prefixParseFns map[lexer.TokenType]prefixParseFn
	infixParseFns  map[lexer.TokenType]infixParseFn
}

// New creates a new Parser for the given lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		errors:         Errors{},
		prefixParseFns: make(map[lexer.TokenType]prefixParseFn),
		infixParseFns:  make(map[lexer.TokenType]infixParseFn),
	}

	// Register prefix parse functions
	p.registerPrefix(lexer.IDENT, p.parseIdentifier)
	p.registerPrefix(lexer.INT, p.parseIntegerLiteral)
	p.registerPrefix(lexer.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(lexer.STRING, p.parseStringLiteral)
	p.registerPrefix(lexer.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(lexer.NONE, p.parseNoneLiteral)
	p.registerPrefix(lexer.MINUS, p.parsePrefixExpression)
	p.registerPrefix(lexer.NOT, p.parsePrefixExpression)
	p.registerPrefix(lexer.LPAREN, p.parseGroupedOrLambda)
	p.registerPrefix(lexer.LBRACK, p.parseListLiteral)
	p.registerPrefix(lexer.LBRACE, p.parseMapOrSetLiteral)
	p.registerPrefix(lexer.RANGE, p.parseRangeExpression)
	p.registerPrefix(lexer.MATCH, p.parseMatchExpression)

	// Register infix parse functions
	p.registerInfix(lexer.PLUS, p.parseInfixExpression)
	p.registerInfix(lexer.MINUS, p.parseInfixExpression)
	p.registerInfix(lexer.STAR, p.parseInfixExpression)
	p.registerInfix(lexer.SLASH, p.parseInfixExpression)
	p.registerInfix(lexer.PERCENT, p.parseInfixExpression)
	p.registerInfix(lexer.STARSTAR, p.parseInfixExpression)
	p.registerInfix(lexer.EQ, p.parseInfixExpression)
	p.registerInfix(lexer.NE, p.parseInfixExpression)
	p.registerInfix(lexer.LT, p.parseInfixExpression)
	p.registerInfix(lexer.GT, p.parseInfixExpression)
	p.registerInfix(lexer.LE, p.parseInfixExpression)
	p.registerInfix(lexer.GE, p.parseInfixExpression)
	p.registerInfix(lexer.AND, p.parseInfixExpression)
	p.registerInfix(lexer.OR, p.parseInfixExpression)
	p.registerInfix(lexer.IN, p.parseInfixExpression)
	p.registerInfix(lexer.PIPE, p.parsePipelineExpression)
	p.registerInfix(lexer.LPAREN, p.parseCallExpression)
	p.registerInfix(lexer.LBRACK, p.parseIndexExpression)
	p.registerInfix(lexer.DOT, p.parseMemberExpression)
	p.registerInfix(lexer.OPTDOT, p.parseMemberExpression)
	p.registerInfix(lexer.OPTLBRACK, p.parseIndexExpression)
	p.registerInfix(lexer.IF, p.parseTernaryExpression)
	p.registerInfix(lexer.ARROW, p.parseLambdaInfix)

	// Read two tokens to set curToken and peekToken
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType lexer.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType lexer.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// isIdentOrKeyword checks if current token is an identifier or a keyword that
// can be used as a kwarg name (like limit, rate, etc.)
func (p *Parser) isIdentOrKeyword() bool {
	switch p.curToken.Type {
	case lexer.IDENT,
		lexer.LIMIT, lexer.RATE, lexer.PARALLEL, lexer.TIMEOUT,
		lexer.TRUE, lexer.FALSE, lexer.NONE,
		lexer.AND, lexer.OR, lexer.NOT,
		lexer.IF, lexer.ELIF, lexer.ELSE,
		lexer.FOR, lexer.IN, lexer.WITH,
		lexer.MATCH, lexer.DEF, lexer.RETURN,
		lexer.EMIT, lexer.STOP,
		lexer.TRY, lexer.CATCH,
		lexer.BREAK, lexer.CONTINUE,
		lexer.RANGE:
		return true
	default:
		return false
	}
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	p.errors = append(p.errors, &Error{
		Token:   p.peekToken,
		Message: fmt.Sprintf("expected %s, got %s", t, p.peekToken.Type),
	})
}

func (p *Parser) curError(msg string) {
	p.errors = append(p.errors, &Error{
		Token:   p.curToken,
		Message: msg,
	})
}

func (p *Parser) noPrefixParseFnError(t lexer.TokenType) {
	p.errors = append(p.errors, &Error{
		Token:   p.curToken,
		Message: fmt.Sprintf("no prefix parse function for %s", t),
	})
}

// Errors returns any parsing errors.
func (p *Parser) Errors() Errors {
	return p.errors
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

// ParseProgram parses the entire program.
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		Statements: []ast.Statement{},
		Modules:    []*ast.Module{},
	}

	for !p.curTokenIs(lexer.EOF) {
		// Skip newlines at top level
		if p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
			continue
		}

		// Check for module definitions
		if p.curTokenIs(lexer.SOURCE) || p.curTokenIs(lexer.USE) || p.curTokenIs(lexer.MAIN) {
			module := p.parseModule()
			if module != nil {
				program.Modules = append(program.Modules, module)
			}
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// parseModule parses a module definition block.
func (p *Parser) parseModule() *ast.Module {
	module := &ast.Module{
		Pos: ast.Position{Line: p.curToken.Line, Column: p.curToken.Column},
	}

	switch p.curToken.Type {
	case lexer.SOURCE:
		module.Type = "SOURCE"
		module.Name = p.curToken.Literal
		p.nextToken() // Skip past SOURCE token

		// Skip newlines after the SOURCE header
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

		// Parse optional header lines (id:, uses:, provides:)
		module.Uses = make(map[string]string)
		for p.curTokenIs(lexer.IDENT) {
			headerName := p.curToken.Literal
			if !p.expectPeek(lexer.COLON) {
				break
			}
			p.nextToken() // move past colon

			switch headerName {
			case "id":
				if p.curTokenIs(lexer.STRING) {
					module.ID = p.curToken.Literal
				}
			case "uses":
				// Parse uses: {local: "id", ...}
				if p.curTokenIs(lexer.LBRACE) {
					module.Uses = p.parseUsesBlock()
				}
			case "provides":
				// Parse provides: [name1, name2, ...]
				if p.curTokenIs(lexer.LBRACK) {
					module.Provides = p.parseProvidesBlock()
				}
			}
			p.nextToken()
			// Skip newlines
			for p.curTokenIs(lexer.NEWLINE) {
				p.nextToken()
			}
		}

		// Look for --- separator
		if p.curToken.Literal == "-" {
			p.skipDashSeparator()
		}

		// Parse body until next module or EOF
		module.Body = p.parseModuleBody()

	case lexer.USE:
		module.Type = "USE"
		module.WithClauses = make(map[string]string)

		// Parse the literal which may contain "with {...}" clause
		// Format: "module/path" or "module/path with {dep: other}"
		literal := p.curToken.Literal
		if idx := strings.Index(literal, " with "); idx != -1 {
			module.Name = strings.TrimSpace(literal[:idx])
			// Parse the with clause from the remaining literal
			withPart := strings.TrimSpace(literal[idx+6:]) // skip " with "
			module.WithClauses = p.parseWithClauseFromString(withPart)
		} else {
			module.Name = literal
		}
		p.nextToken()

		// Skip newlines
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

	case lexer.MAIN:
		module.Type = "MAIN"
		module.Name = "MAIN"
		p.nextToken()

		// Skip newlines
		for p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

		// Parse body until next module or EOF
		module.Body = p.parseModuleBody()
	}

	return module
}

// parseWithClauseFromString parses a with clause like "{dep: other}" from a string.
func (p *Parser) parseWithClauseFromString(s string) map[string]string {
	result := make(map[string]string)

	// Remove braces
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return result
	}
	s = s[1 : len(s)-1]

	// Split by comma
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, ":", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			result[key] = val
		}
	}

	return result
}

// parseUsesBlock parses {local: "id", ...}
func (p *Parser) parseUsesBlock() map[string]string {
	uses := make(map[string]string)

	if !p.curTokenIs(lexer.LBRACE) {
		return uses
	}
	p.nextToken() // skip {

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// Skip newlines and commas
		if p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.COMMA) {
			p.nextToken()
			continue
		}

		if p.curTokenIs(lexer.IDENT) {
			localName := p.curToken.Literal
			if p.expectPeek(lexer.COLON) {
				p.nextToken()
				if p.curTokenIs(lexer.STRING) {
					uses[localName] = p.curToken.Literal
				}
			}
		}
		p.nextToken()
	}

	return uses
}

// parseProvidesBlock parses [name1, name2, ...]
func (p *Parser) parseProvidesBlock() []string {
	provides := []string{}

	if !p.curTokenIs(lexer.LBRACK) {
		return provides
	}
	p.nextToken() // skip [

	for !p.curTokenIs(lexer.RBRACK) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) {
			provides = append(provides, p.curToken.Literal)
		}
		p.nextToken()
	}

	return provides
}

// skipDashSeparator skips a --- separator line.
func (p *Parser) skipDashSeparator() {
	for p.curToken.Literal == "-" {
		p.nextToken()
	}
	// Skip newlines
	for p.curTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
}

// parseModuleBody parses statements until the next module definition or EOF.
func (p *Parser) parseModuleBody() []ast.Statement {
	var statements []ast.Statement

	for !p.curTokenIs(lexer.EOF) {
		// Check for next module definition
		if p.curTokenIs(lexer.SOURCE) || p.curTokenIs(lexer.USE) || p.curTokenIs(lexer.MAIN) {
			break
		}

		// Skip newlines
		if p.curTokenIs(lexer.NEWLINE) {
			p.nextToken()
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.MATCH:
		return p.parseMatchStatement()
	case lexer.DEF:
		return p.parseDefStatement()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.EMIT:
		return p.parseEmitStatement()
	case lexer.STOP:
		return p.parseStopStatement()
	case lexer.TRY:
		return p.parseTryStatement()
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.CONTINUE:
		return p.parseContinueStatement()
	default:
		return p.parseAssignmentOrExpressionStatement()
	}
}

func (p *Parser) parseAssignmentOrExpressionStatement() ast.Statement {
	// Parse the first expression
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	// Check if this is an assignment
	if p.peekTokenIs(lexer.ASSIGN) || p.peekTokenIs(lexer.PLUSEQ) ||
		p.peekTokenIs(lexer.MINUSEQ) || p.peekTokenIs(lexer.STAREQ) ||
		p.peekTokenIs(lexer.SLASHEQ) {

		stmt := &ast.AssignStatement{
			Token:   p.curToken,
			Targets: []ast.Expression{expr},
		}

		p.nextToken() // consume the operator
		stmt.Operator = p.curToken.Literal

		p.nextToken() // move to value
		stmt.Value = p.parseExpression(LOWEST)

		// Skip trailing newline
		if p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}

		return stmt
	}

	// It's an expression statement
	stmt := &ast.ExpressionStatement{
		Token:      p.curToken,
		Expression: expr,
	}

	// Skip trailing newline
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(lexer.NEWLINE) && !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

// ============================================================================
// Prefix Parse Functions
// ============================================================================

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.curError(fmt.Sprintf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.curError(fmt.Sprintf("could not parse %q as float", p.curToken.Literal))
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(lexer.TRUE)}
}

func (p *Parser) parseNoneLiteral() ast.Expression {
	return &ast.NoneLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseGroupedOrLambda() ast.Expression {
	// Could be (expr) or (a, b) -> expr
	startToken := p.curToken
	p.nextToken()

	// Empty parens followed by -> is a no-arg lambda
	if p.curTokenIs(lexer.RPAREN) && p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // consume )
		p.nextToken() // consume ->
		body := p.parseExpression(LOWEST)
		return &ast.LambdaExpression{
			Token:      startToken,
			Parameters: []*ast.Identifier{},
			Body:       body,
		}
	}

	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}

	// Check if this is a tuple (for lambda parameters)
	if p.peekTokenIs(lexer.COMMA) {
		params := []*ast.Identifier{}

		// First must be identifier
		if ident, ok := first.(*ast.Identifier); ok {
			params = append(params, ident)
		} else {
			p.curError("lambda parameters must be identifiers")
			return nil
		}

		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // consume comma
			p.nextToken() // move to next param
			if !p.curTokenIs(lexer.IDENT) {
				p.curError("expected identifier in lambda parameters")
				return nil
			}
			params = append(params, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}

		if !p.expectPeek(lexer.ARROW) {
			return nil
		}

		p.nextToken()
		body := p.parseExpression(LOWEST)
		return &ast.LambdaExpression{
			Token:      startToken,
			Parameters: params,
			Body:       body,
		}
	}

	// Single expression - check if it's a single-param lambda
	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // consume )
		if p.peekTokenIs(lexer.ARROW) {
			// It's a single-param lambda: (x) -> expr
			ident, ok := first.(*ast.Identifier)
			if !ok {
				p.curError("lambda parameter must be an identifier")
				return nil
			}
			p.nextToken() // consume ->
			p.nextToken()
			body := p.parseExpression(LOWEST)
			return &ast.LambdaExpression{
				Token:      startToken,
				Parameters: []*ast.Identifier{ident},
				Body:       body,
			}
		}
		// Just a grouped expression
		return first
	}

	// Must be a grouped expression - need closing paren
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return first
}

func (p *Parser) parseListLiteral() ast.Expression {
	list := &ast.ListLiteral{Token: p.curToken}
	list.Elements = p.parseExpressionList(lexer.RBRACK)
	return list
}

func (p *Parser) parseMapOrSetLiteral() ast.Expression {
	startToken := p.curToken

	if p.peekTokenIs(lexer.RBRACE) {
		// Empty map {}
		p.nextToken()
		return &ast.MapLiteral{
			Token: startToken,
			Pairs: make(map[ast.Expression]ast.Expression),
			Order: []ast.Expression{},
		}
	}

	p.nextToken()
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}

	if p.peekTokenIs(lexer.COLON) {
		// It's a map
		return p.parseMapLiteralRest(startToken, first)
	}

	// It's a set
	return p.parseSetLiteralRest(startToken, first)
}

func (p *Parser) parseMapLiteralRest(startToken lexer.Token, firstKey ast.Expression) ast.Expression {
	pairs := make(map[ast.Expression]ast.Expression)
	order := []ast.Expression{firstKey}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}
	p.nextToken()
	firstValue := p.parseExpression(LOWEST)
	pairs[firstKey] = firstValue

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // comma
		p.nextToken() // key
		key := p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		p.nextToken()
		value := p.parseExpression(LOWEST)
		pairs[key] = value
		order = append(order, key)
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return &ast.MapLiteral{
		Token: startToken,
		Pairs: pairs,
		Order: order,
	}
}

func (p *Parser) parseSetLiteralRest(startToken lexer.Token, first ast.Expression) ast.Expression {
	elements := []ast.Expression{first}

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // comma
		p.nextToken() // element
		elements = append(elements, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.RBRACE) {
		return nil
	}

	return &ast.SetLiteral{
		Token:    startToken,
		Elements: elements,
	}
}

func (p *Parser) parseRangeExpression() ast.Expression {
	tok := p.curToken

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	p.nextToken()

	first := p.parseExpression(LOWEST)

	if p.peekTokenIs(lexer.COMMA) {
		// range(start, end) or range(start, end, step)
		p.nextToken() // comma
		p.nextToken()
		second := p.parseExpression(LOWEST)

		var step ast.Expression
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			step = p.parseExpression(LOWEST)
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}

		return &ast.RangeExpression{
			Token: tok,
			Start: first,
			End:   second,
			Step:  step,
		}
	}

	// range(end)
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return &ast.RangeExpression{
		Token: tok,
		Start: nil,
		End:   first,
		Step:  nil,
	}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	tok := p.curToken
	p.nextToken()

	subject := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	// Parse match arms
	arms := []*ast.MatchArm{}

	// Skip newline and expect indent
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}

	for !p.peekTokenIs(lexer.DEDENT) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		arm := p.parseMatchArm()
		if arm != nil {
			arms = append(arms, arm)
		}
		// Skip newline after arm
		if p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
	}

	if p.peekTokenIs(lexer.DEDENT) {
		p.nextToken()
	}

	return &ast.MatchExpression{
		Token:   tok,
		Subject: subject,
		Arms:    arms,
	}
}

func (p *Parser) parseMatchArm() *ast.MatchArm {
	// Parse pattern at LAMBDA precedence so we don't consume the -> as a lambda
	pattern := p.parseExpression(LAMBDA)

	var guard ast.Expression
	if p.peekTokenIs(lexer.IF) {
		p.nextToken()
		p.nextToken()
		guard = p.parseExpression(LAMBDA)
	}

	if !p.expectPeek(lexer.ARROW) {
		return nil
	}
	p.nextToken()

	// Body can be an expression OR certain statement keywords (continue, break)
	var body ast.Expression
	switch p.curToken.Type {
	case lexer.CONTINUE:
		// Treat continue as an identifier-like expression in match context
		body = &ast.Identifier{Token: p.curToken, Value: "continue"}
	case lexer.BREAK:
		// Treat break as an identifier-like expression in match context
		body = &ast.Identifier{Token: p.curToken, Value: "break"}
	default:
		body = p.parseExpression(LOWEST)
	}

	return &ast.MatchArm{
		Pattern: pattern,
		Guard:   guard,
		Body:    body,
	}
}

// ============================================================================
// Infix Parse Functions
// ============================================================================

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	isRightAssociative := p.curToken.Type == lexer.STARSTAR
	p.nextToken()

	// Right-associative operators use precedence - 1
	if isRightAssociative {
		expression.Right = p.parseExpression(precedence - 1)
	} else {
		expression.Right = p.parseExpression(precedence)
	}

	return expression
}

func (p *Parser) parsePipelineExpression(left ast.Expression) ast.Expression {
	expression := &ast.PipelineExpression{
		Token: p.curToken,
		Left:  left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{
		Token:    p.curToken,
		Function: function,
		Kwargs:   make(map[string]ast.Expression),
	}

	exp.Arguments, exp.Kwargs = p.parseCallArguments()
	return exp
}

func (p *Parser) parseCallArguments() ([]ast.Expression, map[string]ast.Expression) {
	args := []ast.Expression{}
	kwargs := make(map[string]ast.Expression)

	// Skip any whitespace tokens inside parens
	p.skipWhitespaceTokens()

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return args, kwargs
	}

	p.nextToken()
	p.skipWhitespaceTokens()

	for {
		// Skip any whitespace at start of argument
		for p.curTokenIs(lexer.NEWLINE) || p.curTokenIs(lexer.INDENT) || p.curTokenIs(lexer.DEDENT) {
			p.nextToken()
		}

		// Check if we hit closing paren
		if p.curTokenIs(lexer.RPAREN) {
			return args, kwargs
		}

		// Check if this is a keyword argument
		// Keywords like 'limit', 'rate', etc. can also be kwarg names
		if p.isIdentOrKeyword() && p.peekTokenIs(lexer.COLON) {
			name := p.curToken.Literal
			p.nextToken() // colon
			p.nextToken() // value
			kwargs[name] = p.parseExpression(LOWEST)
		} else {
			args = append(args, p.parseExpression(LOWEST))
		}

		// Skip whitespace before comma or closing paren
		p.skipWhitespaceTokens()

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // comma
		p.nextToken() // next arg
	}

	// Skip final whitespace
	p.skipWhitespaceTokens()

	if !p.expectPeek(lexer.RPAREN) {
		return nil, nil
	}

	return args, kwargs
}

// skipWhitespaceTokens skips NEWLINE, INDENT, DEDENT tokens when inside delimiters
func (p *Parser) skipWhitespaceTokens() {
	for p.peekTokenIs(lexer.NEWLINE) || p.peekTokenIs(lexer.INDENT) || p.peekTokenIs(lexer.DEDENT) {
		p.nextToken()
	}
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	optional := p.curTokenIs(lexer.OPTLBRACK)
	tok := p.curToken

	p.nextToken()

	// Check for slice
	if p.curTokenIs(lexer.COLON) {
		return p.parseSliceExpression(left, tok, nil)
	}

	index := p.parseExpression(LOWEST)

	// Check for slice with start
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		return p.parseSliceExpression(left, tok, index)
	}

	if !p.expectPeek(lexer.RBRACK) {
		return nil
	}

	return &ast.IndexExpression{
		Token:    tok,
		Left:     left,
		Index:    index,
		Optional: optional,
	}
}

func (p *Parser) parseSliceExpression(left ast.Expression, tok lexer.Token, start ast.Expression) ast.Expression {
	// We're positioned on ':'
	slice := &ast.SliceExpression{
		Token: tok,
		Left:  left,
		Start: start,
	}

	p.nextToken() // move past ':'

	// Parse end (if not : or ])
	if !p.curTokenIs(lexer.COLON) && !p.curTokenIs(lexer.RBRACK) {
		slice.End = p.parseExpression(LOWEST)
		p.nextToken()
	}

	// Parse step if there's another ':'
	if p.curTokenIs(lexer.COLON) {
		p.nextToken()
		if !p.curTokenIs(lexer.RBRACK) {
			slice.Step = p.parseExpression(LOWEST)
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.RBRACK) {
		p.curError("expected ] in slice expression")
		return nil
	}

	return slice
}

func (p *Parser) parseMemberExpression(left ast.Expression) ast.Expression {
	optional := p.curTokenIs(lexer.OPTDOT)
	tok := p.curToken

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	return &ast.MemberExpression{
		Token:    tok,
		Object:   left,
		Property: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
		Optional: optional,
	}
}

func (p *Parser) parseTernaryExpression(condition ast.Expression) ast.Expression {
	// We're parsing: value if condition else other
	// But we've been called with the condition as 'left' after seeing 'if'
	// Actually, the true value was parsed before 'if', so we need to swap
	tok := p.curToken

	p.nextToken()
	consequence := condition // What we thought was condition is actually the consequence
	actualCondition := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.ELSE) {
		return nil
	}
	p.nextToken()

	alternative := p.parseExpression(LOWEST)

	return &ast.TernaryExpression{
		Token:       tok,
		Condition:   actualCondition,
		Consequence: consequence,
		Alternative: alternative,
	}
}

func (p *Parser) parseLambdaInfix(left ast.Expression) ast.Expression {
	tok := p.curToken

	// Left should be an identifier (single param lambda)
	ident, ok := left.(*ast.Identifier)
	if !ok {
		p.curError("lambda parameter must be an identifier")
		return nil
	}

	p.nextToken()
	body := p.parseExpression(LOWEST)

	return &ast.LambdaExpression{
		Token:      tok,
		Parameters: []*ast.Identifier{ident},
		Body:       body,
	}
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// ============================================================================
// Statement Parse Functions
// ============================================================================

func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfStatement{Token: p.curToken}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	stmt.Consequence = p.parseBlock()

	// Check for elif/else
	if p.peekTokenIs(lexer.ELIF) {
		p.nextToken()
		stmt.Alternative = p.parseIfStatement()
	} else if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()
		if !p.expectPeek(lexer.COLON) {
			return nil
		}
		stmt.Alternative = p.parseBlock()
	}

	return stmt
}

func (p *Parser) parseForStatement() ast.Statement {
	stmt := &ast.ForStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	firstIdent := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for index, variable pattern
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // comma
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Index = firstIdent
		stmt.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		stmt.Variable = firstIdent
	}

	if !p.expectPeek(lexer.IN) {
		return nil
	}
	p.nextToken()

	stmt.Iterable = p.parseExpression(LOWEST)

	// Parse modifiers
	if p.peekTokenIs(lexer.WITH) {
		p.nextToken()
		stmt.Modifiers = p.parseForModifiers()
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	stmt.Body = p.parseBlock()

	return stmt
}

func (p *Parser) parseForModifiers() []*ast.ForModifier {
	modifiers := []*ast.ForModifier{}

	for {
		p.nextToken()
		mod := p.parseForModifier()
		if mod != nil {
			modifiers = append(modifiers, mod)
		}

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // comma
	}

	return modifiers
}

func (p *Parser) parseForModifier() *ast.ForModifier {
	var modType string
	switch p.curToken.Type {
	case lexer.LIMIT:
		modType = "limit"
	case lexer.RATE:
		modType = "rate"
	case lexer.PARALLEL:
		modType = "parallel"
	case lexer.TIMEOUT:
		modType = "timeout"
	default:
		p.curError(fmt.Sprintf("unexpected modifier type: %s", p.curToken.Literal))
		return nil
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}
	p.nextToken()

	// Parse at POWER precedence so we don't consume /s as division
	value := p.parseExpression(POWER)

	mod := &ast.ForModifier{
		Type:  modType,
		Value: value,
	}

	// Check for rate unit (e.g., 10/s)
	if p.peekTokenIs(lexer.SLASH) {
		p.nextToken() // /
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		mod.Unit = p.curToken.Literal
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return mod
}

func (p *Parser) parseMatchStatement() ast.Statement {
	tok := p.curToken
	p.nextToken()

	subject := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	arms := []*ast.MatchArm{}

	// Skip newline and expect indent
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	if !p.expectPeek(lexer.INDENT) {
		return nil
	}

	for !p.peekTokenIs(lexer.DEDENT) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		arm := p.parseMatchArm()
		if arm != nil {
			arms = append(arms, arm)
		}
		if p.peekTokenIs(lexer.NEWLINE) {
			p.nextToken()
		}
	}

	if p.peekTokenIs(lexer.DEDENT) {
		p.nextToken()
	}

	return &ast.MatchStatement{
		Token:   tok,
		Subject: subject,
		Arms:    arms,
	}
}

func (p *Parser) parseDefStatement() ast.Statement {
	stmt := &ast.DefStatement{Token: p.curToken}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseParameters()

	// Optional return type
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken()
		p.nextToken()
		stmt.ReturnType = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	stmt.Body = p.parseBlock()

	return stmt
}

func (p *Parser) parseParameters() []*ast.Parameter {
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	params = append(params, p.parseParameter())

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		params = append(params, p.parseParameter())
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseParameter() *ast.Parameter {
	param := &ast.Parameter{
		Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	// Optional type annotation
	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
		p.nextToken()
		param.Type = p.parseExpression(LOWEST)
	}

	// Optional default value
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken()
		p.nextToken()
		param.Default = p.parseExpression(LOWEST)
	}

	return param
}

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	// Check for value - return can be followed by expression unless it's end of line/block
	if !p.peekTokenIs(lexer.NEWLINE) && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.DEDENT) {
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseEmitStatement() ast.Statement {
	stmt := &ast.EmitStatement{
		Token:  p.curToken,
		Values: []ast.Expression{},
		Named:  make(map[string]ast.Expression),
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return stmt
	}

	p.nextToken()

	for {
		// Check for named argument
		if p.curTokenIs(lexer.IDENT) && p.peekTokenIs(lexer.COLON) {
			name := p.curToken.Literal
			p.nextToken() // colon
			p.nextToken() // value
			stmt.Named[name] = p.parseExpression(LOWEST)
		} else {
			stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
		}

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseStopStatement() ast.Statement {
	stmt := &ast.StopStatement{Token: p.curToken}

	if p.peekTokenIs(lexer.WITH) {
		p.nextToken()
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		if p.curToken.Literal == "rollback" {
			stmt.Rollback = true
		}
	}

	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseTryStatement() ast.Statement {
	stmt := &ast.TryStatement{Token: p.curToken}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	stmt.Body = p.parseBlock()

	// Parse catch clauses
	for p.peekTokenIs(lexer.CATCH) {
		p.nextToken()
		catch := p.parseCatchClause()
		if catch != nil {
			stmt.Catches = append(stmt.Catches, catch)
		}
	}

	return stmt
}

func (p *Parser) parseCatchClause() *ast.CatchClause {
	clause := &ast.CatchClause{Token: p.curToken}

	// Optional error type
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		clause.Type = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

		// Optional "as variable"
		if p.curToken.Literal == "as" || p.peekTokenIs(lexer.IDENT) {
			if p.curToken.Literal == "as" {
				if !p.expectPeek(lexer.IDENT) {
					return nil
				}
			} else if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "as" {
				// Type was specified, now check for "as"
				p.nextToken() // "as"
				if !p.expectPeek(lexer.IDENT) {
					return nil
				}
			}
			clause.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
	}

	if !p.expectPeek(lexer.COLON) {
		return nil
	}

	clause.Body = p.parseBlock()

	return clause
}

func (p *Parser) parseBreakStatement() ast.Statement {
	stmt := &ast.BreakStatement{Token: p.curToken}
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseContinueStatement() ast.Statement {
	stmt := &ast.ContinueStatement{Token: p.curToken}
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseBlock() *ast.Block {
	block := &ast.Block{
		Pos:        ast.Position{Line: p.curToken.Line, Column: p.curToken.Column},
		Statements: []ast.Statement{},
	}

	// Skip newline after colon
	if p.peekTokenIs(lexer.NEWLINE) {
		p.nextToken()
	}

	// Expect indent
	if !p.expectPeek(lexer.INDENT) {
		return block
	}

	for !p.peekTokenIs(lexer.DEDENT) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()

		// Skip blank lines
		if p.curTokenIs(lexer.NEWLINE) {
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
	}

	if p.peekTokenIs(lexer.DEDENT) {
		p.nextToken()
	}

	return block
}

// Parse is a convenience function to parse source code.
func Parse(source string) (*ast.Program, error) {
	l := lexer.New(source)
	p := New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return program, p.Errors()
	}
	return program, nil
}
