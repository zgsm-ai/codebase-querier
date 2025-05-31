package scip

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandExecutor_ExecuteCommand(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
languages:
  - name: typescript
    detection_files: ["package.json"]
    tools:
      - name: scip-typescript
        commands:
          - base: "echo"
            args:
              - "test"
              - "__sourcePath__"
              - "__outputPath__"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(tempDir, configPath)
	if err != nil {
		t.Fatalf("Failed to create command executor: %v", err)
	}

	// Test ExecuteCommand
	ctx := context.Background()
	output, err := executor.ExecuteCommand(ctx, "echo test")
	if err != nil {
		t.Errorf("ExecuteCommand failed: %v", err)
	}
	if output != "test\n" {
		t.Errorf("ExecuteCommand returned wrong output: got %v, want %v", output, "test\n")
	}
}

func TestCommandExecutor_ExecuteCommandStruct(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
languages:
  - name: typescript
    detection_files: ["package.json"]
    tools:
      - name: scip-typescript
        commands:
          - base: "echo"
            args:
              - "test"
              - "__sourcePath__"
              - "__outputPath__"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(tempDir, configPath)
	if err != nil {
		t.Fatalf("Failed to create command executor: %v", err)
	}

	// Test ExecuteCommandStruct
	ctx := context.Background()
	cmd := Command{
		Base: "echo",
		Args: []CommandArg{"test", "__sourcePath__", "__outputPath__"},
	}
	err = executor.ExecuteCommandStruct(ctx, cmd, "")
	if err != nil {
		t.Errorf("ExecuteCommandStruct failed: %v", err)
	}
}

func TestCommandExecutor_BuildCommandString(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
languages:
  - name: typescript
    detection_files: ["package.json"]
    tools:
      - name: scip-typescript
        commands:
          - base: "echo"
            args:
              - "test"
              - "__sourcePath__"
              - "__outputPath__"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(tempDir, configPath)
	if err != nil {
		t.Fatalf("Failed to create command executor: %v", err)
	}

	// Test BuildCommandString
	cmd := Command{
		Base: "echo",
		Args: []CommandArg{"test", "__sourcePath__", "__outputPath__"},
	}
	expected := "echo test " + tempDir + " " + filepath.Join(tempDir, ".codebase_index")
	actual := executor.BuildCommandString(cmd, "")
	if actual != expected {
		t.Errorf("BuildCommandString returned wrong string: got %v, want %v", actual, expected)
	}
}

func TestCommandExecutor_GenerateIndex(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test repository structure
	repoPath := filepath.Join(tempDir, "test-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("Failed to create test repo dir: %v", err)
	}

	// Create a test package.json for TypeScript detection
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"dependencies": {}
	}`
	if err := os.WriteFile(filepath.Join(repoPath, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// Create a test config file
	configPath := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
languages:
  - name: typescript
    detection_files: ["package.json"]
    tools:
      - name: scip-typescript
        commands:
          - base: "echo"
            args:
              - "test"
              - "__sourcePath__"
              - "__outputPath__"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(repoPath, configPath)
	if err != nil {
		t.Fatalf("Failed to create command executor: %v", err)
	}

	// Test GenerateIndex
	ctx := context.Background()
	err = executor.GenerateIndex(ctx)
	if err != nil {
		t.Errorf("GenerateIndex failed: %v", err)
	}

	// Verify output directory was created
	outputDir := filepath.Join(repoPath, ".codebase_index")
	if _, err := os.Stat(outputDir); err != nil {
		t.Errorf("Output directory was not created: %v", err)
	}
} 