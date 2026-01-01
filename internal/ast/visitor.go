package ast

// Visitor defines the interface for AST traversal.
type Visitor interface {
	// Statements
	VisitProgram(*Program) error
	VisitBlock(*Block) error
	VisitExpressionStatement(*ExpressionStatement) error
	VisitAssignStatement(*AssignStatement) error
	VisitIfStatement(*IfStatement) error
	VisitForStatement(*ForStatement) error
	VisitMatchStatement(*MatchStatement) error
	VisitDefStatement(*DefStatement) error
	VisitReturnStatement(*ReturnStatement) error
	VisitEmitStatement(*EmitStatement) error
	VisitStopStatement(*StopStatement) error
	VisitTryStatement(*TryStatement) error
	VisitBreakStatement(*BreakStatement) error
	VisitContinueStatement(*ContinueStatement) error

	// Expressions
	VisitIdentifier(*Identifier) error
	VisitIntegerLiteral(*IntegerLiteral) error
	VisitFloatLiteral(*FloatLiteral) error
	VisitStringLiteral(*StringLiteral) error
	VisitBooleanLiteral(*BooleanLiteral) error
	VisitNoneLiteral(*NoneLiteral) error
	VisitListLiteral(*ListLiteral) error
	VisitMapLiteral(*MapLiteral) error
	VisitSetLiteral(*SetLiteral) error
	VisitPrefixExpression(*PrefixExpression) error
	VisitInfixExpression(*InfixExpression) error
	VisitCallExpression(*CallExpression) error
	VisitIndexExpression(*IndexExpression) error
	VisitSliceExpression(*SliceExpression) error
	VisitMemberExpression(*MemberExpression) error
	VisitLambdaExpression(*LambdaExpression) error
	VisitPipelineExpression(*PipelineExpression) error
	VisitTernaryExpression(*TernaryExpression) error
	VisitMatchExpression(*MatchExpression) error
	VisitListComprehension(*ListComprehension) error
	VisitMapComprehension(*MapComprehension) error
	VisitSchemaExpression(*SchemaExpression) error
	VisitEnumExpression(*EnumExpression) error
	VisitTypeExpression(*TypeExpression) error
	VisitRangeExpression(*RangeExpression) error
}

// Walkable defines the interface for nodes that can traverse themselves.
// This enables each node type to encapsulate its own traversal logic,
// reducing the complexity of the Walk function.
type Walkable interface {
	Node
	WalkNode(v Visitor) error
}

