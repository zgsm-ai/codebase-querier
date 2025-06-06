package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
)

// PhpProcessor implements LanguageProcessor for PHP code
type PhpProcessor struct {
	*BaseProcessor
}

// NewPhpProcessor creates a new PHP language processor
func NewPhpProcessor() *PhpProcessor {
	return &PhpProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"interface_declaration",
			"trait_declaration",
			"function_definition",
			"method_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for PHP
func (p *PhpProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // PHP doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for PHP
func (p *PhpProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for PHP
func (p *PhpProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// PHP doesn't support nested functions in the same way as Python
	return nil
}

// GetPhpConfig returns the configuration for PHP language
func GetPhpConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       PHP,
		SitterLanguage: sitter.NewLanguage(sitterphp.LanguagePHP()),
		chunkQueryPath: makeChunkQueryPath(PHP),
		SupportedExts:  []string{".php", ".phtml"},
		Processor:      NewPhpProcessor(),
	}
}
