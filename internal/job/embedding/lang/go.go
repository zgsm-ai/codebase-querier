package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// GoProcessor implements LanguageProcessor for Go
type GoProcessor struct {
	*BaseProcessor
}

// NewGoProcessor creates a new Go language processor
func NewGoProcessor() *GoProcessor {
	return &GoProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"function_declaration",
				"method_declaration",
				"type_declaration",
				"const_declaration",
				"var_declaration",
			},
			[]string{
				"function",
				"struct",
				"interface",
				"type_alias",
				"variable",
			},
		),
	}
}

// ProcessMatch processes a match for Go language
func (p *GoProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in Go
func (p *GoProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "type_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in Go
func (p *GoProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_declaration" || curr.Kind() == "method_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Go
func (p *GoProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetGoConfig returns the configuration for Go language
func GetGoConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           Go,
		SitterLanguage:     sitter.NewLanguage(sittergo.Language()),
		chunkQueryPath:     makeChunkQueryPath(Go),
		structureQueryPath: makeStructureQueryPath(Go),
		SupportedExts:      []string{".go"},
		Processor:          NewGoProcessor(),
	}
}
