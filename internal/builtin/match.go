// Package builtin provides built-in functions for the SLOP language.
// This file implements the match() function for template pattern extraction.
package builtin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/anthropics/slop/internal/evaluator"
)

// PatternSpec defines a named pattern with its regex and optional conversion.
type PatternSpec struct {
	Regex     string
	Greedy    bool
	Validator func(s string) error
	Converter func(s string) (evaluator.Value, error)
}

// TemplateToken represents a parsed token from the template.
type TemplateToken struct {
	IsPlaceholder bool
	Literal       string
	Name          string
	TypeHint      string
	Optional      bool
}

// CompiledTemplate holds a parsed and compiled template.
type CompiledTemplate struct {
	Tokens    []TemplateToken
	Regex     *regexp.Regexp
	Names     []string
	TypeHints map[string]string
	Optional  map[string]bool
}

// defaultPatterns holds the built-in pattern registry.
var defaultPatterns = map[string]PatternSpec{
	// Basic types
	"int": {
		Regex: `[+-]?\d+`,
		Converter: func(s string) (evaluator.Value, error) {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid integer: %s", err)
			}
			return &evaluator.IntValue{Value: i}, nil
		},
	},
	"float": {
		Regex: `[+-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?`,
		Converter: func(s string) (evaluator.Value, error) {
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid float: %s", err)
			}
			return &evaluator.FloatValue{Value: f}, nil
		},
	},
	"word": {
		Regex: `\S+`,
	},
	"rest": {
		Regex:  `.*`,
		Greedy: true,
	},
	"json": {
		Regex: `(?:\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}|\[[^\[\]]*(?:\[[^\[\]]*\][^\[\]]*)*\])`,
		Converter: func(s string) (evaluator.Value, error) {
			var data interface{}
			if err := json.Unmarshal([]byte(s), &data); err != nil {
				return nil, fmt.Errorf("invalid JSON: %s", err)
			}
			return jsonToValue(data), nil
		},
	},

	// Named patterns
	"IP": {
		Regex: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		Validator: func(s string) error {
			parts := strings.Split(s, ".")
			for _, p := range parts {
				n, err := strconv.Atoi(p)
				if err != nil || n < 0 || n > 255 {
					return fmt.Errorf("invalid IP octet: %s", p)
				}
			}
			return nil
		},
	},
	"iso8601": {
		Regex: `\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`,
	},
	"email": {
		Regex: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	},
	"uuid": {
		Regex: `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	},
	"url": {
		Regex: `https?://[^\s]+`,
	},
}

// parseTemplate parses a template string into tokens.
func parseTemplate(template string) ([]TemplateToken, error) {
	tokens := []TemplateToken{}
	i := 0
	literalStart := 0
	runes := []rune(template)

	for i < len(runes) {
		if runes[i] == '{' {
			// Check for escaped brace
			if i+1 < len(runes) && runes[i+1] == '{' {
				// Add literal including first brace
				if literalStart <= i {
					tokens = append(tokens, TemplateToken{
						IsPlaceholder: false,
						Literal:       string(runes[literalStart : i+1]),
					})
				}
				i += 2
				literalStart = i
				continue
			}

			// Save any preceding literal
			if literalStart < i {
				tokens = append(tokens, TemplateToken{
					IsPlaceholder: false,
					Literal:       string(runes[literalStart:i]),
				})
			}

			// Parse placeholder
			j := i + 1
			optional := false

			// Check for optional marker
			if j < len(runes) && runes[j] == '?' {
				optional = true
				j++
			}

			// Find closing brace
			end := -1
			for k := j; k < len(runes); k++ {
				if runes[k] == '}' {
					end = k
					break
				}
			}
			if end == -1 {
				return nil, fmt.Errorf("unclosed brace at position %d", i)
			}

			// Parse name and type hint
			content := string(runes[j:end])
			name, typeHint := parseNameAndType(content)

			if name == "" {
				return nil, fmt.Errorf("empty placeholder name at position %d", i)
			}

			// Validate name (must be valid identifier)
			if !isValidIdentifier(name) {
				return nil, fmt.Errorf("invalid placeholder name '%s' at position %d", name, i)
			}

			tokens = append(tokens, TemplateToken{
				IsPlaceholder: true,
				Name:          name,
				TypeHint:      typeHint,
				Optional:      optional,
			})

			i = end + 1
			literalStart = i
			continue
		}

		// Handle escaped closing brace
		if runes[i] == '}' && i+1 < len(runes) && runes[i+1] == '}' {
			if literalStart <= i {
				tokens = append(tokens, TemplateToken{
					IsPlaceholder: false,
					Literal:       string(runes[literalStart : i+1]),
				})
			}
			i += 2
			literalStart = i
			continue
		}

		i++
	}

	// Add remaining literal
	if literalStart < len(runes) {
		tokens = append(tokens, TemplateToken{
			IsPlaceholder: false,
			Literal:       string(runes[literalStart:]),
		})
	}

	return tokens, nil
}

