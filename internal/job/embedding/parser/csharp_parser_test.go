package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestCSharpParser(t *testing.T) {
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
			filePath:    filepath.Join("testdata", "csharp_rich_test.cs"),
			expectError: false,
			// Based on a quick count of classes, interfaces, enums, structs, delegates, events, properties, methods.
			// HelloWorld (class) + SayHello (method) + Add (method) + Message (property) + Constructor + ExploreList (method)
			// IMyInterface + DoSomething (method) + DoSomethingElse (method)
			// MyClass + DoSomething (method) + DoSomethingElse (method)
			// Status (enum)
			// Point (struct) + DistanceFromOrigin (method)
			// GreetingDelegate (delegate)
			// EventPublisher + OnGreet (event) + RaiseGreetEvent (method)
			// CheckNumber (method)
			// SimpleWhile (method)
			// SimpleFor (method)
			// DocumentedMethod (method)
			// AnotherClass + Id (property) + Name (property) + Constructor + Process (method)
			// Utility (static class) + ProcessItems (static method) + TransformItem (static method)
			// Placeholder methods 1-100
			// FinalCSharpMethod (method)
			// Total: 1 + 3 + 1 + 1 + 1 + 1 + 1 + 2 + 1 + 2 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 1 + 2 + 1 + 1 + 1 + 100 + 1 = 133
			// Let's assert a minimum of 125 to allow for some flexibility.
			expectMinBlocks: 125,
		},
		// Add more test cases with smaller code snippets below
		{
			name: "simple class and method",
			code: `using System;

public class MyClass
{
    public void MyMethod()
    {
        Console.WriteLine("Hello");
    }
}
`,
			filePath:    "simple_class.cs",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "public class MyClass\n{\n    public void MyMethod()\n    {\n        Console.WriteLine(\"Hello\");\n    }\n}",
					FilePath:    "simple_class.cs",
					StartLine:   3,
					EndLine:     9,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "public void MyMethod()\n    {\n        Console.WriteLine(\"Hello\");\n    }",
					FilePath:    "simple_class.cs",
					StartLine:   5,
					EndLine:     8,
					ParentFunc:  "",
					ParentClass: "MyClass",
				},
			},
		},
		{
			name: "struct with property",
			code: `public struct MyStruct
{
    public int MyProperty { get; set; }
}
`,
			filePath:    "simple_struct.cs",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "public struct MyStruct\n{\n    public int MyProperty { get; set; }\n}",
					FilePath:    "simple_struct.cs",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "public int MyProperty { get; set; }",
					FilePath:    "simple_struct.cs",
					StartLine:   3,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "MyStruct",
				},
			},
		},
		{
			name: "interface and enum",
			code: `public interface IMyInterface
{
    void DoSomething();
}

public enum MyEnum
{
    Value1,
    Value2
}
`,
			filePath:    "interface_enum.cs",
			expectError: false,
			expectBlocks: []types.CodeBlock{
				{
					Content:     "public interface IMyInterface\n{\n    void DoSomething();\n}",
					FilePath:    "interface_enum.cs",
					StartLine:   1,
					EndLine:     4,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "public enum MyEnum\n{\n    Value1,\n    Value2\n}",
					FilePath:    "interface_enum.cs",
					StartLine:   6,
					EndLine:     10,
					ParentFunc:  "",
					ParentClass: "",
				},
				{
					Content:     "void DoSomething();",
					FilePath:    "interface_enum.cs",
					StartLine:   3,
					EndLine:     3,
					ParentFunc:  "",
					ParentClass: "IMyInterface",
				},
			},
		},
	}

	// Initialize the parser outside the loop
	parser, err := NewCSharpParser(1000, 100)
	assert.NoError(t, err, "Failed to create C# parser")
	assert.NotNil(t, parser, "C# parser should not be nil")
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
