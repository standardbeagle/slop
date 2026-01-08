package builtin

import (
	"fmt"

	"github.com/standardbeagle/slop/internal/evaluator"
)

// PipelineFuncCaller is an interface for calling functions during pipeline operations.
// This is needed because pipeline functions need to invoke user-defined functions.
type PipelineFuncCaller interface {
	CallFunction(fn evaluator.Value, args []evaluator.Value) (evaluator.Value, error)
}

// SetFuncCaller sets the function caller for pipeline operations.
// This should be called by the evaluator before using pipeline functions.
var pipelineFuncCaller PipelineFuncCaller

func SetPipelineFuncCaller(caller PipelineFuncCaller) {
	pipelineFuncCaller = caller
}

func (r *Registry) registerPipelineFunctions() {
	// Transformation
	r.Register("map", builtinMap)
	r.Register("flat_map", builtinFlatMap)

	// Filtering
	r.Register("filter", builtinFilter)
	r.Register("reject", builtinReject)
	r.Register("compact", builtinCompact)
	r.Register("unique", builtinUnique)
	r.Register("dedup", builtinDedup)

	// Selection
	r.Register("take", builtinTake)
	r.Register("drop", builtinDrop)
	r.Register("take_while", builtinTakeWhile)
	r.Register("drop_while", builtinDropWhile)
	r.Register("nth", builtinNth)

	// Grouping
	r.Register("group", builtinGroup)
	r.Register("group_by", builtinGroupBy)
	r.Register("partition", builtinPartition)
	r.Register("chunk", builtinChunk)
	r.Register("window", builtinWindow)

	// Aggregation
	r.Register("reduce", builtinReduce)
	r.Register("avg", builtinAvg)
	r.Register("any", builtinAny)
	r.Register("all", builtinAll)
	r.Register("none", builtinNoneFunc)
	r.Register("find", builtinFindValue)
	r.Register("find_index", builtinFindIndex)

	// Combination
	r.Register("concat", builtinConcat)
	r.Register("zip_with", builtinZipWith)
	r.Register("interleave", builtinInterleave)
}

func callFunction(fn evaluator.Value, args []evaluator.Value) (evaluator.Value, error) {
	if pipelineFuncCaller == nil {
		return nil, fmt.Errorf("pipeline function caller not set")
	}
	return pipelineFuncCaller.CallFunction(fn, args)
}

// Transformation

