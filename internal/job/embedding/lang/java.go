package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// JavaProcessor implements LanguageProcessor for Java
type JavaProcessor struct {
	*BaseProcessor
}

// NewJavaProcessor creates a new Java language processor
func NewJavaProcessor() *JavaProcessor {
	return &JavaProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_declaration",
				"interface_declaration",
				"method_declaration",
				"constructor_declaration",
				"field_declaration",
				"enum_declaration",
				"type_parameter",
			},
			[]string{
				"class",
				"interface",
				"function",
				"variable",
				"enum",
				"type_alias",
			},
		),
	}
}

// ProcessMatch processes a match for Java language
func (p *JavaProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in Java
func (p *JavaProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in Java
func (p *JavaProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "method_declaration" || curr.Kind() == "constructor_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Java
func (p *JavaProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetJavaConfig returns the configuration for Java language
func GetJavaConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           Java,
		SitterLanguage:     sitter.NewLanguage(sitterjava.Language()),
		chunkQueryPath:     makeChunkQueryPath(Java),
		structureQueryPath: makeStructureQueryPath(Java),
		SupportedExts:      []string{".java"},
		Processor:          NewJavaProcessor(),
	}
}
