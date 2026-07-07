package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/standardbeagle/slop/internal/ast"
)

// CheckpointManager handles checkpoint save/load operations.
type CheckpointManager struct {
	checkpointDir string
	program       *ast.Program
	script        string
}

// NewCheckpointManager creates a new checkpoint manager.
func NewCheckpointManager(checkpointDir string) *CheckpointManager {
	return &CheckpointManager{
		checkpointDir: checkpointDir,
	}
}

// SetProgram sets the program for position tracking.
func (cm *CheckpointManager) SetProgram(program *ast.Program, script string) {
	cm.program = program
	cm.script = script
}

// GetProgram returns the current program.
func (cm *CheckpointManager) GetProgram() *ast.Program {
	return cm.program
}

// SaveCheckpoint saves a checkpoint to disk.
func (cm *CheckpointManager) SaveCheckpoint(ctx *Context, pos Position, name string) (string, error) {
	if cm.checkpointDir == "" {
		return "", fmt.Errorf("checkpoint directory not configured")
	}
	if cm.program == nil {
		return "", fmt.Errorf("program not set for checkpoint manager")
	}

	// Create checkpoint directory if it doesn't exist
	if err := os.MkdirAll(cm.checkpointDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Create serializer
	s := NewSerializer(cm.program)

	// Get checkpoint message
	message := ctx.GetPauseMessage()
	if message == "" && name != "" {
		message = name
	}

	// Create checkpoint
	checkpoint, err := s.CreateCheckpoint(ctx, cm.script, pos, message)
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// Set checkpoint name
	if name != "" {
		checkpoint.CheckpointName = name
	}

	// Serialize to JSON
	data, err := SaveCheckpoint(checkpoint)
	if err != nil {
		return "", fmt.Errorf("failed to serialize checkpoint: %w", err)
	}

	// Generate filename
	filename := cm.generateFilename(checkpoint)
	path := filepath.Join(cm.checkpointDir, filename)

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return path, nil
}

// LoadCheckpoint loads a checkpoint from disk.
func (cm *CheckpointManager) LoadCheckpoint(path string, builtins map[string]*BuiltinValue, services map[string]Service) (*Checkpoint, *Context, error) {
	// Read checkpoint file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	// Load and deserialize
	checkpoint, ctx, err := LoadCheckpoint(data, cm.program, builtins, services)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	return checkpoint, ctx, nil
}

// ValidateCheckpoint checks if a checkpoint is compatible with the current script.
func (cm *CheckpointManager) ValidateCheckpoint(checkpoint *Checkpoint) error {
	if cm.script == "" {
		return nil // No script to validate against
	}

	currentHash := HashScript(cm.script)
	if checkpoint.ScriptHash != currentHash {
		return fmt.Errorf("script has been modified since checkpoint was created")
	}

	return nil
}

// ListCheckpoints returns all checkpoints in the checkpoint directory.
func (cm *CheckpointManager) ListCheckpoints() ([]CheckpointInfo, error) {
	if cm.checkpointDir == "" {
		return nil, fmt.Errorf("checkpoint directory not configured")
	}

	entries, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CheckpointInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var infos []CheckpointInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(cm.checkpointDir, entry.Name())
		info, err := cm.readCheckpointInfo(path)
		if err != nil {
			continue // Skip invalid files
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// CheckpointInfo contains summary information about a checkpoint.
type CheckpointInfo struct {
	Path            string   `json:"path"`
	Name            string   `json:"name,omitempty"`
	Message         string   `json:"message,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Line            int      `json:"line"`
	Column          int      `json:"column"`
	ScriptHash      string   `json:"script_hash"`
}

func (cm *CheckpointManager) readCheckpointInfo(path string) (CheckpointInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CheckpointInfo{}, err
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return CheckpointInfo{}, err
	}

	return CheckpointInfo{
		Path:       path,
		Name:       checkpoint.CheckpointName,
		Message:    checkpoint.CheckpointMessage,
		CreatedAt:  checkpoint.CreatedAt.Format("2006-01-02 15:04:05"),
		Line:       checkpoint.Position.Line,
		Column:     checkpoint.Position.Column,
		ScriptHash: checkpoint.ScriptHash,
	}, nil
}

func (cm *CheckpointManager) generateFilename(checkpoint *Checkpoint) string {
	// Use name if provided, otherwise use timestamp
	name := checkpoint.CheckpointName
	if name == "" {
		name = checkpoint.CreatedAt.Format("20060102_150405")
	}
	// Sanitize name for filename
	safe := sanitizeFilename(name)
	return safe + ".json"
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters with underscores
	result := make([]byte, 0, len(name))
	hasAlphanumeric := false
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			result = append(result, ch)
			hasAlphanumeric = true
		} else if ch == ' ' {
			result = append(result, '_')
		}
	}
	// Return default if empty or only contains converted spaces
	if len(result) == 0 || !hasAlphanumeric {
		return "checkpoint"
	}
	return string(result)
}

// ResumableEvaluator extends Evaluator with checkpoint support.
type ResumableEvaluator struct {
	*Evaluator
	checkpointMgr  *CheckpointManager
	currentPos     Position
	statementIndex int
}

// NewResumableEvaluator creates an evaluator with checkpoint support.
func NewResumableEvaluator(checkpointDir string) *ResumableEvaluator {
	return &ResumableEvaluator{
		Evaluator:     New(),
		checkpointMgr: NewCheckpointManager(checkpointDir),
	}
}

// NewResumableEvaluatorWithEvaluator creates a resumable evaluator using an existing evaluator.
// This preserves the existing evaluator's context, services, and builtins.
func NewResumableEvaluatorWithEvaluator(eval *Evaluator, checkpointDir string) *ResumableEvaluator {
	return &ResumableEvaluator{
		Evaluator:     eval,
		checkpointMgr: NewCheckpointManager(checkpointDir),
	}
}

// SetProgram sets the program and script for checkpoint support.
func (re *ResumableEvaluator) SetProgram(program *ast.Program, script string) {
	re.checkpointMgr.SetProgram(program, script)
}

// EvalWithCheckpoints evaluates a program with checkpoint support.
// If a pause is encountered, it saves a checkpoint and returns.
// Returns the result, checkpoint path (if paused), and any error.
func (re *ResumableEvaluator) EvalWithCheckpoints(program *ast.Program) (Value, string, error) {
	re.checkpointMgr.SetProgram(program, re.checkpointMgr.script)

	// Use resumeExecution starting from index 0 to track statement index
	result, err := re.resumeExecution(program, 0)
	if err != nil {
		return nil, "", err
	}

	// Check if we paused
	if re.ctx.ShouldPause() {
		// Save checkpoint with the current statement index
		pos := Position{
			Line:           re.currentPos.Line,
			Column:         re.currentPos.Column,
			StatementIndex: re.statementIndex,
		}

		checkpointPath, saveErr := re.checkpointMgr.SaveCheckpoint(re.ctx, pos, "")
		if saveErr != nil {
			return nil, "", fmt.Errorf("failed to save checkpoint: %w", saveErr)
		}
		return result, checkpointPath, nil
	}

	return result, "", nil
}

// ResumeFromCheckpoint resumes execution from a checkpoint.
func (re *ResumableEvaluator) ResumeFromCheckpoint(checkpointPath string, builtins map[string]*BuiltinValue, services map[string]Service) (Value, string, error) {
	// Load checkpoint
	checkpoint, ctx, err := re.checkpointMgr.LoadCheckpoint(checkpointPath, builtins, services)
	if err != nil {
		return nil, "", err
	}

	// Validate checkpoint against current script
	if err := re.checkpointMgr.ValidateCheckpoint(checkpoint); err != nil {
		return nil, "", err
	}

	// Replace evaluator context
	re.ctx = ctx
	re.ctx.ClearPause() // Clear pause flag to continue execution

	// Update position
	re.currentPos = checkpoint.Position
	re.statementIndex = checkpoint.Position.StatementIndex

	// Resume execution from the checkpoint position
	// This requires re-parsing the program and skipping to the right statement
	if re.checkpointMgr.program == nil {
		return nil, "", fmt.Errorf("program not set - call SetProgram first")
	}

	// Resume from the statement AFTER the pause statement
	result, err := re.resumeExecution(re.checkpointMgr.program, checkpoint.Position.StatementIndex+1)
	if err != nil {
		return nil, "", err
	}

	// Check if we paused again
	if re.ctx.ShouldPause() {
		pos := Position{
			Line:           re.currentPos.Line,
			Column:         re.currentPos.Column,
			StatementIndex: re.statementIndex,
		}
		checkpointPath, saveErr := re.checkpointMgr.SaveCheckpoint(re.ctx, pos, "")
		if saveErr != nil {
			return nil, "", fmt.Errorf("failed to save checkpoint: %w", saveErr)
		}
		return result, checkpointPath, nil
	}

	return result, "", nil
}

// resumeExecution continues evaluation from a specific statement index.
func (re *ResumableEvaluator) resumeExecution(program *ast.Program, startIndex int) (Value, error) {
	// If program has modules, this is more complex - for now handle simple case
	if len(program.Modules) > 0 {
		// TODO: Handle module-based programs
		return nil, fmt.Errorf("resume not yet supported for programs with modules")
	}

	var result Value = NONE

	for i := startIndex; i < len(program.Statements); i++ {
		stmt := program.Statements[i]
		re.statementIndex = i

		val, err := re.Eval(stmt)
		if err != nil {
			return nil, err
		}
		result = val

		// Check for control flow
		if re.ctx.ShouldReturn() {
			result, _ = re.ctx.GetReturn()
			break
		}
		if re.ctx.ShouldStop() || re.ctx.ShouldPause() {
			break
		}
	}

	return result, nil
}

// GetCheckpointManager returns the checkpoint manager.
func (re *ResumableEvaluator) GetCheckpointManager() *CheckpointManager {
	return re.checkpointMgr
}
