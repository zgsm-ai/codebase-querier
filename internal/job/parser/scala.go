package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterscala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
)

// ScalaProcessor implements LanguageProcessor for Scala code
type ScalaProcessor struct {
	*BaseProcessor
}

// NewScalaProcessor creates a new Scala language processor
func NewScalaProcessor() *ScalaProcessor {
	return &ScalaProcessor{
		BaseProcessor: NewBaseProcessor(
			[]string{
				"class_declaration",
				"object_declaration",
				"trait_declaration",
				"method_declaration",
				"type_alias",
				"enum_declaration",
			},
			[]string{
				"class",
				"interface",
				"function",
				"type_alias",
				"enum",
			},
		),
	}
}

// ProcessMatch implements LanguageProcessor for Scala
func (p *ScalaProcessor) ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error) {
	return p.CommonMatchProcessor(
		match,
		root,
		content,
		p.GetDefinitionKinds(),
		p.FindEnclosingType,
		p.FindEnclosingFunction,
	)
}

// FindEnclosingType implements LanguageProcessor for Scala
func (p *ScalaProcessor) FindEnclosingType(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		switch curr.Kind() {
		case "class_declaration", "object_declaration", "trait_declaration":
			return curr
		}
		curr = curr.Parent()
	}
	return nil
}

// FindEnclosingFunction implements LanguageProcessor for Scala
func (p *ScalaProcessor) FindEnclosingFunction(node *sitter.Node) *sitter.Node {
	curr := node.Parent()
	for curr != nil && !curr.IsMissing() {
		if curr.Kind() == "method_declaration" {
			return curr
		}
		// Stop at class/object/trait Definition
		if curr.Kind() == "class_declaration" || curr.Kind() == "object_declaration" || curr.Kind() == "trait_declaration" {
			return nil
		}
		curr = curr.Parent()
	}
	return nil
}

// ProcessStructureMatch processes a structure match for Scala
func (p *ScalaProcessor) ProcessStructureMatch(match *sitter.QueryMatch, query *sitter.Query, root *sitter.Node, content []byte) (*codegraphpb.Definition, error) {
	return p.CommonStructureProcessor(match, query, root, content)
}

// GetScalaConfig returns the configuration for Scala language
func GetScalaConfig() *LanguageConfig {
	return &LanguageConfig{
		Language:       Scala,
		SitterLanguage: sitter.NewLanguage(sitterscala.Language()),

		structureQueryPath: makeStructureQueryPath(Scala),
		SupportedExts:      []string{".scala"},
		Processor:          NewScalaProcessor(),
	}
}
