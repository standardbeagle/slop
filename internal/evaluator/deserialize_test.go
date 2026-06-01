package evaluator

import (
	"encoding/json"
	"testing"
)

func TestDeserializeValue_Primitives(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	tests := []struct {
		name     string
		sv       *SerializedValue
		wantType string
		check    func(Value) bool
	}{
		{
			"none",
			&SerializedValue{Type: "none", Data: json.RawMessage("null")},
			"none",
			func(v Value) bool { return IsNone(v) },
		},
		{
			"bool_true",
			&SerializedValue{Type: "bool", Data: json.RawMessage("true")},
			"bool",
			func(v Value) bool {
				b, ok := v.(*BoolValue)
				return ok && b.Value == true
			},
		},
		{
			"bool_false",
			&SerializedValue{Type: "bool", Data: json.RawMessage("false")},
			"bool",
			func(v Value) bool {
				b, ok := v.(*BoolValue)
				return ok && b.Value == false
			},
		},
		{
			"int",
			&SerializedValue{Type: "int", Data: json.RawMessage(`"42"`)},
			"int",
			func(v Value) bool {
				i, ok := v.(*IntValue)
				return ok && i.Value == 42
			},
		},
		{
			"int_large",
			&SerializedValue{Type: "int", Data: json.RawMessage(`"9223372036854775807"`)},
			"int",
			func(v Value) bool {
				i, ok := v.(*IntValue)
				return ok && i.Value == 9223372036854775807
			},
		},
		{
			"float",
			&SerializedValue{Type: "float", Data: json.RawMessage("3.14159")},
			"float",
			func(v Value) bool {
				f, ok := v.(*FloatValue)
				return ok && f.Value == 3.14159
			},
		},
		{
			"string",
			&SerializedValue{Type: "string", Data: json.RawMessage(`"hello"`)},
			"string",
			func(v Value) bool {
				s, ok := v.(*StringValue)
				return ok && s.Value == "hello"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := d.DeserializeValue(tt.sv)
			if err != nil {
				t.Fatalf("DeserializeValue() error = %v", err)
			}
			if v.Type() != tt.wantType {
				t.Errorf("DeserializeValue() type = %v, want %v", v.Type(), tt.wantType)
			}
			if !tt.check(v) {
				t.Errorf("DeserializeValue() value check failed")
			}
		})
	}
}

func TestDeserializeValue_List(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	// Create serialized list
	elementsData, _ := json.Marshal([]*SerializedValue{
		{Type: "int", Data: json.RawMessage(`"1"`)},
		{Type: "string", Data: json.RawMessage(`"two"`)},
	})

	sv := &SerializedValue{Type: "list", Data: elementsData}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	list, ok := v.(*ListValue)
	if !ok {
		t.Fatal("DeserializeValue() did not return ListValue")
	}

	if len(list.Elements) != 2 {
		t.Errorf("list has %d elements, want 2", len(list.Elements))
	}

	// Check first element
	if i, ok := list.Elements[0].(*IntValue); !ok || i.Value != 1 {
		t.Error("first element is not int 1")
	}

	// Check second element
	if s, ok := list.Elements[1].(*StringValue); !ok || s.Value != "two" {
		t.Error("second element is not string 'two'")
	}
}

func TestDeserializeValue_Map(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	// Create serialized map
	sm := &SerializedMap{
		Pairs: map[string]*SerializedValue{
			"name":  {Type: "string", Data: json.RawMessage(`"test"`)},
			"count": {Type: "int", Data: json.RawMessage(`"42"`)},
		},
		Order: []string{"name", "count"},
	}
	mapData, _ := json.Marshal(sm)

	sv := &SerializedValue{Type: "map", Data: mapData}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	m, ok := v.(*MapValue)
	if !ok {
		t.Fatal("DeserializeValue() did not return MapValue")
	}

	if len(m.Pairs) != 2 {
		t.Errorf("map has %d pairs, want 2", len(m.Pairs))
	}

	// Check order is preserved
	if m.Order[0] != "name" || m.Order[1] != "count" {
		t.Errorf("map order = %v, want [name, count]", m.Order)
	}

	// Check values
	nameVal, ok := m.Get("name")
	if !ok {
		t.Error("map missing 'name' key")
	}
	if s, ok := nameVal.(*StringValue); !ok || s.Value != "test" {
		t.Error("name value is not 'test'")
	}
}

