package scip

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestParseSCIPFileForGraph tests the SCIP index parsing function.
func TestParseSCIPFileForGraph(t *testing.T) {
	// Get the directory containing the test file.
	_, _, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get current file information")
	}
	//testFileDir := filepath.Dir(filename)

	// Assuming the test file is at internal/job/codegraph/scip/scip_parser_test.go,
	// the project root is four levels up.
	projectRoot := "/home/zk/projects/codebase-indexer/"

	// Construct the path to the generated index.scip file
	scipFilePath := filepath.Join(projectRoot, ".codebase_index", "index.scip")

	// Check if the scip file exists before attempting to parse
	if _, err := os.Stat(scipFilePath); os.IsNotExist(err) {
		t.Fatalf("SCIP index file not found at: %s. Please run SCIP index generation first.", scipFilePath)
	}

	fmt.Printf("Attempting to parse SCIP file: %s\n", scipFilePath)

	// Parse the SCIP file
	symbolNodes, err := ParseSCIPFileForGraph(scipFilePath)

	// Check for parsing errors
	if err != nil {
		t.Errorf("Failed to parse SCIP file: %v", err)
		return
	}

	// Basic check: ensure some symbol nodes were parsed
	if len(symbolNodes) == 0 {
		t.Errorf("Parsed graph contains no symbol nodes.")
	}

	fmt.Printf("Successfully parsed SCIP file. Found %d symbol nodes.\n", len(symbolNodes))

	// TODO: Add more specific assertions here to check the content of the parsed graph
	// For example, check for expected symbols, definitions, or relationships.
}
