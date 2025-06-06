package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// JavaScriptProcessor implements LanguageProcessor for JavaScript
type JavaScriptProcessor struct {
	*BaseProcessor
}

// NewJavaScriptProcessor creates a new JavaScript language processor
func NewJavaScriptProcessor() *JavaScriptProcessor {
	return &JavaScriptProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"function_declaration",
				"class_declaration",
				"method_definition",
				"variable_declarator",
				"interface_declaration",
				"type_alias_declaration",
				"enum_declaration",
			},
			[]string{
				"function",
				"class",
				"interface",
				"type_alias",
				"variable",
				"enum",
			},
		),
	}
}

// ProcessMatch processes a match for JavaScript language
func (p *JavaScriptProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node in JavaScript
func (p *JavaScriptProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node in JavaScript
func (p *JavaScriptProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_declaration" || curr.Kind() == "method_definition" {
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for JavaScript
func (p *JavaScriptProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetJavaScriptConfig returns the configuration for JavaScript language
func GetJavaScriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           JavaScript,
		SitterLanguage:     sitter.NewLanguage(sitterjavascript.Language()),
		chunkQueryPath:     makeChunkQueryPath(JavaScript),
		structureQueryPath: makeStructureQueryPath(JavaScript),
		SupportedExts:      []string{".js", ".jsx"},
		Processor:          NewJavaScriptProcessor(),
	}
}

// GetTypeScriptConfig returns the configuration for TypeScript language
func GetTypeScriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           TypeScript,
		SitterLanguage:     sitter.NewLanguage(sittertypescript.LanguageTypescript()),
		chunkQueryPath:     makeChunkQueryPath(TypeScript),
		structureQueryPath: makeStructureQueryPath(TypeScript),
		SupportedExts:      []string{".ts"},
		Processor:          NewJavaScriptProcessor(),
	}
}

// GetTSXConfig returns the configuration for TSX language
func GetTSXConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           TSX,
		SitterLanguage:     sitter.NewLanguage(sittertypescript.LanguageTSX()),
		chunkQueryPath:     makeChunkQueryPath(TSX),
		structureQueryPath: makeStructureQueryPath(TSX),
		SupportedExts:      []string{".tsx"},
		Processor:          NewJavaScriptProcessor(),
	}
}