func TestDeserializeValue_Set(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	// Create serialized set
	elementsData, _ := json.Marshal([]*SerializedValue{
		{Type: "string", Data: json.RawMessage(`"a"`)},
		{Type: "string", Data: json.RawMessage(`"b"`)},
	})

	sv := &SerializedValue{Type: "set", Data: elementsData}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	set, ok := v.(*SetValue)
	if !ok {
		t.Fatal("DeserializeValue() did not return SetValue")
	}

	if len(set.Elements) != 2 {
		t.Errorf("set has %d elements, want 2", len(set.Elements))
	}

	if !set.Has(&StringValue{Value: "a"}) {
		t.Error("set missing 'a'")
	}
	if !set.Has(&StringValue{Value: "b"}) {
		t.Error("set missing 'b'")
	}
}

func TestDeserializeValue_Error(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	// Create serialized error
	se := &SerializedError{
		Message: "test error",
		Data:    &SerializedValue{Type: "int", Data: json.RawMessage(`"42"`)},
	}
	errorData, _ := json.Marshal(se)

	sv := &SerializedValue{Type: "error", Data: errorData}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	slopErr, ok := v.(*SlopError)
	if !ok {
		t.Fatal("DeserializeValue() did not return SlopError")
	}

	if slopErr.Message != "test error" {
		t.Errorf("error message = %v, want 'test error'", slopErr.Message)
	}
	if slopErr.Data == nil {
		t.Error("error data is nil")
	}
}

func TestDeserializeValue_Iterator(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	// Range iterator
	si := &SerializedIterator{
		IterType: "range",
		Current:  5,
		End:      10,
		Step:     1,
	}
	iterData, _ := json.Marshal(si)

	sv := &SerializedValue{Type: "iterator", Data: iterData}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	iter, ok := v.(*IteratorValue)
	if !ok {
		t.Fatal("DeserializeValue() did not return IteratorValue")
	}

	if iter.Current != 5 {
		t.Errorf("iterator Current = %d, want 5", iter.Current)
	}
	if iter.End != 10 {
		t.Errorf("iterator End = %d, want 10", iter.End)
	}
}

func TestDeserializeValue_Builtin(t *testing.T) {
	builtins := map[string]*BuiltinValue{
		"len": {Name: "len", Fn: func(args []Value, kwargs map[string]Value) (Value, error) {
			return &IntValue{Value: 0}, nil
		}},
	}

	d := NewDeserializer(nil, builtins, nil)

	sv := &SerializedValue{Type: "builtin", Data: json.RawMessage(`"len"`)}

	v, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	builtin, ok := v.(*BuiltinValue)
	if !ok {
		t.Fatal("DeserializeValue() did not return BuiltinValue")
	}

	if builtin.Name != "len" {
		t.Errorf("builtin name = %v, want 'len'", builtin.Name)
	}
	if builtin.Fn == nil {
		t.Error("builtin function is nil")
	}
}

func TestDeserializeLimits(t *testing.T) {
	sl := &SerializedLimits{
		MaxIterations:  1000,
		MaxLLMCalls:    50,
		IterationCount: 500,
		TotalCost:      5.0,
	}

	limits, err := DeserializeLimits(sl)
	if err != nil {
		t.Fatalf("DeserializeLimits() unexpected error: %v", err)
	}

	if limits.MaxIterations != 1000 {
		t.Errorf("MaxIterations = %d, want 1000", limits.MaxIterations)
	}
	if limits.IterationCount != 500 {
		t.Errorf("IterationCount = %d, want 500", limits.IterationCount)
	}
}

