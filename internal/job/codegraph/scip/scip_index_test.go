package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSCIPIndexGeneration_GoProject tests the SCIP index generation for a Go project.
// It uses the current codebase as the test project.
func TestSCIPIndexGeneration_GoProject(t *testing.T) {
	// Get the directory containing the test file.
	_, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get current file information")
	}
	//testFileDir := filepath.Dir(filename)

	// Assuming the test file is at internal/job/codegraph/scip/scip_index_test.go,
	// the project root is four levels up.

	codebasePath := "/home/zk/projects/codebase-indexer"

	fmt.Printf("Testing with codebase path: %s\n", codebasePath)

	generator := &SCIPIndexGenerator{}

	// Run the index generation process
	indexPath, err := generator.Generate(context.Background(), codebasePath)

	// Check the results
	if err != nil {
		t.Errorf("SCIP index generation failed: %v", err)
		return
	}

	// Verify the output path
	expectedOutputPath := filepath.Join(codebasePath, ".codebase_index", "index.scip")
	if indexPath != expectedOutputPath {
		t.Errorf("Unexpected index path. Got: %s, Want: %s", indexPath, expectedOutputPath)
	}

	// Optional: Check if the file actually exists (requires the command to succeed and create the file)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("Generated index file does not exist at: %s", indexPath)
	}

	fmt.Printf("Successfully generated SCIP index at: %s\n", indexPath)
}
