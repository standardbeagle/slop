package scripts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/slop/internal/lexer"
	"github.com/anthropics/slop/internal/parser"
)

// TestScriptsParse verifies that all example scripts parse without errors
func TestScriptsParse(t *testing.T) {
	scriptsDir := "."
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		t.Fatalf("failed to read scripts directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".slop" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			scriptPath := filepath.Join(scriptsDir, entry.Name())
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("failed to read script: %v", err)
			}

			l := lexer.New(string(content))
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Errorf("parse errors in %s:", entry.Name())
				for _, err := range p.Errors() {
					t.Errorf("  %s", err)
				}
				t.FailNow()
			}

			if program == nil {
				t.Fatal("parser returned nil program")
			}
		})
	}
}

// TestSimpleChatScriptStructure tests specific parsing of simple_chat.slop
func TestSimpleChatScriptStructure(t *testing.T) {
	content, err := os.ReadFile("simple_chat.slop")
	if err != nil {
		t.Skipf("simple_chat.slop not found: %v", err)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Errorf("parse errors:")
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
	}

	// Verify we have at least a few statements
	if program != nil && len(program.Statements) < 2 {
		t.Errorf("expected at least 2 statements, got %d", len(program.Statements))
	}
}

// TestResearchScriptStructure tests research.slop parsing
func TestResearchScriptStructure(t *testing.T) {
	content, err := os.ReadFile("research.slop")
	if err != nil {
		t.Skipf("research.slop not found: %v", err)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Errorf("parse errors:")
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
	}

	if program == nil {
		t.Fatal("parser returned nil program")
	}
}

// TestCodeReviewScriptStructure tests code_review.slop parsing
func TestCodeReviewScriptStructure(t *testing.T) {
	content, err := os.ReadFile("code_review.slop")
	if err != nil {
		t.Skipf("code_review.slop not found: %v", err)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Errorf("parse errors:")
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
	}

	if program == nil {
		t.Fatal("parser returned nil program")
	}
}

// TestToolAgentScriptStructure tests tool_agent.slop parsing
func TestToolAgentScriptStructure(t *testing.T) {
	content, err := os.ReadFile("tool_agent.slop")
	if err != nil {
		t.Skipf("tool_agent.slop not found: %v", err)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Errorf("parse errors:")
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
	}

	if program == nil {
		t.Fatal("parser returned nil program")
	}
}

// TestMultilineStringNotSupported tests that multiline strings fail as expected
func TestMultilineStringNotSupported(t *testing.T) {
	// This test documents that SLOP doesn't support multiline strings
	input := `prompt: """
line1
line2
"""`

	l := lexer.New(input)
	p := parser.New(l)
	p.ParseProgram()

	// We expect this to have parse errors
	if len(p.Errors()) == 0 {
		t.Error("expected parse errors for multiline string, but got none")
	}
}
