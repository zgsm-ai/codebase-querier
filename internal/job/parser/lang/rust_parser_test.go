package lang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestRustParser(t *testing.T) {
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
			Path:        filepath.Join("testdata", "rust_rich_test.rs"),
			expectError: false,
			// Based on a quick count of functions and structs.
			// Estimated blocks: 100 placeholder functions + global functions + methods + structs.
			// Includes: main, hello_world, add, User (struct), new (method), get_username (method), Product (struct), MyTrait, do_something (trait method), MyStruct (implements trait), do_something (struct method), explore_vector, check_number, simple_while, simple_for, documented_function, process_list, filter_even_numbers, complex_logic, Point (struct), distance_from_origin (method), MathResult (enum), Success (variant), Error (variant), safe_division, safe_sqrt, placeholder_function_rust_1-100, final_rust_function.
			// Total: 18 standard + 100 placeholders = 118.
			// Let's assert a minimum of 110 to allow for some flexibility.
			expectMinBlocks: 110,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `fn my_function() {
    println!("Hello");
}
`,
			Path:        "test.rs",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "fn my_function() {\n    println!(\"Hello\");\n}",
					FilePath:    "test.rs",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "", // Top level function
				},
			},
		},
		{
			name: "simple struct and method",
			code: `struct MyStruct {
    field: i32,
}

impl MyStruct {
    fn my_method(&self) -> i32 {
        self.field
    }
}
`,
			Path:        "struct_test.rs",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "struct MyStruct {\n    field: i32,\n}",
					FilePath:    "struct_test.rs",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "MyStruct", // Struct definition is a block
				},
				{
					Content:     "fn my_method(&self) -> i32 {\n        self.field\n    }",
					FilePath:    "struct_test.rs",
					StartLine:   6,
					EndLine:     8,
					ParentFunc:  "",
					ParentClass: "MyStruct", // Method inside impl block for MyStruct
				},
			},
		},
		{
			name: "enum",
			code: `enum MyEnum {
    Value1,
    Value2,
}
`,
			Path:        "enum_test.rs",
			expectError: false,
			expectBlocks: []types.CodeChunk{
				{
					Content:     "enum MyEnum {\n    Value1,\n    Value2,\n}",
					FilePath:    "enum_test.rs",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyEnum", // Enum definition is a block
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewRustParser()
	assert.NoError(t, err, "Failed to create Rust parser")
	assert.NotNil(t, parser, "Rust parser should not be nil")
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
