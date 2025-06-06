package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// RubyProcessor implements LanguageProcessor for Ruby
type RubyProcessor struct {
	*BaseProcessor
}

// NewRubyProcessor creates a new Ruby language processor
func NewRubyProcessor() *RubyProcessor {
	return &RubyProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class",
				"module",
				"method",
				"singleton_method",
				"assignment",
			},
			[]string{
				"class",
				"function",
				"variable",
			},
		),
	}
}

// ProcessMatch processes a match for Ruby language
func (p *RubyProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in Ruby
func (p *RubyProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class" || curr.Kind() == "module" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in Ruby
func (p *RubyProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "method" || curr.Kind() == "singleton_method" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Ruby
func (p *RubyProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetRubyConfig returns the configuration for Ruby language
func GetRubyConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           Ruby,
		SitterLanguage:     sitter.NewLanguage(sitterruby.Language()),
		chunkQueryPath:     makeChunkQueryPath(Ruby),
		structureQueryPath: makeStructureQueryPath(Ruby),
		SupportedExts:      []string{".rb"},
		Processor:          NewRubyProcessor(),
	}
}
