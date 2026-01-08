package builtin

import (
	"fmt"
	"strconv"

	"github.com/standardbeagle/slop/internal/evaluator"
)

func (r *Registry) registerCoreFunctions() {
	// Schema type identifiers (for use in llm.call schema definitions)
	r.RegisterConstant("string", &evaluator.StringValue{Value: "string"})
	r.RegisterConstant("number", &evaluator.StringValue{Value: "number"})
	r.RegisterConstant("integer", &evaluator.StringValue{Value: "integer"})
	r.RegisterConstant("boolean", &evaluator.StringValue{Value: "boolean"})
	r.RegisterConstant("object", &evaluator.StringValue{Value: "object"})
	r.RegisterConstant("array", &evaluator.StringValue{Value: "array"})

	// Type conversion
	r.Register("int", builtinInt)
	r.Register("float", builtinFloat)
	r.Register("str", builtinStr)
	r.Register("bool", builtinBool)
	r.Register("list", builtinList)
	r.Register("set", builtinSet)
	r.Register("dict", builtinDict)

	// Type checking
	r.Register("type", builtinType)
	r.Register("is_none", builtinIsNone)
	r.Register("is_bool", builtinIsBool)
	r.Register("is_int", builtinIsInt)
	r.Register("is_float", builtinIsFloat)
	r.Register("is_number", builtinIsNumber)
	r.Register("is_string", builtinIsString)
	r.Register("is_list", builtinIsList)
	r.Register("is_map", builtinIsMap)
	r.Register("is_set", builtinIsSet)
	r.Register("is_callable", builtinIsCallable)

	// Basic functions
	r.Register("len", builtinLen)
	r.Register("print", builtinPrint)
	r.Register("range", builtinRange)
	r.Register("enumerate", builtinEnumerate)
	r.Register("zip", builtinZip)
}

// Type conversion functions

func builtinInt(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("int", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.IntValue:
		return v, nil
	case *evaluator.FloatValue:
		return &evaluator.IntValue{Value: int64(v.Value)}, nil
	case *evaluator.StringValue:
		i, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("int() cannot convert string '%s' to int", v.Value)
		}
		return &evaluator.IntValue{Value: i}, nil
	case *evaluator.BoolValue:
		if v.Value {
			return &evaluator.IntValue{Value: 1}, nil
		}
		return &evaluator.IntValue{Value: 0}, nil
	default:
		return nil, fmt.Errorf("int() cannot convert %s to int", v.Type())
	}
}

func builtinFloat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("float", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.FloatValue:
		return v, nil
	case *evaluator.IntValue:
		return &evaluator.FloatValue{Value: float64(v.Value)}, nil
	case *evaluator.StringValue:
		f, err := strconv.ParseFloat(v.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("float() cannot convert string '%s' to float", v.Value)
		}
		return &evaluator.FloatValue{Value: f}, nil
	case *evaluator.BoolValue:
		if v.Value {
			return &evaluator.FloatValue{Value: 1.0}, nil
		}
		return &evaluator.FloatValue{Value: 0.0}, nil
	default:
		return nil, fmt.Errorf("float() cannot convert %s to float", v.Type())
	}
}

func builtinStr(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("str", args, 1); err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: args[0].String()}, nil
}

func builtinBool(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("bool", args, 1); err != nil {
		return nil, err
	}
	if args[0].IsTruthy() {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinList(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("list", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.ListValue:
		// Return a copy
		items := make([]evaluator.Value, len(v.Elements))
		copy(items, v.Elements)
		return &evaluator.ListValue{Elements: items}, nil
	case *evaluator.SetValue:
		items := make([]evaluator.Value, 0, len(v.Elements))
		for _, val := range v.Elements {
			items = append(items, val)
		}
		return &evaluator.ListValue{Elements: items}, nil
	case *evaluator.StringValue:
		items := make([]evaluator.Value, len(v.Value))
		for i, ch := range v.Value {
			items[i] = &evaluator.StringValue{Value: string(ch)}
		}
		return &evaluator.ListValue{Elements: items}, nil
	case *evaluator.MapValue:
		// Return list of keys
		items := make([]evaluator.Value, 0, len(v.Pairs))
		for key := range v.Pairs {
			items = append(items, &evaluator.StringValue{Value: key})
		}
		return &evaluator.ListValue{Elements: items}, nil
	case *evaluator.IteratorValue:
		items := make([]evaluator.Value, 0)
		for {
			item, hasNext := v.Next()
			if !hasNext {
				break
			}
			items = append(items, item)
		}
		return &evaluator.ListValue{Elements: items}, nil
	default:
		return nil, fmt.Errorf("list() cannot convert %s to list", v.Type())
	}
}

func builtinSet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("set", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.SetValue:
		// Return a copy
		items := make(map[string]evaluator.Value, len(v.Elements))
		for k, val := range v.Elements {
			items[k] = val
		}
		return &evaluator.SetValue{Elements: items}, nil
	case *evaluator.ListValue:
		items := make(map[string]evaluator.Value)
		for _, item := range v.Elements {
			key := item.String()
			items[key] = item
		}
		return &evaluator.SetValue{Elements: items}, nil
	case *evaluator.StringValue:
		items := make(map[string]evaluator.Value)
		for _, ch := range v.Value {
			s := string(ch)
			items[s] = &evaluator.StringValue{Value: s}
		}
		return &evaluator.SetValue{Elements: items}, nil
	default:
		return nil, fmt.Errorf("set() cannot convert %s to set", v.Type())
	}
}

func builtinDict(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if len(args) == 0 {
		return &evaluator.MapValue{Pairs: make(map[string]evaluator.Value)}, nil
	}

	if err := requireArgs("dict", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.MapValue:
		// Return a copy
		pairs := make(map[string]evaluator.Value, len(v.Pairs))
		for k, val := range v.Pairs {
			pairs[k] = val
		}
		return &evaluator.MapValue{Pairs: pairs}, nil
	case *evaluator.ListValue:
		// Expect list of [key, value] pairs
		pairs := make(map[string]evaluator.Value)
		for _, item := range v.Elements {
			pair, ok := item.(*evaluator.ListValue)
			if !ok || len(pair.Elements) != 2 {
				return nil, fmt.Errorf("dict() expects list of [key, value] pairs")
			}
			key := pair.Elements[0].String()
			pairs[key] = pair.Elements[1]
		}
		return &evaluator.MapValue{Pairs: pairs}, nil
	default:
		return nil, fmt.Errorf("dict() cannot convert %s to dict", v.Type())
	}
}

// Type checking functions

func builtinType(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("type", args, 1); err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: args[0].Type()}, nil
}

