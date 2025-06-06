package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
)

// CppProcessor implements LanguageProcessor for C++ code
type CppProcessor struct {
	*BaseProcessor
}

// NewCppProcessor creates a new C++ language processor
func NewCppProcessor() *CppProcessor {
	return &CppProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_definition",
			"class_specifier",
			"struct_specifier",
			"method_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for C++
func (p *CppProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		nil, // C++ doesn't have nested functions in the same way as Python
	)
}

// FindEnclosingType implements LanguageProcessor for C++
func (p *CppProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_specifier", "struct_specifier":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for C++
func (p *CppProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	// C++ doesn't support nested functions in the same way as Python
	return nil
}

// GetCppConfig returns the configuration for C++ language
func GetCppConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       CPP,
		SitterLanguage: sitter.NewLanguage(sittercpp.Language()),
		chunkQueryPath: makeChunkQueryPath(CPP),
		SupportedExts:  []string{".cpp", ".cc", ".cxx", ".hpp"},
		Processor:      NewCppProcessor(),
	}
}
