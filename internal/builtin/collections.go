package builtin

import (
	"fmt"
	"sort"

	"github.com/standardbeagle/slop/internal/evaluator"
)

func (r *Registry) registerListFunctions() {
	r.Register("append", builtinAppend)
	r.Register("extend", builtinExtend)
	r.Register("insert", builtinInsert)
	r.Register("remove", builtinRemove)
	r.Register("pop", builtinPop)
	r.Register("clear", builtinClear)
	r.Register("index", builtinIndex)
	r.Register("sorted", builtinSorted)
	r.Register("reversed", builtinReversed)
	r.Register("copy", builtinCopy)
	r.Register("first", builtinFirst)
	r.Register("last", builtinLast)
	r.Register("flatten", builtinFlatten)
}

func (r *Registry) registerMapFunctions() {
	r.Register("keys", builtinKeys)
	r.Register("values", builtinValues)
	r.Register("items", builtinItems)
	r.Register("get", builtinGet)
	r.Register("has_key", builtinHasKey)
	r.Register("merge", builtinMerge)
}

func (r *Registry) registerSetFunctions() {
	r.Register("add", builtinAdd)
	r.Register("discard", builtinDiscard)
	r.Register("union", builtinUnion)
	r.Register("intersection", builtinIntersection)
	r.Register("difference", builtinDifference)
	r.Register("symmetric_difference", builtinSymmetricDifference)
	r.Register("issubset", builtinIsSubset)
	r.Register("issuperset", builtinIsSuperset)
}

// List functions

func builtinAppend(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("append", args, 2); err != nil {
		return nil, err
	}
	list, err := requireList("append", args[0])
	if err != nil {
		return nil, err
	}
	// Mutate the list
	lv := args[0].(*evaluator.ListValue)
	lv.Elements = append(list, args[1])
	return lv, nil
}

func builtinExtend(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("extend", args, 2); err != nil {
		return nil, err
	}
	list, err := requireList("extend", args[0])
	if err != nil {
		return nil, err
	}
	other, err := requireList("extend", args[1])
	if err != nil {
		return nil, err
	}
	// Mutate the list
	lv := args[0].(*evaluator.ListValue)
	lv.Elements = append(list, other...)
	return lv, nil
}

func builtinInsert(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("insert", args, 3); err != nil {
		return nil, err
	}
	list, err := requireList("insert", args[0])
	if err != nil {
		return nil, err
	}
	idx, err := requireInt("insert", args[1])
	if err != nil {
		return nil, err
	}
	if idx < 0 {
		idx = int64(len(list)) + idx
	}
	if idx < 0 {
		idx = 0
	}
	if idx > int64(len(list)) {
		idx = int64(len(list))
	}

	// Mutate the list
	lv := args[0].(*evaluator.ListValue)
	newItems := make([]evaluator.Value, len(list)+1)
	copy(newItems[:idx], list[:idx])
	newItems[idx] = args[2]
	copy(newItems[idx+1:], list[idx:])
	lv.Elements = newItems
	return lv, nil
}

func builtinRemove(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("remove", args, 2); err != nil {
		return nil, err
	}
	list, err := requireList("remove", args[0])
	if err != nil {
		return nil, err
	}

	// Find and remove first occurrence
	for i, item := range list {
		if evaluator.Equal(item, args[1]) {
			lv := args[0].(*evaluator.ListValue)
			lv.Elements = append(list[:i], list[i+1:]...)
			return lv, nil
		}
	}
	return nil, fmt.Errorf("remove() element not in list")
}

func builtinPop(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("pop", args, 1, 2); err != nil {
		return nil, err
	}
	list, err := requireList("pop", args[0])
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("pop() from empty list")
	}

	idx := int64(len(list) - 1)
	if len(args) == 2 {
		idx, err = requireInt("pop", args[1])
		if err != nil {
			return nil, err
		}
		if idx < 0 {
			idx = int64(len(list)) + idx
		}
		if idx < 0 || idx >= int64(len(list)) {
			return nil, fmt.Errorf("pop() index out of range")
		}
	}

	item := list[idx]
	lv := args[0].(*evaluator.ListValue)
	lv.Elements = append(list[:idx], list[idx+1:]...)
	return item, nil
}

