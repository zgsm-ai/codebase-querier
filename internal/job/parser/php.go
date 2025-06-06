package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// PHPProcessor implements LanguageProcessor for PHP
type PHPProcessor struct {
	*BaseProcessor
}

// NewPHPProcessor creates a new PHP language processor
func NewPHPProcessor() *PHPProcessor {
	return &PHPProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_declaration",
				"interface_declaration",
				"trait_declaration",
				"function_definition",
				"method_declaration",
				"const_declaration",
				"property_declaration",
				"namespace_definition",
				"use_declaration",
				"enum_declaration",
			},
			[]string{
				"class",
				"interface",
				"function",
				"variable",
				"type_alias",
				"enum",
			},
		),
	}
}

// ProcessMatch processes a match for PHP language
func (p *PHPProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in PHP
func (p *PHPProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" || curr.Kind() == "trait_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in PHP
func (p *PHPProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" || curr.Kind() == "method_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for PHP
func (p *PHPProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetPhpConfig returns the configuration for PHP language
func GetPhpConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           PHP,
		SitterLanguage:     sitter.NewLanguage(sitterphp.LanguagePHP()),
		chunkQueryPath:     makeChunkQueryPath(PHP),
		structureQueryPath: makeStructureQueryPath(PHP),
		SupportedExts:      []string{".php", ".phtml"},
		Processor:          NewPHPProcessor(),
	}
}
