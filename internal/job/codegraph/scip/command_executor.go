package scip

import (
	"bytes"
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

// NewCommandExecutor creates a new CommandExecutor
func NewCommandExecutor(outputPath string) (*CommandExecutor, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("output path is required")
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &CommandExecutor{
		outputPath: outputPath,
		processing: make(map[string]struct{}),
	}, nil
}

// Execute executes a command
func (e *CommandExecutor) Execute(ctx context.Context, cmd *Command) error {
	cmdStr := e.BuildCommandString(cmd, "", e.outputPath)
	_, err := e.ExecuteCommand(ctx, cmdStr)
	return err
}

// ExecuteCommand executes a command string
func (e *CommandExecutor) ExecuteCommand(ctx context.Context, cmdStr string) (string, error) {
	LogIndexInfo("Executing command: %s", cmdStr)

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	cmd.Dir = e.outputPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String()
		if stderr.String() != "" {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}
		LogIndexError("Command execution failed: %v, output: %s", err, output)
		return "", err
	}

	output := stdout.String()
	if stderr.String() != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	LogIndexInfo("Command executed successfully, output: %s", output)
	return output, nil
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

	// Execute index commands
	for _, tool := range langConfig.Tools {
		LogIndexInfo("Executing index commands for tool: %s", tool.Name)
		for _, cmd := range tool.Commands {
			_, err := e.ExecuteCommand(ctx, e.BuildCommandString(cmd, "", codebasePath))
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
func (e *CommandExecutor) BuildCommandString(cmd *Command, buildArgs string, sourcePath string) string {
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		arg = strings.ReplaceAll(arg, "__sourcePath__", sourcePath)
		arg = strings.ReplaceAll(arg, "__outputPath__", e.outputPath)
		arg = strings.ReplaceAll(arg, "__buildArgs__", buildArgs)
		args[i] = arg
	}

	return fmt.Sprintf("%s %s", cmd.Base, strings.Join(args, " "))
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