// Walk traverses the AST calling the visitor for each node.
// If the node implements Walkable, it delegates to the node's WalkNode method.
// Otherwise, it uses the traditional switch-based traversal.
func Walk(v Visitor, node Node) error {
	// Check if node implements Walkable interface
	if walkable, ok := node.(Walkable); ok {
		return walkable.WalkNode(v)
	}

	// Fall back to switch-based traversal for nodes that don't implement Walkable
	switch n := node.(type) {
	case *Program:
		if err := v.VisitProgram(n); err != nil {
			return err
		}
		for _, stmt := range n.Statements {
			if err := Walk(v, stmt); err != nil {
				return err
			}
		}

	case *Block:
		if err := v.VisitBlock(n); err != nil {
			return err
		}
		for _, stmt := range n.Statements {
			if err := Walk(v, stmt); err != nil {
				return err
			}
		}

	case *ExpressionStatement:
		if err := v.VisitExpressionStatement(n); err != nil {
			return err
		}
		if n.Expression != nil {
			if err := Walk(v, n.Expression); err != nil {
				return err
			}
		}

	case *AssignStatement:
		if err := v.VisitAssignStatement(n); err != nil {
			return err
		}
		for _, t := range n.Targets {
			if err := Walk(v, t); err != nil {
				return err
			}
		}
		if err := Walk(v, n.Value); err != nil {
			return err
		}

	case *IfStatement:
		if err := v.VisitIfStatement(n); err != nil {
			return err
		}
		if err := Walk(v, n.Condition); err != nil {
			return err
		}
		if err := Walk(v, n.Consequence); err != nil {
			return err
		}
		if n.Alternative != nil {
			if err := Walk(v, n.Alternative); err != nil {
				return err
			}
		}

	case *ForStatement:
		if err := v.VisitForStatement(n); err != nil {
			return err
		}
		if err := Walk(v, n.Variable); err != nil {
			return err
		}
		if n.Index != nil {
			if err := Walk(v, n.Index); err != nil {
				return err
			}
		}
		if err := Walk(v, n.Iterable); err != nil {
			return err
		}
		if err := Walk(v, n.Body); err != nil {
			return err
		}

	case *MatchStatement:
		if err := v.VisitMatchStatement(n); err != nil {
			return err
		}
		if err := Walk(v, n.Subject); err != nil {
			return err
		}
		for _, arm := range n.Arms {
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

	case *DefStatement:
		if err := v.VisitDefStatement(n); err != nil {
			return err
		}
		if err := Walk(v, n.Name); err != nil {
			return err
		}
		if err := Walk(v, n.Body); err != nil {
			return err
		}

	case *ReturnStatement:
		if err := v.VisitReturnStatement(n); err != nil {
			return err
		}
		if n.Value != nil {
			if err := Walk(v, n.Value); err != nil {
				return err
			}
		}

	case *EmitStatement:
		if err := v.VisitEmitStatement(n); err != nil {
			return err
		}
		for _, val := range n.Values {
			if err := Walk(v, val); err != nil {
				return err
			}
		}
		for _, val := range n.Named {
			if err := Walk(v, val); err != nil {
				return err
			}
		}

	case *StopStatement:
		if err := v.VisitStopStatement(n); err != nil {
			return err
		}

	case *TryStatement:
		if err := v.VisitTryStatement(n); err != nil {
			return err
		}
		if err := Walk(v, n.Body); err != nil {
			return err
		}
		for _, c := range n.Catches {
			if err := Walk(v, c.Body); err != nil {
				return err
			}
		}

	case *BreakStatement:
		if err := v.VisitBreakStatement(n); err != nil {
			return err
		}

	case *ContinueStatement:
		if err := v.VisitContinueStatement(n); err != nil {
			return err
		}

	// Expressions
	case *Identifier:
		return v.VisitIdentifier(n)

	case *IntegerLiteral:
		return v.VisitIntegerLiteral(n)

	case *FloatLiteral:
		return v.VisitFloatLiteral(n)

	case *StringLiteral:
		return v.VisitStringLiteral(n)

	case *BooleanLiteral:
		return v.VisitBooleanLiteral(n)

	case *NoneLiteral:
		return v.VisitNoneLiteral(n)

	case *ListLiteral:
		if err := v.VisitListLiteral(n); err != nil {
			return err
		}
		for _, e := range n.Elements {
			if err := Walk(v, e); err != nil {
				return err
			}
		}

	case *MapLiteral:
		if err := v.VisitMapLiteral(n); err != nil {
			return err
		}
		for k, val := range n.Pairs {
			if err := Walk(v, k); err != nil {
				return err
			}
			if err := Walk(v, val); err != nil {
				return err
			}
		}

	case *SetLiteral:
		if err := v.VisitSetLiteral(n); err != nil {
			return err
		}
		for _, e := range n.Elements {
			if err := Walk(v, e); err != nil {
				return err
			}
		}

	case *PrefixExpression:
		if err := v.VisitPrefixExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Right); err != nil {
			return err
		}

	case *InfixExpression:
		if err := v.VisitInfixExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Left); err != nil {
			return err
		}
		if err := Walk(v, n.Right); err != nil {
			return err
		}

	case *CallExpression:
		if err := v.VisitCallExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Function); err != nil {
			return err
		}
		for _, a := range n.Arguments {
			if err := Walk(v, a); err != nil {
				return err
			}
		}
		for _, val := range n.Kwargs {
			if err := Walk(v, val); err != nil {
				return err
			}
		}

	case *IndexExpression:
		if err := v.VisitIndexExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Left); err != nil {
			return err
		}
		if err := Walk(v, n.Index); err != nil {
			return err
		}

	case *SliceExpression:
		if err := v.VisitSliceExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Left); err != nil {
			return err
		}
		if n.Start != nil {
			if err := Walk(v, n.Start); err != nil {
				return err
			}
		}
		if n.End != nil {
			if err := Walk(v, n.End); err != nil {
				return err
			}
		}
		if n.Step != nil {
			if err := Walk(v, n.Step); err != nil {
				return err
			}
		}

	case *MemberExpression:
		if err := v.VisitMemberExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Object); err != nil {
			return err
		}
		if err := Walk(v, n.Property); err != nil {
			return err
		}

	case *LambdaExpression:
		if err := v.VisitLambdaExpression(n); err != nil {
			return err
		}
		for _, p := range n.Parameters {
			if err := Walk(v, p); err != nil {
				return err
			}
		}
		if err := Walk(v, n.Body); err != nil {
			return err
		}

	case *PipelineExpression:
		if err := v.VisitPipelineExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Left); err != nil {
			return err
		}
		if err := Walk(v, n.Right); err != nil {
			return err
		}

	case *TernaryExpression:
		if err := v.VisitTernaryExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Condition); err != nil {
			return err
		}
		if err := Walk(v, n.Consequence); err != nil {
			return err
		}
		if err := Walk(v, n.Alternative); err != nil {
			return err
		}

	case *MatchExpression:
		if err := v.VisitMatchExpression(n); err != nil {
			return err
		}
		if err := Walk(v, n.Subject); err != nil {
			return err
		}
		for _, arm := range n.Arms {
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

	case *ListComprehension:
		if err := v.VisitListComprehension(n); err != nil {
			return err
		}
		if err := Walk(v, n.Element); err != nil {
			return err
		}
		if err := Walk(v, n.Variable); err != nil {
			return err
		}
		if n.Index != nil {
			if err := Walk(v, n.Index); err != nil {
				return err
			}
		}
		if err := Walk(v, n.Iterable); err != nil {
			return err
		}
		if n.Filter != nil {
			if err := Walk(v, n.Filter); err != nil {
				return err
			}
		}

	case *MapComprehension:
		if err := v.VisitMapComprehension(n); err != nil {
			return err
		}
		if err := Walk(v, n.Key); err != nil {
			return err
		}
		if err := Walk(v, n.Value); err != nil {
			return err
		}
		if err := Walk(v, n.KeyVar); err != nil {
			return err
		}
		if err := Walk(v, n.ValueVar); err != nil {
			return err
		}
		if err := Walk(v, n.Iterable); err != nil {
			return err
		}
		if n.Filter != nil {
			if err := Walk(v, n.Filter); err != nil {
				return err
			}
		}

	case *SchemaExpression:
		if err := v.VisitSchemaExpression(n); err != nil {
			return err
		}
		for _, f := range n.Fields {
			if err := Walk(v, f.Type); err != nil {
				return err
			}
		}

	case *EnumExpression:
		return v.VisitEnumExpression(n)

	case *TypeExpression:
		if err := v.VisitTypeExpression(n); err != nil {
			return err
		}
		if n.Inner != nil {
			if err := Walk(v, n.Inner); err != nil {
				return err
			}
		}
		for _, c := range n.Constraints {
			if err := Walk(v, c); err != nil {
				return err
			}
		}

	case *RangeExpression:
		if err := v.VisitRangeExpression(n); err != nil {
			return err
		}
		if n.Start != nil {
			if err := Walk(v, n.Start); err != nil {
				return err
			}
		}
		if err := Walk(v, n.End); err != nil {
			return err
		}
		if n.Step != nil {
			if err := Walk(v, n.Step); err != nil {
				return err
			}
		}
	}

	return nil
}

