package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/standardbeagle/slop/internal/ast"
	"github.com/standardbeagle/slop/internal/lexer"
	"github.com/standardbeagle/slop/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointManager_SaveAndLoad(t *testing.T) {
	// Create temp directory for checkpoints
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 1
pause "test checkpoint"
x = 2`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	// Create evaluator and run until pause
	e := New()
	_, err = e.Eval(program)
	require.NoError(t, err)
	require.True(t, e.ctx.ShouldPause())

	// Create checkpoint manager
	cm := NewCheckpointManager(tmpDir)
	cm.SetProgram(program, script)

	// Save checkpoint
	pos := Position{Line: 2, Column: 1, StatementIndex: 1}
	path, err := cm.SaveCheckpoint(e.ctx, pos, "test")
	require.NoError(t, err)
	require.FileExists(t, path)

	// Load checkpoint
	builtins := make(map[string]*BuiltinValue)
	services := make(map[string]Service)
	checkpoint, ctx, err := cm.LoadCheckpoint(path, builtins, services)
	require.NoError(t, err)
	require.NotNil(t, checkpoint)
	require.NotNil(t, ctx)

	// Verify checkpoint data
	assert.Equal(t, "1.0", checkpoint.Version)
	assert.Equal(t, "test checkpoint", checkpoint.CheckpointMessage)
	assert.Equal(t, 2, checkpoint.Position.Line)
	assert.Equal(t, 1, checkpoint.Position.StatementIndex)

	// Verify context was restored
	x, ok := ctx.Scope.Get("x")
	require.True(t, ok)
	assert.Equal(t, int64(1), x.(*IntValue).Value)
}

func TestCheckpointManager_ListCheckpoints(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 1
pause "checkpoint 1"
x = 2`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	// Create and save multiple checkpoints
	cm := NewCheckpointManager(tmpDir)
	cm.SetProgram(program, script)

	e := New()
	_, err = e.Eval(program)
	require.NoError(t, err)

	// Save first checkpoint
	pos1 := Position{Line: 2, Column: 1, StatementIndex: 1}
	_, err = cm.SaveCheckpoint(e.ctx, pos1, "first")
	require.NoError(t, err)

	// Modify and save second
	e.ctx.Scope.Set("x", &IntValue{Value: 100})
	pos2 := Position{Line: 3, Column: 1, StatementIndex: 2}
	_, err = cm.SaveCheckpoint(e.ctx, pos2, "second")
	require.NoError(t, err)

	// List checkpoints
	infos, err := cm.ListCheckpoints()
	require.NoError(t, err)
	require.Len(t, infos, 2)

	// Verify info content
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name
	}
	assert.Contains(t, names, "first")
	assert.Contains(t, names, "second")
}

func TestCheckpointManager_ValidateCheckpoint(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 1
pause`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	cm := NewCheckpointManager(tmpDir)
	cm.SetProgram(program, script)

	e := New()
	_, err = e.Eval(program)
	require.NoError(t, err)

	pos := Position{Line: 2, Column: 1, StatementIndex: 1}
	path, err := cm.SaveCheckpoint(e.ctx, pos, "validate-test")
	require.NoError(t, err)

	// Load and validate - should pass
	builtins := make(map[string]*BuiltinValue)
	services := make(map[string]Service)
	checkpoint, _, err := cm.LoadCheckpoint(path, builtins, services)
	require.NoError(t, err)

	err = cm.ValidateCheckpoint(checkpoint)
	assert.NoError(t, err)

	// Change script and validate - should fail
	cm.SetProgram(program, "modified script")
	err = cm.ValidateCheckpoint(checkpoint)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "script has been modified")
}

func TestResumableEvaluator_BasicPauseResume(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 1
pause "checkpoint"
x = 2
x`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	// Create resumable evaluator
	re := NewResumableEvaluator(tmpDir)
	re.checkpointMgr.SetProgram(program, script)

	// First run - should pause
	result, checkpointPath, err := re.EvalWithCheckpoints(program)
	require.NoError(t, err)
	assert.NotEmpty(t, checkpointPath)
	assert.True(t, re.ctx.ShouldPause())

	// Verify x is still 1
	x, ok := re.ctx.Scope.Get("x")
	require.True(t, ok)
	assert.Equal(t, int64(1), x.(*IntValue).Value)

	// Resume from checkpoint
	builtins := make(map[string]*BuiltinValue)
	services := make(map[string]Service)
	result, newCheckpointPath, err := re.ResumeFromCheckpoint(checkpointPath, builtins, services)
	require.NoError(t, err)
	assert.Empty(t, newCheckpointPath) // No more pauses

	// Verify final value
	assert.Equal(t, int64(2), result.(*IntValue).Value)
}

