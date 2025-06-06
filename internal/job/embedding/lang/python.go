package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

// PythonProcessor implements LanguageProcessor for Python code
type PythonProcessor struct {
	*BaseProcessor
}

// NewPythonProcessor creates a new Python language processor
func NewPythonProcessor() *PythonProcessor {
	return &PythonProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"class_definition",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Python
func (p *PythonProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Python
func (p *PythonProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Python
func (p *PythonProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" {
			return curr
		}
		// Stop at class definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_definition" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// GetPythonConfig returns the configuration for Python language
func GetPythonConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Python,
		SitterLanguage: sitter.NewLanguage(sitterpython.Language()),
		chunkQueryPath: makeChunkQueryPath(Python),
		SupportedExts:  []string{".py"},
		Processor:      NewPythonProcessor(),
	}
}
