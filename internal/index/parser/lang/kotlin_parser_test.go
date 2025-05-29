package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestKotlinParser(t *testing.T) {
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
			name:        "rich test data file",
			Path:        filepath.Join("testdata", "kotlin_rich_test.kt"),
			expectError: false,
			// Based on a quick count of functions, classes, data classes, objects, interfaces.
			// The kotlin_rich_test.kt file has helloWorld, add, User (class), Product (data class), Constants (object), MyInterface, MyClass (implements), doSomething (method), doSomethingElse (method), exploreHashMap, checkNumber, simpleWhile, simpleFor, describe (when), documentedFunction, processList, filterEvenNumbers, complexLogic, Point (class), distanceFromOrigin (method), MathResult (sealed class), Success (data class), Error (data class), safeDivision, safeSqrt, placeholderFunction1-100, finalKotlinFunction.
			// That's 26 standard + 100 placeholders = 126 functions/methods/types/objects/interfaces.
			// Let's assert a minimum of 120.
			expectMinBlocks: 120,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `fun myFunction() {
    println("Hello")
}
`,
			Path:        "test.kt",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "fun myFunction() {\n    println(\"Hello\")\n}",
					FilePath:    "test.kt",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
		{
			name: "class and method",
			code: `class MyClass {
    fun myMethod(): Int {
        return 1
    }
}
`,
			Path:        "class_test.kt",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "class MyClass {\n    fun myMethod(): Int {\n        return 1\n    }\n}",
					FilePath:    "class_test.kt",
					StartLine:   1,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "fun myMethod(): Int {\n        return 1\n    }",
					FilePath:    "class_test.kt",
					StartLine:   2,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyClass",
				},
			},
		},
		{
			name: "data class and object",
			code: `data class MyData(val name: String)

object MyObject {
    const val VERSION = 1
}
`,
			Path:        "data_object_test.kt",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "data class MyData(val name: String)",
					FilePath:    "data_object_test.kt",
					StartLine:   1,
					EndLine:     1,
					ParentFunc:  "",
					ParentClass: "", // Data classes are top level
				},
				{
					Content:     "object MyObject {\n    const val VERSION = 1\n}",
					FilePath:    "data_object_test.kt",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Objects are top level
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := parser.NewKotlinParser()
	assert.NoError(t, err, "Failed to create Kotlin parser")
	assert.NotNil(t, parser, "Kotlin parser should not be nil")
	defer parser.Close()

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
