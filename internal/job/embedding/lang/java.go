package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// JavaProcessor implements LanguageProcessor for Java code
type JavaProcessor struct {
	*BaseProcessor
}

// NewJavaProcessor creates a new Java language processor
func NewJavaProcessor() *JavaProcessor {
	return &JavaProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"interface_declaration",
			"enum_declaration",
			"method_declaration",
			"constructor_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Java
func (p *JavaProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Java doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for Java
func (p *JavaProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Java
func (p *JavaProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Java doesn't support nested functions
	return nil
}

// GetJavaConfig returns the configuration for Java language
func GetJavaConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Java,
		SitterLanguage: sitter.NewLanguage(sitterjava.Language()),
		chunkQueryPath: makeChunkQueryPath(Java),
		SupportedExts:  []string{".java"},
		Processor:      NewJavaProcessor(),
	}
}
