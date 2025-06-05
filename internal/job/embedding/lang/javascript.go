package lang

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// JavaScriptProcessor implements LanguageProcessor for JavaScript/TypeScript code
type JavaScriptProcessor struct {
	*BaseProcessor
}

// NewJavaScriptProcessor creates a new JavaScript/TypeScript language processor
func NewJavaScriptProcessor() *JavaScriptProcessor {
	return &JavaScriptProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"function_declaration",
			"class_declaration",
			"method_definition",
			"interface_declaration",
			"enum_declaration",
			"type_alias_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "interface_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for JavaScript/TypeScript
func (p *JavaScriptProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "function_declaration", "arrow_function":
			return curr
		}
		// Stop at class/interface definition to correctly identify methods vs nested functions
		if curr.Kind() == "class_declaration" || curr.Kind() == "interface_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// GetJavaScriptConfig returns the configuration for JavaScript language
func GetJavaScriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       JavaScript,
		SitterLanguage: sitter.NewLanguage(sitterjavascript.Language()),
		queryPath:      makeQueryPath(JavaScript),
		SupportedExts:  []string{".js", ".jsx"},
		Processor:      NewJavaScriptProcessor(),
	}
}

// GetTypeScriptConfig returns the configuration for TypeScript language
func GetTypeScriptConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       TypeScript,
		SitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTypescript()),
		queryPath:      makeQueryPath(TypeScript),
		SupportedExts:  []string{".ts"},
		Processor:      NewJavaScriptProcessor(),
	}
}

// GetTSXConfig returns the configuration for TSX language
func GetTSXConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       TSX,
		SitterLanguage: sitter.NewLanguage(sittertypescript.LanguageTSX()),
		queryPath:      makeQueryPath(TSX),
		SupportedExts:  []string{".tsx"},
		Processor:      NewJavaScriptProcessor(),
	}
}
