package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestTypeScriptParser(t *testing.T) {
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
			Path:        filepath.Join("testdata", "typescript_rich_test.ts"),
			expectError: false,
			// Based on a quick count of functions, classes, interfaces, enums, types.
			// helloWorld, add, User (interface), UserService (class), processUser (method), greet (method), IUserProcessor (interface), Status (enum), UserId (type), exploreMap, checkNumber, simpleWhile, simpleFor, documentedFunction, DataProcessor (interface), NumberProcessor (class), process (method), filterArray, transformArray, Point (class), constructor, distanceFromOrigin (method), MathResult (type), safeDivision, safeSqrt, placeholderFunctionTs1-100, finalTypescriptFunction.
			// Total: 25 standard + 100 placeholders = 125
			// Let's assert a minimum of 118 to allow for some flexibility.
			expectMinBlocks: 118,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `function myFunction(): void {
    console.log("Hello");
}
`,
			Path:        "test.ts",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "function myFunction(): void {\n    console.log(\"Hello\");\n}",
					FilePath:    "test.ts",
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
    myMethod(): number {
        return 1;
    }
}
`,
			Path:        "class_test.ts",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "class MyClass {\n    myMethod(): number {\n        return 1;\n    }\n}",
					FilePath:    "class_test.ts",
					StartLine:   1,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself doesn't have a parent class
				},
				{
					Content:     "myMethod(): number {\n        return 1;\n    }",
					FilePath:    "class_test.ts",
					StartLine:   2,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method is inside MyClass
				},
			},
		},
		{
			name: "interface",
			code: `interface MyInterface {
    prop: string;
    method(): void;
}
`,
			Path:        "interface_test.ts",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "interface MyInterface {\n    prop: string;\n    method(): void;\n}",
					FilePath:    "interface_test.ts",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "", // Interface definition doesn't have a parent class
				},
				{
					Content:     "method(): void;",
					FilePath:    "interface_test.ts",
					StartLine:   3,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "MyInterface", // Interface method is inside MyInterface
				},
				// Note: properties might or might not be parsed as blocks depending on query
			},
		},
		{
			name: "enum and type alias",
			code: `enum MyEnum {
    Value1,
    Value2
}

type MyType = string | number;
`,
			Path:        "enum_type_test.ts",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "enum MyEnum {\n    Value1,\n    Value2\n}",
					FilePath:    "enum_type_test.ts",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "type MyType = string | number;",
					FilePath:    "enum_type_test.ts",
					StartLine:   6,
					EndLine:     6,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewTypeScriptTSParser()
	assert.NoError(t, err)
	assert.NotNil(t, parser, "TypeScript parser should not be nil")

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
				assert.FailNow(t, "Test case must provide either code or codeFile.FullPath")
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
