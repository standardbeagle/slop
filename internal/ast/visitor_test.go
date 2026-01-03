package ast

import (
	"errors"
	"testing"
)

// testVisitor is a visitor that tracks visited nodes and can return errors.
type testVisitor struct {
	BaseVisitor
	visited    []string
	errorOn    string // If set, returns an error when visiting this node type
	errorAfter int    // If > 0, returns an error after visiting this many nodes
}

func newTestVisitor() *testVisitor {
	return &testVisitor{
		visited: make([]string, 0),
	}
}

func (tv *testVisitor) checkError(nodeType string) error {
	if tv.errorOn == nodeType {
		return errors.New("test error on " + nodeType)
	}
	if tv.errorAfter > 0 && len(tv.visited) >= tv.errorAfter {
		return errors.New("test error after limit")
	}
	return nil
}

func (tv *testVisitor) VisitIdentifier(n *Identifier) error {
	tv.visited = append(tv.visited, "Identifier:"+n.Value)
	return tv.checkError("Identifier")
}

func (tv *testVisitor) VisitIntegerLiteral(n *IntegerLiteral) error {
	tv.visited = append(tv.visited, "IntegerLiteral")
	return tv.checkError("IntegerLiteral")
}

func (tv *testVisitor) VisitStringLiteral(n *StringLiteral) error {
	tv.visited = append(tv.visited, "StringLiteral:"+n.Value)
	return tv.checkError("StringLiteral")
}

func (tv *testVisitor) VisitBooleanLiteral(n *BooleanLiteral) error {
	tv.visited = append(tv.visited, "BooleanLiteral")
	return tv.checkError("BooleanLiteral")
}

func (tv *testVisitor) VisitBlock(n *Block) error {
	tv.visited = append(tv.visited, "Block")
	return tv.checkError("Block")
}

func (tv *testVisitor) VisitExpressionStatement(n *ExpressionStatement) error {
	tv.visited = append(tv.visited, "ExpressionStatement")
	return tv.checkError("ExpressionStatement")
}

