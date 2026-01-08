package builtin

import (
	"fmt"
	"math"

	"github.com/standardbeagle/slop/internal/evaluator"
)

func (r *Registry) registerMathFunctions() {
	r.Register("abs", builtinAbs)
	r.Register("min", builtinMin)
	r.Register("max", builtinMax)
	r.Register("sum", builtinSum)
	r.Register("round", builtinRound)
	r.Register("floor", builtinFloor)
	r.Register("ceil", builtinCeil)
	r.Register("pow", builtinPow)
	r.Register("sqrt", builtinSqrt)
	r.Register("log", builtinLog)
	r.Register("sin", builtinSin)
	r.Register("cos", builtinCos)
	r.Register("tan", builtinTan)
	r.Register("asin", builtinAsin)
	r.Register("acos", builtinAcos)
	r.Register("atan", builtinAtan)
	r.Register("atan2", builtinAtan2)
	r.Register("exp", builtinExp)
	r.Register("log10", builtinLog10)
	r.Register("log2", builtinLog2)
}

func builtinAbs(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("abs", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.IntValue:
		if v.Value < 0 {
			return &evaluator.IntValue{Value: -v.Value}, nil
		}
		return v, nil
	case *evaluator.FloatValue:
		return &evaluator.FloatValue{Value: math.Abs(v.Value)}, nil
	default:
		return nil, fmt.Errorf("abs() requires numeric argument, got %s", v.Type())
	}
}

func builtinMin(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("min() requires at least 1 argument")
	}

	// If single argument is a list, find min in the list
	if len(args) == 1 {
		list, err := requireList("min", args[0])
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("min() argument is an empty sequence")
		}
		args = list
	}

	minVal := args[0]
	for _, arg := range args[1:] {
		cmp, err := evaluator.Compare(arg, minVal)
		if err != nil {
			return nil, err
		}
		if cmp < 0 {
			minVal = arg
		}
	}

	return minVal, nil
}

func builtinMax(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("max() requires at least 1 argument")
	}

	// If single argument is a list, find max in the list
	if len(args) == 1 {
		list, err := requireList("max", args[0])
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("max() argument is an empty sequence")
		}
		args = list
	}

	maxVal := args[0]
	for _, arg := range args[1:] {
		cmp, err := evaluator.Compare(arg, maxVal)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			maxVal = arg
		}
	}

	return maxVal, nil
}

func builtinSum(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("sum", args, 1, 2); err != nil {
		return nil, err
	}

	list, err := requireList("sum", args[0])
	if err != nil {
		return nil, err
	}

	var startVal evaluator.Value = &evaluator.IntValue{Value: 0}
	if len(args) == 2 {
		startVal = args[1]
	}

	// Determine if we're working with ints or floats
	hasFloat := false
	if _, ok := startVal.(*evaluator.FloatValue); ok {
		hasFloat = true
	}
	for _, item := range list {
		if _, ok := item.(*evaluator.FloatValue); ok {
			hasFloat = true
			break
		}
	}

	if hasFloat {
		sum, _ := toFloat(startVal)
		for _, item := range list {
			f, ok := toFloat(item)
			if !ok {
				return nil, fmt.Errorf("sum() requires numeric elements, got %s", item.Type())
			}
			sum += f
		}
		return &evaluator.FloatValue{Value: sum}, nil
	}

	sum, _ := toInt(startVal)
	for _, item := range list {
		i, ok := toInt(item)
		if !ok {
			return nil, fmt.Errorf("sum() requires numeric elements, got %s", item.Type())
		}
		sum += i
	}
	return &evaluator.IntValue{Value: sum}, nil
}

func builtinRound(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("round", args, 1, 2); err != nil {
		return nil, err
	}

	f, err := requireFloat("round", args[0])
	if err != nil {
		return nil, err
	}

	var decimals int64 = 0
	if len(args) == 2 {
		decimals, err = requireInt("round", args[1])
		if err != nil {
			return nil, err
		}
	}

	if decimals == 0 {
		return &evaluator.IntValue{Value: int64(math.Round(f))}, nil
	}

	mult := math.Pow(10, float64(decimals))
	rounded := math.Round(f*mult) / mult
	return &evaluator.FloatValue{Value: rounded}, nil
}

