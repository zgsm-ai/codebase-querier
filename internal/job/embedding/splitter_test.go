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
	splitter, err := NewCodeSplitter(context.Background(), 1000, 100)
	assert.NoError(t, err)
	assert.NotNil(t, splitter)

	// Split the code
	chunks, err := splitter.Split(codeFile)
	assert.NoError(t, err)
	assert.NotNil(t, chunks)

	assert.Len(t, chunks, 2)

	assert.Contains(t, chunks[0].Content, "func main")
	assert.Equal(t, 5, chunks[0].StartLine) // Assuming main starts on line 5
	// assert.Equal(t, "main", chunks[0].Language) // Assuming Language field is populated in DefinitionNodeInfo

	assert.Contains(t, chunks[1].Content, "func anotherFunc")
	assert.Equal(t, 9, chunks[1].StartLine) // Assuming anotherFunc starts on line 10
	// assert.Equal(t, "anotherFunc", chunks[1].Language)

}