// parseNameAndType splits "name:type" into name and type hint.
func parseNameAndType(content string) (name, typeHint string) {
	parts := strings.SplitN(content, ":", 2)
	name = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		typeHint = strings.TrimSpace(parts[1])
	}
	return
}

// isValidIdentifier checks if a string is a valid SLOP identifier.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	runes := []rune(s)
	// First character must be letter or underscore
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		return false
	}
	// Rest can be letters, digits, or underscores
	for _, r := range runes[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// compileTemplate converts parsed tokens to a compiled template with regex.
func compileTemplate(tokens []TemplateToken) (*CompiledTemplate, error) {
	var regexBuilder strings.Builder
	regexBuilder.WriteString("^")

	names := []string{}
	typeHints := map[string]string{}
	optionals := map[string]bool{}

	for _, token := range tokens {
		if !token.IsPlaceholder {
			regexBuilder.WriteString(regexp.QuoteMeta(token.Literal))
			continue
		}

		// Get pattern for type hint
		var pattern string
		spec, hasSpec := defaultPatterns[token.TypeHint]

		if hasSpec {
			if spec.Greedy {
				pattern = fmt.Sprintf("(%s)", spec.Regex)
			} else {
				// For non-greedy patterns, use non-greedy quantifier
				pattern = fmt.Sprintf("(%s)", spec.Regex)
			}
		} else if token.TypeHint != "" {
			// Unknown type hint - treat as default pattern
			pattern = "(.+?)"
		} else {
			// Default: non-greedy match
			pattern = "(.+?)"
		}

		if token.Optional {
			// Make the capture group optional
			innerPattern := pattern[1 : len(pattern)-1] // Remove outer parens
			pattern = fmt.Sprintf("(%s)?", innerPattern)
		}

		regexBuilder.WriteString(pattern)
		names = append(names, token.Name)
		typeHints[token.Name] = token.TypeHint
		optionals[token.Name] = token.Optional
	}

	regexBuilder.WriteString("$")

	regex, err := regexp.Compile(regexBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("regex compilation failed: %s", err)
	}

	return &CompiledTemplate{
		Tokens:    tokens,
		Regex:     regex,
		Names:     names,
		TypeHints: typeHints,
		Optional:  optionals,
	}, nil
}

// convertValue applies type conversion based on type hint.
func convertValue(captured string, typeHint string) (evaluator.Value, error) {
	if typeHint == "" {
		return &evaluator.StringValue{Value: captured}, nil
	}

	spec, exists := defaultPatterns[typeHint]
	if !exists {
		// Unknown type hint - return as string
		return &evaluator.StringValue{Value: captured}, nil
	}

	// Run validator if present
	if spec.Validator != nil {
		if err := spec.Validator(captured); err != nil {
			return nil, err
		}
	}

	// Run converter if present
	if spec.Converter != nil {
		return spec.Converter(captured)
	}

	return &evaluator.StringValue{Value: captured}, nil
}

// builtinMatch implements the match(text, pattern, **kwargs) function.
// It extracts values from text using a template pattern.
//
// Examples:
//
//	match("The sum of 5 and 10 is 15.", "The sum of {x} and {y} is {z}.")
//	  -> {x: "5", y: "10", z: "15"}
//
//	match("value: 42", "value: {n:int}")
//	  -> {n: 42}  (integer, not string)
//
//	match(line, "{timestamp:iso8601} [{level:word}] {message:rest}")
//	  -> extracts timestamp, level, and message
func builtinMatch(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("match", args, 2); err != nil {
		return nil, err
	}

	text, err := requireString("match", args[0])
	if err != nil {
		return nil, err
	}

	pattern, err := requireString("match", args[1])
	if err != nil {
		return nil, err
	}

	// Check for strict mode
	strict := false
	if sv, ok := kwargs["strict"]; ok {
		if bv, ok := sv.(*evaluator.BoolValue); ok {
			strict = bv.Value
		}
	}

	// Parse and compile template
	tokens, err := parseTemplate(pattern)
	if err != nil {
		return nil, fmt.Errorf("match() invalid pattern: %s", err)
	}

	compiled, err := compileTemplate(tokens)
	if err != nil {
		return nil, fmt.Errorf("match() pattern compilation error: %s", err)
	}

	// Execute regex match
	matches := compiled.Regex.FindStringSubmatch(text)
	if matches == nil {
		if strict {
			return nil, fmt.Errorf("match() pattern did not match text")
		}
		return evaluator.NewMapValue(), nil
	}

	// Build result map
	result := evaluator.NewMapValue()
	for i, name := range compiled.Names {
		if i+1 >= len(matches) {
			continue
		}

		captured := matches[i+1]

		// Skip empty optional matches
		if captured == "" && compiled.Optional[name] {
			continue
		}

		// Convert value based on type hint
		typeHint := compiled.TypeHints[name]
		value, err := convertValue(captured, typeHint)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("match() conversion error for '%s': %s", name, err)
			}
			// On conversion error in non-strict mode, use string value
			value = &evaluator.StringValue{Value: captured}
		}

		result.Set(name, value)
	}

	return result, nil
}