// TestWalkSlice_Empty tests that WalkSlice handles empty slices correctly.
func TestWalkSlice_Empty(t *testing.T) {
	v := newTestVisitor()

	// Empty slice of statements
	err := WalkSlice(v, []Statement{})
	if err != nil {
		t.Errorf("WalkSlice on empty slice returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkSlice_WithNodes tests that WalkSlice visits all nodes in order.
func TestWalkSlice_WithNodes(t *testing.T) {
	v := newTestVisitor()

	nodes := []Expression{
		&Identifier{Value: "a"},
		&Identifier{Value: "b"},
		&Identifier{Value: "c"},
	}

	err := WalkSlice(v, nodes)
	if err != nil {
		t.Errorf("WalkSlice returned error: %v", err)
	}

	expected := []string{"Identifier:a", "Identifier:b", "Identifier:c"}
	if len(v.visited) != len(expected) {
		t.Errorf("Expected %d visited nodes, got %d", len(expected), len(v.visited))
	}

	for i, exp := range expected {
		if v.visited[i] != exp {
			t.Errorf("visited[%d] = %s, expected %s", i, v.visited[i], exp)
		}
	}
}

// TestWalkSlice_ErrorPropagation tests that WalkSlice stops and propagates errors.
func TestWalkSlice_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorAfter = 2 // Error after visiting 2 nodes

	nodes := []Expression{
		&Identifier{Value: "a"},
		&Identifier{Value: "b"},
		&Identifier{Value: "c"},
	}

	err := WalkSlice(v, nodes)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Should have stopped after 2 nodes
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes before error, got %d", len(v.visited))
	}
}

// TestWalkOptional_Nil tests that WalkOptional handles nil correctly.
func TestWalkOptional_Nil(t *testing.T) {
	v := newTestVisitor()

	err := WalkOptional(v, nil)
	if err != nil {
		t.Errorf("WalkOptional on nil returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes for nil, got %d", len(v.visited))
	}
}

// TestWalkOptional_NonNil tests that WalkOptional traverses non-nil nodes.
func TestWalkOptional_NonNil(t *testing.T) {
	v := newTestVisitor()

	node := &Identifier{Value: "test"}
	err := WalkOptional(v, node)
	if err != nil {
		t.Errorf("WalkOptional returned error: %v", err)
	}

	if len(v.visited) != 1 {
		t.Errorf("Expected 1 visited node, got %d", len(v.visited))
	}

	if v.visited[0] != "Identifier:test" {
		t.Errorf("Expected Identifier:test, got %s", v.visited[0])
	}
}

// TestWalkOptional_ErrorPropagation tests that WalkOptional propagates errors.
func TestWalkOptional_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "Identifier"

	node := &Identifier{Value: "test"}
	err := WalkOptional(v, node)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestWalkMatchArms_Empty tests walkMatchArms with empty arms.
func TestWalkMatchArms_Empty(t *testing.T) {
	v := newTestVisitor()

	err := walkMatchArms(v, []*MatchArm{})
	if err != nil {
		t.Errorf("walkMatchArms on empty slice returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkMatchArms_WithGuard tests walkMatchArms with guard expressions.
func TestWalkMatchArms_WithGuard(t *testing.T) {
	v := newTestVisitor()

	arms := []*MatchArm{
		{
			Pattern: &Identifier{Value: "pattern1"},
			Guard:   &Identifier{Value: "guard1"},
			Body:    &Identifier{Value: "body1"},
		},
		{
			Pattern: &Identifier{Value: "pattern2"},
			Guard:   nil, // No guard
			Body:    &Identifier{Value: "body2"},
		},
	}

	err := walkMatchArms(v, arms)
	if err != nil {
		t.Errorf("walkMatchArms returned error: %v", err)
	}

	// Should visit: pattern1, guard1, body1, pattern2, body2
	expected := []string{
		"Identifier:pattern1",
		"Identifier:guard1",
		"Identifier:body1",
		"Identifier:pattern2",
		"Identifier:body2",
	}

	if len(v.visited) != len(expected) {
		t.Errorf("Expected %d visited nodes, got %d: %v", len(expected), len(v.visited), v.visited)
	}

	for i, exp := range expected {
		if i < len(v.visited) && v.visited[i] != exp {
			t.Errorf("visited[%d] = %s, expected %s", i, v.visited[i], exp)
		}
	}
}

// TestWalkMatchArms_ErrorOnPattern tests error propagation on pattern.
func TestWalkMatchArms_ErrorOnPattern(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "Identifier"

	arms := []*MatchArm{
		{
			Pattern: &Identifier{Value: "pattern"},
			Body:    &IntegerLiteral{Value: 1},
		},
	}

	err := walkMatchArms(v, arms)
	if err == nil {
		t.Error("Expected error on pattern, got nil")
	}
}

// TestWalkMatchArms_ErrorOnGuard tests error propagation on guard.
func TestWalkMatchArms_ErrorOnGuard(t *testing.T) {
	v := newTestVisitor()
	v.errorAfter = 2 // Error after pattern and guard

	arms := []*MatchArm{
		{
			Pattern: &IntegerLiteral{Value: 1},
			Guard:   &Identifier{Value: "guard"},
			Body:    &Identifier{Value: "body"},
		},
	}

	err := walkMatchArms(v, arms)
	if err == nil {
		t.Error("Expected error on guard, got nil")
	}

	// Should have visited pattern and guard before error
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkNamedMap_Empty tests walkNamedMap with empty map.
func TestWalkNamedMap_Empty(t *testing.T) {
	v := newTestVisitor()

	err := walkNamedMap(v, map[string]Expression{})
	if err != nil {
		t.Errorf("walkNamedMap on empty map returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkNamedMap_WithEntries tests walkNamedMap with entries.
func TestWalkNamedMap_WithEntries(t *testing.T) {
	v := newTestVisitor()

	named := map[string]Expression{
		"a": &Identifier{Value: "val_a"},
		"b": &Identifier{Value: "val_b"},
	}

	err := walkNamedMap(v, named)
	if err != nil {
		t.Errorf("walkNamedMap returned error: %v", err)
	}

	// Should have visited 2 values (order is non-deterministic)
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkNamedMap_ErrorPropagation tests error propagation in walkNamedMap.
func TestWalkNamedMap_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "Identifier"

	named := map[string]Expression{
		"a": &Identifier{Value: "val"},
	}

	err := walkNamedMap(v, named)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestWalkCatchClauses_Empty tests walkCatchClauses with empty clauses.
func TestWalkCatchClauses_Empty(t *testing.T) {
	v := newTestVisitor()

	err := walkCatchClauses(v, []*CatchClause{})
	if err != nil {
		t.Errorf("walkCatchClauses on empty slice returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkCatchClauses_WithClauses tests walkCatchClauses with clauses.
func TestWalkCatchClauses_WithClauses(t *testing.T) {
	v := newTestVisitor()

	catches := []*CatchClause{
		{
			Body: &Block{Statements: []Statement{}},
		},
		{
			Body: &Block{Statements: []Statement{}},
		},
	}

	err := walkCatchClauses(v, catches)
	if err != nil {
		t.Errorf("walkCatchClauses returned error: %v", err)
	}

	// Should have visited 2 blocks
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkCatchClauses_ErrorPropagation tests error propagation.
func TestWalkCatchClauses_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "Block"

	catches := []*CatchClause{
		{
			Body: &Block{Statements: []Statement{}},
		},
	}

	err := walkCatchClauses(v, catches)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestWalkSchemaFields_Empty tests walkSchemaFields with empty fields.
func TestWalkSchemaFields_Empty(t *testing.T) {
	v := newTestVisitor()

	err := walkSchemaFields(v, []*SchemaField{})
	if err != nil {
		t.Errorf("walkSchemaFields on empty slice returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkSchemaFields_WithFields tests walkSchemaFields with fields.
func TestWalkSchemaFields_WithFields(t *testing.T) {
	v := newTestVisitor()

	fields := []*SchemaField{
		{Name: "field1", Type: &Identifier{Value: "string"}},
		{Name: "field2", Type: &Identifier{Value: "int"}},
	}

	err := walkSchemaFields(v, fields)
	if err != nil {
		t.Errorf("walkSchemaFields returned error: %v", err)
	}

	// Should have visited 2 type expressions
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkSchemaFields_ErrorPropagation tests error propagation.
func TestWalkSchemaFields_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "Identifier"

	fields := []*SchemaField{
		{Name: "field1", Type: &Identifier{Value: "string"}},
	}

	err := walkSchemaFields(v, fields)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestWalkConstraints_Empty tests walkConstraints with empty constraints.
func TestWalkConstraints_Empty(t *testing.T) {
	v := newTestVisitor()

	err := walkConstraints(v, map[string]Expression{})
	if err != nil {
		t.Errorf("walkConstraints on empty map returned error: %v", err)
	}

	if len(v.visited) != 0 {
		t.Errorf("Expected 0 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkConstraints_WithConstraints tests walkConstraints with constraints.
func TestWalkConstraints_WithConstraints(t *testing.T) {
	v := newTestVisitor()

	constraints := map[string]Expression{
		"min": &IntegerLiteral{Value: 0},
		"max": &IntegerLiteral{Value: 100},
	}

	err := walkConstraints(v, constraints)
	if err != nil {
		t.Errorf("walkConstraints returned error: %v", err)
	}

	// Should have visited 2 constraints (order is non-deterministic)
	if len(v.visited) != 2 {
		t.Errorf("Expected 2 visited nodes, got %d", len(v.visited))
	}
}

// TestWalkConstraints_ErrorPropagation tests error propagation.
func TestWalkConstraints_ErrorPropagation(t *testing.T) {
	v := newTestVisitor()
	v.errorOn = "IntegerLiteral"

	constraints := map[string]Expression{
		"min": &IntegerLiteral{Value: 0},
	}

	err := walkConstraints(v, constraints)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestWalk_NilNode tests that Walk handles non-Walkable nodes through the switch.
func TestWalk_NilNode(t *testing.T) {
	v := newTestVisitor()

	// The walkNodeSwitch should handle nodes that don't implement Walkable
	// by returning nil for unknown types
	err := walkNodeSwitch(v, nil)
	if err != nil {
		t.Errorf("walkNodeSwitch on nil returned error: %v", err)
	}
}

// TestWalkableInterface tests that nodes implementing Walkable use their WalkNode.
func TestWalkableInterface(t *testing.T) {
	v := newTestVisitor()

	// Identifier implements Walkable
	node := &Identifier{Value: "test"}
	err := Walk(v, node)
	if err != nil {
		t.Errorf("Walk returned error: %v", err)
	}

	if len(v.visited) != 1 || v.visited[0] != "Identifier:test" {
		t.Errorf("Expected [Identifier:test], got %v", v.visited)
	}
}

// TestBaseVisitor tests that BaseVisitor provides no-op implementations.
func TestBaseVisitor(t *testing.T) {
	bv := &BaseVisitor{}

	// All methods should return nil
	if err := bv.VisitProgram(nil); err != nil {
		t.Error("VisitProgram should return nil")
	}
	if err := bv.VisitBlock(nil); err != nil {
		t.Error("VisitBlock should return nil")
	}
	if err := bv.VisitIdentifier(nil); err != nil {
		t.Error("VisitIdentifier should return nil")
	}
	if err := bv.VisitIntegerLiteral(nil); err != nil {
		t.Error("VisitIntegerLiteral should return nil")
	}
	if err := bv.VisitFloatLiteral(nil); err != nil {
		t.Error("VisitFloatLiteral should return nil")
	}
	if err := bv.VisitStringLiteral(nil); err != nil {
		t.Error("VisitStringLiteral should return nil")
	}
	if err := bv.VisitBooleanLiteral(nil); err != nil {
		t.Error("VisitBooleanLiteral should return nil")
	}
	if err := bv.VisitNoneLiteral(nil); err != nil {
		t.Error("VisitNoneLiteral should return nil")
	}
}
