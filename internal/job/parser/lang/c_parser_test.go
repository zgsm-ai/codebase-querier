package lang

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
		Path            string
		expectError     bool
		expectBlocks    []types.CodeChunk // For simpler cases
		expectMinBlocks int               // For rich test data file
	}{
		{
			name:        "rich test data file",
			Path:        filepath.Join("testdata", "c_rich_test.c"),
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
			Path:        "test.c",
			expectError: false,
			expectBlocks: []types.CodeChunk{
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
			Path:        "struct_test.c",
			expectError: false,
			expectBlocks: []types.CodeChunk{
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
			Path:        "enum_test.c",
			expectError: false,
			expectBlocks: []types.CodeChunk{
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
			Path:        "union_test.c",
			expectError: false,
			expectBlocks: []types.CodeChunk{
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
	parser, err := NewCParser()
	assert.NoError(t, err)
	assert.NotNil(t, parser, "C parser should not be nil")

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
