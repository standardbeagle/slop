package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model != "claude-sonnet" {
		t.Errorf("expected default model 'claude-sonnet', got '%s'", cfg.Model)
	}

	if cfg.MaxTokens != 4096 {
		t.Errorf("expected default max_tokens 4096, got %d", cfg.MaxTokens)
	}

	if cfg.Temperature != 0.7 {
		t.Errorf("expected default temperature 0.7, got %f", cfg.Temperature)
	}

	if cfg.ScriptsDir != "scripts" {
		t.Errorf("expected default scripts_dir 'scripts', got '%s'", cfg.ScriptsDir)
	}

	if cfg.MCPServers == nil {
		t.Error("expected MCPServers to be initialized")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"model": "claude-opus",
		"max_tokens": 8192,
		"temperature": 0.5,
		"scripts_dir": "my-scripts",
		"mcp_servers": {
			"test-server": {
				"command": "test-cmd",
				"args": ["arg1", "arg2"],
				"transport": "stdio"
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Model != "claude-opus" {
		t.Errorf("expected model 'claude-opus', got '%s'", cfg.Model)
	}

	if cfg.MaxTokens != 8192 {
		t.Errorf("expected max_tokens 8192, got %d", cfg.MaxTokens)
	}

	if cfg.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %f", cfg.Temperature)
	}

	if cfg.ScriptsDir != "my-scripts" {
		t.Errorf("expected scripts_dir 'my-scripts', got '%s'", cfg.ScriptsDir)
	}

	if len(cfg.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server, got %d", len(cfg.MCPServers))
	}

	server, ok := cfg.MCPServers["test-server"]
	if !ok {
		t.Error("expected 'test-server' in MCPServers")
	}

	if server.Command != "test-cmd" {
		t.Errorf("expected command 'test-cmd', got '%s'", server.Command)
	}

	if len(server.Args) != 2 || server.Args[0] != "arg1" || server.Args[1] != "arg2" {
		t.Errorf("unexpected args: %v", server.Args)
	}

	if server.Transport != "stdio" {
		t.Errorf("expected transport 'stdio', got '%s'", server.Transport)
	}
}

func TestLoadConfigEnvOverride(t *testing.T) {
	// Set API key via environment
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.APIKey != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got '%s'", cfg.APIKey)
	}
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	cfg := &Config{
		Model:       "claude-opus",
		APIKey:      "secret-key",
		MaxTokens:   8192,
		Temperature: 0.3,
		ScriptsDir:  "my-scripts",
		MCPServers: map[string]MCPServerConfig{
			"my-server": {
				Command:   "server-cmd",
				Args:      []string{"--port", "8080"},
				Transport: "stdio",
				Env:       map[string]string{"KEY": "value"},
			},
		},
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load and verify content
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.Model != cfg.Model {
		t.Errorf("expected model '%s', got '%s'", cfg.Model, loaded.Model)
	}

	if loaded.MaxTokens != cfg.MaxTokens {
		t.Errorf("expected max_tokens %d, got %d", cfg.MaxTokens, loaded.MaxTokens)
	}

	if len(loaded.MCPServers) != 1 {
		t.Errorf("expected 1 MCP server, got %d", len(loaded.MCPServers))
	}

	server := loaded.MCPServers["my-server"]
	if server.Env["KEY"] != "value" {
		t.Errorf("expected env KEY='value', got '%s'", server.Env["KEY"])
	}
}

func TestLoadDefaultConfigWithNoFile(t *testing.T) {
	// Unset any environment variables
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	// Load with empty path should return defaults when no file exists
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}

	if cfg.Model != "claude-sonnet" {
		t.Errorf("expected default model, got '%s'", cfg.Model)
	}
}

func TestMCPServerConfigWithHTTPTransport(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"model": "claude-sonnet",
		"mcp_servers": {
			"http-server": {
				"transport": "http",
				"url": "http://localhost:8080/mcp"
			}
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	server := cfg.MCPServers["http-server"]
	if server.Transport != "http" {
		t.Errorf("expected transport 'http', got '%s'", server.Transport)
	}

	if server.URL != "http://localhost:8080/mcp" {
		t.Errorf("expected URL 'http://localhost:8080/mcp', got '%s'", server.URL)
	}
}