func builtinClear(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("clear", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.ListValue:
		v.Elements = []evaluator.Value{}
		return v, nil
	case *evaluator.MapValue:
		v.Pairs = make(map[string]evaluator.Value)
		return v, nil
	case *evaluator.SetValue:
		v.Elements = make(map[string]evaluator.Value)
		return v, nil
	default:
		return nil, fmt.Errorf("clear() argument must be list, map, or set, got %s", v.Type())
	}
}

func builtinIndex(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("index", args, 2); err != nil {
		return nil, err
	}
	list, err := requireList("index", args[0])
	if err != nil {
		return nil, err
	}

	for i, item := range list {
		if evaluator.Equal(item, args[1]) {
			return &evaluator.IntValue{Value: int64(i)}, nil
		}
	}
	return nil, fmt.Errorf("index() element not in list")
}

func builtinSorted(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("sorted", args, 1, 2); err != nil {
		return nil, err
	}
	list, err := requireList("sorted", args[0])
	if err != nil {
		return nil, err
	}

	// Make a copy
	items := make([]evaluator.Value, len(list))
	copy(items, list)

	// Check for reverse kwarg
	reverse := false
	if rv, ok := kwargs["reverse"]; ok {
		if bv, ok := rv.(*evaluator.BoolValue); ok {
			reverse = bv.Value
		}
	}

	// Get key function if provided
	var keyFn evaluator.Value
	if len(args) == 2 {
		keyFn = args[1]
	}
	if kv, ok := kwargs["key"]; ok {
		keyFn = kv
	}

	// Sort with comparison
	var sortErr error
	sort.SliceStable(items, func(i, j int) bool {
		if sortErr != nil {
			return false
		}
		a, b := items[i], items[j]

		// Apply key function if provided
		if keyFn != nil {
			// This is simplified - in real implementation we'd need to call the function
			// through the evaluator
		}

		cmp, err := evaluator.Compare(a, b)
		if err != nil {
			sortErr = err
			return false
		}
		if reverse {
			return cmp > 0
		}
		return cmp < 0
	})

	if sortErr != nil {
		return nil, sortErr
	}

	return &evaluator.ListValue{Elements: items}, nil
}

func builtinReversed(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("reversed", args, 1); err != nil {
		return nil, err
	}
	list, err := requireList("reversed", args[0])
	if err != nil {
		return nil, err
	}

	items := make([]evaluator.Value, len(list))
	for i, j := 0, len(list)-1; j >= 0; i, j = i+1, j-1 {
		items[i] = list[j]
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinCopy(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("copy", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.ListValue:
		items := make([]evaluator.Value, len(v.Elements))
		copy(items, v.Elements)
		return &evaluator.ListValue{Elements: items}, nil
	case *evaluator.MapValue:
		pairs := make(map[string]evaluator.Value, len(v.Pairs))
		for k, val := range v.Pairs {
			pairs[k] = val
		}
		return &evaluator.MapValue{Pairs: pairs}, nil
	case *evaluator.SetValue:
		items := make(map[string]evaluator.Value, len(v.Elements))
		for k, val := range v.Elements {
			items[k] = val
		}
		return &evaluator.SetValue{Elements: items}, nil
	default:
		return nil, fmt.Errorf("copy() argument must be list, map, or set, got %s", v.Type())
	}
}

func builtinFirst(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("first", args, 1); err != nil {
		return nil, err
	}
	list, err := requireList("first", args[0])
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return evaluator.NONE, nil
	}
	return list[0], nil
}

func builtinLast(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("last", args, 1); err != nil {
		return nil, err
	}
	list, err := requireList("last", args[0])
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return evaluator.NONE, nil
	}
	return list[len(list)-1], nil
}

func builtinFlatten(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("flatten", args, 1, 2); err != nil {
		return nil, err
	}
	list, err := requireList("flatten", args[0])
	if err != nil {
		return nil, err
	}

	depth := int64(1)
	if len(args) == 2 {
		depth, err = requireInt("flatten", args[1])
		if err != nil {
			return nil, err
		}
	}

	result := flattenList(list, depth)
	return &evaluator.ListValue{Elements: result}, nil
}

func flattenList(items []evaluator.Value, depth int64) []evaluator.Value {
	if depth == 0 {
		return items
	}

	result := make([]evaluator.Value, 0)
	for _, item := range items {
		if lv, ok := item.(*evaluator.ListValue); ok {
			result = append(result, flattenList(lv.Elements, depth-1)...)
		} else {
			result = append(result, item)
		}
	}
	return result
}

// Map functions

