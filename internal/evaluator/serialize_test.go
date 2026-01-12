package evaluator

import (
	"encoding/json"
	"testing"
)

func TestSerializeValue_Primitives(t *testing.T) {
	s := NewSerializer(nil)

	tests := []struct {
		name     string
		value    Value
		wantType string
	}{
		{"none", NONE, "none"},
		{"true", TRUE, "bool"},
		{"false", FALSE, "bool"},
		{"int", &IntValue{Value: 42}, "int"},
		{"int_large", &IntValue{Value: 9223372036854775807}, "int"},
		{"int_negative", &IntValue{Value: -9223372036854775808}, "int"},
		{"float", &FloatValue{Value: 3.14159}, "float"},
		{"string", &StringValue{Value: "hello world"}, "string"},
		{"string_empty", &StringValue{Value: ""}, "string"},
		{"string_unicode", &StringValue{Value: "こんにちは"}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := s.SerializeValue(tt.value)
			if err != nil {
				t.Fatalf("SerializeValue() error = %v", err)
			}
			if sv.Type != tt.wantType {
				t.Errorf("SerializeValue() type = %v, want %v", sv.Type, tt.wantType)
			}
		})
	}
}

func TestSerializeValue_List(t *testing.T) {
	s := NewSerializer(nil)

	list := &ListValue{
		Elements: []Value{
			&IntValue{Value: 1},
			&StringValue{Value: "two"},
			&BoolValue{Value: true},
		},
	}

	sv, err := s.SerializeValue(list)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "list" {
		t.Errorf("SerializeValue() type = %v, want list", sv.Type)
	}

	// Verify JSON structure
	var elements []*SerializedValue
	if err := json.Unmarshal(sv.Data, &elements); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(elements) != 3 {
		t.Errorf("list has %d elements, want 3", len(elements))
	}
}

func TestSerializeValue_Map(t *testing.T) {
	s := NewSerializer(nil)

	m := NewMapValue()
	m.Set("name", &StringValue{Value: "test"})
	m.Set("count", &IntValue{Value: 42})

	sv, err := s.SerializeValue(m)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "map" {
		t.Errorf("SerializeValue() type = %v, want map", sv.Type)
	}

	// Verify JSON structure
	var sm SerializedMap
	if err := json.Unmarshal(sv.Data, &sm); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(sm.Pairs) != 2 {
		t.Errorf("map has %d pairs, want 2", len(sm.Pairs))
	}
	if len(sm.Order) != 2 {
		t.Errorf("map has %d order entries, want 2", len(sm.Order))
	}
	// Verify order is preserved
	if sm.Order[0] != "name" || sm.Order[1] != "count" {
		t.Errorf("map order = %v, want [name, count]", sm.Order)
	}
}

func TestSerializeValue_NestedStructures(t *testing.T) {
	s := NewSerializer(nil)

	// Create nested structure: {"items": [1, {"nested": true}]}
	innerMap := NewMapValue()
	innerMap.Set("nested", TRUE)

	innerList := &ListValue{
		Elements: []Value{
			&IntValue{Value: 1},
			innerMap,
		},
	}

	outerMap := NewMapValue()
	outerMap.Set("items", innerList)

	sv, err := s.SerializeValue(outerMap)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "map" {
		t.Errorf("SerializeValue() type = %v, want map", sv.Type)
	}
}

func TestSerializeValue_Set(t *testing.T) {
	s := NewSerializer(nil)

	set := NewSetValue()
	set.Add(&StringValue{Value: "a"})
	set.Add(&StringValue{Value: "b"})
	set.Add(&StringValue{Value: "a"}) // Duplicate

	sv, err := s.SerializeValue(set)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "set" {
		t.Errorf("SerializeValue() type = %v, want set", sv.Type)
	}

	// Verify JSON structure - should have only 2 elements (deduplicated)
	var elements []*SerializedValue
	if err := json.Unmarshal(sv.Data, &elements); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(elements) != 2 {
		t.Errorf("set has %d elements, want 2", len(elements))
	}
}

func TestSerializeValue_Error(t *testing.T) {
	s := NewSerializer(nil)

	err := &SlopError{
		Message: "test error",
		Data:    &IntValue{Value: 42},
	}

	sv, serErr := s.SerializeValue(err)
	if serErr != nil {
		t.Fatalf("SerializeValue() error = %v", serErr)
	}
	if sv.Type != "error" {
		t.Errorf("SerializeValue() type = %v, want error", sv.Type)
	}

	// Verify JSON structure
	var se SerializedError
	if err := json.Unmarshal(sv.Data, &se); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if se.Message != "test error" {
		t.Errorf("error message = %v, want 'test error'", se.Message)
	}
	if se.Data == nil {
		t.Error("error data is nil, want non-nil")
	}
}

func TestSerializeValue_Iterator(t *testing.T) {
	s := NewSerializer(nil)

	// Range iterator
	rangeIter := &IteratorValue{
		Type_:   "range",
		Current: 5,
		End:     10,
		Step:    1,
	}

	sv, err := s.SerializeValue(rangeIter)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "iterator" {
		t.Errorf("SerializeValue() type = %v, want iterator", sv.Type)
	}

	// List iterator
	listIter := &IteratorValue{
		Current: 1,
		Items: []Value{
			&StringValue{Value: "a"},
			&StringValue{Value: "b"},
		},
	}

	sv, err = s.SerializeValue(listIter)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "iterator" {
		t.Errorf("SerializeValue() type = %v, want iterator", sv.Type)
	}
}

