package evaluator

import (
	"fmt"

	"github.com/standardbeagle/slop/internal/ast"
)

// ValidateMachineProgram conservatively verifies that every node can be
// evaluated by Machine. It performs no evaluation and has no side effects.
func ValidateMachineProgram(program *ast.Program) error {
	if program == nil {
		return fmt.Errorf("resumable program is nil")
	}
	if len(program.Modules) != 0 {
		return fmt.Errorf("resumable execution does not support modules")
	}
	return validateMachineNode(program)
}

func validateMachineNode(node ast.Node) error {
	if node == nil {
		return nil
	}
	check := func(child ast.Node) error { return validateMachineNode(child) }
	checkAll := func(children []ast.Expression) error {
		for _, child := range children {
			if err := check(child); err != nil {
				return err
			}
		}
		return nil
	}
	switch n := node.(type) {
	case *ast.Program:
		for _, child := range n.Statements {
			if err := check(child); err != nil {
				return err
			}
		}
	case *ast.Block:
		for _, child := range n.Statements {
			if err := check(child); err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		return check(n.Expression)
	case *ast.AssignStatement:
		if len(n.Targets) != 1 || n.Operator != "=" {
			return fmt.Errorf("unsupported resumable assignment at %s", n.Token.Literal)
		}
		if _, ok := n.Targets[0].(*ast.Identifier); !ok {
			return fmt.Errorf("unsupported resumable assignment target %T", n.Targets[0])
		}
		return check(n.Value)
	case *ast.InfixExpression:
		if err := check(n.Left); err != nil {
			return err
		}
		return check(n.Right)
	case *ast.PrefixExpression:
		return check(n.Right)
	case *ast.ListLiteral:
		return checkAll(n.Elements)
	case *ast.SetLiteral:
		return checkAll(n.Elements)
	case *ast.MapLiteral:
		for _, key := range n.Order {
			if err := check(n.Pairs[key]); err != nil {
				return err
			}
		}
	case *ast.IndexExpression:
		if err := check(n.Left); err != nil {
			return err
		}
		return check(n.Index)
	case *ast.TernaryExpression:
		if err := check(n.Condition); err != nil {
			return err
		}
		if err := check(n.Consequence); err != nil {
			return err
		}
		return check(n.Alternative)
	case *ast.RangeExpression:
		if err := check(n.Start); err != nil {
			return err
		}
		if err := check(n.End); err != nil {
			return err
		}
		return check(n.Step)
	case *ast.IfStatement:
		if err := check(n.Condition); err != nil {
			return err
		}
		if err := check(n.Consequence); err != nil {
			return err
		}
		return check(n.Alternative)
	case *ast.EmitStatement:
		return checkAll(n.Values)
	case *ast.ReturnStatement:
		return check(n.Value)
	case *ast.PauseStatement:
		return check(n.Message)
	case *ast.MemberExpression:
		if err := check(n.Object); err != nil {
			return err
		}
		return check(n.Property)
	case *ast.CallExpression:
		if _, ok := n.Function.(*ast.MemberExpression); !ok {
			return fmt.Errorf("unsupported resumable call target %T (only service method calls are supported)", n.Function)
		}
		if err := check(n.Function); err != nil {
			return err
		}
		if err := checkAll(n.Arguments); err != nil {
			return err
		}
		for _, child := range n.Kwargs {
			if err := check(child); err != nil {
				return err
			}
		}
	case *ast.Identifier, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.StringLiteral, *ast.BooleanLiteral, *ast.NoneLiteral:
		return nil
	default:
		return fmt.Errorf("unsupported resumable node %T", node)
	}
	return nil
}
