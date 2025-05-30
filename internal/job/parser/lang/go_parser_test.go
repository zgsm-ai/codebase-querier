package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestGoParser(t *testing.T) {
	// Define test cases
	tests := []struct {
		name            string
		code            string
		Path            string
		expectError     bool
		expectBlocks    []types.CodeChunk // For simpler cases
		expectMinBlocks int               // For rich test data file
	}{
		{
			name:            "rich test data file",
			Path:            filepath.Join("testdata", "go_rich_test.go"),
			expectError:     false,
			expectMinBlocks: 115, // Adjusted minimum based on previous analysis
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `package main

func myFunction() {
	fmt.Println("Hello")
}
`,
			Path:        "test.go",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "func myFunction() {\n\tfmt.Println(\"Hello\")\n}",
					FilePath:    "test.go",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
		{
			name: "struct and method",
			code: `package main

type MyStruct struct {
	Field int
}

func (m MyStruct) MyMethod() int {
	return m.Field
}
`,
			Path:        "struct_test.go",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "type MyStruct struct {\n\tField int\n}",
					FilePath:    "struct_test.go",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "func (m MyStruct) MyMethod() int {\n\treturn m.Field\n}",
					FilePath:    "struct_test.go",
					StartLine:   7,
					EndLine:     9,
					ParentFunc:  "",
					ParentClass: "MyStruct",
				},
			},
		},
	}

	// Initialize the parser outside the loop for efficiency
	parser, err := NewGoParser()
	assert.NoError(t, err)
	// Using arbitrary values for maxTokensPerChunk and overlapTokens
	assert.NotNil(t, parser, "Go parser should not be nil")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte{}
			if tt.code != "" {
				code = []byte(tt.code)
			} else if tt.Path != "" {
				// Load code from file for the rich test case
				fileCode, readErr := os.ReadFile(tt.Path)
				assert.NoError(t, readErr, "Failed to read test data file")
				code = fileCode
			} else {
				assert.FailNow(t, "Test case must provide either code or codeFile.Path")
			}

			// Create a types.CodeFile object
			codeFile := &types.CodeFile{
				Path:    tt.Path,
				Content: string(code),
				// Language and other fields can be zero values for this test
			}

			// Using arbitrary values for maxTokensPerChunk and overlapTokens for the test call
			maxTokensPerChunk := 1000
			overlapTokens := 100

			// Call the Split method with the new signature
			blocks, parseErr := parser.Split(codeFile, maxTokensPerChunk, overlapTokens)

			if tt.expectError {
				assert.Error(t, parseErr)
			} else {
				assert.NoError(t, parseErr)
				assert.NotEmpty(t, blocks)

				if tt.expectMinBlocks > 0 {
					assert.GreaterOrEqual(t, len(blocks), tt.expectMinBlocks, "Should find minimum number of code blocks")
				} else if len(tt.expectBlocks) > 0 {
					assert.Equal(t, len(tt.expectBlocks), len(blocks), "Should have the expected number of blocks")
					for _, expectedBlock := range tt.expectBlocks {
						// Find the corresponding block in the actual results and assert properties
						foundBlock := findBlockByContentSubstring(blocks, strings.TrimSpace(expectedBlock.Content)) // Use substring for flexibility
						assert.NotNil(t, foundBlock, "Expected block not found: %s", expectedBlock.Content)
						if foundBlock != nil {
							assert.Equal(t, expectedBlock.FilePath, foundBlock.FilePath)
							assert.Equal(t, expectedBlock.StartLine, foundBlock.StartLine)
							assert.Equal(t, expectedBlock.EndLine, foundBlock.EndLine)
							assert.Equal(t, expectedBlock.ParentFunc, foundBlock.ParentFunc)
							assert.Equal(t, expectedBlock.ParentClass, foundBlock.ParentClass)
							// Optional: assert.Equal(t, strings.TrimSpace(expectedBlock.Content), strings.TrimSpace(foundBlock.Content))
						}
					}
				}
			}
		})
	}
}
