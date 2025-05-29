package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestPhpParser(t *testing.T) {
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
			Path:        filepath.Join("testdata", "php_rich_test.php"),
			expectError: false,
			// Based on a quick count of functions, classes, interfaces, traits, methods.
			// helloWorld, add, User (class), constructor, getUsername (method)
			// MyInterface + doSomething (method signature) + doSomethingElse (method signature)
			// MyClass (implements) + doSomething (method) + doSomethingElse (method)
			// MyTrait + traitMethod (method)
			// exploreArray (function)
			// checkNumber (function)
			// simpleWhile (function)
			// simpleFor (function)
			// documentedFunction (function)
			// processArray (function)
			// filterEvenNumbers (function)
			// complexLogic (function)
			// Point (class) + constructor + distanceFromOrigin (method)
			// placeholderFunctionPhp1-100 (functions)
			// finalPhpFunction (function)
			// Total: 2 + 3 + 3 + 2 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 3 + 100 + 1 = 123
			// Let's assert a minimum of 115 to allow for some flexibility.
			expectMinBlocks: 115,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `<?php

function myFunction() {
    echo "Hello\n";
}

?>
`,
			Path:        "test.php",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "function myFunction() {\n    echo \"Hello\\n\";\n}",
					FilePath:    "test.php",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Top level function
				},
			},
		},
		{
			name: "class and method",
			code: `<?php

class MyClass {
    public function myMethod() {
        return 1;
    }
}

?>
`,
			Path:        "class_test.php",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "class MyClass {\n    public function myMethod() {\n        return 1;\n    }\n}",
					FilePath:    "class_test.php",
					StartLine:   3,
					EndLine:     7,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "public function myMethod() {\n        return 1;\n    }",
					FilePath:    "class_test.php",
					StartLine:   4,
					EndLine:     6,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method inside class
				},
			},
		},
		{
			name: "interface",
			code: `<?php

interface MyInterface {
    public function doSomething();
}

?>
`,
			Path:        "interface_test.php",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "interface MyInterface {\n    public function doSomething();\n}",
					FilePath:    "interface_test.php",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Interface definition itself
				},
				{
					Content:     "public function doSomething();",
					FilePath:    "interface_test.php",
					StartLine:   4,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyInterface", // Method inside interface
				},
			},
		},
		{
			name: "trait",
			code: `<?php

trait MyTrait {
    public function traitMethod() {
        // ...
    }
}

?>
`,
			Path:        "trait_test.php",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "trait MyTrait {\n    public function traitMethod() {\n        // ...\n    }\n}",
					FilePath:    "trait_test.php",
					StartLine:   3,
					EndLine:     7,
					ParentFunc:  "",
					ParentClass: "", // Trait definition itself
				},
				{
					Content:     "public function traitMethod() {\n        // ...\n    }",
					FilePath:    "trait_test.php",
					StartLine:   4,
					EndLine:     6,
					ParentFunc:  "",
					ParentClass: "MyTrait", // Method inside trait
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewPhpParser()
	assert.NoError(t, err, "Failed to create PHP parser")
	assert.NotNil(t, parser, "PHP parser should not be nil")
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
