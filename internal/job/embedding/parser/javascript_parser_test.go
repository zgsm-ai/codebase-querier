package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestJavaScriptParser(t *testing.T) {
	// Define test cases
	tests := []struct {
		name            string
		code            string
		filePath        string
		expectError     bool
		expectBlocks    []types.CodeBlock // For simpler cases
		expectMinBlocks int               // For rich test data file
	}{
		{
			name:        "rich test data file",
			filePath:    filepath.Join("testdata", "javascript_rich_test.js"),
			expectError: false,
			// Based on a quick count of functions, classes, and methods.
			// Estimated blocks based on the structure of javascript_rich_test.js.
			// Includes: helloWorld, add, User (class), constructor, getUsername (method), MyClass, myMethod (method), exploreArray, checkNumber, simpleWhile, simpleFor, documentedFunction, processArray, filterEvenNumbers, complexLogic, Point (class), constructor, distanceFromOrigin (method), placeholderFunctionJs1-100, finalJavascriptFunction.
			// Total: 19 standard + 100 placeholders = 119.
			// Let's assert a minimum of 110 to allow for some flexibility.
			expectMinBlocks: 110,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `function myFunction() {
  console.log("Hello");
}
`,
			filePath:    "test.js",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "function myFunction() {\n  console.log(\"Hello\");\n}",
					FilePath:    "test.js",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Top level function
				},
			},
		},
		{
			name: "simple class and method",
			code: `class MyClass {
  myMethod() {
    return 1;
  }
}
`,
			filePath:    "class_test.js",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "class MyClass {\n  myMethod() {\n    return 1;\n  }\n}",
					FilePath:    "class_test.js",
					StartLine:   1,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "myMethod() {\n    return 1;\n  }",
					FilePath:    "class_test.js",
					StartLine:   2,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method inside class
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewJavaScriptParser(1000, 100)
	assert.NoError(t, err, "Failed to create JavaScript parser")
	assert.NotNil(t, parser, "JavaScript parser should not be nil")
	defer parser.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte{}
			if tt.code != "" {
				code = []byte(tt.code)
			} else if tt.filePath != "" {
				// Load code from file for the rich test case
				fileCode, readErr := os.ReadFile(tt.filePath)
				assert.NoError(t, readErr, "Failed to read test data file")
				code = fileCode
			} else {
				assert.FailNow(t, "Test case must provide either code or filePath")
			}

			blocks, parseErr := parser.Parse(string(code), tt.filePath)

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
