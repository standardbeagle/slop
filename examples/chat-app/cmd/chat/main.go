package main

import (
	"fmt"
	"os"

	"github.com/anthropics/slop/examples/chat-app/internal/chat"
	"github.com/anthropics/slop/examples/chat-app/internal/config"
	"github.com/spf13/cobra"
)

var (
	configFile string
	mcpServer  string
	model      string
	verbose    bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "chat",
		Short: "A CLI LLM chat application with MCP support powered by SLOP",
		Long: `A command-line chat application that uses SLOP scripts to orchestrate
LLM conversations with MCP (Model Context Protocol) tool support.

Examples:
  # Start interactive chat
  chat

  # Chat with a specific MCP server
  chat --mcp-server weather

  # Use a specific model
  chat --model claude-sonnet

  # Run a SLOP script directly
  chat run scripts/research.slop
`,
		RunE: runChat,
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&mcpServer, "mcp-server", "", "MCP server to connect to")
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "claude-sonnet", "LLM model to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	runCmd := &cobra.Command{
		Use:   "run [script]",
		Short: "Run a SLOP script",
		Args:  cobra.ExactArgs(1),
		RunE:  runScript,
	}
	rootCmd.AddCommand(runCmd)

	listCmd := &cobra.Command{
		Use:   "list-tools",
		Short: "List available MCP tools",
		RunE:  listTools,
	}
	rootCmd.AddCommand(listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if model != "" {
		cfg.Model = model
	}

	app, err := chat.New(cfg, verbose)
	if err != nil {
		return fmt.Errorf("failed to create chat app: %w", err)
	}
	defer app.Close()

	// Connect to MCP server if specified
	if mcpServer != "" {
		if err := app.ConnectMCP(mcpServer); err != nil {
			return fmt.Errorf("failed to connect to MCP server: %w", err)
		}
	}

	return app.RunInteractive()
}

func runScript(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if model != "" {
		cfg.Model = model
	}

	app, err := chat.New(cfg, verbose)
	if err != nil {
		return fmt.Errorf("failed to create chat app: %w", err)
	}
	defer app.Close()

	// Connect to MCP server if specified
	if mcpServer != "" {
		if err := app.ConnectMCP(mcpServer); err != nil {
			return fmt.Errorf("failed to connect to MCP server: %w", err)
		}
	}

	return app.RunScript(args[0])
}

func listTools(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	app, err := chat.New(cfg, verbose)
	if err != nil {
		return fmt.Errorf("failed to create chat app: %w", err)
	}
	defer app.Close()

	tools := app.ListTools()
	if len(tools) == 0 {
		fmt.Println("No tools available. Connect to an MCP server to see tools.")
		return nil
	}

	fmt.Println("Available tools:")
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	return nil
}