func builtinKeys(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("keys", args, 1); err != nil {
		return nil, err
	}
	m, err := requireMap("keys", args[0])
	if err != nil {
		return nil, err
	}

	items := make([]evaluator.Value, 0, len(m))
	for k := range m {
		items = append(items, &evaluator.StringValue{Value: k})
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinValues(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("values", args, 1); err != nil {
		return nil, err
	}
	m, err := requireMap("values", args[0])
	if err != nil {
		return nil, err
	}

	items := make([]evaluator.Value, 0, len(m))
	for _, v := range m {
		items = append(items, v)
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinItems(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("items", args, 1); err != nil {
		return nil, err
	}
	m, err := requireMap("items", args[0])
	if err != nil {
		return nil, err
	}

	items := make([]evaluator.Value, 0, len(m))
	for k, v := range m {
		pair := &evaluator.ListValue{
			Elements: []evaluator.Value{
				&evaluator.StringValue{Value: k},
				v,
			},
		}
		items = append(items, pair)
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinGet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("get", args, 2, 3); err != nil {
		return nil, err
	}
	m, err := requireMap("get", args[0])
	if err != nil {
		return nil, err
	}

	key := args[1].String()
	if v, ok := m[key]; ok {
		return v, nil
	}

	if len(args) == 3 {
		return args[2], nil
	}
	return evaluator.NONE, nil
}

func builtinHasKey(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("has_key", args, 2); err != nil {
		return nil, err
	}
	m, err := requireMap("has_key", args[0])
	if err != nil {
		return nil, err
	}

	key := args[1].String()
	if _, ok := m[key]; ok {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinMerge(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("merge", args, 2); err != nil {
		return nil, err
	}

	result := make(map[string]evaluator.Value)

	for _, arg := range args {
		m, err := requireMap("merge", arg)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			result[k] = v
		}
	}

	return &evaluator.MapValue{Pairs: result}, nil
}

// Set functions

func builtinAdd(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("add", args, 2); err != nil {
		return nil, err
	}
	s, err := requireSet("add", args[0])
	if err != nil {
		return nil, err
	}

	key := args[1].String()
	s[key] = args[1]
	return args[0], nil
}

func builtinDiscard(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("discard", args, 2); err != nil {
		return nil, err
	}
	s, err := requireSet("discard", args[0])
	if err != nil {
		return nil, err
	}

	key := args[1].String()
	delete(s, key)
	return args[0], nil
}

func builtinUnion(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("union", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("union", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("union", args[1])
	if err != nil {
		return nil, err
	}

	result := make(map[string]evaluator.Value)
	for k, v := range s1 {
		result[k] = v
	}
	for k, v := range s2 {
		result[k] = v
	}
	return &evaluator.SetValue{Elements: result}, nil
}

func builtinIntersection(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("intersection", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("intersection", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("intersection", args[1])
	if err != nil {
		return nil, err
	}

	result := make(map[string]evaluator.Value)
	for k, v := range s1 {
		if _, ok := s2[k]; ok {
			result[k] = v
		}
	}
	return &evaluator.SetValue{Elements: result}, nil
}

func builtinDifference(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("difference", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("difference", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("difference", args[1])
	if err != nil {
		return nil, err
	}

	result := make(map[string]evaluator.Value)
	for k, v := range s1 {
		if _, ok := s2[k]; !ok {
			result[k] = v
		}
	}
	return &evaluator.SetValue{Elements: result}, nil
}

func builtinSymmetricDifference(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("symmetric_difference", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("symmetric_difference", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("symmetric_difference", args[1])
	if err != nil {
		return nil, err
	}

	result := make(map[string]evaluator.Value)
	for k, v := range s1 {
		if _, ok := s2[k]; !ok {
			result[k] = v
		}
	}
	for k, v := range s2 {
		if _, ok := s1[k]; !ok {
			result[k] = v
		}
	}
	return &evaluator.SetValue{Elements: result}, nil
}

func builtinIsSubset(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("issubset", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("issubset", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("issubset", args[1])
	if err != nil {
		return nil, err
	}

	for k := range s1 {
		if _, ok := s2[k]; !ok {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}

func builtinIsSuperset(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("issuperset", args, 2); err != nil {
		return nil, err
	}
	s1, err := requireSet("issuperset", args[0])
	if err != nil {
		return nil, err
	}
	s2, err := requireSet("issuperset", args[1])
	if err != nil {
		return nil, err
	}

	for k := range s2 {
		if _, ok := s1[k]; !ok {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}