// TestDeserializeLimits_InjectionRejected verifies that crafted checkpoint
// state that would bypass budget limits is rejected.
func TestDeserializeLimits_InjectionRejected(t *testing.T) {
	cases := []struct {
		name string
		sl   *SerializedLimits
	}{
		{
			"negative iteration_count",
			&SerializedLimits{IterationCount: -1},
		},
		{
			"negative llm_call_count",
			&SerializedLimits{LLMCallCount: -1},
		},
		{
			"negative api_call_count",
			&SerializedLimits{APICallCount: -1},
		},
		{
			"negative call_depth",
			&SerializedLimits{CallDepth: -1},
		},
		{
			"negative total_cost",
			&SerializedLimits{TotalCost: -1.0},
		},
		{
			"total_cost exceeds max_cost",
			&SerializedLimits{MaxCost: 10.0, TotalCost: 20.0},
		},
		{
			"iteration_count exceeds max_iterations",
			&SerializedLimits{MaxIterations: 100, IterationCount: 200},
		},
		{
			"llm_call_count exceeds max_llm_calls",
			&SerializedLimits{MaxLLMCalls: 10, LLMCallCount: 50},
		},
		{
			"api_call_count exceeds max_api_calls",
			&SerializedLimits{MaxAPICalls: 5, APICallCount: 99},
		},
		{
			"call_depth exceeds max_call_depth",
			&SerializedLimits{MaxCallDepth: 100, CallDepth: 101},
		},
		{
			"call_depth exceeds default max when max_call_depth unset",
			&SerializedLimits{CallDepth: DefaultMaxCallDepth + 1},
		},
		{
			"total_cost exceeds hard limit",
			&SerializedLimits{TotalCost: maxCheckpointCost + 1.0},
		},
		{
			"negative max_cost",
			&SerializedLimits{MaxCost: -1.0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DeserializeLimits(tc.sl)
			if err == nil {
				t.Errorf("DeserializeLimits() expected error for %q, got nil", tc.name)
			}
		})
	}
}

// TestDeserializeIterator_InjectionRejected verifies that crafted iterator
// state that would cause unbounded loops or negative indices is rejected.
func TestDeserializeIterator_InjectionRejected(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	cases := []struct {
		name string
		si   *SerializedIterator
	}{
		{
			"zero step range",
			&SerializedIterator{IterType: "range", Current: 0, End: 10, Step: 0},
		},
		{
			"positive step but current > end",
			&SerializedIterator{IterType: "range", Current: 100, End: 10, Step: 1},
		},
		{
			"negative step but current < end",
			&SerializedIterator{IterType: "range", Current: 0, End: 100, Step: -1},
		},
		{
			"iteration count exceeds limit",
			&SerializedIterator{IterType: "range", Current: 0, End: maxCheckpointIteratorItems + 1, Step: 1},
		},
		{
			"negative current list iterator",
			&SerializedIterator{IterType: "list", Current: -1},
		},
		{
			"current exceeds item count",
			&SerializedIterator{IterType: "list", Current: 5, Items: []*SerializedValue{}},
		},
		{
			"unknown iterator type",
			&SerializedIterator{IterType: "bogus", Current: 0},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, _ := json.Marshal(tc.si)
			_, err := d.deserializeIterator(data)
			if err == nil {
				t.Errorf("deserializeIterator() expected error for %q, got nil", tc.name)
			}
		})
	}
}

// TestLoadCheckpoint_MissingRequiredFields verifies that checkpoints missing
// mandatory schema fields are rejected before any state is restored.
func TestLoadCheckpoint_MissingRequiredFields(t *testing.T) {
	// Start from a valid minimal checkpoint JSON and remove required fields.
	validJSON := `{
		"version": "1.0",
		"script_hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"position": {"line": 1, "column": 1},
		"context": {
			"scopes": [{"id": "s1", "variables": {}, "is_global": true}],
			"current_scope_id": "s1",
			"limits": {},
			"emitted": [],
			"control_flow": {}
		},
		"created_at": "2026-01-01T00:00:00Z"
	}`

	requiredFields := []string{"version", "script_hash", "position", "context", "created_at"}
	for _, field := range requiredFields {
		t.Run("missing_"+field, func(t *testing.T) {
			var obj map[string]json.RawMessage
			_ = json.Unmarshal([]byte(validJSON), &obj)
			delete(obj, field)
			data, _ := json.Marshal(obj)
			_, _, err := LoadCheckpoint(data, nil, nil, nil)
			if err == nil {
				t.Errorf("LoadCheckpoint() expected error when %q is absent, got nil", field)
			}
		})
	}
}

