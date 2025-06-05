package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
)

// RustProcessor implements LanguageProcessor for Rust code
type RustProcessor struct {
	*BaseProcessor
}

// NewRustProcessor creates a new Rust language processor
func NewRustProcessor() *RustProcessor {
	return &RustProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_item",
			"struct_item",
			"enum_item",
			"trait_item",
			"impl_item",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Rust
func (p *RustProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // Rust doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for Rust
func (p *RustProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "struct_item", "enum_item", "trait_item", "impl_item":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Rust
func (p *RustProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// Rust doesn't support nested functions in the same way as Python
	return nil
}

// GetRustConfig returns the configuration for Rust language
func GetRustConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Rust,
		SitterLanguage: sitter.NewLanguage(sitterrust.Language()),
		queryPath:      makeQueryPath(Rust),
		SupportedExts:  []string{".rs"},
		Processor:      NewRustProcessor(),
	}
}
