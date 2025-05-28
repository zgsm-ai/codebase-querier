package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCParser(t *testing.T) {
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
			filePath:    filepath.Join("testdata", "c_rich_test.c"),
			expectError: false,
			// Based on a quick count of functions, structs, enums, unions.
			// hello_world, add, User (struct), init_user, print_user_info, Status (enum), Data (union), modify_value, modify_value_ref, check_number, simple_while, simple_for, documented_function, process_array, manipulate_string, placeholder_function_c_1-100, main, final_c_function.
			// Total: 15 standard + 100 placeholders + main + final_c_function = 117
			// Let's assert a minimum of 110 to allow for some flexibility.
			expectMinBlocks: 110,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function",
			code: `void myFunction() {
    printf("Hello\n");
}
`,
			filePath:    "test.c",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "void myFunction() {\n    printf(\"Hello\\n\");\n}",
					FilePath:    "test.c",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
		{
			name: "struct definition",
			code: `struct MyStruct {
    int field;
};
`,
			filePath:    "struct_test.c",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "struct MyStruct {\n    int field;\n};",
					FilePath:    "struct_test.c",
					StartLine:   1,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "MyStruct",
				},
			},
		},
		{
			name: "enum definition",
			code: `enum MyEnum {
    VALUE1,
    VALUE2
};
`,
			filePath:    "enum_test.c",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "enum MyEnum {\n    VALUE1,\n    VALUE2\n};",
					FilePath:    "enum_test.c",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyEnum",
				},
			},
		},
		{
			name: "union definition",
			code: `union MyUnion {
    int i;
    float f;
};
`,
			filePath:    "union_test.c",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "union MyUnion {\n    int i;\n    float f;\n};",
					FilePath:    "union_test.c",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "MyUnion",
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewCParser(1000, 100)
	assert.NoError(t, err)
	assert.NotNil(t, parser, "C parser should not be nil")

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
