package chat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/slop/examples/chat-app/internal/config"
)

// TestRunScriptSimpleChat tests running the simple_chat.slop script
func TestRunScriptSimpleChat(t *testing.T) {
	// Get absolute path to scripts directory
	scriptsDir, err := filepath.Abs("../../scripts")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ScriptsDir = scriptsDir
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Verify the script file exists
	scriptPath := filepath.Join(scriptsDir, "simple_chat.slop")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("script not found: %s", scriptPath)
	}

	// Run the script
	err = app.RunScript(scriptPath)
	if err != nil {
		t.Errorf("failed to run simple_chat script: %v", err)
	}
}

// TestRunScriptResearch tests running the research.slop script
func TestRunScriptResearch(t *testing.T) {
	scriptsDir, err := filepath.Abs("../../scripts")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ScriptsDir = scriptsDir
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	scriptPath := filepath.Join(scriptsDir, "research.slop")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("script not found: %s", scriptPath)
	}

	err = app.RunScript(scriptPath)
	if err != nil {
		t.Errorf("failed to run research script: %v", err)
	}
}

// TestRunScriptCodeReview tests running the code_review.slop script
func TestRunScriptCodeReview(t *testing.T) {
	scriptsDir, err := filepath.Abs("../../scripts")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ScriptsDir = scriptsDir
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	scriptPath := filepath.Join(scriptsDir, "code_review.slop")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("script not found: %s", scriptPath)
	}

	err = app.RunScript(scriptPath)
	if err != nil {
		t.Errorf("failed to run code_review script: %v", err)
	}
}

// TestRunScriptToolAgent tests running the tool_agent.slop script
func TestRunScriptToolAgent(t *testing.T) {
	scriptsDir, err := filepath.Abs("../../scripts")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ScriptsDir = scriptsDir
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	scriptPath := filepath.Join(scriptsDir, "tool_agent.slop")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skipf("script not found: %s", scriptPath)
	}

	err = app.RunScript(scriptPath)
	if err != nil {
		t.Errorf("failed to run tool_agent script: %v", err)
	}
}

// TestRunScriptNonExistent tests error handling for non-existent script
func TestRunScriptNonExistent(t *testing.T) {
	cfg := config.DefaultConfig()
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	err = app.RunScript("nonexistent.slop")
	if err == nil {
		t.Error("expected error for non-existent script")
	}
}

// TestRunScriptRelativePath tests running a script with relative path
func TestRunScriptRelativePath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ScriptsDir = "../../scripts"
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Should resolve relative to ScriptsDir
	err = app.RunScript("simple_chat.slop")
	if err != nil {
		t.Errorf("failed to run script with relative path: %v", err)
	}
}

// TestScriptWithInvalidSyntax tests error handling for invalid SLOP
func TestScriptWithInvalidSyntax(t *testing.T) {
	cfg := config.DefaultConfig()
	app, err := New(cfg, false)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	// Create a temp script with invalid syntax
	tmpDir := t.TempDir()
	badScript := filepath.Join(tmpDir, "bad.slop")
	if err := os.WriteFile(badScript, []byte("this is not valid SLOP ( ( ("), 0644); err != nil {
		t.Fatalf("failed to create bad script: %v", err)
	}

	err = app.RunScript(badScript)
	if err == nil {
		t.Error("expected error for invalid SLOP syntax")
	}
}
