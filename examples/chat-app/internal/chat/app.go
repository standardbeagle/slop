package chat

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/slop/examples/chat-app/internal/config"
	"github.com/anthropics/slop/examples/chat-app/internal/mcp"
	"github.com/anthropics/slop/internal/evaluator"
	"github.com/anthropics/slop/internal/lexer"
	"github.com/anthropics/slop/internal/parser"
	"github.com/anthropics/slop/pkg/slop"
)

// ToolInfo holds information about an available tool.
type ToolInfo struct {
	Name        string
	Description string
}

// App is the main chat application.
type App struct {
	cfg       *config.Config
	verbose   bool
	runtime   *slop.Runtime
	mcpClient *mcp.Client
	history   []Message
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// New creates a new chat application.
func New(cfg *config.Config, verbose bool) (*App, error) {
	rt := slop.NewRuntime()

	app := &App{
		cfg:     cfg,
		verbose: verbose,
		runtime: rt,
		history: []Message{},
	}

	// Register built-in services
	app.registerBuiltinServices()

	return app, nil
}

// Close closes the chat application.
func (a *App) Close() error {
	if a.mcpClient != nil {
		return a.mcpClient.Close()
	}
	return nil
}

// ConnectMCP connects to an MCP server.
func (a *App) ConnectMCP(serverName string) error {
	serverCfg, ok := a.cfg.MCPServers[serverName]
	if !ok {
		return fmt.Errorf("unknown MCP server: %s", serverName)
	}

	a.mcpClient = mcp.NewClient()
	if err := a.mcpClient.Connect(serverCfg.Command, serverCfg.Args); err != nil {
		return err
	}

	// Register MCP tools as a SLOP service
	mcpService := mcp.NewService(a.mcpClient, serverName)
	a.runtime.RegisterService(serverName, mcpService)

	if a.verbose {
		tools := a.mcpClient.Tools()
		fmt.Printf("Connected to MCP server '%s' with %d tools\n", serverName, len(tools))
		for _, tool := range tools {
			fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
		}
	}

	return nil
}

// ListTools returns the available tools.
func (a *App) ListTools() []ToolInfo {
	if a.mcpClient == nil {
		return nil
	}

	tools := a.mcpClient.Tools()
	result := make([]ToolInfo, len(tools))
	for i, t := range tools {
		result[i] = ToolInfo{
			Name:        t.Name,
			Description: t.Description,
		}
	}
	return result
}

// RunInteractive runs the interactive chat loop.
func (a *App) RunInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("SLOP Chat - Type 'quit' to exit, 'run <script>' to run a script")
	fmt.Println()

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle special commands
		switch {
		case input == "quit" || input == "exit":
			fmt.Println("Goodbye!")
			return nil

		case strings.HasPrefix(input, "run "):
			scriptPath := strings.TrimPrefix(input, "run ")
			if err := a.RunScript(scriptPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue

		case input == "history":
			a.printHistory()
			continue

		case input == "clear":
			a.history = []Message{}
			fmt.Println("History cleared.")
			continue

		case input == "tools":
			a.printTools()
			continue

		case input == "help":
			a.printHelp()
			continue
		}

		// Process as chat message
		if err := a.processMessage(input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}

// RunScript runs a SLOP script.
func (a *App) RunScript(scriptPath string) error {
	// Resolve script path
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(a.cfg.ScriptsDir, scriptPath)
	}

	// Read script
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	// Parse script
	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse errors: %v", p.Errors())
	}

	// Use the runtime's context which has builtins and services already registered
	ctx := a.runtime.Context()

	// Push a new scope for script-specific variables
	ctx.PushScope()
	defer ctx.PopScope()

	// Set script-specific variables in the new scope
	ctx.Scope.Set("history", a.historyToValue())
	ctx.Scope.Set("model", &evaluator.StringValue{Value: a.cfg.Model})
	ctx.Scope.Set("input", evaluator.NewMapValue())

	// Use the runtime's evaluator directly
	eval := a.runtime
	result, err := eval.Eval(program)
	if err != nil {
		return fmt.Errorf("evaluation error: %w", err)
	}

	// Print emitted values
	emitted := a.runtime.Emitted()
	for _, emission := range emitted {
		fmt.Println(emission.String())
	}

	// Clear emitted values for next run
	ctx.Emitted = []evaluator.Value{}

	// If result is a string, add to history
	if str, ok := result.(*evaluator.StringValue); ok && str.Value != "" {
		a.history = append(a.history, Message{Role: "assistant", Content: str.Value})
	}

	return nil
}

func (a *App) processMessage(input string) error {
	// Add user message to history
	a.history = append(a.history, Message{Role: "user", Content: input})

	// Create a simple response script
	script := fmt.Sprintf(`
# Process user message using LLM
response = llm.call(
    prompt: """
You are a helpful assistant. Respond to the user's message.

%s

User: %s
""",
    schema: {response: string}
)

emit(response.response)
`, a.formatHistory(), input)

	// Parse and run
	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return fmt.Errorf("parse errors: %v", p.Errors())
	}

	ctx := evaluator.NewContext()
	for name, svc := range a.runtime.Services() {
		ctx.Services[name] = svc
	}

	eval := evaluator.NewWithContext(ctx)
	_, err := eval.Eval(program)
	if err != nil {
		return err
	}

	// Print and save response
	for _, emission := range ctx.Emitted {
		response := emission.String()
		fmt.Printf("\n%s\n\n", response)
		a.history = append(a.history, Message{Role: "assistant", Content: response})
	}

	return nil
}

func (a *App) formatHistory() string {
	if len(a.history) == 0 {
		return "No previous conversation."
	}

	var sb strings.Builder
	sb.WriteString("Previous conversation:\n")
	for _, msg := range a.history {
		sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	return sb.String()
}

func (a *App) historyToValue() evaluator.Value {
	items := make([]evaluator.Value, len(a.history))
	for i, msg := range a.history {
		items[i] = &evaluator.MapValue{
			Pairs: map[string]evaluator.Value{
				"role":    &evaluator.StringValue{Value: msg.Role},
				"content": &evaluator.StringValue{Value: msg.Content},
			},
		}
	}
	return &evaluator.ListValue{Elements: items}
}

func (a *App) registerBuiltinServices() {
	// LLM service is already registered by the runtime
	// Add any additional built-in services here
}

func (a *App) printHistory() {
	if len(a.history) == 0 {
		fmt.Println("No conversation history.")
		return
	}

	fmt.Println("Conversation history:")
	for _, msg := range a.history {
		fmt.Printf("  [%s] %s\n", msg.Role, msg.Content)
	}
}

func (a *App) printTools() {
	tools := a.ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available. Connect to an MCP server first.")
		return
	}

	fmt.Println("Available tools:")
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
}

func (a *App) printHelp() {
	fmt.Println(`Commands:
  quit, exit  - Exit the chat
  run <file>  - Run a SLOP script
  history     - Show conversation history
  clear       - Clear conversation history
  tools       - List available MCP tools
  help        - Show this help message

Or just type a message to chat with the AI.`)
}
