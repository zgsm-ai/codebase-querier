package embedding

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCodeSplitter_Split_Go(t *testing.T) {
	// Simple Go code with a function
	goCode := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}

func anotherFunc() int {
	return 1
}
`

	// Create a dummy CodeFile
	codeFile := &types.CodeFile{
		Path:    filepath.Join("testdata", "go", "simple.go"), // Use a realistic path for testing
		Content: goCode,
	}

	// Create a new CodeSplitter
	// Use reasonable default options for maxTokens and overlap
	splitter, err := NewCodeSplitter(context.Background(), WithMaxTokensPerChunk(1000), WithOverlapTokens(100))
	assert.NoError(t, err)
	assert.NotNil(t, splitter)

	// Split the code
	chunks, err := splitter.Split(codeFile)
	assert.NoError(t, err)
	assert.NotNil(t, chunks)

	// Assert that chunks were generated (should be at least the two functions)
	assert.True(t, len(chunks) > 0, "Expected chunks to be generated")

	// Optional: More specific assertions about the chunks can be added here
	// For example, check the number of chunks, their content, start/end lines, or parent info
	// assert.Len(t, chunks, 2) // Example assertion: expect 2 chunks for main and anotherFunc

	// Example assertions about specific chunks (adjust indices and content based on actual output)
	// if len(chunks) > 0 {
	//     assert.Contains(t, chunks[0].Content, "func main")
	//     assert.Equal(t, 5, chunks[0].StartLine) // Assuming main starts on line 5
	//     assert.Equal(t, "main", chunks[0].Name) // Assuming Name field is populated in DefinitionNodeInfo
	// }
	// if len(chunks) > 1 {
	//     assert.Contains(t, chunks[1].Content, "func anotherFunc")
	//     assert.Equal(t, 10, chunks[1].StartLine) // Assuming anotherFunc starts on line 10
	//     assert.Equal(t, "anotherFunc", chunks[1].Name)
	// }

	// Don't forget to close the splitter (which closes the registry and parsers)
	// Assuming the CodeSplitter has a Close method or provides access to close the registry
	// Based on current CodeSplitter implementation, it does not have a Close method.
	// The registry has a Close method, but CodeSplitter doesn't expose it.
	// For a proper test cleanup, CodeSplitter should have a Close method that calls registry.Close().
	// For now, we will skip explicit closing in the test to avoid introducing new code changes.
	// In a real application or more complete test, managing resource cleanup is important.

}
