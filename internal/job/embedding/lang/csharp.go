package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittercsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
)

// CSharpProcessor implements LanguageProcessor for C# code
type CSharpProcessor struct {
	*BaseProcessor
}

// NewCSharpProcessor creates a new C# language processor
func NewCSharpProcessor() *CSharpProcessor {
	return &CSharpProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_declaration",
				"struct_declaration",
				"interface_declaration",
				"enum_declaration",
				"method_declaration",
				"property_declaration",
				"field_declaration",
				"event_declaration",
				"delegate_declaration",
			},
			[]string{
				"class",
				"struct",
				"interface",
				"function",
				"variable",
				"enum",
				"type_alias",
			},
		),
	}
}

// ProcessMatch implements LanguageProcessor for C#
func (p *CSharpProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C# doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for C#
func (p *CSharpProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C#
func (p *CSharpProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C# doesn't support nested functions in the same way as Python
	return nil
}

// ProcessStructureMatch processes a structure match for C#
func (p *CSharpProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetCSharpConfig returns the configuration for C# language
func GetCSharpConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           CSharp,
		SitterLanguage:     sitter.NewLanguage(sittercsharp.Language()),
		chunkQueryPath:     makeChunkQueryPath(CSharp),
		structureQueryPath: makeStructureQueryPath(CSharp),
		SupportedExts:      []string{".cs"},
		Processor:          NewCSharpProcessor(),
	}
}
