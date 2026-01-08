// Package analyzer provides semantic analysis for SLOP programs.
// It verifies that programs are well-formed, will terminate, and
// have valid bounds.
package analyzer

import (
	"fmt"

	"github.com/standardbeagle/slop/internal/ast"
)

// Analyzer performs semantic analysis on SLOP programs.
type Analyzer struct {
	program *ast.Program
	errors  []AnalysisError

	// Function definitions for call graph analysis
	functions map[string]*ast.DefStatement

	// Analysis results
	bounds *BoundsInfo
}

// AnalysisError represents a semantic analysis error.
type AnalysisError struct {
	Line    int
	Column  int
	Message string
	Kind    ErrorKind
}

// ErrorKind categorizes analysis errors.
type ErrorKind string

const (
	ErrorRecursion     ErrorKind = "recursion"
	ErrorCycle         ErrorKind = "cycle"
	ErrorUnbounded     ErrorKind = "unbounded"
	ErrorType          ErrorKind = "type"
	ErrorUndefined     ErrorKind = "undefined"
	ErrorInvalidBounds ErrorKind = "invalid_bounds"
)

func (e AnalysisError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Line, e.Column, e.Message)
}

// BoundsInfo contains computed bounds for the program.
type BoundsInfo struct {
	MaxIterations int64   // Total maximum loop iterations
	MaxLLMCalls   int64   // Maximum LLM calls possible
	MaxAPICalls   int64   // Maximum API calls possible
	MaxCost       float64 // Estimated max cost (if estimable)
}

// New creates a new Analyzer.
func New() *Analyzer {
	return &Analyzer{
		functions: make(map[string]*ast.DefStatement),
		bounds:    &BoundsInfo{},
	}
}

// Analyze performs all semantic analysis passes on a program.
func (a *Analyzer) Analyze(program *ast.Program) []AnalysisError {
	a.program = program
	a.errors = nil

	// Pass 1: Collect function definitions
	a.collectFunctions(program)

	// Pass 2: Check for recursion and cycles
	a.checkTermination(program)

	// Pass 3: Compute bounds
	a.computeBounds(program)

	return a.errors
}

// Bounds returns the computed bounds for the program.
func (a *Analyzer) Bounds() *BoundsInfo {
	return a.bounds
}

// collectFunctions extracts all function definitions.
func (a *Analyzer) collectFunctions(program *ast.Program) {
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.DefStatement); ok {
			a.functions[fn.Name.Value] = fn
		}
	}
}

// checkTermination verifies the program will terminate.
func (a *Analyzer) checkTermination(program *ast.Program) {
	// Build call graph
	callGraph := buildCallGraph(a.functions)

	// Check for cycles (recursion)
	cycles := findCycles(callGraph)
	for _, cycle := range cycles {
		a.addError(0, 0, ErrorCycle,
			fmt.Sprintf("recursive call cycle detected: %s", formatCycle(cycle)))
	}

	// Check for direct self-recursion
	for name, fn := range a.functions {
		if callsSelf(fn, name) {
			a.addError(fn.Token.Line, fn.Token.Column, ErrorRecursion,
				fmt.Sprintf("function %q calls itself (recursion not allowed)", name))
		}
	}
}

// computeBounds calculates maximum iterations and calls.
func (a *Analyzer) computeBounds(program *ast.Program) {
	a.walkNode(program, 0)
}

