package lang

import (
	sitterkotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// KotlinProcessor implements LanguageProcessor for Kotlin
type KotlinProcessor struct {
	*BaseProcessor
}

// NewKotlinProcessor creates a new Kotlin processor
func NewKotlinProcessor() *KotlinProcessor {
	return &KotlinProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_declaration",
				"interface_declaration",
				"object_declaration",
				"function_declaration",
				"property_declaration",
				"type_alias",
				"enum_class",
				"companion_object",
				"secondary_constructor",
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

// ProcessMatch processes a match for Kotlin
func (p *KotlinProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(match, root, content, p.GetDefinitionKinds(), p.FindEnclosingType, p.FindEnclosingFunction)
}

// FindEnclosingType finds the enclosing type for a node
func (p *KotlinProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil && !current.IsMissing() {
		switch current.Kind() {
		case "class_declaration", "interface_declaration", "object_declaration", "enum_class":
			return current
		}
		current = current.Parent()
	}
	return nil
}

// FindEnclosingFunction finds the enclosing function for a node
func (p *KotlinProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	current := node
	for current != nil && !current.IsMissing() {
		switch current.Kind() {
		case "function_declaration", "secondary_constructor":
			return current
		}
		current = current.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Kotlin
func (p *KotlinProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetKotlinConfig returns the configuration for Kotlin language
func GetKotlinConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:           Kotlin,
		SitterLanguage:     sitter.NewLanguage(sitterkotlin.Language()),
		chunkQueryPath:     makeChunkQueryPath(Kotlin),
		structureQueryPath: makeStructureQueryPath(Kotlin),
		SupportedExts:      []string{".kt", ".kts"},
		Processor:          NewKotlinProcessor(),
	}
}
