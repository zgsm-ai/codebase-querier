package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// GoProcessor implements LanguageProcessor for Go code
type GoProcessor struct {
	*BaseProcessor
}

// NewGoProcessor creates a new Go language processor
func NewGoProcessor() *GoProcessor {
	return &GoProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_declaration",
			"method_declaration",
			"type_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Go
func (p *GoProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Go doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for Go
func (p *GoProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "type_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Go
func (p *GoProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Go doesn't support nested functions
	return nil
}

// GetGoConfig returns the configuration for Go language
func GetGoConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Go,
		SitterLanguage: sitter.NewLanguage(sittergo.Language()),
		chunkQueryPath: makeChunkQueryPath(Go),
		SupportedExts:  []string{".go"},
		Processor:      NewGoProcessor(),
	}
}
