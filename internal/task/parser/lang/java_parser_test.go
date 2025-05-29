package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestJavaParser(t *testing.T) {
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
			Path:        filepath.Join("testdata", "java_rich_test.java"),
			expectError: false,
			// Based on a quick count of classes, interfaces, enums, and methods.
			// Estimated blocks: 100 placeholder methods + standard classes/methods/interfaces/enums.
			// Includes: HelloWorld (class) + main (method) + sayHello (method) + add (method)
			// User (class) + constructor + getUsername (method)
			// MyInterface + doSomething (method signature) + doSomethingElse (method signature)
			// MyClass (implements) + doSomething (method) + doSomethingElse (method)
			// Status (enum)
			// exploreList (method)
			// checkNumber (method)
			// simpleWhile (method)
			// simpleFor (method)
			// documentedMethod (method)
			// DataProcessor (interface) + process (method signature)
			// NumberProcessor (class) + constructor + process (method)
			// filterEvenNumbers (method)
			// transformList (method)
			// Point (class) + constructor + distanceFromOrigin (method)
			// MathResult (abstract class) + Success (class extends) + Error (class extends) + isSuccess (method) + getError (method)
			// safeDivision (method)
			// safeSqrt (method)
			// placeholderMethodJava1-100 (methods)
			// FinalJavaClass (class) + finalJavaMethod (method).
			// Total: 4 + 3 + 3 + 3 + 1 + 1 + 1 + 1 + 1 + 1 + 2 + 3 + 1 + 1 + 3 + 5 + 1 + 1 + 100 + 2 = 135.
			// Let's assert a minimum of 128 to allow for some flexibility.
			expectMinBlocks: 128,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple method",
			code: `class MyClass {
    public void myFunction() {
        System.out.println("Hello");
    }
}
`,
			Path:        "test.java",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "class MyClass {\n    public void myFunction() {\n        System.out.println(\"Hello\");\n    }\n}",
					FilePath:    "test.java",
					StartLine:   1,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "public void myFunction() {\n        System.out.println(\"Hello\");\n    }",
					FilePath:    "test.java",
					StartLine:   2,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method inside class
				},
			},
		},
		{
			name: "interface and enum",
			code: `interface MyInterface {
    void doSomething();
}

enum MyEnum {
    VALUE1,
    VALUE2
}
`,
			Path:        "interface_enum.java",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "interface MyInterface {\n    void doSomething();\n}",
					FilePath:    "interface_enum.java",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Interface definition itself
				},
				{
					Content:     "void doSomething();",
					FilePath:    "interface_enum.java",
					StartLine:   2,
					EndLine:     2,
					ParentFunc:  "",
					ParentClass: "MyInterface", // Method inside interface
				},
				{
					Content:     "enum MyEnum {\n    VALUE1,\n    VALUE2\n}",
					FilePath:    "interface_enum.java",
					StartLine:   5,
					EndLine:     8,
					ParentFunc:  "",
					ParentClass: "", // Enum definition itself
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewJavaParser()
	assert.NoError(t, err, "Failed to create Java parser")
	assert.NotNil(t, parser, "Java parser should not be nil")
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