func builtinIsNone(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_none", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsNone(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsBool(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_bool", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsBool(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsInt(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_int", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsInt(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsFloat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_float", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsFloat(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsNumber(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_number", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsNumber(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsString(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_string", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsString(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsList(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_list", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsList(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsMap(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_map", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsMap(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsSet(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_set", args, 1); err != nil {
		return nil, err
	}
	if _, ok := args[0].(*evaluator.SetValue); ok {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinIsCallable(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("is_callable", args, 1); err != nil {
		return nil, err
	}
	if evaluator.IsCallable(args[0]) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

// Basic functions

func builtinLen(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("len", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.StringValue:
		return &evaluator.IntValue{Value: int64(len(v.Value))}, nil
	case *evaluator.ListValue:
		return &evaluator.IntValue{Value: int64(len(v.Elements))}, nil
	case *evaluator.MapValue:
		return &evaluator.IntValue{Value: int64(len(v.Pairs))}, nil
	case *evaluator.SetValue:
		return &evaluator.IntValue{Value: int64(len(v.Elements))}, nil
	default:
		return nil, fmt.Errorf("len() argument must be a sequence, got %s", v.Type())
	}
}

func builtinPrint(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	for i, arg := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(arg.String())
	}
	fmt.Println()
	return evaluator.NONE, nil
}

func builtinRange(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("range", args, 1, 3); err != nil {
		return nil, err
	}

	var start, stop, step int64

	switch len(args) {
	case 1:
		var err error
		stop, err = requireInt("range", args[0])
		if err != nil {
			return nil, err
		}
		start = 0
		step = 1
	case 2:
		var err error
		start, err = requireInt("range", args[0])
		if err != nil {
			return nil, err
		}
		stop, err = requireInt("range", args[1])
		if err != nil {
			return nil, err
		}
		step = 1
	case 3:
		var err error
		start, err = requireInt("range", args[0])
		if err != nil {
			return nil, err
		}
		stop, err = requireInt("range", args[1])
		if err != nil {
			return nil, err
		}
		step, err = requireInt("range", args[2])
		if err != nil {
			return nil, err
		}
		if step == 0 {
			return nil, fmt.Errorf("range() step argument must not be zero")
		}
	}

	items := make([]evaluator.Value, 0)
	if step > 0 {
		for i := start; i < stop; i += step {
			items = append(items, &evaluator.IntValue{Value: i})
		}
	} else {
		for i := start; i > stop; i += step {
			items = append(items, &evaluator.IntValue{Value: i})
		}
	}

	return &evaluator.ListValue{Elements: items}, nil
}

func builtinEnumerate(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("enumerate", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("enumerate", args[0])
	if err != nil {
		return nil, err
	}

	var start int64 = 0
	if len(args) == 2 {
		start, err = requireInt("enumerate", args[1])
		if err != nil {
			return nil, err
		}
	}

	items := make([]evaluator.Value, len(list))
	for i, v := range list {
		items[i] = &evaluator.ListValue{
			Elements: []evaluator.Value{
				&evaluator.IntValue{Value: start + int64(i)},
				v,
			},
		}
	}

	return &evaluator.ListValue{Elements: items}, nil
}

func builtinZip(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("zip", args, 2); err != nil {
		return nil, err
	}

	lists := make([][]evaluator.Value, len(args))
	minLen := -1

	for i, arg := range args {
		list, err := requireList("zip", arg)
		if err != nil {
			return nil, err
		}
		lists[i] = list
		if minLen < 0 || len(list) < minLen {
			minLen = len(list)
		}
	}

	items := make([]evaluator.Value, minLen)
	for i := 0; i < minLen; i++ {
		tuple := make([]evaluator.Value, len(lists))
		for j, list := range lists {
			tuple[j] = list[i]
		}
		items[i] = &evaluator.ListValue{Elements: tuple}
	}

	return &evaluator.ListValue{Elements: items}, nil
}