// TestLoadCheckpoint_InvalidScriptHash verifies that malformed script_hash
// values are rejected at load time.
func TestLoadCheckpoint_InvalidScriptHash(t *testing.T) {
	base := map[string]interface{}{
		"version":     "1.0",
		"position":    map[string]int{"line": 1, "column": 1},
		"created_at":  "2026-01-01T00:00:00Z",
		"context": map[string]interface{}{
			"scopes":           []interface{}{map[string]interface{}{"id": "s1", "variables": map[string]interface{}{}, "is_global": true}},
			"current_scope_id": "s1",
			"limits":           map[string]interface{}{},
			"emitted":          []interface{}{},
			"control_flow":     map[string]interface{}{},
		},
	}

	cases := []struct {
		name string
		hash string
	}{
		{"too short", "aabbcc"},
		{"wrong chars", "GGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG"},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obj := make(map[string]interface{})
			for k, v := range base {
				obj[k] = v
			}
			obj["script_hash"] = tc.hash
			data, _ := json.Marshal(obj)
			_, _, err := LoadCheckpoint(data, nil, nil, nil)
			if err == nil {
				t.Errorf("LoadCheckpoint() expected error for hash %q, got nil", tc.hash)
			}
		})
	}
}

// TestLoadCheckpoint_InjectedLimitsRejected verifies end-to-end that a
// checkpoint with out-of-bounds limits is rejected at load time.
func TestLoadCheckpoint_InjectedLimitsRejected(t *testing.T) {
	// Craft a checkpoint where iteration_count exceeds max_iterations.
	payload := map[string]interface{}{
		"version":     "1.0",
		"script_hash": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"position":    map[string]int{"line": 1, "column": 1},
		"created_at":  "2026-01-01T00:00:00Z",
		"context": map[string]interface{}{
			"scopes":           []interface{}{map[string]interface{}{"id": "s1", "variables": map[string]interface{}{}, "is_global": true}},
			"current_scope_id": "s1",
			"limits": map[string]interface{}{
				"max_iterations":  10,
				"iteration_count": 9999, // injection: far exceeds the max
			},
			"emitted":      []interface{}{},
			"control_flow": map[string]interface{}{},
		},
	}
	data, _ := json.Marshal(payload)
	_, _, err := LoadCheckpoint(data, nil, nil, nil)
	if err == nil {
		t.Error("LoadCheckpoint() expected error for injected iteration_count, got nil")
	}
}

func TestDeserializeScopeChain(t *testing.T) {
	d := NewDeserializer(nil, nil, nil)

	parentID := "scope_1"
	scopes := []*SerializedScope{
		{
			ID:        "scope_1",
			ParentID:  nil,
			Variables: map[string]*SerializedValue{},
			IsGlobal:  true,
		},
		{
			ID:       "scope_2",
			ParentID: &parentID,
			Variables: map[string]*SerializedValue{
				"x": {Type: "int", Data: json.RawMessage(`"42"`)},
			},
			IsGlobal: false,
		},
	}

	current, globals, err := d.DeserializeScopeChain(scopes, "scope_2")
	if err != nil {
		t.Fatalf("DeserializeScopeChain() error = %v", err)
	}

	if globals == nil {
		t.Error("globals is nil")
	}

	if current == nil {
		t.Error("current is nil")
	}

	// Check variable in current scope
	val, ok := current.Get("x")
	if !ok {
		t.Error("variable 'x' not found in current scope")
	}
	if i, ok := val.(*IntValue); !ok || i.Value != 42 {
		t.Error("variable 'x' is not int 42")
	}

	// Check parent relationship
	if current.parent != globals {
		t.Error("current scope parent is not globals")
	}
}

// Round-trip tests: serialize → deserialize → compare

