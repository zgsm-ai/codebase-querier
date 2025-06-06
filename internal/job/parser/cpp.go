package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// CppProcessor implements LanguageProcessor for C++
type CppProcessor struct {
	*BaseProcessor
}

// NewCppProcessor creates a new C++ language processor
func NewCppProcessor() *CppProcessor {
	return &CppProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_specifier",
				"struct_specifier",
				"union_specifier",
				"function_definition",
				"declaration",
				"enum_specifier",
				"type_alias_declaration",
				"type_definition",
			},
			[]string{
				"class",
				"struct",
				"function",
				"variable",
				"enum",
				"type_alias",
			},
		),
	}
}

// ProcessMatch processes a match for C++ language
func (p *CppProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in C++
func (p *CppProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_specifier" || curr.Kind() == "struct_specifier" || curr.Kind() == "union_specifier" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in C++
func (p *CppProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for C++
func (p *CppProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetCppConfig returns the configuration for C++ language
func GetCppConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           CPP,
		SitterLanguage:     sitter.NewLanguage(sittercpp.Language()),
		chunkQueryPath:     makeChunkQueryPath(CPP),
		structureQueryPath: makeStructureQueryPath(CPP),
		SupportedExts:      []string{".cpp", ".cc", ".cxx", ".hpp"},
		Processor:          NewCppProcessor(),
	}
}
