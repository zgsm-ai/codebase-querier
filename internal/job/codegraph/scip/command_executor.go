package scip

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// CommandExecutor handles command execution for SCIP indexing
type CommandExecutor struct {
	outputPath string
	mu         sync.Mutex
	processing map[string]struct{}
}

var (
	processingLock  sync.Mutex
	processingRepos = make(map[string]struct{})
)

// NewCommandExecutor creates a new CommandExecutor
func NewCommandExecutor(outputPath string) (*CommandExecutor, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, NewError(ErrCodeResource, "failed to create output directory", err)
	}

	return &CommandExecutor{
		outputPath: outputPath,
		processing: make(map[string]struct{}),
	}, nil
}

// ExecuteCommand executes a command and returns its output
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, command string) (string, error) {
	LogIndexInfo("Executing command: %s", command)

	// Create command with context
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		LogIndexError("Command execution failed: %v, output: %s", err, string(output))
		return "", NewError(ErrCodeCommand, fmt.Sprintf("command execution failed: %s", command), err)
	}

	LogIndexInfo("Command executed successfully, output: %s", string(output))
	return string(output), nil
}

// GenerateIndex generates the SCIP index for a codebase
func (e *CommandExecutor) GenerateIndex(ctx context.Context, codebasePath string, langConfig *LanguageConfig) error {
	// Check if already processing this codebase
	e.mu.Lock()
	if _, exists := e.processing[codebasePath]; exists {
		e.mu.Unlock()
		return NewError(ErrCodeConcurrent, "codebase is already being processed", nil)
	}
	e.processing[codebasePath] = struct{}{}
	e.mu.Unlock()

	// Cleanup function to remove from processing map
	defer func() {
		e.mu.Lock()
		delete(e.processing, codebasePath)
		e.mu.Unlock()
	}()

	// Detect language
	LogIndexInfo("Detected language: %s", langConfig.Name)

	// Detect build tool
	buildTool, err := langConfig.FindBuildTool(codebasePath)
	if err != nil {
		return NewError(ErrCodeBuildTool, "failed to find build tool", err)
	}
	LogIndexInfo("Detected build tool: %s", buildTool.Name)

	// Execute build commands if any
	if len(buildTool.BuildCommands) > 0 {
		LogIndexInfo("Executing build commands for %s", buildTool.Name)
		for _, cmd := range buildTool.BuildCommands {
			buildArgs := strings.Join(cmd.Args, " ")
			_, err := e.ExecuteCommand(ctx, e.BuildCommandString(cmd, buildArgs))
			if err != nil {
				return NewError(ErrCodeCommand, "build command failed", err)
			}
		}
	}

	// Execute index commands
	for _, tool := range langConfig.Tools {
		LogIndexInfo("Executing index commands for tool: %s", tool.Name)
		for _, cmd := range tool.Commands {
			_, err := e.ExecuteCommand(ctx, e.BuildCommandString(cmd, ""))
			if err != nil {
				return NewError(ErrCodeCommand, "index command failed", err)
			}
		}
	}

	// Verify index file was created
	indexPath := filepath.Join(e.outputPath, "index.scip")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return NewError(ErrCodeResource, "index file was not created", err)
	}

	LogIndexInfo("Successfully generated index at: %s", indexPath)
	return nil
}

// BuildCommandString builds a command string with proper argument substitution
func (e *CommandExecutor) BuildCommandString(cmd Command, buildArgs string) string {
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		args[i] = e.replacePlaceholders(arg, buildArgs)
	}
	return fmt.Sprintf("%s %s", cmd.Base, strings.Join(args, " "))
}

// replacePlaceholders replaces placeholders in command arguments
func (e *CommandExecutor) replacePlaceholders(arg, buildArgs string) string {
	arg = strings.ReplaceAll(arg, "__sourcePath__", e.outputPath)
	arg = strings.ReplaceAll(arg, "__outputPath__", e.outputPath)
	arg = strings.ReplaceAll(arg, "__buildArgs__", buildArgs)
	return arg
}

// Cleanup removes the output directory and releases any locks
func (e *CommandExecutor) Cleanup() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Clear processing map
	e.processing = make(map[string]struct{})

	// Remove output directory
	if err := os.RemoveAll(e.outputPath); err != nil {
		return NewError(ErrCodeResource, "failed to cleanup output directory", err)
	}

	return nil
}