func TestRoundTrip_Primitives(t *testing.T) {
	s := NewSerializer(nil)
	d := NewDeserializer(nil, nil, nil)

	tests := []struct {
		name  string
		value Value
	}{
		{"none", NONE},
		{"true", TRUE},
		{"false", FALSE},
		{"int", &IntValue{Value: 42}},
		{"int_large", &IntValue{Value: 9223372036854775807}},
		{"int_negative", &IntValue{Value: -9223372036854775808}},
		{"float", &FloatValue{Value: 3.14159}},
		{"string", &StringValue{Value: "hello world"}},
		{"string_unicode", &StringValue{Value: "こんにちは 🎉"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			sv, err := s.SerializeValue(tt.value)
			if err != nil {
				t.Fatalf("SerializeValue() error = %v", err)
			}

			// Deserialize
			result, err := d.DeserializeValue(sv)
			if err != nil {
				t.Fatalf("DeserializeValue() error = %v", err)
			}

			// Compare
			if !Equal(tt.value, result) {
				t.Errorf("round-trip failed: %v != %v", tt.value, result)
			}
		})
	}
}

func TestRoundTrip_List(t *testing.T) {
	s := NewSerializer(nil)
	d := NewDeserializer(nil, nil, nil)

	original := &ListValue{
		Elements: []Value{
			&IntValue{Value: 1},
			&StringValue{Value: "two"},
			&BoolValue{Value: true},
			NONE,
		},
	}

	// Serialize
	sv, err := s.SerializeValue(original)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}

	// Deserialize
	result, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	list, ok := result.(*ListValue)
	if !ok {
		t.Fatal("result is not ListValue")
	}

	if len(list.Elements) != len(original.Elements) {
		t.Errorf("list length = %d, want %d", len(list.Elements), len(original.Elements))
	}

	for i := range original.Elements {
		if !Equal(original.Elements[i], list.Elements[i]) {
			t.Errorf("element %d: %v != %v", i, original.Elements[i], list.Elements[i])
		}
	}
}

func TestRoundTrip_Map(t *testing.T) {
	s := NewSerializer(nil)
	d := NewDeserializer(nil, nil, nil)

	original := NewMapValue()
	original.Set("name", &StringValue{Value: "test"})
	original.Set("count", &IntValue{Value: 42})
	original.Set("active", TRUE)

	// Serialize
	sv, err := s.SerializeValue(original)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}

	// Deserialize
	result, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	m, ok := result.(*MapValue)
	if !ok {
		t.Fatal("result is not MapValue")
	}

	// Check values
	for _, key := range original.Order {
		origVal, _ := original.Get(key)
		resultVal, ok := m.Get(key)
		if !ok {
			t.Errorf("missing key: %s", key)
			continue
		}
		if !Equal(origVal, resultVal) {
			t.Errorf("key %s: %v != %v", key, origVal, resultVal)
		}
	}

	// Check order preserved
	for i, key := range original.Order {
		if m.Order[i] != key {
			t.Errorf("order[%d] = %s, want %s", i, m.Order[i], key)
		}
	}
}

func TestRoundTrip_NestedStructures(t *testing.T) {
	s := NewSerializer(nil)
	d := NewDeserializer(nil, nil, nil)

	// Create: {"items": [1, {"nested": true}], "meta": {"count": 2}}
	innerMap := NewMapValue()
	innerMap.Set("nested", TRUE)

	innerList := &ListValue{
		Elements: []Value{
			&IntValue{Value: 1},
			innerMap,
		},
	}

	metaMap := NewMapValue()
	metaMap.Set("count", &IntValue{Value: 2})

	original := NewMapValue()
	original.Set("items", innerList)
	original.Set("meta", metaMap)

	// Serialize
	sv, err := s.SerializeValue(original)
	if err != nil {
		t.Fatalf("SerializeValue() error = %v", err)
	}

	// Deserialize
	result, err := d.DeserializeValue(sv)
	if err != nil {
		t.Fatalf("DeserializeValue() error = %v", err)
	}

	m, ok := result.(*MapValue)
	if !ok {
		t.Fatal("result is not MapValue")
	}

	// Verify nested list
	items, ok := m.Get("items")
	if !ok {
		t.Fatal("missing 'items' key")
	}
	itemsList, ok := items.(*ListValue)
	if !ok {
		t.Fatal("items is not ListValue")
	}
	if len(itemsList.Elements) != 2 {
		t.Errorf("items has %d elements, want 2", len(itemsList.Elements))
	}

	// Verify nested map in list
	nestedMap, ok := itemsList.Elements[1].(*MapValue)
	if !ok {
		t.Fatal("items[1] is not MapValue")
	}
	nestedVal, ok := nestedMap.Get("nested")
	if !ok {
		t.Fatal("missing 'nested' key")
	}
	if b, ok := nestedVal.(*BoolValue); !ok || !b.Value {
		t.Error("nested value is not true")
	}
}

