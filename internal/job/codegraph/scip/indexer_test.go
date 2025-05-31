package scip

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexer_GenerateIndex(t *testing.T) {
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

	// Create indexer
	indexer := NewIndexer(configPath)

	// Test GenerateIndex
	ctx := context.Background()
	err = indexer.GenerateIndex(ctx, repoPath)
	if err != nil {
		t.Errorf("GenerateIndex failed: %v", err)
	}

	// Verify output directory was created
	outputDir := filepath.Join(repoPath, ".codebase_index")
	if _, err := os.Stat(outputDir); err != nil {
		t.Errorf("Output directory was not created: %v", err)
	}
}

func TestIndexer_GetIndexPath(t *testing.T) {
	indexer := NewIndexer("test-config.yaml")
	repoPath := "/test/repo"
	expectedPath := filepath.Join(repoPath, ".codebase_index", "index.scip")
	
	actualPath := indexer.GetIndexPath(repoPath)
	if actualPath != expectedPath {
		t.Errorf("GetIndexPath returned wrong path: got %v, want %v", actualPath, expectedPath)
	}
}

func TestIndexer_Cleanup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test repository with output directory
	repoPath := filepath.Join(tempDir, "test-repo")
	outputDir := filepath.Join(repoPath, ".codebase_index")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Create a test file in the output directory
	testFile := filepath.Join(outputDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create indexer
	indexer := NewIndexer("test-config.yaml")

	// Test Cleanup
	if err := indexer.Cleanup(repoPath); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Verify output directory was removed
	if _, err := os.Stat(outputDir); err == nil {
		t.Error("Output directory was not removed")
	}
}

func TestIndexer_InvalidRepoPath(t *testing.T) {
	indexer := NewIndexer("test-config.yaml")
	ctx := context.Background()
	
	err := indexer.GenerateIndex(ctx, "/nonexistent/path")
	if err == nil {
		t.Error("Expected error for nonexistent repository path")
	}
} 