package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterc "github.com/tree-sitter/tree-sitter-c/bindings/go"
)

// CProcessor implements LanguageProcessor for C code
type CProcessor struct {
	*BaseProcessor
}

// NewCProcessor creates a new C language processor
func NewCProcessor() *CProcessor {
	return &CProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"struct_specifier",
			"enum_specifier",
			"union_specifier",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for C
func (p *CProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C doesn't have nested functions
	)
}

// FindEnclosingType implements LanguageProcessor for C
func (p *CProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "struct_specifier", "enum_specifier", "union_specifier":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C
func (p *CProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C doesn't support nested functions
	return nil
}

// GetCConfig returns the configuration for C language
func GetCConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       C,
		SitterLanguage: sitter.NewLanguage(sitterc.Language()),
		chunkQueryPath: makeChunkQueryPath(C),
		SupportedExts:  []string{".c", ".h"},
		Processor:      NewCProcessor(),
	}
}
