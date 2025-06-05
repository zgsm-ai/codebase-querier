package lang

import (
	sitterkotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// KotlinProcessor implements LanguageProcessor for Kotlin code
type KotlinProcessor struct {
	*BaseProcessor
}

// NewKotlinProcessor creates a new Kotlin language processor
func NewKotlinProcessor() *KotlinProcessor {
	return &KotlinProcessor{
		BaseProcessor: NewBaseProcessor([]string{
			"class_declaration",
			"object_declaration",
			"interface_declaration",
			"function_declaration",
			"property_declaration",
		}),
	}
}

// ProcessMatch implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "interface_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Kotlin
func (p *KotlinProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "function_declaration" {
			return curr
		}
		// Stop at class/object/interface definition
		if curr.Kind() == "class_declaration" || curr.Kind() == "object_declaration" || curr.Kind() == "interface_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// GetKotlinConfig returns the configuration for Kotlin language
func GetKotlinConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Kotlin,
		SitterLanguage: sitter.NewLanguage(sitterkotlin.Language()),
		queryPath:      makeQueryPath(Kotlin),
		SupportedExts:  []string{".kt", ".kts"},
		Processor:      NewKotlinProcessor(),
	}
}
