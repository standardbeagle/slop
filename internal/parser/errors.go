// Package parser provides parsing for the SLOP language.
package parser

import (
	"fmt"

	"github.com/anthropics/slop/internal/lexer"
)

// Error represents a parsing error.
type Error struct {
	Token   lexer.Token
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("parse error at %d:%d: %s", e.Token.Line, e.Token.Column, e.Message)
}

// Errors is a collection of parsing errors.
type Errors []*Error

func (e Errors) Error() string {
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	msg := fmt.Sprintf("%d parse errors:\n", len(e))
	for _, err := range e {
		msg += "  " + err.Error() + "\n"
	}
	return msg
}
