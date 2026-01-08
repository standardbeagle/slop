// Package main provides the CLI entry point for the SLOP interpreter.
package main

import (
	"fmt"
	"os"

	"github.com/standardbeagle/slop/internal/analyzer"
	"github.com/standardbeagle/slop/internal/evaluator"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/standardbeagle/slop/pkg/slop"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "run":
		runCmd(os.Args[2:])
	case "check":
		checkCmd(os.Args[2:])
	case "plan":
		planCmd(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("slop version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`SLOP - Script Language for Orchestrating Protocols

Usage:
  slop <command> [arguments]

Commands:
  run <file>      Execute a SLOP script
  check <file>    Validate a script without running it
  plan <file>     Show execution bounds analysis
  version         Print version information
  help            Show this help message

Examples:
  slop run script.slop
  slop check script.slop
  slop plan script.slop

For more information, visit: https://github.com/standardbeagle/slop`)
}

func runCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing script file")
		fmt.Fprintln(os.Stderr, "Usage: slop run <file>")
		os.Exit(1)
	}

	filename := args[0]
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse config from args
	cfg := slop.Config{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--max-iterations":
			if i+1 < len(args) {
				if _, err := fmt.Sscanf(args[i+1], "%d", &cfg.MaxIterations); err != nil {
					fmt.Fprintf(os.Stderr, "invalid max-iterations value: %v\n", err)
				}
				i++
			}
		case "--max-llm-calls":
			if i+1 < len(args) {
				if _, err := fmt.Sscanf(args[i+1], "%d", &cfg.MaxLLMCalls); err != nil {
					fmt.Fprintf(os.Stderr, "invalid max-llm-calls value: %v\n", err)
				}
				i++
			}
		}
	}

	// Create runtime
	var rt *slop.Runtime
	if cfg.MaxIterations > 0 || cfg.MaxLLMCalls > 0 {
		rt = slop.NewRuntimeWithConfig(cfg)
	} else {
		rt = slop.NewRuntime()
	}

	// Execute
	result, err := rt.Execute(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print emitted values
	emitted := rt.Emitted()
	if len(emitted) > 0 {
		for _, v := range emitted {
			printValue(v)
		}
	} else if result != nil {
		printValue(result)
	}
}

func checkCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing script file")
		fmt.Fprintln(os.Stderr, "Usage: slop check <file>")
		os.Exit(1)
	}

	filename := args[0]
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Lex
	l := lexer.New(string(source))

	// Parse
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parse errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  %s\n", err)
		}
		os.Exit(1)
	}

	// Analyze
	a := analyzer.New()
	errors := a.Analyze(program)

	if len(errors) > 0 {
		fmt.Println("Analysis errors:")
		for _, err := range errors {
			fmt.Printf("  %s\n", err.Error())
		}
		os.Exit(1)
	}

	fmt.Printf("✓ %s: valid\n", filename)

	// Show bounds
	bounds := a.Bounds()
	if bounds.MaxIterations > 0 {
		fmt.Printf("  Max iterations: %d\n", bounds.MaxIterations)
	}
	if bounds.MaxLLMCalls > 0 {
		fmt.Printf("  Max LLM calls: %d\n", bounds.MaxLLMCalls)
	}
	if bounds.MaxAPICalls > 0 {
		fmt.Printf("  Max API calls: %d\n", bounds.MaxAPICalls)
	}
}

func planCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing script file")
		fmt.Fprintln(os.Stderr, "Usage: slop plan <file>")
		os.Exit(1)
	}

	filename := args[0]
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Lex
	l := lexer.New(string(source))

	// Parse
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		fmt.Println("Parse errors:")
		for _, err := range p.Errors() {
			fmt.Printf("  %s\n", err)
		}
		os.Exit(1)
	}

	// Analyze
	a := analyzer.New()
	errors := a.Analyze(program)

	fmt.Printf("Execution Plan: %s\n", filename)
	fmt.Println("=" + string(make([]byte, len(filename)+17)))
	fmt.Println()

	// Bounds
	bounds := a.Bounds()
	fmt.Println("Resource Bounds:")
	fmt.Printf("  Max iterations:  %d\n", bounds.MaxIterations)
	fmt.Printf("  Max LLM calls:   %d\n", bounds.MaxLLMCalls)
	fmt.Printf("  Max API calls:   %d\n", bounds.MaxAPICalls)
	if bounds.MaxCost > 0 {
		fmt.Printf("  Max cost:        $%.2f\n", bounds.MaxCost)
	}
	fmt.Println()

	// Termination guarantee
	if len(errors) == 0 {
		fmt.Println("Termination: Guaranteed (no recursion detected)")
	} else {
		fmt.Println("Termination: NOT guaranteed")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err.Error())
		}
	}
	fmt.Println()

	// Summary
	fmt.Println("Summary:")
	if len(errors) == 0 {
		fmt.Println("  ✓ Script is valid and will terminate")
		fmt.Printf("  ✓ Worst-case: %d iterations, %d LLM calls, %d API calls\n",
			bounds.MaxIterations, bounds.MaxLLMCalls, bounds.MaxAPICalls)
	} else {
		fmt.Println("  ✗ Script has issues that need to be fixed")
	}
}

func printValue(v evaluator.Value) {
	switch val := v.(type) {
	case *evaluator.NoneValue:
		// Don't print none
	case *evaluator.StringValue:
		fmt.Println(val.Value)
	case *evaluator.IntValue:
		fmt.Println(val.Value)
	case *evaluator.FloatValue:
		fmt.Println(val.Value)
	case *evaluator.BoolValue:
		fmt.Println(val.Value)
	case *evaluator.ListValue:
		fmt.Println(formatList(val))
	case *evaluator.MapValue:
		fmt.Println(formatMap(val))
	default:
		fmt.Println(v.String())
	}
}

func formatList(list *evaluator.ListValue) string {
	result := "["
	for i, elem := range list.Elements {
		if i > 0 {
			result += ", "
		}
		result += formatValue(elem)
	}
	result += "]"
	return result
}

func formatMap(m *evaluator.MapValue) string {
	result := "{"
	first := true
	for _, key := range m.Order {
		if !first {
			result += ", "
		}
		first = false
		val, _ := m.Get(key)
		result += fmt.Sprintf("%s: %s", key, formatValue(val))
	}
	result += "}"
	return result
}

func formatValue(v evaluator.Value) string {
	switch val := v.(type) {
	case *evaluator.StringValue:
		return fmt.Sprintf("%q", val.Value)
	case *evaluator.IntValue:
		return fmt.Sprintf("%d", val.Value)
	case *evaluator.FloatValue:
		return fmt.Sprintf("%g", val.Value)
	case *evaluator.BoolValue:
		return fmt.Sprintf("%t", val.Value)
	case *evaluator.NoneValue:
		return "none"
	case *evaluator.ListValue:
		return formatList(val)
	case *evaluator.MapValue:
		return formatMap(val)
	default:
		return v.String()
	}
}