func builtinFloor(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("floor", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("floor", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.IntValue{Value: int64(math.Floor(f))}, nil
}

func builtinCeil(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("ceil", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("ceil", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.IntValue{Value: int64(math.Ceil(f))}, nil
}

func builtinPow(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("pow", args, 2); err != nil {
		return nil, err
	}

	base, err := requireFloat("pow", args[0])
	if err != nil {
		return nil, err
	}

	exp, err := requireFloat("pow", args[1])
	if err != nil {
		return nil, err
	}

	result := math.Pow(base, exp)

	// Return int if both args were ints and result is whole number
	if _, ok := args[0].(*evaluator.IntValue); ok {
		if _, ok := args[1].(*evaluator.IntValue); ok {
			if result == math.Floor(result) {
				return &evaluator.IntValue{Value: int64(result)}, nil
			}
		}
	}

	return &evaluator.FloatValue{Value: result}, nil
}

func builtinSqrt(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("sqrt", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("sqrt", args[0])
	if err != nil {
		return nil, err
	}

	if f < 0 {
		return nil, fmt.Errorf("sqrt() argument must be non-negative")
	}

	return &evaluator.FloatValue{Value: math.Sqrt(f)}, nil
}

func builtinLog(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("log", args, 1, 2); err != nil {
		return nil, err
	}

	x, err := requireFloat("log", args[0])
	if err != nil {
		return nil, err
	}

	if x <= 0 {
		return nil, fmt.Errorf("log() argument must be positive")
	}

	if len(args) == 1 {
		// Natural log
		return &evaluator.FloatValue{Value: math.Log(x)}, nil
	}

	base, err := requireFloat("log", args[1])
	if err != nil {
		return nil, err
	}

	if base <= 0 || base == 1 {
		return nil, fmt.Errorf("log() base must be positive and not 1")
	}

	return &evaluator.FloatValue{Value: math.Log(x) / math.Log(base)}, nil
}

func builtinSin(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("sin", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("sin", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Sin(f)}, nil
}

func builtinCos(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("cos", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("cos", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Cos(f)}, nil
}

func builtinTan(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("tan", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("tan", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Tan(f)}, nil
}

func builtinAsin(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("asin", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("asin", args[0])
	if err != nil {
		return nil, err
	}

	if f < -1 || f > 1 {
		return nil, fmt.Errorf("asin() argument must be in range [-1, 1]")
	}

	return &evaluator.FloatValue{Value: math.Asin(f)}, nil
}

func builtinAcos(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("acos", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("acos", args[0])
	if err != nil {
		return nil, err
	}

	if f < -1 || f > 1 {
		return nil, fmt.Errorf("acos() argument must be in range [-1, 1]")
	}

	return &evaluator.FloatValue{Value: math.Acos(f)}, nil
}

func builtinAtan(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("atan", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("atan", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Atan(f)}, nil
}

func builtinAtan2(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("atan2", args, 2); err != nil {
		return nil, err
	}

	y, err := requireFloat("atan2", args[0])
	if err != nil {
		return nil, err
	}

	x, err := requireFloat("atan2", args[1])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Atan2(y, x)}, nil
}

func builtinExp(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("exp", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("exp", args[0])
	if err != nil {
		return nil, err
	}

	return &evaluator.FloatValue{Value: math.Exp(f)}, nil
}

func builtinLog10(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("log10", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("log10", args[0])
	if err != nil {
		return nil, err
	}

	if f <= 0 {
		return nil, fmt.Errorf("log10() argument must be positive")
	}

	return &evaluator.FloatValue{Value: math.Log10(f)}, nil
}

func builtinLog2(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("log2", args, 1); err != nil {
		return nil, err
	}

	f, err := requireFloat("log2", args[0])
	if err != nil {
		return nil, err
	}

	if f <= 0 {
		return nil, fmt.Errorf("log2() argument must be positive")
	}

	return &evaluator.FloatValue{Value: math.Log2(f)}, nil
}
