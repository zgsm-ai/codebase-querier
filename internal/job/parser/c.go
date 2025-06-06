package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// CProcessor implements LanguageProcessor for C
type CProcessor struct {
	*BaseProcessor
}

// NewCProcessor creates a new C language processor
func NewCProcessor() *CProcessor {
	return &CProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"function_definition",
				"struct_specifier",
				"union_specifier",
				"declaration",
				"enum_specifier",
				"type_definition",
			},
			[]string{
				"function",
				"struct",
				"variable",
				"enum",
				"type_alias",
			},
		),
	}
}

// ProcessMatch processes a match for C language
func (p *CProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in C
func (p *CProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "struct_specifier" || curr.Kind() == "union_specifier" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in C
func (p *CProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for C
func (p *CProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetCConfig returns the configuration for C language
func GetCConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           C,
		SitterLanguage:     sitter.NewLanguage(sitterc.Language()),
		chunkQueryPath:     makeChunkQueryPath(C),
		structureQueryPath: makeStructureQueryPath(C),
		SupportedExts:      []string{".c", ".h"},
		Processor:          NewCProcessor(),
	}
}
