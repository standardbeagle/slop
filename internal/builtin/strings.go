package builtin

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/anthropics/slop/internal/evaluator"
)

func (r *Registry) registerStringFunctions() {
	// String methods are implemented as method calls on string values
	// These are global functions that can also be used
	r.Register("upper", builtinUpper)
	r.Register("lower", builtinLower)
	r.Register("strip", builtinStrip)
	r.Register("lstrip", builtinLstrip)
	r.Register("rstrip", builtinRstrip)
	r.Register("split", builtinSplit)
	r.Register("join", builtinJoin)
	r.Register("replace", builtinReplace)
	r.Register("startswith", builtinStartswith)
	r.Register("endswith", builtinEndswith)
	r.Register("contains", builtinContains)
	r.Register("find", builtinFind)
	r.Register("count", builtinCount)
	r.Register("format", builtinFormat)
	r.Register("pad_left", builtinPadLeft)
	r.Register("pad_right", builtinPadRight)
	r.Register("slice", builtinSlice)
	r.Register("repeat", builtinRepeat)
	r.Register("reverse", builtinReverse)
	r.Register("lines", builtinLines)
	r.Register("words", builtinWords)
	r.Register("title", builtinTitle)
	r.Register("capitalize", builtinCapitalize)
	r.Register("isdigit", builtinIsDigit)
	r.Register("isalpha", builtinIsAlpha)
	r.Register("isalnum", builtinIsAlnum)
	r.Register("isspace", builtinIsSpace)
	r.Register("ord", builtinOrd)
	r.Register("chr", builtinChr)
}

func builtinUpper(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("upper", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("upper", args[0])
	if err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: strings.ToUpper(s)}, nil
}

func builtinLower(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("lower", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("lower", args[0])
	if err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: strings.ToLower(s)}, nil
}