// walkNode recursively walks the AST computing bounds.
func (a *Analyzer) walkNode(node ast.Node, loopDepth int) {
	switch n := node.(type) {
	case *ast.Program:
		for _, stmt := range n.Statements {
			a.walkNode(stmt, loopDepth)
		}

	case *ast.Block:
		for _, stmt := range n.Statements {
			a.walkNode(stmt, loopDepth)
		}

	case *ast.DefStatement:
		a.walkNode(n.Body, loopDepth)

	case *ast.ForStatement:
		// Track loop bounds
		limit := extractLimit(n)
		if loopDepth == 0 {
			a.bounds.MaxIterations += limit
		} else {
			// Nested loop multiplies
			a.bounds.MaxIterations *= limit
		}
		a.walkNode(n.Body, loopDepth+1)

	case *ast.IfStatement:
		a.walkNode(n.Consequence, loopDepth)
		if n.Alternative != nil {
			a.walkNode(n.Alternative, loopDepth)
		}

	case *ast.MatchStatement:
		for _, arm := range n.Arms {
			a.walkNode(arm.Body, loopDepth)
		}

	case *ast.TryStatement:
		a.walkNode(n.Body, loopDepth)
		for _, c := range n.Catches {
			a.walkNode(c.Body, loopDepth)
		}

	case *ast.ExpressionStatement:
		a.walkExpr(n.Expression, loopDepth)

	case *ast.AssignStatement:
		a.walkExpr(n.Value, loopDepth)
	}
}

// walkExpr walks an expression looking for service calls.
func (a *Analyzer) walkExpr(expr ast.Expression, loopDepth int) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		// Check for LLM calls
		if member, ok := e.Function.(*ast.MemberExpression); ok {
			if ident, ok := member.Object.(*ast.Identifier); ok {
				if ident.Value == "llm" && member.Property.Value == "call" {
					a.bounds.MaxLLMCalls++
				} else {
					// Any other service call
					a.bounds.MaxAPICalls++
				}
			}
		}
		// Walk arguments
		for _, arg := range e.Arguments {
			a.walkExpr(arg, loopDepth)
		}
		for _, val := range e.Kwargs {
			a.walkExpr(val, loopDepth)
		}

	case *ast.InfixExpression:
		a.walkExpr(e.Left, loopDepth)
		a.walkExpr(e.Right, loopDepth)

	case *ast.PrefixExpression:
		a.walkExpr(e.Right, loopDepth)

	case *ast.IndexExpression:
		a.walkExpr(e.Left, loopDepth)
		a.walkExpr(e.Index, loopDepth)

	case *ast.MemberExpression:
		a.walkExpr(e.Object, loopDepth)

	case *ast.ListLiteral:
		for _, el := range e.Elements {
			a.walkExpr(el, loopDepth)
		}

	case *ast.MapLiteral:
		for k, v := range e.Pairs {
			a.walkExpr(k, loopDepth)
			a.walkExpr(v, loopDepth)
		}

	case *ast.PipelineExpression:
		a.walkExpr(e.Left, loopDepth)
		a.walkExpr(e.Right, loopDepth)

	case *ast.TernaryExpression:
		a.walkExpr(e.Condition, loopDepth)
		a.walkExpr(e.Consequence, loopDepth)
		a.walkExpr(e.Alternative, loopDepth)

	case *ast.LambdaExpression:
		a.walkExpr(e.Body, loopDepth)
	}
}

// addError adds an analysis error.
func (a *Analyzer) addError(line, column int, kind ErrorKind, message string) {
	a.errors = append(a.errors, AnalysisError{
		Line:    line,
		Column:  column,
		Kind:    kind,
		Message: message,
	})
}

// buildCallGraph builds a graph of function calls.
func buildCallGraph(functions map[string]*ast.DefStatement) map[string][]string {
	graph := make(map[string][]string)

	for name, fn := range functions {
		calls := findCalls(fn.Body)
		graph[name] = calls
	}

	return graph
}

// findCalls finds all function calls in a node.
func findCalls(node ast.Node) []string {
	var calls []string
	findCallsIn(node, &calls)
	return calls
}

