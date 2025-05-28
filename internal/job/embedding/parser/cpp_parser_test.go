package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCPPParser(t *testing.T) {
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
			filePath:    filepath.Join("testdata", "cpp_rich_test.cpp"),
			expectError: false,
			// Based on a quick count of functions, methods, classes, structs.
			// Estimated blocks: 100 placeholder functions + global functions + methods + classes + structs = ~100+
			// Let's assert a minimum of 120 to allow for some flexibility.
			expectMinBlocks: 120,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `void myFunction() {
    std::cout << "Hello" << std::endl;
}
`,
			filePath:    "test.cpp",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "void myFunction() {\n    std::cout << \"Hello\" << std::endl;\n}",
					FilePath:    "test.cpp",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Global function
				},
			},
		},
		{
			name: "simple class and method",
			code: `class MyClass {
public:
    int myMethod() {
        return 1;
    }
};
`,
			filePath:    "class_test.cpp",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "class MyClass {\npublic:\n    int myMethod() {\n        return 1;\n    }\n};",
					FilePath:    "class_test.cpp",
					StartLine:   1,
					EndLine:     6,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "int myMethod() {\n        return 1;\n    }",
					FilePath:    "class_test.cpp",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "MyClass", // Method inside class
				},
			},
		},
		{
			name: "struct",
			code: `struct MyStruct {
    int field;
};
`,
			filePath:    "struct_test.cpp",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "struct MyStruct {\n    int field;\n};",
					FilePath:    "struct_test.cpp",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "MyStruct", // Struct definition is also a block, parentClass is its name
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewCPPParser(1000, 100)
	assert.NoError(t, err, "Failed to create C++ parser")
	assert.NotNil(t, parser, "C++ parser should not be nil")
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