func builtinStrip(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("strip", args, 1, 2); err != nil {
		return nil, err
	}
	s, err := requireString("strip", args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 2 {
		chars, err := requireString("strip", args[1])
		if err != nil {
			return nil, err
		}
		return &evaluator.StringValue{Value: strings.Trim(s, chars)}, nil
	}
	return &evaluator.StringValue{Value: strings.TrimSpace(s)}, nil
}

func builtinLstrip(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("lstrip", args, 1, 2); err != nil {
		return nil, err
	}
	s, err := requireString("lstrip", args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 2 {
		chars, err := requireString("lstrip", args[1])
		if err != nil {
			return nil, err
		}
		return &evaluator.StringValue{Value: strings.TrimLeft(s, chars)}, nil
	}
	return &evaluator.StringValue{Value: strings.TrimLeftFunc(s, unicode.IsSpace)}, nil
}

func builtinRstrip(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("rstrip", args, 1, 2); err != nil {
		return nil, err
	}
	s, err := requireString("rstrip", args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 2 {
		chars, err := requireString("rstrip", args[1])
		if err != nil {
			return nil, err
		}
		return &evaluator.StringValue{Value: strings.TrimRight(s, chars)}, nil
	}
	return &evaluator.StringValue{Value: strings.TrimRightFunc(s, unicode.IsSpace)}, nil
}

func builtinSplit(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("split", args, 1, 2); err != nil {
		return nil, err
	}
	s, err := requireString("split", args[0])
	if err != nil {
		return nil, err
	}

	var parts []string
	if len(args) == 2 {
		sep, err := requireString("split", args[1])
		if err != nil {
			return nil, err
		}
		parts = strings.Split(s, sep)
	} else {
		// Split on whitespace
		parts = strings.Fields(s)
	}

	items := make([]evaluator.Value, len(parts))
	for i, part := range parts {
		items[i] = &evaluator.StringValue{Value: part}
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinJoin(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("join", args, 2); err != nil {
		return nil, err
	}
	sep, err := requireString("join", args[0])
	if err != nil {
		return nil, err
	}
	list, err := requireList("join", args[1])
	if err != nil {
		return nil, err
	}

	parts := make([]string, len(list))
	for i, item := range list {
		parts[i] = item.String()
	}
	return &evaluator.StringValue{Value: strings.Join(parts, sep)}, nil
}

func builtinReplace(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("replace", args, 3, 4); err != nil {
		return nil, err
	}
	s, err := requireString("replace", args[0])
	if err != nil {
		return nil, err
	}
	old, err := requireString("replace", args[1])
	if err != nil {
		return nil, err
	}
	newStr, err := requireString("replace", args[2])
	if err != nil {
		return nil, err
	}

	n := -1 // Replace all by default
	if len(args) == 4 {
		nVal, err := requireInt("replace", args[3])
		if err != nil {
			return nil, err
		}
		n = int(nVal)
	}

	return &evaluator.StringValue{Value: strings.Replace(s, old, newStr, n)}, nil
}

func builtinStartswith(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("startswith", args, 2); err != nil {
		return nil, err
	}
	s, err := requireString("startswith", args[0])
	if err != nil {
		return nil, err
	}
	prefix, err := requireString("startswith", args[1])
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(s, prefix) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinEndswith(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("endswith", args, 2); err != nil {
		return nil, err
	}
	s, err := requireString("endswith", args[0])
	if err != nil {
		return nil, err
	}
	suffix, err := requireString("endswith", args[1])
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(s, suffix) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinContains(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("contains", args, 2); err != nil {
		return nil, err
	}
	s, err := requireString("contains", args[0])
	if err != nil {
		return nil, err
	}
	substr, err := requireString("contains", args[1])
	if err != nil {
		return nil, err
	}
	if strings.Contains(s, substr) {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinFind(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("find", args, 2); err != nil {
		return nil, err
	}
	s, err := requireString("find", args[0])
	if err != nil {
		return nil, err
	}
	substr, err := requireString("find", args[1])
	if err != nil {
		return nil, err
	}
	return &evaluator.IntValue{Value: int64(strings.Index(s, substr))}, nil
}

func builtinCount(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("count", args, 2); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.StringValue:
		substr, err := requireString("count", args[1])
		if err != nil {
			return nil, err
		}
		return &evaluator.IntValue{Value: int64(strings.Count(v.Value, substr))}, nil
	case *evaluator.ListValue:
		count := 0
		for _, item := range v.Elements {
			if evaluator.Equal(item, args[1]) {
				count++
			}
		}
		return &evaluator.IntValue{Value: int64(count)}, nil
	default:
		return nil, fmt.Errorf("count() first argument must be string or list, got %s", v.Type())
	}
}

func builtinFormat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireMinArgs("format", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("format", args[0])
	if err != nil {
		return nil, err
	}

	// Simple placeholder replacement: {0}, {1}, etc.
	result := s
	for i := 1; i < len(args); i++ {
		placeholder := fmt.Sprintf("{%d}", i-1)
		result = strings.Replace(result, placeholder, args[i].String(), -1)
	}

	// Also replace {} in order
	for i := 1; i < len(args); i++ {
		result = strings.Replace(result, "{}", args[i].String(), 1)
	}

	return &evaluator.StringValue{Value: result}, nil
}

func builtinPadLeft(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("pad_left", args, 2, 3); err != nil {
		return nil, err
	}
	s, err := requireString("pad_left", args[0])
	if err != nil {
		return nil, err
	}
	width, err := requireInt("pad_left", args[1])
	if err != nil {
		return nil, err
	}

	char := " "
	if len(args) == 3 {
		char, err = requireString("pad_left", args[2])
		if err != nil {
			return nil, err
		}
		if len(char) == 0 {
			char = " "
		}
	}

	if int64(len(s)) >= width {
		return &evaluator.StringValue{Value: s}, nil
	}

	padding := ""
	for int64(len(padding)+len(s)) < width {
		padding += char
	}
	// Trim if we overshot
	if int64(len(padding)+len(s)) > width {
		padding = padding[:int(width)-len(s)]
	}

	return &evaluator.StringValue{Value: padding + s}, nil
}

func builtinPadRight(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("pad_right", args, 2, 3); err != nil {
		return nil, err
	}
	s, err := requireString("pad_right", args[0])
	if err != nil {
		return nil, err
	}
	width, err := requireInt("pad_right", args[1])
	if err != nil {
		return nil, err
	}

	char := " "
	if len(args) == 3 {
		char, err = requireString("pad_right", args[2])
		if err != nil {
			return nil, err
		}
		if len(char) == 0 {
			char = " "
		}
	}

	if int64(len(s)) >= width {
		return &evaluator.StringValue{Value: s}, nil
	}

	result := s
	for int64(len(result)) < width {
		result += char
	}
	// Trim if we overshot
	if int64(len(result)) > width {
		result = result[:width]
	}

	return &evaluator.StringValue{Value: result}, nil
}

func builtinSlice(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireRangeArgs("slice", args, 2, 3); err != nil {
		return nil, err
	}

	start, err := requireInt("slice", args[1])
	if err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.StringValue:
		length := int64(len(v.Value))
		if start < 0 {
			start = length + start
		}
		if start < 0 {
			start = 0
		}
		if start > length {
			start = length
		}

		end := length
		if len(args) == 3 {
			end, err = requireInt("slice", args[2])
			if err != nil {
				return nil, err
			}
			if end < 0 {
				end = length + end
			}
			if end < start {
				end = start
			}
			if end > length {
				end = length
			}
		}

		return &evaluator.StringValue{Value: v.Value[start:end]}, nil

	case *evaluator.ListValue:
		length := int64(len(v.Elements))
		if start < 0 {
			start = length + start
		}
		if start < 0 {
			start = 0
		}
		if start > length {
			start = length
		}

		end := length
		if len(args) == 3 {
			end, err = requireInt("slice", args[2])
			if err != nil {
				return nil, err
			}
			if end < 0 {
				end = length + end
			}
			if end < start {
				end = start
			}
			if end > length {
				end = length
			}
		}

		return &evaluator.ListValue{Elements: v.Elements[start:end]}, nil

	default:
		return nil, fmt.Errorf("slice() first argument must be string or list, got %s", v.Type())
	}
}

func builtinRepeat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("repeat", args, 2); err != nil {
		return nil, err
	}
	s, err := requireString("repeat", args[0])
	if err != nil {
		return nil, err
	}
	n, err := requireInt("repeat", args[1])
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, fmt.Errorf("repeat() count must be non-negative")
	}
	return &evaluator.StringValue{Value: strings.Repeat(s, int(n))}, nil
}

func builtinReverse(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("reverse", args, 1); err != nil {
		return nil, err
	}

	switch v := args[0].(type) {
	case *evaluator.StringValue:
		runes := []rune(v.Value)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return &evaluator.StringValue{Value: string(runes)}, nil

	case *evaluator.ListValue:
		items := make([]evaluator.Value, len(v.Elements))
		for i, j := 0, len(v.Elements)-1; j >= 0; i, j = i+1, j-1 {
			items[i] = v.Elements[j]
		}
		return &evaluator.ListValue{Elements: items}, nil

	default:
		return nil, fmt.Errorf("reverse() argument must be string or list, got %s", v.Type())
	}
}

func builtinLines(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("lines", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("lines", args[0])
	if err != nil {
		return nil, err
	}

	lines := strings.Split(s, "\n")
	items := make([]evaluator.Value, len(lines))
	for i, line := range lines {
		items[i] = &evaluator.StringValue{Value: line}
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinWords(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("words", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("words", args[0])
	if err != nil {
		return nil, err
	}

	// Use word boundary regex for better word detection
	wordRegex := regexp.MustCompile(`\b\w+\b`)
	words := wordRegex.FindAllString(s, -1)
	items := make([]evaluator.Value, len(words))
	for i, word := range words {
		items[i] = &evaluator.StringValue{Value: word}
	}
	return &evaluator.ListValue{Elements: items}, nil
}

func builtinTitle(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("title", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("title", args[0])
	if err != nil {
		return nil, err
	}
	return &evaluator.StringValue{Value: strings.Title(s)}, nil
}

func builtinCapitalize(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("capitalize", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("capitalize", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) == 0 {
		return &evaluator.StringValue{Value: ""}, nil
	}
	return &evaluator.StringValue{Value: strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])}, nil
}

func builtinIsDigit(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("isdigit", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("isdigit", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) == 0 {
		return evaluator.FALSE, nil
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}

func builtinIsAlpha(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("isalpha", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("isalpha", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) == 0 {
		return evaluator.FALSE, nil
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}

func builtinIsAlnum(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("isalnum", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("isalnum", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) == 0 {
		return evaluator.FALSE, nil
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}

func builtinIsSpace(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("isspace", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("isspace", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) == 0 {
		return evaluator.FALSE, nil
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return evaluator.FALSE, nil
		}
	}
	return evaluator.TRUE, nil
}

func builtinOrd(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("ord", args, 1); err != nil {
		return nil, err
	}
	s, err := requireString("ord", args[0])
	if err != nil {
		return nil, err
	}
	if len(s) != 1 {
		return nil, fmt.Errorf("ord() expected a character, but string of length %d found", len(s))
	}
	return &evaluator.IntValue{Value: int64(s[0])}, nil
}

func builtinChr(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("chr", args, 1); err != nil {
		return nil, err
	}
	n, err := requireInt("chr", args[0])
	if err != nil {
		return nil, err
	}
	if n < 0 || n > 0x10FFFF {
		return nil, fmt.Errorf("chr() arg not in range(0x110000)")
	}
	return &evaluator.StringValue{Value: string(rune(n))}, nil
}
