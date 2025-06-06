package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// RustProcessor implements LanguageProcessor for Rust
type RustProcessor struct {
	*BaseProcessor
}

// NewRustProcessor creates a new Rust language processor
func NewRustProcessor() *RustProcessor {
	return &RustProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"struct_item",
				"enum_item",
				"function_item",
				"impl_item",
				"trait_item",
				"type_item",
				"const_item",
				"static_item",
				"mod_item",
				"macro_definition",
			},
			[]string{
				"struct",
				"enum",
				"function",
				"interface",
				"type_alias",
				"variable",
				"class",
			},
		),
	}
}

// ProcessMatch processes a match for Rust language
func (p *RustProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in Rust
func (p *RustProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "struct_item" || curr.Kind() == "enum_item" || curr.Kind() == "trait_item" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in Rust
func (p *RustProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_item" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Rust
func (p *RustProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetRustConfig returns the configuration for Rust language
func GetRustConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           Rust,
		SitterLanguage:     sitter.NewLanguage(sitterrust.Language()),
		chunkQueryPath:     makeChunkQueryPath(Rust),
		structureQueryPath: makeStructureQueryPath(Rust),
		SupportedExts:      []string{".rs"},
		Processor:          NewRustProcessor(),
	}
}
