package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
)

// RubyProcessor implements LanguageProcessor for Ruby code
type RubyProcessor struct {
	*BaseProcessor
}

// NewRubyProcessor creates a new Ruby language processor
func NewRubyProcessor() *RubyProcessor {
	return &RubyProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"method_declaration",
			"class_declaration",
			"module_declaration",
			"singleton_method",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Ruby
func (p *RubyProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Ruby doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for Ruby
func (p *RubyProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "module_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Ruby
func (p *RubyProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Ruby doesn't support nested functions in the same way as Python
	return nil
}

// GetRubyConfig returns the configuration for Ruby language
func GetRubyConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Ruby,
		SitterLanguage: sitter.NewLanguage(sitterruby.Language()),
		chunkQueryPath: makeChunkQueryPath(Ruby),
		SupportedExts:  []string{".rb"},
		Processor:      NewRubyProcessor(),
	}
}
