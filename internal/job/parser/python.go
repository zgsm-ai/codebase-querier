package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// PythonProcessor implements LanguageProcessor for Python
type PythonProcessor struct {
	*BaseProcessor
}

// NewPythonProcessor creates a new Python language processor
func NewPythonProcessor() *PythonProcessor {
	return &PythonProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"function_definition",
				"class_definition",
				"assignment",
			},
			[]string{
				"function",
				"class",
				"type_alias",
				"variable",
				"enum",
			},
		),
	}
}

// ProcessMatch processes a match for Python language
func (p *PythonProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in Python
func (p *PythonProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in Python
func (p *PythonProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Python
func (p *PythonProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetPythonConfig returns the configuration for Python language
func GetPythonConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Python,
		SitterLanguage: sitter.NewLanguage(sitterpython.Language()),

		structureQueryPath: makeStructureQueryPath(Python),
		SupportedExts:      []string{".py"},
		Processor:          NewPythonProcessor(),
	}
}
