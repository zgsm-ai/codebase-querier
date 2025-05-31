package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandExecutor(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	executor, err := NewCommandExecutor(tempDir)
	require.NoError(t, err)

	// Test simple command execution
	output, err := executor.ExecuteCommand(context.Background(), "echo 'test'")
	require.NoError(t, err)
	assert.Equal(t, "test\n", output)

	// Test command with error
	_, err = executor.ExecuteCommand(context.Background(), "nonexistent-command")
	assert.Error(t, err)
}

func TestBuildCommandString(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	executor, err := NewCommandExecutor(tempDir)
	require.NoError(t, err)

	cmd := &Command{
		Base: "test-command",
		Args: []string{
			"--source",
			"__sourcePath__",
			"--output",
			"__outputPath__",
			"--build-args",
			"__buildArgs__",
		},
	}

	// Test without build args
	expected := "test-command --source " + tempDir + " --output " + tempDir + " --build-args "
	result := executor.BuildCommandString(cmd, "", tempDir)
	assert.Equal(t, expected, result)

	// Test with build args
	buildArgs := "build arg1 arg2"
	expected = "test-command --source " + tempDir + " --output " + tempDir + " --build-args " + buildArgs
	result = executor.BuildCommandString(cmd, buildArgs, tempDir)
	assert.Equal(t, expected, result)
}

func TestCommandExecutor_Timeout(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	executor, err := NewCommandExecutor(tempDir)
	require.NoError(t, err)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	// Test command timeout
	_, err = executor.ExecuteCommand(ctx, "sleep 2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestCommandExecutor_GenerateIndex(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test repository structure
	testRepo := filepath.Join(tempDir, "test-repo")
	err = os.MkdirAll(testRepo, 0755)
	require.NoError(t, err)

	// Create a test Go module
	err = os.WriteFile(filepath.Join(testRepo, "go.mod"), []byte(`
		module github.com/test/project

		go 1.21
	`), 0644)
	require.NoError(t, err)

	// Create executor
	executor, err := NewCommandExecutor(testRepo)
	require.NoError(t, err)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Generate index
	outputPath := filepath.Join(testRepo, ".codebase_index")
	err = os.MkdirAll(outputPath, 0755)
	require.NoError(t, err)

	cmd := fmt.Sprintf("scip-go --project-root %s --output %s/index.scip", testRepo, outputPath)
	_, err = executor.ExecuteCommand(ctx, cmd)
	if err != nil {
		// If scip-go is not installed, skip the test
		if strings.Contains(err.Error(), "exit status 127") {
			t.Skip("scip-go command not found, skipping test")
		}
		t.Errorf("index command failed: %v", err)
		return
	}

	// Verify the output directory exists
	assert.DirExists(t, outputPath)

	// Verify the index file exists
	indexFile := filepath.Join(outputPath, "index.scip")
	assert.FileExists(t, indexFile)

	// Verify the index file is not empty
	fileInfo, err := os.Stat(indexFile)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}
