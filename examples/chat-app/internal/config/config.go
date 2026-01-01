package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	// Model is the LLM model to use (e.g., "claude-sonnet", "claude-opus")
	Model string `json:"model"`

	// APIKey is the Anthropic API key (can also be set via ANTHROPIC_API_KEY env)
	APIKey string `json:"api_key,omitempty"`

	// MCPServers is a list of MCP server configurations
	MCPServers map[string]MCPServerConfig `json:"mcp_servers,omitempty"`

	// MaxTokens is the maximum number of tokens for LLM responses
	MaxTokens int `json:"max_tokens"`

	// Temperature controls randomness in LLM responses
	Temperature float64 `json:"temperature"`

	// ScriptsDir is the directory containing SLOP scripts
	ScriptsDir string `json:"scripts_dir"`
}

// MCPServerConfig holds configuration for an MCP server.
type MCPServerConfig struct {
	// Command is the command to run the MCP server
	Command string `json:"command"`

	// Args are the arguments to pass to the command
	Args []string `json:"args,omitempty"`

	// Env is additional environment variables for the server
	Env map[string]string `json:"env,omitempty"`

	// Transport is the transport type ("stdio" or "http")
	Transport string `json:"transport"`

	// URL is the URL for HTTP transport
	URL string `json:"url,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Model:       "claude-sonnet",
		MaxTokens:   4096,
		Temperature: 0.7,
		ScriptsDir:  "scripts",
		MCPServers:  make(map[string]MCPServerConfig),
	}
}

// Load loads the configuration from a file.
// If path is empty, it looks for config in default locations.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	// Try to load .env file (ignore errors if not found)
	_ = godotenv.Load()
	_ = godotenv.Load("../../.env") // Try parent directories
	_ = godotenv.Load("../.env")

	// Check for API key in environment
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}

	// If no path specified, try default locations
	if path == "" {
		paths := []string{
			"chat-config.json",
			filepath.Join(os.Getenv("HOME"), ".config", "slop-chat", "config.json"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	// If still no path, return default config
	if path == "" {
		return cfg, nil
	}

	// Load from file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves the configuration to a file.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