func TestSerializeValue_Builtin(t *testing.T) {
	s := NewSerializer(nil)

	builtin := &BuiltinValue{
		Name: "len",
		Fn:   nil, // Function not serialized
	}

	sv, err := s.SerializeValue(builtin)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}
	if sv.Type != "builtin" {
		t.Errorf("SerializeValue() type = %v, want builtin", sv.Type)
	}

	var name string
	if err := json.Unmarshal(sv.Data, &name); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if name != "len" {
		t.Errorf("builtin name = %v, want 'len'", name)
	}
}

func TestSerializeScope(t *testing.T) {
	s := NewSerializer(nil)

	scope := NewScope()
	scope.Set("x", &IntValue{Value: 42})
	scope.Set("name", &StringValue{Value: "test"})

	ss, err := s.SerializeScope(scope, false)
	if err != nil {
		t.Fatalf("SerializeScope() error = %v", err)
	}

	if ss.ID == "" {
		t.Error("scope ID is empty")
	}
	if len(ss.Variables) != 2 {
		t.Errorf("scope has %d variables, want 2", len(ss.Variables))
	}
	if ss.IsGlobal {
		t.Error("scope IsGlobal = true, want false")
	}
}

func TestSerializeScopeChain(t *testing.T) {
	s := NewSerializer(nil)

	// Create scope chain: global -> parent -> current
	global := NewScope()
	global.Set("builtin", &StringValue{Value: "builtin_value"})

	parent := NewEnclosedScope(global)
	parent.Set("x", &IntValue{Value: 1})

	current := NewEnclosedScope(parent)
	current.Set("y", &IntValue{Value: 2})

	scopes, currentID, err := s.SerializeScopeChain(current, global)
	if err != nil {
		t.Fatalf("SerializeScopeChain() error = %v", err)
	}

	if len(scopes) != 3 {
		t.Errorf("scope chain has %d scopes, want 3", len(scopes))
	}

	// First scope should be global (root)
	if !scopes[0].IsGlobal {
		t.Error("first scope should be global")
	}
	if scopes[0].ParentID != nil {
		t.Error("global scope should not have parent")
	}

	// Verify current scope ID
	if currentID != scopes[2].ID {
		t.Errorf("currentID = %v, want %v", currentID, scopes[2].ID)
	}

	// Verify parent references
	if scopes[1].ParentID == nil || *scopes[1].ParentID != scopes[0].ID {
		t.Error("parent scope should reference global")
	}
	if scopes[2].ParentID == nil || *scopes[2].ParentID != scopes[1].ID {
		t.Error("current scope should reference parent")
	}
}

func TestSerializeLimits(t *testing.T) {
	limits := &ExecutionLimits{
		MaxIterations:  1000,
		MaxLLMCalls:    50,
		MaxAPICalls:    100,
		MaxDuration:    3600,
		MaxCost:        10.0,
		IterationCount: 500,
		LLMCallCount:   25,
		APICallCount:   50,
		StartTime:      1234567890,
		TotalCost:      5.0,
	}

	sl := SerializeLimits(limits)

	if sl.MaxIterations != 1000 {
		t.Errorf("MaxIterations = %d, want 1000", sl.MaxIterations)
	}
	if sl.IterationCount != 500 {
		t.Errorf("IterationCount = %d, want 500", sl.IterationCount)
	}
	if sl.TotalCost != 5.0 {
		t.Errorf("TotalCost = %f, want 5.0", sl.TotalCost)
	}
}

func TestSerializeContext(t *testing.T) {
	s := NewSerializer(nil)
	ctx := NewContext()

	// Set up some state
	ctx.Scope.Set("x", &IntValue{Value: 42})
	ctx.Emit(&StringValue{Value: "result"})

	sc, err := s.SerializeContext(ctx)
	if err != nil {
		t.Fatalf("SerializeContext() error = %v", err)
	}

	if len(sc.Scopes) == 0 {
		t.Error("serialized context has no scopes")
	}
	if sc.CurrentScopeID == "" {
		t.Error("serialized context has no current scope ID")
	}
	if len(sc.Emitted) != 1 {
		t.Errorf("serialized context has %d emitted values, want 1", len(sc.Emitted))
	}
}

func TestCreateCheckpoint(t *testing.T) {
	s := NewSerializer(nil)
	ctx := NewContext()
	ctx.Scope.Set("x", &IntValue{Value: 42})

	script := "x = 42\npause \"test\""
	pos := Position{Line: 2, Column: 1, StatementIndex: 1}

	checkpoint, err := s.CreateCheckpoint(ctx, script, pos, "test checkpoint")
	if err != nil {
		t.Fatalf("CreateCheckpoint() error = %v", err)
	}

	if checkpoint.Version != CheckpointVersion {
		t.Errorf("Version = %v, want %v", checkpoint.Version, CheckpointVersion)
	}
	if checkpoint.ScriptHash == "" {
		t.Error("ScriptHash is empty")
	}
	if checkpoint.Position.Line != 2 {
		t.Errorf("Position.Line = %d, want 2", checkpoint.Position.Line)
	}
	if checkpoint.CheckpointMessage != "test checkpoint" {
		t.Errorf("CheckpointMessage = %v, want 'test checkpoint'", checkpoint.CheckpointMessage)
	}
	if checkpoint.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestHashScript(t *testing.T) {
	script := "x = 42"
	hash1 := HashScript(script)
	hash2 := HashScript(script)

	if hash1 != hash2 {
		t.Error("HashScript not deterministic")
	}
	if len(hash1) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash1))
	}

	// Different script should have different hash
	hash3 := HashScript("y = 43")
	if hash1 == hash3 {
		t.Error("different scripts have same hash")
	}
}
