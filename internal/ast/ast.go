// Package ast provides the Abstract Syntax Tree for the SLOP language.
package ast

import (
	"fmt"
	"strings"

	"github.com/standardbeagle/slop/internal/lexer"
)

// Node is the interface that all AST nodes implement.
type Node interface {
	node()
	TokenLiteral() string
	String() string
}

// Expression is a node that produces a value.
type Expression interface {
	Node
	expressionNode()
}

// Statement is a node that performs an action.
type Statement interface {
	Node
	statementNode()
}

// Position represents a location in source code.
type Position struct {
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Program is the root node of the AST.
type Program struct {
	Statements []Statement
	Modules    []*Module // Module definitions (===SOURCE:, etc.)
}

func (p *Program) node()            {}
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}
func (p *Program) String() string {
	var out strings.Builder
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// WalkNode implements the Walkable interface for Program.
func (p *Program) WalkNode(v Visitor) error {
	if err := v.VisitProgram(p); err != nil {
		return err
	}
	for _, stmt := range p.Statements {
		if err := Walk(v, stmt); err != nil {
			return err
		}
	}
	return nil
}

// Module represents a module definition block.
type Module struct {
	Type       string // "SOURCE", "USE", "MAIN", "EXPORT", "INPUT", "OUTPUT"
	Name       string
	ID         string            // Full module ID like "mycompany/utils@v1"
	Uses       map[string]string // Dependencies: local name -> required ID
	Provides   []string          // Exported names
	WithClauses map[string]string // Remapping clauses
	Body       []Statement
	Pos        Position
}

func (m *Module) node()               {}
func (m *Module) TokenLiteral() string { return m.Type }
func (m *Module) String() string {
	return fmt.Sprintf("===%s: %s===", m.Type, m.Name)
}

// Block represents a sequence of statements.
type Block struct {
	Statements []Statement
	Pos        Position
}

func (b *Block) node()               {}
func (b *Block) statementNode()      {}
func (b *Block) TokenLiteral() string {
	if len(b.Statements) > 0 {
		return b.Statements[0].TokenLiteral()
	}
	return ""
}
func (b *Block) String() string {
	var out strings.Builder
	for _, s := range b.Statements {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	return out.String()
}

// WalkNode implements the Walkable interface for Block.
func (b *Block) WalkNode(v Visitor) error {
	if err := v.VisitBlock(b); err != nil {
		return err
	}
	for _, stmt := range b.Statements {
		if err := Walk(v, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// Statements
// ============================================================================

// ExpressionStatement wraps an expression as a statement.
type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) node()            {}
func (es *ExpressionStatement) statementNode()   {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// WalkNode implements the Walkable interface for ExpressionStatement.
func (es *ExpressionStatement) WalkNode(v Visitor) error {
	if err := v.VisitExpressionStatement(es); err != nil {
		return err
	}
	if es.Expression != nil {
		if err := Walk(v, es.Expression); err != nil {
			return err
		}
	}
	return nil
}

// AssignStatement represents variable assignment.
type AssignStatement struct {
	Token    lexer.Token
	Targets  []Expression // Left-hand side (can be multiple for tuple unpacking)
	Operator string       // "=", "+=", "-=", "*=", "/="
	Value    Expression
}

func (as *AssignStatement) node()            {}
func (as *AssignStatement) statementNode()   {}
func (as *AssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignStatement) String() string {
	var out strings.Builder
	targets := make([]string, len(as.Targets))
	for i, t := range as.Targets {
		targets[i] = t.String()
	}
	out.WriteString(strings.Join(targets, ", "))
	out.WriteString(" ")
	out.WriteString(as.Operator)
	out.WriteString(" ")
	out.WriteString(as.Value.String())
	return out.String()
}

// WalkNode implements the Walkable interface for AssignStatement.
func (as *AssignStatement) WalkNode(v Visitor) error {
	if err := v.VisitAssignStatement(as); err != nil {
		return err
	}
	for _, t := range as.Targets {
		if err := Walk(v, t); err != nil {
			return err
		}
	}
	if err := Walk(v, as.Value); err != nil {
		return err
	}
	return nil
}

// IfStatement represents if/elif/else.
type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *Block
	Alternative Statement // Either another IfStatement (elif) or Block (else) or nil
}

func (is *IfStatement) node()            {}
func (is *IfStatement) statementNode()   {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if ")
	out.WriteString(is.Condition.String())
	out.WriteString(":\n")
	out.WriteString(is.Consequence.String())
	if is.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(is.Alternative.String())
	}
	return out.String()
}

// WalkNode implements the Walkable interface for IfStatement.
func (is *IfStatement) WalkNode(v Visitor) error {
	if err := v.VisitIfStatement(is); err != nil {
		return err
	}
	if err := Walk(v, is.Condition); err != nil {
		return err
	}
	if err := Walk(v, is.Consequence); err != nil {
		return err
	}
	if is.Alternative != nil {
		if err := Walk(v, is.Alternative); err != nil {
			return err
		}
	}
	return nil
}

// ForStatement represents a for loop with optional modifiers.
type ForStatement struct {
	Token     lexer.Token
	Variable  *Identifier      // Loop variable
	Index     *Identifier      // Optional index variable (for enumerate)
	Iterable  Expression
	Modifiers []*ForModifier   // limit, rate, parallel, timeout
	Body      *Block
}

func (fs *ForStatement) node()            {}
func (fs *ForStatement) statementNode()   {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for ")
	if fs.Index != nil {
		out.WriteString(fs.Index.String())
		out.WriteString(", ")
	}
	out.WriteString(fs.Variable.String())
	out.WriteString(" in ")
	out.WriteString(fs.Iterable.String())
	if len(fs.Modifiers) > 0 {
		out.WriteString(" with ")
		mods := make([]string, len(fs.Modifiers))
		for i, m := range fs.Modifiers {
			mods[i] = m.String()
		}
		out.WriteString(strings.Join(mods, ", "))
	}
	out.WriteString(":\n")
	out.WriteString(fs.Body.String())
	return out.String()
}

// WalkNode implements the Walkable interface for ForStatement.
func (fs *ForStatement) WalkNode(v Visitor) error {
	if err := v.VisitForStatement(fs); err != nil {
		return err
	}
	if err := Walk(v, fs.Variable); err != nil {
		return err
	}
	if fs.Index != nil {
		if err := Walk(v, fs.Index); err != nil {
			return err
		}
	}
	if err := Walk(v, fs.Iterable); err != nil {
		return err
	}
	if err := Walk(v, fs.Body); err != nil {
		return err
	}
	return nil
}

// ForModifier represents a loop modifier (limit, rate, parallel, timeout).
type ForModifier struct {
	Type  string // "limit", "rate", "parallel", "timeout"
	Value Expression
	Unit  string // For rate: "s", "m", "h"; For timeout: "s", "m"
}

func (fm *ForModifier) String() string {
	if fm.Unit != "" {
		return fmt.Sprintf("%s(%s/%s)", fm.Type, fm.Value.String(), fm.Unit)
	}
	return fmt.Sprintf("%s(%s)", fm.Type, fm.Value.String())
}

// MatchStatement represents a match expression.
type MatchStatement struct {
	Token   lexer.Token
	Subject Expression
	Arms    []*MatchArm
}

func (ms *MatchStatement) node()            {}
func (ms *MatchStatement) statementNode()   {}
func (ms *MatchStatement) TokenLiteral() string { return ms.Token.Literal }
func (ms *MatchStatement) String() string {
	var out strings.Builder
	out.WriteString("match ")
	out.WriteString(ms.Subject.String())
	out.WriteString(":\n")
	for _, arm := range ms.Arms {
		out.WriteString("    ")
		out.WriteString(arm.String())
		out.WriteString("\n")
	}
	return out.String()
}

// WalkNode implements the Walkable interface for MatchStatement.
func (ms *MatchStatement) WalkNode(v Visitor) error {
	if err := v.VisitMatchStatement(ms); err != nil {
		return err
	}
	if err := Walk(v, ms.Subject); err != nil {
		return err
	}
	for _, arm := range ms.Arms {
		if err := Walk(v, arm.Pattern); err != nil {
			return err
		}
		if arm.Guard != nil {
			if err := Walk(v, arm.Guard); err != nil {
				return err
			}
		}
		if err := Walk(v, arm.Body); err != nil {
			return err
		}
	}
	return nil
}

// MatchArm represents a single arm in a match expression.
type MatchArm struct {
	Pattern Expression
	Guard   Expression // Optional "if" guard
	Body    Expression
}

func (ma *MatchArm) String() string {
	var out strings.Builder
	out.WriteString(ma.Pattern.String())
	if ma.Guard != nil {
		out.WriteString(" if ")
		out.WriteString(ma.Guard.String())
	}
	out.WriteString(" -> ")
	out.WriteString(ma.Body.String())
	return out.String()
}

// DefStatement represents a function definition.
type DefStatement struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Parameter
	ReturnType Expression // Optional return type annotation
	Body       *Block
}

func (ds *DefStatement) node()            {}
func (ds *DefStatement) statementNode()   {}
func (ds *DefStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DefStatement) String() string {
	var out strings.Builder
	out.WriteString("def ")
	out.WriteString(ds.Name.String())
	out.WriteString("(")
	params := make([]string, len(ds.Parameters))
	for i, p := range ds.Parameters {
		params[i] = p.String()
	}
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	if ds.ReturnType != nil {
		out.WriteString(" -> ")
		out.WriteString(ds.ReturnType.String())
	}
	out.WriteString(":\n")
	out.WriteString(ds.Body.String())
	return out.String()
}

// WalkNode implements the Walkable interface for DefStatement.
func (ds *DefStatement) WalkNode(v Visitor) error {
	if err := v.VisitDefStatement(ds); err != nil {
		return err
	}
	if err := Walk(v, ds.Name); err != nil {
		return err
	}
	if err := Walk(v, ds.Body); err != nil {
		return err
	}
	return nil
}

// Parameter represents a function parameter.
type Parameter struct {
	Name    *Identifier
	Type    Expression // Optional type annotation
	Default Expression // Optional default value
}

func (p *Parameter) String() string {
	var out strings.Builder
	out.WriteString(p.Name.String())
	if p.Type != nil {
		out.WriteString(": ")
		out.WriteString(p.Type.String())
	}
	if p.Default != nil {
		out.WriteString(" = ")
		out.WriteString(p.Default.String())
	}
	return out.String()
}

// ReturnStatement represents a return statement.
type ReturnStatement struct {
	Token lexer.Token
	Value Expression // nil for bare "return"
}

func (rs *ReturnStatement) node()            {}
func (rs *ReturnStatement) statementNode()   {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out strings.Builder
	out.WriteString("return")
	if rs.Value != nil {
		out.WriteString(" ")
		out.WriteString(rs.Value.String())
	}
	return out.String()
}

// WalkNode implements the Walkable interface for ReturnStatement.
func (rs *ReturnStatement) WalkNode(v Visitor) error {
	if err := v.VisitReturnStatement(rs); err != nil {
		return err
	}
	if rs.Value != nil {
		if err := Walk(v, rs.Value); err != nil {
			return err
		}
	}
	return nil
}

// EmitStatement represents the emit statement for output.
type EmitStatement struct {
	Token  lexer.Token
	Values []Expression          // Positional values
	Named  map[string]Expression // Named values
}

func (es *EmitStatement) node()            {}
func (es *EmitStatement) statementNode()   {}
func (es *EmitStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EmitStatement) String() string {
	var out strings.Builder
	out.WriteString("emit(")
	parts := make([]string, 0, len(es.Values)+len(es.Named))
	for _, v := range es.Values {
		parts = append(parts, v.String())
	}
	for k, v := range es.Named {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString(")")
	return out.String()
}

// WalkNode implements the Walkable interface for EmitStatement.
func (es *EmitStatement) WalkNode(v Visitor) error {
	if err := v.VisitEmitStatement(es); err != nil {
		return err
	}
	for _, val := range es.Values {
		if err := Walk(v, val); err != nil {
			return err
		}
	}
	for _, val := range es.Named {
		if err := Walk(v, val); err != nil {
			return err
		}
	}
	return nil
}

// StopStatement represents the stop statement.
type StopStatement struct {
	Token    lexer.Token
	Rollback bool
}

func (ss *StopStatement) node()            {}
func (ss *StopStatement) statementNode()   {}
func (ss *StopStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StopStatement) String() string {
	if ss.Rollback {
		return "stop with rollback"
	}
	return "stop"
}

// WalkNode implements the Walkable interface for StopStatement.
func (ss *StopStatement) WalkNode(v Visitor) error {
	return v.VisitStopStatement(ss)
}

// TryStatement represents try/catch error handling.
type TryStatement struct {
	Token   lexer.Token
	Body    *Block
	Catches []*CatchClause
}

func (ts *TryStatement) node()            {}
func (ts *TryStatement) statementNode()   {}
func (ts *TryStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryStatement) String() string {
	var out strings.Builder
	out.WriteString("try:\n")
	out.WriteString(ts.Body.String())
	for _, c := range ts.Catches {
		out.WriteString(c.String())
	}
	return out.String()
}

// WalkNode implements the Walkable interface for TryStatement.
func (ts *TryStatement) WalkNode(v Visitor) error {
	if err := v.VisitTryStatement(ts); err != nil {
		return err
	}
	if err := Walk(v, ts.Body); err != nil {
		return err
	}
	for _, c := range ts.Catches {
		if err := Walk(v, c.Body); err != nil {
			return err
		}
	}
	return nil
}

// CatchClause represents a catch clause.
type CatchClause struct {
	Token    lexer.Token
	Type     *Identifier // Error type (optional)
	Variable *Identifier // Bound variable (optional)
	Body     *Block
}

func (cc *CatchClause) String() string {
	var out strings.Builder
	out.WriteString("catch")
	if cc.Type != nil {
		out.WriteString(" ")
		out.WriteString(cc.Type.String())
	}
	if cc.Variable != nil {
		out.WriteString(" as ")
		out.WriteString(cc.Variable.String())
	}
	out.WriteString(":\n")
	out.WriteString(cc.Body.String())
	return out.String()
}

// BreakStatement represents the break statement.
type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) node()            {}
func (bs *BreakStatement) statementNode()   {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

// WalkNode implements the Walkable interface for BreakStatement.
func (bs *BreakStatement) WalkNode(v Visitor) error {
	return v.VisitBreakStatement(bs)
}

// ContinueStatement represents the continue statement.
type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) node()            {}
func (cs *ContinueStatement) statementNode()   {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue" }

// WalkNode implements the Walkable interface for ContinueStatement.
func (cs *ContinueStatement) WalkNode(v Visitor) error {
	return v.VisitContinueStatement(cs)
}

// PauseStatement represents the pause statement for checkpointing.
// Syntax: pause or pause "optional message"
type PauseStatement struct {
	Token   lexer.Token
	Message Expression // Optional message/checkpoint name
}

func (ps *PauseStatement) node()            {}
func (ps *PauseStatement) statementNode()   {}
func (ps *PauseStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PauseStatement) String() string {
	if ps.Message != nil {
		return "pause " + ps.Message.String()
	}
	return "pause"
}

// WalkNode implements the Walkable interface for PauseStatement.
func (ps *PauseStatement) WalkNode(v Visitor) error {
	if err := v.VisitPauseStatement(ps); err != nil {
		return err
	}
	if ps.Message != nil {
		if err := Walk(v, ps.Message); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// Expressions
// ============================================================================

// Identifier represents an identifier.
type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) node()            {}
func (i *Identifier) expressionNode()  {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// WalkNode implements the Walkable interface for Identifier.
func (i *Identifier) WalkNode(v Visitor) error {
	return v.VisitIdentifier(i)
}

// IntegerLiteral represents an integer.
type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) node()            {}
func (il *IntegerLiteral) expressionNode()  {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// WalkNode implements the Walkable interface for IntegerLiteral.
func (il *IntegerLiteral) WalkNode(v Visitor) error {
	return v.VisitIntegerLiteral(il)
}

// FloatLiteral represents a floating-point number.
type FloatLiteral struct {
	Token lexer.Token
	Value float64
}

func (fl *FloatLiteral) node()            {}
func (fl *FloatLiteral) expressionNode()  {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// WalkNode implements the Walkable interface for FloatLiteral.
func (fl *FloatLiteral) WalkNode(v Visitor) error {
	return v.VisitFloatLiteral(fl)
}

// StringLiteral represents a string.
type StringLiteral struct {
	Token lexer.Token
	Value string
}

func (sl *StringLiteral) node()            {}
func (sl *StringLiteral) expressionNode()  {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return fmt.Sprintf("%q", sl.Value) }

// WalkNode implements the Walkable interface for StringLiteral.
func (sl *StringLiteral) WalkNode(v Visitor) error {
	return v.VisitStringLiteral(sl)
}

// BooleanLiteral represents true or false.
type BooleanLiteral struct {
	Token lexer.Token
	Value bool
}

func (bl *BooleanLiteral) node()            {}
func (bl *BooleanLiteral) expressionNode()  {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string {
	if bl.Value {
		return "true"
	}
	return "false"
}

// WalkNode implements the Walkable interface for BooleanLiteral.
func (bl *BooleanLiteral) WalkNode(v Visitor) error {
	return v.VisitBooleanLiteral(bl)
}

// NoneLiteral represents the none value.
type NoneLiteral struct {
	Token lexer.Token
}

func (nl *NoneLiteral) node()            {}
func (nl *NoneLiteral) expressionNode()  {}
func (nl *NoneLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NoneLiteral) String() string       { return "none" }

// WalkNode implements the Walkable interface for NoneLiteral.
func (nl *NoneLiteral) WalkNode(v Visitor) error {
	return v.VisitNoneLiteral(nl)
}

// ListLiteral represents a list [a, b, c].
type ListLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (ll *ListLiteral) node()            {}
func (ll *ListLiteral) expressionNode()  {}
func (ll *ListLiteral) TokenLiteral() string { return ll.Token.Literal }
func (ll *ListLiteral) String() string {
	elements := make([]string, len(ll.Elements))
	for i, e := range ll.Elements {
		elements[i] = e.String()
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// WalkNode implements the Walkable interface for ListLiteral.
func (ll *ListLiteral) WalkNode(v Visitor) error {
	if err := v.VisitListLiteral(ll); err != nil {
		return err
	}
	for _, e := range ll.Elements {
		if err := Walk(v, e); err != nil {
			return err
		}
	}
	return nil
}

// MapLiteral represents a map {a: 1, b: 2}.
type MapLiteral struct {
	Token lexer.Token
	Pairs map[Expression]Expression
	Order []Expression // To maintain insertion order
}

func (ml *MapLiteral) node()            {}
func (ml *MapLiteral) expressionNode()  {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MapLiteral) String() string {
	pairs := make([]string, 0, len(ml.Pairs))
	for _, k := range ml.Order {
		v := ml.Pairs[k]
		pairs = append(pairs, fmt.Sprintf("%s: %s", k.String(), v.String()))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// WalkNode implements the Walkable interface for MapLiteral.
func (ml *MapLiteral) WalkNode(v Visitor) error {
	if err := v.VisitMapLiteral(ml); err != nil {
		return err
	}
	for k, val := range ml.Pairs {
		if err := Walk(v, k); err != nil {
			return err
		}
		if err := Walk(v, val); err != nil {
			return err
		}
	}
	return nil
}

// SetLiteral represents a set {a, b, c}.
type SetLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (sl *SetLiteral) node()            {}
func (sl *SetLiteral) expressionNode()  {}
func (sl *SetLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *SetLiteral) String() string {
	elements := make([]string, len(sl.Elements))
	for i, e := range sl.Elements {
		elements[i] = e.String()
	}
	return "{" + strings.Join(elements, ", ") + "}"
}

// WalkNode implements the Walkable interface for SetLiteral.
func (sl *SetLiteral) WalkNode(v Visitor) error {
	if err := v.VisitSetLiteral(sl); err != nil {
		return err
	}
	for _, e := range sl.Elements {
		if err := Walk(v, e); err != nil {
			return err
		}
	}
	return nil
}

// PrefixExpression represents a unary prefix operation.
type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) node()            {}
func (pe *PrefixExpression) expressionNode()  {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

// WalkNode implements the Walkable interface for PrefixExpression.
func (pe *PrefixExpression) WalkNode(v Visitor) error {
	if err := v.VisitPrefixExpression(pe); err != nil {
		return err
	}
	if err := Walk(v, pe.Right); err != nil {
		return err
	}
	return nil
}

// InfixExpression represents a binary operation.
type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) node()            {}
func (ie *InfixExpression) expressionNode()  {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

// WalkNode implements the Walkable interface for InfixExpression.
func (ie *InfixExpression) WalkNode(v Visitor) error {
	if err := v.VisitInfixExpression(ie); err != nil {
		return err
	}
	if err := Walk(v, ie.Left); err != nil {
		return err
	}
	if err := Walk(v, ie.Right); err != nil {
		return err
	}
	return nil
}

// CallExpression represents a function call.
type CallExpression struct {
	Token     lexer.Token
	Function  Expression   // Identifier or member expression
	Arguments []Expression // Positional arguments
	Kwargs    map[string]Expression // Keyword arguments
}

func (ce *CallExpression) node()            {}
func (ce *CallExpression) expressionNode()  {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	args := make([]string, 0, len(ce.Arguments)+len(ce.Kwargs))
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	for k, v := range ce.Kwargs {
		args = append(args, fmt.Sprintf("%s: %s", k, v.String()))
	}
	return ce.Function.String() + "(" + strings.Join(args, ", ") + ")"
}

// WalkNode implements the Walkable interface for CallExpression.
func (ce *CallExpression) WalkNode(v Visitor) error {
	if err := v.VisitCallExpression(ce); err != nil {
		return err
	}
	if err := Walk(v, ce.Function); err != nil {
		return err
	}
	for _, a := range ce.Arguments {
		if err := Walk(v, a); err != nil {
			return err
		}
	}
	for _, val := range ce.Kwargs {
		if err := Walk(v, val); err != nil {
			return err
		}
	}
	return nil
}

// IndexExpression represents array/map indexing a[b].
type IndexExpression struct {
	Token    lexer.Token
	Left     Expression
	Index    Expression
	Optional bool // Whether this is ?[ optional access
}

func (ie *IndexExpression) node()            {}
func (ie *IndexExpression) expressionNode()  {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	if ie.Optional {
		return "(" + ie.Left.String() + "?[" + ie.Index.String() + "])"
	}
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// WalkNode implements the Walkable interface for IndexExpression.
func (ie *IndexExpression) WalkNode(v Visitor) error {
	if err := v.VisitIndexExpression(ie); err != nil {
		return err
	}
	if err := Walk(v, ie.Left); err != nil {
		return err
	}
	if err := Walk(v, ie.Index); err != nil {
		return err
	}
	return nil
}

// SliceExpression represents array slicing a[start:end:step].
type SliceExpression struct {
	Token lexer.Token
	Left  Expression
	Start Expression // nil means from beginning
	End   Expression // nil means to end
	Step  Expression // nil means step of 1
}

func (se *SliceExpression) node()            {}
func (se *SliceExpression) expressionNode()  {}
func (se *SliceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SliceExpression) String() string {
	var out strings.Builder
	out.WriteString(se.Left.String())
	out.WriteString("[")
	if se.Start != nil {
		out.WriteString(se.Start.String())
	}
	out.WriteString(":")
	if se.End != nil {
		out.WriteString(se.End.String())
	}
	if se.Step != nil {
		out.WriteString(":")
		out.WriteString(se.Step.String())
	}
	out.WriteString("]")
	return out.String()
}

// WalkNode implements the Walkable interface for SliceExpression.
func (se *SliceExpression) WalkNode(v Visitor) error {
	if err := v.VisitSliceExpression(se); err != nil {
		return err
	}
	if err := Walk(v, se.Left); err != nil {
		return err
	}
	if se.Start != nil {
		if err := Walk(v, se.Start); err != nil {
			return err
		}
	}
	if se.End != nil {
		if err := Walk(v, se.End); err != nil {
			return err
		}
	}
	if se.Step != nil {
		if err := Walk(v, se.Step); err != nil {
			return err
		}
	}
	return nil
}

// MemberExpression represents attribute access a.b or a?.b.
type MemberExpression struct {
	Token    lexer.Token
	Object   Expression
	Property *Identifier
	Optional bool // Whether this is ?. optional access
}

func (me *MemberExpression) node()            {}
func (me *MemberExpression) expressionNode()  {}
func (me *MemberExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MemberExpression) String() string {
	if me.Optional {
		return me.Object.String() + "?." + me.Property.String()
	}
	return me.Object.String() + "." + me.Property.String()
}

// WalkNode implements the Walkable interface for MemberExpression.
func (me *MemberExpression) WalkNode(v Visitor) error {
	if err := v.VisitMemberExpression(me); err != nil {
		return err
	}
	if err := Walk(v, me.Object); err != nil {
		return err
	}
	if err := Walk(v, me.Property); err != nil {
		return err
	}
	return nil
}

// LambdaExpression represents a lambda x -> x * 2.
type LambdaExpression struct {
	Token      lexer.Token
	Parameters []*Identifier
	Body       Expression
}

func (le *LambdaExpression) node()            {}
func (le *LambdaExpression) expressionNode()  {}
func (le *LambdaExpression) TokenLiteral() string { return le.Token.Literal }
func (le *LambdaExpression) String() string {
	params := make([]string, len(le.Parameters))
	for i, p := range le.Parameters {
		params[i] = p.String()
	}
	if len(params) == 1 {
		return params[0] + " -> " + le.Body.String()
	}
	return "(" + strings.Join(params, ", ") + ") -> " + le.Body.String()
}

// WalkNode implements the Walkable interface for LambdaExpression.
func (le *LambdaExpression) WalkNode(v Visitor) error {
	if err := v.VisitLambdaExpression(le); err != nil {
		return err
	}
	for _, p := range le.Parameters {
		if err := Walk(v, p); err != nil {
			return err
		}
	}
	if err := Walk(v, le.Body); err != nil {
		return err
	}
	return nil
}

// PipelineExpression represents a pipeline a | b | c.
type PipelineExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (pe *PipelineExpression) node()            {}
func (pe *PipelineExpression) expressionNode()  {}
func (pe *PipelineExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipelineExpression) String() string {
	return pe.Left.String() + " | " + pe.Right.String()
}

// WalkNode implements the Walkable interface for PipelineExpression.
func (pe *PipelineExpression) WalkNode(v Visitor) error {
	if err := v.VisitPipelineExpression(pe); err != nil {
		return err
	}
	if err := Walk(v, pe.Left); err != nil {
		return err
	}
	if err := Walk(v, pe.Right); err != nil {
		return err
	}
	return nil
}

// TernaryExpression represents value if condition else other.
type TernaryExpression struct {
	Token       lexer.Token
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

func (te *TernaryExpression) node()            {}
func (te *TernaryExpression) expressionNode()  {}
func (te *TernaryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TernaryExpression) String() string {
	return te.Consequence.String() + " if " + te.Condition.String() + " else " + te.Alternative.String()
}

// WalkNode implements the Walkable interface for TernaryExpression.
func (te *TernaryExpression) WalkNode(v Visitor) error {
	if err := v.VisitTernaryExpression(te); err != nil {
		return err
	}
	if err := Walk(v, te.Condition); err != nil {
		return err
	}
	if err := Walk(v, te.Consequence); err != nil {
		return err
	}
	if err := Walk(v, te.Alternative); err != nil {
		return err
	}
	return nil
}

// MatchExpression represents an inline match expression.
type MatchExpression struct {
	Token   lexer.Token
	Subject Expression
	Arms    []*MatchArm
}

func (me *MatchExpression) node()            {}
func (me *MatchExpression) expressionNode()  {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	var out strings.Builder
	out.WriteString("match ")
	out.WriteString(me.Subject.String())
	out.WriteString(": ")
	arms := make([]string, len(me.Arms))
	for i, arm := range me.Arms {
		arms[i] = arm.String()
	}
	out.WriteString(strings.Join(arms, "; "))
	return out.String()
}

// WalkNode implements the Walkable interface for MatchExpression.
func (me *MatchExpression) WalkNode(v Visitor) error {
	if err := v.VisitMatchExpression(me); err != nil {
		return err
	}
	if err := Walk(v, me.Subject); err != nil {
		return err
	}
	for _, arm := range me.Arms {
		if err := Walk(v, arm.Pattern); err != nil {
			return err
		}
		if arm.Guard != nil {
			if err := Walk(v, arm.Guard); err != nil {
				return err
			}
		}
		if err := Walk(v, arm.Body); err != nil {
			return err
		}
	}
	return nil
}

// ListComprehension represents [expr for x in iter if cond].
type ListComprehension struct {
	Token    lexer.Token
	Element  Expression
	Variable *Identifier
	Index    *Identifier // Optional (for i, x in items)
	Iterable Expression
	Filter   Expression // Optional if condition
}

func (lc *ListComprehension) node()            {}
func (lc *ListComprehension) expressionNode()  {}
func (lc *ListComprehension) TokenLiteral() string { return lc.Token.Literal }
func (lc *ListComprehension) String() string {
	var out strings.Builder
	out.WriteString("[")
	out.WriteString(lc.Element.String())
	out.WriteString(" for ")
	if lc.Index != nil {
		out.WriteString(lc.Index.String())
		out.WriteString(", ")
	}
	out.WriteString(lc.Variable.String())
	out.WriteString(" in ")
	out.WriteString(lc.Iterable.String())
	if lc.Filter != nil {
		out.WriteString(" if ")
		out.WriteString(lc.Filter.String())
	}
	out.WriteString("]")
	return out.String()
}

// WalkNode implements the Walkable interface for ListComprehension.
func (lc *ListComprehension) WalkNode(v Visitor) error {
	if err := v.VisitListComprehension(lc); err != nil {
		return err
	}
	if err := Walk(v, lc.Element); err != nil {
		return err
	}
	if err := Walk(v, lc.Variable); err != nil {
		return err
	}
	if lc.Index != nil {
		if err := Walk(v, lc.Index); err != nil {
			return err
		}
	}
	if err := Walk(v, lc.Iterable); err != nil {
		return err
	}
	if lc.Filter != nil {
		if err := Walk(v, lc.Filter); err != nil {
			return err
		}
	}
	return nil
}

// MapComprehension represents {k: v for k, v in pairs if cond}.
type MapComprehension struct {
	Token    lexer.Token
	Key      Expression
	Value    Expression
	KeyVar   *Identifier
	ValueVar *Identifier
	Iterable Expression
	Filter   Expression // Optional if condition
}

func (mc *MapComprehension) node()            {}
func (mc *MapComprehension) expressionNode()  {}
func (mc *MapComprehension) TokenLiteral() string { return mc.Token.Literal }
func (mc *MapComprehension) String() string {
	var out strings.Builder
	out.WriteString("{")
	out.WriteString(mc.Key.String())
	out.WriteString(": ")
	out.WriteString(mc.Value.String())
	out.WriteString(" for ")
	out.WriteString(mc.KeyVar.String())
	out.WriteString(", ")
	out.WriteString(mc.ValueVar.String())
	out.WriteString(" in ")
	out.WriteString(mc.Iterable.String())
	if mc.Filter != nil {
		out.WriteString(" if ")
		out.WriteString(mc.Filter.String())
	}
	out.WriteString("}")
	return out.String()
}

// WalkNode implements the Walkable interface for MapComprehension.
func (mc *MapComprehension) WalkNode(v Visitor) error {
	if err := v.VisitMapComprehension(mc); err != nil {
		return err
	}
	if err := Walk(v, mc.Key); err != nil {
		return err
	}
	if err := Walk(v, mc.Value); err != nil {
		return err
	}
	if err := Walk(v, mc.KeyVar); err != nil {
		return err
	}
	if err := Walk(v, mc.ValueVar); err != nil {
		return err
	}
	if err := Walk(v, mc.Iterable); err != nil {
		return err
	}
	if mc.Filter != nil {
		if err := Walk(v, mc.Filter); err != nil {
			return err
		}
	}
	return nil
}

// SchemaExpression represents a schema definition for LLM calls.
type SchemaExpression struct {
	Token  lexer.Token
	Fields []*SchemaField
}

func (se *SchemaExpression) node()            {}
func (se *SchemaExpression) expressionNode()  {}
func (se *SchemaExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SchemaExpression) String() string {
	fields := make([]string, len(se.Fields))
	for i, f := range se.Fields {
		fields[i] = f.String()
	}
	return "{" + strings.Join(fields, ", ") + "}"
}

// WalkNode implements the Walkable interface for SchemaExpression.
func (se *SchemaExpression) WalkNode(v Visitor) error {
	if err := v.VisitSchemaExpression(se); err != nil {
		return err
	}
	for _, f := range se.Fields {
		if err := Walk(v, f.Type); err != nil {
			return err
		}
	}
	return nil
}

// SchemaField represents a field in a schema.
type SchemaField struct {
	Name     string
	Type     Expression
	Optional bool
}

func (sf *SchemaField) String() string {
	opt := ""
	if sf.Optional {
		opt = "?"
	}
	return sf.Name + opt + ": " + sf.Type.String()
}

// EnumExpression represents enum(a, b, c).
type EnumExpression struct {
	Token  lexer.Token
	Values []string
}

func (ee *EnumExpression) node()            {}
func (ee *EnumExpression) expressionNode()  {}
func (ee *EnumExpression) TokenLiteral() string { return ee.Token.Literal }
func (ee *EnumExpression) String() string {
	return "enum(" + strings.Join(ee.Values, ", ") + ")"
}

// WalkNode implements the Walkable interface for EnumExpression.
func (ee *EnumExpression) WalkNode(v Visitor) error {
	return v.VisitEnumExpression(ee)
}

// TypeExpression represents a type annotation like list(string) or int(min: 1, max: 10).
type TypeExpression struct {
	Token       lexer.Token
	Name        string
	Inner       Expression            // For list(T), the inner type
	Constraints map[string]Expression // For int(min: 1, max: 10)
}

func (te *TypeExpression) node()            {}
func (te *TypeExpression) expressionNode()  {}
func (te *TypeExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TypeExpression) String() string {
	if te.Inner != nil {
		return te.Name + "(" + te.Inner.String() + ")"
	}
	if len(te.Constraints) > 0 {
		parts := make([]string, 0, len(te.Constraints))
		for k, v := range te.Constraints {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
		}
		return te.Name + "(" + strings.Join(parts, ", ") + ")"
	}
	return te.Name
}

// WalkNode implements the Walkable interface for TypeExpression.
func (te *TypeExpression) WalkNode(v Visitor) error {
	if err := v.VisitTypeExpression(te); err != nil {
		return err
	}
	if te.Inner != nil {
		if err := Walk(v, te.Inner); err != nil {
			return err
		}
	}
	for _, c := range te.Constraints {
		if err := Walk(v, c); err != nil {
			return err
		}
	}
	return nil
}

// RangeExpression represents range(start, end, step).
type RangeExpression struct {
	Token lexer.Token
	Start Expression
	End   Expression
	Step  Expression
}

func (re *RangeExpression) node()            {}
func (re *RangeExpression) expressionNode()  {}
func (re *RangeExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RangeExpression) String() string {
	var out strings.Builder
	out.WriteString("range(")
	if re.Start != nil {
		out.WriteString(re.Start.String())
		out.WriteString(", ")
	}
	out.WriteString(re.End.String())
	if re.Step != nil {
		out.WriteString(", ")
		out.WriteString(re.Step.String())
	}
	out.WriteString(")")
	return out.String()
}

// WalkNode implements the Walkable interface for RangeExpression.
func (re *RangeExpression) WalkNode(v Visitor) error {
	if err := v.VisitRangeExpression(re); err != nil {
		return err
	}
	if re.Start != nil {
		if err := Walk(v, re.Start); err != nil {
			return err
		}
	}
	if err := Walk(v, re.End); err != nil {
		return err
	}
	if re.Step != nil {
		if err := Walk(v, re.Step); err != nil {
			return err
		}
	}
	return nil
}
