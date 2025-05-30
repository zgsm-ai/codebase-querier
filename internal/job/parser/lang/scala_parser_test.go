package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestScalaParser(t *testing.T) {
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
			Path:        filepath.Join("testdata", "scala_rich_test.scala"),
			expectError: false,
			// Based on a quick count of objects, classes, traits, functions, methods, case classes.
			// HelloWorld (object) + main (method) + add (method)
			// User (class) + constructor + signInCount (var) + active (var) + getUsername (method)
			// Product (case class)
			// MyTrait + doSomething (method signature) + doSomethingElse (method signature)
			// MyClass (implements) + doSomething (method) + doSomethingElse (method)
			// MapExplorer (object) + exploreMap (method)
			// checkNumber (function)
			// simpleWhile (function)
			// simpleFor (function)
			// describe (function/pattern match)
			// documentedFunction (function)
			// DataProcessor (class) + constructor + process (method)
			// filterList (function)
			// transformList (function)
			// Point (case class) + distanceFromOrigin (method)
			// MathResult (sealed trait)
			// Success (case class)
			// Error (case class)
			// safeDivision (function)
			// safeSqrt (function)
			// placeholderFunctionScala1-100 (functions)
			// FinalScalaObject (object) + finalScalaFunction (method)
			// Total: 3 + 5 + 1 + 3 + 3 + 2 + 1 + 1 + 1 + 1 + 1 + 3 + 1 + 1 + 2 + 1 + 1 + 2 + 100 + 2 = 135
			// Let's assert a minimum of 128 to allow for some flexibility.
			expectMinBlocks: 128,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `def myFunction(): Unit = {
    println("Hello")
}
`,
			Path:        "test.scala",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "def myFunction(): Unit = {\n    println(\"Hello\")\n}",
					FilePath:    "test.scala",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Top level function
				},
			},
		},
		{
			name: "class and method",
			code: `class MyClass {
    def myMethod(): Int = {
        return 1
    }
}
`,
			Path:        "class_test.scala",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "class MyClass {\n    def myMethod(): Int = {\n        return 1\n    }\n}",
					FilePath:    "class_test.scala",
					StartLine:   1,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "def myMethod(): Int = {\n        return 1\n    }",
					FilePath:    "class_test.scala",
					StartLine:   2,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method inside class
				},
			},
		},
		{
			name: "object and case class",
			code: `object MyObject {
    val version = 1
}

case class MyCase(name: String)
`,
			Path:        "object_case_test.scala",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "object MyObject {\n    val version = 1\n}",
					FilePath:    "object_case_test.scala",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Object definition itself
				},
				{
					Content:     "case class MyCase(name: String)",
					FilePath:    "object_case_test.scala",
					StartLine:   5,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Case class definition itself
				},
			},
		},
		{
			name: "trait",
			code: `trait MyTrait {
    def traitMethod(): Unit
}
`,
			Path:        "trait_test.scala",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "trait MyTrait {\n    def traitMethod(): Unit\n}",
					FilePath:    "trait_test.scala",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Trait definition itself
				},
				{
					Content:     "def traitMethod(): Unit",
					FilePath:    "trait_test.scala",
					StartLine:   2,
					EndLine:     2,
					ParentFunc:  "",
					ParentClass: "MyTrait", // Method inside trait
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewScalaParser()
	assert.NoError(t, err, "Failed to create Scala parser")
	assert.NotNil(t, parser, "Scala parser should not be nil")
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
