package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestTypeScriptTSXParser(t *testing.T) {
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
			filePath:    filepath.Join("testdata", "typescript_tsx_rich_test.tsx"),
			expectError: false,
			// Based on a quick count of functions, classes, interfaces, enums, types, and JSX components.
			// Estimated blocks based on typescript_rich_test.ts structure + JSX components.
			// helloWorld, add, User (interface), UserService (class), processUser (method), greet (method), IUserProcessor (interface), Status (enum), UserId (type), exploreMap, checkNumber, simpleWhile, simpleFor, documentedFunction, DataProcessor (interface), NumberProcessor (class), process (method), filterArray, transformArray, Point (class), constructor, distanceFromOrigin (method), MathResult (type), safeDivision, safeSqrt, placeholderFunctionTsx1-100, FinalTypescriptTSXComponent (function component), AnotherComponent (class component), JsxExample (function component).
			// Total: 25 standard JS/TS + 100 placeholders + 3 React components = 128.
			// Let's assert a minimum of 120 to allow for some flexibility.
			expectMinBlocks: 120,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple function component",
			code: `import React from 'react';

const MyComponent: React.FC = () => {
  return <div>Hello, World!</div>;
};

export default MyComponent;
`,
			filePath:    "MyComponent.tsx",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "const MyComponent: React.FC = () => {\n  return <div>Hello, World!</div>;\n};",
					FilePath:    "MyComponent.tsx",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Top level variable declaration with function component
				},
				// Depending on query, the export might also be a block
				{
					Content:     "export default MyComponent;",
					FilePath:    "MyComponent.tsx",
					StartLine:   7,
					EndLine:     7,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
		{
			name: "simple class component",
			code: `import React from 'react';

interface MyProps {
  name: string;
}

class MyClassComponent extends React.Component<MyProps> {
  render() {
    return <div>Hello, {this.props.name}!</div>;
  }
}

export default MyClassComponent;
`,
			filePath:    "MyClassComponent.tsx",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "interface MyProps {\n  name: string;\n}",
					FilePath:    "MyClassComponent.tsx",
					StartLine:   3,
					EndLine:     5,
					ParentFunc:  "",
					ParentClass: "", // Top level interface
				},
				{
					Content:     "class MyClassComponent extends React.Component<MyProps> {\n  render() {\n    return <div>Hello, {this.props.name}!</div>;\n  }\n}",
					FilePath:    "MyClassComponent.tsx",
					StartLine:   7,
					EndLine:     11,
					ParentFunc:  "",
					ParentClass: "", // Class definition itself
				},
				{
					Content:     "render() {\n    return <div>Hello, {this.props.name}!</div>;\n  }",
					FilePath:    "MyClassComponent.tsx",
					StartLine:   8,
					EndLine:     10,
					ParentFunc:  "",
					ParentClass: "MyClassComponent", // Method inside class
				},
				// Export might be a block too
				{
					Content:     "export default MyClassComponent;",
					FilePath:    "MyClassComponent.tsx",
					StartLine:   13,
					EndLine:     13,
					ParentFunc:  "",
					ParentClass: "",
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewTypeScriptTSXParser(1000, 100)
	assert.NoError(t, err, "Failed to create TypeScript TSX parser")
	assert.NotNil(t, parser, "TypeScript TSX parser should not be nil")
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