func TestResumableEvaluator_MultiplePauses(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 1
pause "first"
x = x + 1
pause "second"
x = x + 1
x`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	re := NewResumableEvaluator(tmpDir)
	re.checkpointMgr.SetProgram(program, script)

	// First run - should pause at "first"
	_, path1, err := re.EvalWithCheckpoints(program)
	require.NoError(t, err)
	assert.NotEmpty(t, path1)

	// Resume - should pause at "second"
	builtins := make(map[string]*BuiltinValue)
	services := make(map[string]Service)
	_, path2, err := re.ResumeFromCheckpoint(path1, builtins, services)
	require.NoError(t, err)
	assert.NotEmpty(t, path2)

	// Verify x is 2 after first resume
	x, ok := re.ctx.Scope.Get("x")
	require.True(t, ok)
	assert.Equal(t, int64(2), x.(*IntValue).Value)

	// Final resume
	result, path3, err := re.ResumeFromCheckpoint(path2, builtins, services)
	require.NoError(t, err)
	assert.Empty(t, path3)

	// Verify final value
	assert.Equal(t, int64(3), result.(*IntValue).Value)
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with-dashes", "with-dashes"},
		{"with_underscores", "with_underscores"},
		{"UPPERCASE", "UPPERCASE"},
		{"with/slashes", "withslashes"},
		{"with:colons", "withcolons"},
		{"with*special!chars", "withspecialchars"},
		{"", "checkpoint"},
		{"   ", "checkpoint"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckpointInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slop-checkpoints-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	script := `x = 42
pause "info test"`

	l := lexer.New(script)
	p := parser.New(l)
	program := p.ParseProgram()
	require.Empty(t, p.Errors())

	cm := NewCheckpointManager(tmpDir)
	cm.SetProgram(program, script)

	e := New()
	_, err = e.Eval(program)
	require.NoError(t, err)

	pos := Position{Line: 2, Column: 1, StatementIndex: 1}
	path, err := cm.SaveCheckpoint(e.ctx, pos, "info-test")
	require.NoError(t, err)

	// Read info
	infos, err := cm.ListCheckpoints()
	require.NoError(t, err)
	require.Len(t, infos, 1)

	info := infos[0]
	assert.Equal(t, "info-test", info.Name)
	assert.Equal(t, "info test", info.Message)
	assert.Equal(t, 2, info.Line)
	assert.Equal(t, path, info.Path)
	assert.NotEmpty(t, info.ScriptHash)
	assert.NotEmpty(t, info.CreatedAt)
}

func TestCheckpointDirectory(t *testing.T) {
	// Test with non-existent directory
	tmpDir := filepath.Join(os.TempDir(), "slop-test-nonexistent")
	defer os.RemoveAll(tmpDir)

	cm := NewCheckpointManager(tmpDir)
	cm.SetProgram(&ast.Program{}, "x = 1")

	ctx := NewContext()
	ctx.Scope.Set("x", &IntValue{Value: 1})

	// Save should create directory
	pos := Position{Line: 1, Column: 1}
	_, err := cm.SaveCheckpoint(ctx, pos, "test")
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, tmpDir)
}