func TestRoundTrip_Context(t *testing.T) {
	s := NewSerializer(nil)

	// Create a context with state
	ctx := NewContext()
	ctx.Scope.Set("x", &IntValue{Value: 42})
	ctx.Scope.Set("name", &StringValue{Value: "test"})
	ctx.Emit(&StringValue{Value: "result1"})
	ctx.Emit(&IntValue{Value: 100})
	ctx.Limits.IterationCount = 50

	// Serialize
	sc, err := s.SerializeContext(ctx)
	if err != nil {
		t.Fatalf("SerializeContext() error = %v", err)
	}

	// Deserialize
	d := NewDeserializer(nil, nil, nil)
	resultCtx, err := d.DeserializeContext(sc)
	if err != nil {
		t.Fatalf("DeserializeContext() error = %v", err)
	}

	// Verify variables
	xVal, ok := resultCtx.Scope.Get("x")
	if !ok {
		t.Error("variable 'x' not found")
	}
	if i, ok := xVal.(*IntValue); !ok || i.Value != 42 {
		t.Errorf("x = %v, want 42", xVal)
	}

	nameVal, ok := resultCtx.Scope.Get("name")
	if !ok {
		t.Error("variable 'name' not found")
	}
	if s, ok := nameVal.(*StringValue); !ok || s.Value != "test" {
		t.Errorf("name = %v, want 'test'", nameVal)
	}

	// Verify emitted values
	if len(resultCtx.Emitted) != 2 {
		t.Errorf("emitted has %d values, want 2", len(resultCtx.Emitted))
	}

	// Verify limits
	if resultCtx.Limits.IterationCount != 50 {
		t.Errorf("IterationCount = %d, want 50", resultCtx.Limits.IterationCount)
	}
}

func TestRoundTrip_Checkpoint(t *testing.T) {
	s := NewSerializer(nil)
	ctx := NewContext()
	ctx.Scope.Set("x", &IntValue{Value: 42})

	script := "x = 42\npause \"test\""
	pos := Position{Line: 2, Column: 1, StatementIndex: 1}

	// Create checkpoint
	checkpoint, err := s.CreateCheckpoint(ctx, script, pos, "test message")
	if err != nil {
		t.Fatalf("CreateCheckpoint() error = %v", err)
	}

	// Serialize to JSON
	data, err := SaveCheckpoint(checkpoint)
	if err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}

	// Load from JSON
	loadedCheckpoint, loadedCtx, err := LoadCheckpoint(data, nil, nil, nil)
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}

	// Verify checkpoint metadata
	if loadedCheckpoint.Version != CheckpointVersion {
		t.Errorf("Version = %v, want %v", loadedCheckpoint.Version, CheckpointVersion)
	}
	if loadedCheckpoint.ScriptHash != checkpoint.ScriptHash {
		t.Errorf("ScriptHash mismatch")
	}
	if loadedCheckpoint.Position.Line != 2 {
		t.Errorf("Position.Line = %d, want 2", loadedCheckpoint.Position.Line)
	}
	if loadedCheckpoint.CheckpointMessage != "test message" {
		t.Errorf("CheckpointMessage = %v, want 'test message'", loadedCheckpoint.CheckpointMessage)
	}

	// Verify context
	xVal, ok := loadedCtx.Scope.Get("x")
	if !ok {
		t.Error("variable 'x' not found in loaded context")
	}
	if i, ok := xVal.(*IntValue); !ok || i.Value != 42 {
		t.Errorf("x = %v, want 42", xVal)
	}
}