// BaseVisitor provides default no-op implementations for all Visitor methods.
// Embed this in your visitor to only implement the methods you care about.
type BaseVisitor struct{}

func (b *BaseVisitor) VisitProgram(*Program) error                     { return nil }
func (b *BaseVisitor) VisitBlock(*Block) error                         { return nil }
func (b *BaseVisitor) VisitExpressionStatement(*ExpressionStatement) error { return nil }
func (b *BaseVisitor) VisitAssignStatement(*AssignStatement) error     { return nil }
func (b *BaseVisitor) VisitIfStatement(*IfStatement) error             { return nil }
func (b *BaseVisitor) VisitForStatement(*ForStatement) error           { return nil }
func (b *BaseVisitor) VisitMatchStatement(*MatchStatement) error       { return nil }
func (b *BaseVisitor) VisitDefStatement(*DefStatement) error           { return nil }
func (b *BaseVisitor) VisitReturnStatement(*ReturnStatement) error     { return nil }
func (b *BaseVisitor) VisitEmitStatement(*EmitStatement) error         { return nil }
func (b *BaseVisitor) VisitStopStatement(*StopStatement) error         { return nil }
func (b *BaseVisitor) VisitTryStatement(*TryStatement) error           { return nil }
func (b *BaseVisitor) VisitBreakStatement(*BreakStatement) error       { return nil }
func (b *BaseVisitor) VisitContinueStatement(*ContinueStatement) error { return nil }
func (b *BaseVisitor) VisitIdentifier(*Identifier) error               { return nil }
func (b *BaseVisitor) VisitIntegerLiteral(*IntegerLiteral) error       { return nil }
func (b *BaseVisitor) VisitFloatLiteral(*FloatLiteral) error           { return nil }
func (b *BaseVisitor) VisitStringLiteral(*StringLiteral) error         { return nil }
func (b *BaseVisitor) VisitBooleanLiteral(*BooleanLiteral) error       { return nil }
func (b *BaseVisitor) VisitNoneLiteral(*NoneLiteral) error             { return nil }
func (b *BaseVisitor) VisitListLiteral(*ListLiteral) error             { return nil }
func (b *BaseVisitor) VisitMapLiteral(*MapLiteral) error               { return nil }
func (b *BaseVisitor) VisitSetLiteral(*SetLiteral) error               { return nil }
func (b *BaseVisitor) VisitPrefixExpression(*PrefixExpression) error   { return nil }
func (b *BaseVisitor) VisitInfixExpression(*InfixExpression) error     { return nil }
func (b *BaseVisitor) VisitCallExpression(*CallExpression) error       { return nil }
func (b *BaseVisitor) VisitIndexExpression(*IndexExpression) error     { return nil }
func (b *BaseVisitor) VisitSliceExpression(*SliceExpression) error     { return nil }
func (b *BaseVisitor) VisitMemberExpression(*MemberExpression) error   { return nil }
func (b *BaseVisitor) VisitLambdaExpression(*LambdaExpression) error   { return nil }
func (b *BaseVisitor) VisitPipelineExpression(*PipelineExpression) error { return nil }
func (b *BaseVisitor) VisitTernaryExpression(*TernaryExpression) error { return nil }
func (b *BaseVisitor) VisitMatchExpression(*MatchExpression) error     { return nil }
func (b *BaseVisitor) VisitListComprehension(*ListComprehension) error { return nil }
func (b *BaseVisitor) VisitMapComprehension(*MapComprehension) error   { return nil }
func (b *BaseVisitor) VisitSchemaExpression(*SchemaExpression) error   { return nil }
func (b *BaseVisitor) VisitEnumExpression(*EnumExpression) error       { return nil }
func (b *BaseVisitor) VisitTypeExpression(*TypeExpression) error       { return nil }
func (b *BaseVisitor) VisitRangeExpression(*RangeExpression) error     { return nil }