func findCallsIn(node ast.Node, calls *[]string) {
	switch n := node.(type) {
	case *ast.Block:
		for _, stmt := range n.Statements {
			findCallsIn(stmt, calls)
		}

	case *ast.ExpressionStatement:
		findCallsInExpr(n.Expression, calls)

	case *ast.AssignStatement:
		findCallsInExpr(n.Value, calls)

	case *ast.IfStatement:
		findCallsInExpr(n.Condition, calls)
		findCallsIn(n.Consequence, calls)
		if n.Alternative != nil {
			findCallsIn(n.Alternative, calls)
		}

	case *ast.ForStatement:
		findCallsInExpr(n.Iterable, calls)
		findCallsIn(n.Body, calls)

	case *ast.ReturnStatement:
		if n.Value != nil {
			findCallsInExpr(n.Value, calls)
		}
	}
}

func findCallsInExpr(expr ast.Expression, calls *[]string) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			*calls = append(*calls, ident.Value)
		}
		for _, arg := range e.Arguments {
			findCallsInExpr(arg, calls)
		}
		for _, val := range e.Kwargs {
			findCallsInExpr(val, calls)
		}

	case *ast.InfixExpression:
		findCallsInExpr(e.Left, calls)
		findCallsInExpr(e.Right, calls)

	case *ast.PrefixExpression:
		findCallsInExpr(e.Right, calls)

	case *ast.IndexExpression:
		findCallsInExpr(e.Left, calls)
		findCallsInExpr(e.Index, calls)

	case *ast.MemberExpression:
		findCallsInExpr(e.Object, calls)

	case *ast.PipelineExpression:
		findCallsInExpr(e.Left, calls)
		findCallsInExpr(e.Right, calls)

	case *ast.TernaryExpression:
		findCallsInExpr(e.Condition, calls)
		findCallsInExpr(e.Consequence, calls)
		findCallsInExpr(e.Alternative, calls)

	case *ast.LambdaExpression:
		findCallsInExpr(e.Body, calls)
	}
}

// callsSelf checks if a function calls itself directly.
func callsSelf(fn *ast.DefStatement, name string) bool {
	calls := findCalls(fn.Body)
	for _, call := range calls {
		if call == name {
			return true
		}
	}
	return false
}

// findCycles detects cycles in the call graph using DFS.
func findCycles(graph map[string][]string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycle = append(cycle, neighbor)
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return false
	}

	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

func formatCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	result := cycle[0]
	for i := 1; i < len(cycle); i++ {
		result += " -> " + cycle[i]
	}
	return result
}

// extractLimit extracts the limit from a for statement.
func extractLimit(forStmt *ast.ForStatement) int64 {
	// Check for RangeExpression (range(10) or range(0, 10))
	if rangeExpr, ok := forStmt.Iterable.(*ast.RangeExpression); ok {
		// For range(end), End contains the limit
		// For range(start, end), we need End - Start
		if rangeExpr.Start == nil {
			// range(end) - just use End
			if intLit, ok := rangeExpr.End.(*ast.IntegerLiteral); ok {
				return intLit.Value
			}
		} else {
			// range(start, end) - compute end - start
			startLit, startOk := rangeExpr.Start.(*ast.IntegerLiteral)
			endLit, endOk := rangeExpr.End.(*ast.IntegerLiteral)
			if startOk && endOk {
				return endLit.Value - startLit.Value
			}
		}
	}

	// Check for CallExpression (backwards compatibility)
	if call, ok := forStmt.Iterable.(*ast.CallExpression); ok {
		if ident, ok := call.Function.(*ast.Identifier); ok {
			if ident.Value == "range" && len(call.Arguments) >= 1 {
				if intLit, ok := call.Arguments[0].(*ast.IntegerLiteral); ok {
					return intLit.Value
				}
			}
		}
	}

	// Check for limit modifier
	for _, mod := range forStmt.Modifiers {
		if mod.Type == "limit" {
			if intLit, ok := mod.Value.(*ast.IntegerLiteral); ok {
				return intLit.Value
			}
		}
	}

	// Default bound (unknown, use large but finite default)
	return 1000
}