func builtinMap(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("map", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | map(fn) -> map(list, fn)
	list, err := requireList("map", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("map", fn); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, len(list))
	for i, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		result[i] = val
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinFlatMap(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("flat_map", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | flat_map(fn) -> flat_map(list, fn)
	list, err := requireList("flat_map", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("flat_map", fn); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if lv, ok := val.(*evaluator.ListValue); ok {
			result = append(result, lv.Elements...)
		} else {
			result = append(result, val)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

// Filtering

func builtinFilter(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("filter", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | filter(fn) -> filter(list, fn)
	list, err := requireList("filter", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("filter", fn); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if val.IsTruthy() {
			result = append(result, item)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinReject(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("reject", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | reject(fn) -> reject(list, fn)
	list, err := requireList("reject", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("reject", fn); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if !val.IsTruthy() {
			result = append(result, item)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinCompact(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("compact", args, 1); err != nil {
		return nil, err
	}

	list, err := requireList("compact", args[0])
	if err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, item := range list {
		if item.IsTruthy() {
			result = append(result, item)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinUnique(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("unique", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("unique", args[0])
	if err != nil {
		return nil, err
	}

	var keyFn evaluator.Value
	if len(args) == 2 {
		keyFn = args[1]
		if err := requireCallable("unique", keyFn); err != nil {
			return nil, err
		}
	}

	seen := make(map[string]bool)
	result := make([]evaluator.Value, 0)

	for _, item := range list {
		var key string
		if keyFn != nil {
			keyVal, err := callFunction(keyFn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
			key = keyVal.String()
		} else {
			key = item.String()
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinDedup(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	// dedup is an alias for unique with optional "by" kwarg
	if err := requireArgs("dedup", args, 1); err != nil {
		return nil, err
	}

	if byFn, ok := kwargs["by"]; ok {
		return builtinUnique([]evaluator.Value{args[0], byFn}, nil)
	}

	return builtinUnique(args, nil)
}

// Selection

func builtinTake(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("take", args, 2); err != nil {
		return nil, err
	}

	n, err := requireInt("take", args[0])
	if err != nil {
		return nil, err
	}

	list, err := requireList("take", args[1])
	if err != nil {
		return nil, err
	}

	if n < 0 {
		n = 0
	}
	if n > int64(len(list)) {
		n = int64(len(list))
	}

	return &evaluator.ListValue{Elements: list[:n]}, nil
}

func builtinDrop(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("drop", args, 2); err != nil {
		return nil, err
	}

	n, err := requireInt("drop", args[0])
	if err != nil {
		return nil, err
	}

	list, err := requireList("drop", args[1])
	if err != nil {
		return nil, err
	}

	if n < 0 {
		n = 0
	}
	if n > int64(len(list)) {
		n = int64(len(list))
	}

	return &evaluator.ListValue{Elements: list[n:]}, nil
}

func builtinTakeWhile(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("take_while", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | take_while(fn)
	list, err := requireList("take_while", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("take_while", fn); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if !val.IsTruthy() {
			break
		}
		result = append(result, item)
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinDropWhile(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("drop_while", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | drop_while(fn)
	list, err := requireList("drop_while", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("drop_while", fn); err != nil {
		return nil, err
	}

	dropping := true
	result := make([]evaluator.Value, 0)
	for _, item := range list {
		if dropping {
			val, err := callFunction(fn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
			if !val.IsTruthy() {
				dropping = false
				result = append(result, item)
			}
		} else {
			result = append(result, item)
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinNth(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("nth", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | nth(n)
	list, err := requireList("nth", args[0])
	if err != nil {
		return nil, err
	}

	n, err := requireInt("nth", args[1])
	if err != nil {
		return nil, err
	}

	if n < 0 {
		n = int64(len(list)) + n
	}
	if n < 0 || n >= int64(len(list)) {
		return evaluator.NONE, nil
	}

	return list[n], nil
}

// Grouping

func builtinGroup(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("group", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | group(fn)
	list, err := requireList("group", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("group", fn); err != nil {
		return nil, err
	}

	groups := make(map[string][]evaluator.Value)
	for _, item := range list {
		keyVal, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		key := keyVal.String()
		groups[key] = append(groups[key], item)
	}

	result := make(map[string]evaluator.Value)
	for k, v := range groups {
		result[k] = &evaluator.ListValue{Elements: v}
	}

	return &evaluator.MapValue{Pairs: result}, nil
}

func builtinGroupBy(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	// group_by is an alias for group
	return builtinGroup(args, nil)
}

func builtinPartition(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("partition", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | partition(fn)
	list, err := requireList("partition", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("partition", fn); err != nil {
		return nil, err
	}

	matches := make([]evaluator.Value, 0)
	nonMatches := make([]evaluator.Value, 0)

	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if val.IsTruthy() {
			matches = append(matches, item)
		} else {
			nonMatches = append(nonMatches, item)
		}
	}

	return &evaluator.ListValue{
		Elements: []evaluator.Value{
			&evaluator.ListValue{Elements: matches},
			&evaluator.ListValue{Elements: nonMatches},
		},
	}, nil
}

func builtinChunk(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("chunk", args, 2); err != nil {
		return nil, err
	}

	n, err := requireInt("chunk", args[0])
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, fmt.Errorf("chunk() size must be positive")
	}

	list, err := requireList("chunk", args[1])
	if err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for i := 0; i < len(list); i += int(n) {
		end := i + int(n)
		if end > len(list) {
			end = len(list)
		}
		result = append(result, &evaluator.ListValue{Elements: list[i:end]})
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinWindow(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("window", args, 2); err != nil {
		return nil, err
	}

	n, err := requireInt("window", args[0])
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, fmt.Errorf("window() size must be positive")
	}

	list, err := requireList("window", args[1])
	if err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for i := 0; i <= len(list)-int(n); i++ {
		window := make([]evaluator.Value, n)
		copy(window, list[i:i+int(n)])
		result = append(result, &evaluator.ListValue{Elements: window})
	}

	return &evaluator.ListValue{Elements: result}, nil
}

// Aggregation

func builtinReduce(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("reduce", args, 3); err != nil {
		return nil, err
	}

	// Pipeline style: list | reduce(fn, init) -> reduce(list, fn, init)
	list, err := requireList("reduce", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("reduce", fn); err != nil {
		return nil, err
	}

	init := args[2]

	acc := init
	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{acc, item})
		if err != nil {
			return nil, err
		}
		acc = val
	}

	return acc, nil
}

func builtinAvg(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("avg", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("avg", args[0])
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("avg() empty sequence")
	}

	var keyFn evaluator.Value
	if len(args) == 2 {
		keyFn = args[1]
		if err := requireCallable("avg", keyFn); err != nil {
			return nil, err
		}
	}

	var sum float64
	for _, item := range list {
		var val evaluator.Value = item
		if keyFn != nil {
			val, err = callFunction(keyFn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
		}
		f, ok := toFloat(val)
		if !ok {
			return nil, fmt.Errorf("avg() requires numeric elements, got %s", val.Type())
		}
		sum += f
	}

	return &evaluator.FloatValue{Value: sum / float64(len(list))}, nil
}

func builtinAny(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("any", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("any", args[0])
	if err != nil {
		return nil, err
	}

	var fn evaluator.Value
	if len(args) == 2 {
		fn = args[1]
		if err := requireCallable("any", fn); err != nil {
			return nil, err
		}
	}

	for _, item := range list {
		var result evaluator.Value = item
		if fn != nil {
			result, err = callFunction(fn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
		}
		if result.IsTruthy() {
			return evaluator.TRUE, nil
		}
	}

	return evaluator.FALSE, nil
}

func builtinAll(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("all", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("all", args[0])
	if err != nil {
		return nil, err
	}

	var fn evaluator.Value
	if len(args) == 2 {
		fn = args[1]
		if err := requireCallable("all", fn); err != nil {
			return nil, err
		}
	}

	for _, item := range list {
		var result evaluator.Value = item
		if fn != nil {
			result, err = callFunction(fn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
		}
		if !result.IsTruthy() {
			return evaluator.FALSE, nil
		}
	}

	return evaluator.TRUE, nil
}

func builtinNoneFunc(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("none", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("none", args[0])
	if err != nil {
		return nil, err
	}

	var fn evaluator.Value
	if len(args) == 2 {
		fn = args[1]
		if err := requireCallable("none", fn); err != nil {
			return nil, err
		}
	}

	for _, item := range list {
		var result evaluator.Value = item
		if fn != nil {
			result, err = callFunction(fn, []evaluator.Value{item})
			if err != nil {
				return nil, err
			}
		}
		if result.IsTruthy() {
			return evaluator.FALSE, nil
		}
	}

	return evaluator.TRUE, nil
}

func builtinFindValue(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("find", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | find(fn)
	list, err := requireList("find", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("find", fn); err != nil {
		return nil, err
	}

	for _, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if val.IsTruthy() {
			return item, nil
		}
	}

	return evaluator.NONE, nil
}

func builtinFindIndex(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("find_index", args, 2); err != nil {
		return nil, err
	}

	// Pipeline style: list | find_index(fn)
	list, err := requireList("find_index", args[0])
	if err != nil {
		return nil, err
	}

	fn := args[1]
	if err := requireCallable("find_index", fn); err != nil {
		return nil, err
	}

	for i, item := range list {
		val, err := callFunction(fn, []evaluator.Value{item})
		if err != nil {
			return nil, err
		}
		if val.IsTruthy() {
			return &evaluator.IntValue{Value: int64(i)}, nil
		}
	}

	return &evaluator.IntValue{Value: -1}, nil
}

// Combination

func builtinConcat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("concat", args, 1); err != nil {
		return nil, err
	}

	result := make([]evaluator.Value, 0)
	for _, arg := range args {
		list, err := requireList("concat", arg)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinZipWith(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("zip_with", args, 3); err != nil {
		return nil, err
	}

	// Pipeline style: list1 | zip_with(list2, fn)
	list1, err := requireList("zip_with", args[0])
	if err != nil {
		return nil, err
	}

	list2, err := requireList("zip_with", args[1])
	if err != nil {
		return nil, err
	}

	fn := args[2]
	if err := requireCallable("zip_with", fn); err != nil {
		return nil, err
	}

	minLen := len(list1)
	if len(list2) < minLen {
		minLen = len(list2)
	}

	result := make([]evaluator.Value, minLen)
	for i := 0; i < minLen; i++ {
		val, err := callFunction(fn, []evaluator.Value{list1[i], list2[i]})
		if err != nil {
			return nil, err
		}
		result[i] = val
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinInterleave(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("interleave", args, 2); err != nil {
		return nil, err
	}

	list1, err := requireList("interleave", args[0])
	if err != nil {
		return nil, err
	}

	list2, err := requireList("interleave", args[1])
	if err != nil {
		return nil, err
	}

	maxLen := len(list1)
	if len(list2) > maxLen {
		maxLen = len(list2)
	}

	result := make([]evaluator.Value, 0, len(list1)+len(list2))
	for i := 0; i < maxLen; i++ {
		if i < len(list1) {
			result = append(result, list1[i])
		}
		if i < len(list2) {
			result = append(result, list2[i])
		}
	}

	return &evaluator.ListValue{Elements: result}, nil
}
