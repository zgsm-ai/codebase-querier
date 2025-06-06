package lang

import (
	"errors"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

const name = "name"

// Custom errors
var (
	ErrNoCaptures   = errors.New("no captures in match")
	ErrMissingNode  = errors.New("captured node is missing")
	ErrNoDefinition = errors.New("no definition node found")
	ErrInvalidNode  = errors.New("invalid node")
)

// LanguageProcessor defines the interface for language-specific AST processing
type LanguageProcessor interface {
	ProcessMatch(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error)
	GetDefinitionKinds() []string
	FindEnclosingType(node *sitter.Node) *sitter.Node
	FindEnclosingFunction(node *sitter.Node) *sitter.Node
}

// DefinitionNodeInfo holds information about a definition node
type DefinitionNodeInfo struct {
	Node        *sitter.Node
	Kind        string
	Name        string
	ParentClass string
	ParentFunc  string
}

// BaseProcessor provides common functionality for all language processors
type BaseProcessor struct {
	definitionKinds []string
}

// NewBaseProcessor creates a new base processor with object pooling
func NewBaseProcessor(definitionKinds []string) *BaseProcessor {
	return &BaseProcessor{
		definitionKinds: definitionKinds,
	}
}

// GetDefinitionKinds returns the list of definition kinds for this language
func (p *BaseProcessor) GetDefinitionKinds() []string {
	return p.definitionKinds
}

// CommonMatchProcessor provides shared functionality for processing matches
func (p *BaseProcessor) CommonMatchProcessor(
	match *sitter.QueryMatch,
	root *sitter.Node,
	content []byte,
	definitionKinds []string,
	findEnclosingType func(*sitter.Node) *sitter.Node,
	findEnclosingFunction func(*sitter.Node) *sitter.Node,
) ([]*DefinitionNodeInfo, error) {
	if len(match.Captures) == 0 || match.Captures[0].Node.IsMissing() {
		return nil, ErrNoCaptures
	}
	capturedNode := &match.Captures[0].Node
	definitionNode := p.findDefinitionNode(capturedNode, definitionKinds)
	if definitionNode == nil {
		return nil, ErrNoDefinition
	}
	if definitionNode.IsMissing() {
		return nil, ErrMissingNode
	}

	nodeDefInfo := &DefinitionNodeInfo{
		Node: definitionNode,
		Kind: getDefinitionKindFromNodeKind(definitionNode.Kind()),
	}

	// Extract name
	if nameNode := definitionNode.ChildByFieldName(name); nameNode != nil && !nameNode.IsMissing() {
		nodeDefInfo.Name = nameNode.Utf8Text(content)
	} else {
		nodeDefInfo.Name = definitionNode.Utf8Text(content)
	}

	// Find parent context
	if enclosingType := findEnclosingType(definitionNode); enclosingType != nil {
		if nameNode := enclosingType.ChildByFieldName(name); nameNode != nil {
			nodeDefInfo.ParentClass = nameNode.Utf8Text(content)
		}
	}

	if findEnclosingFunction != nil {
		if enclosingFunc := findEnclosingFunction(definitionNode); enclosingFunc != nil {
			if nameNode := enclosingFunc.ChildByFieldName(name); nameNode != nil {
				nodeDefInfo.ParentFunc = nameNode.Utf8Text(content)
			}
		}
	}

	return []*DefinitionNodeInfo{nodeDefInfo}, nil
}

// findDefinitionNode traverses up the AST to find a definition node of the specified kinds
func (p *BaseProcessor) findDefinitionNode(node *sitter.Node, kinds []string) *sitter.Node {
	curr := node
	for curr != nil && !curr.IsMissing() {
		for _, kind := range kinds {
			if curr.Kind() == kind {
				return curr
			}
		}
		curr = curr.Parent()
	}
	return nil
}

// getDefinitionKindFromNodeKind converts a node kind to a definition kind
func getDefinitionKindFromNodeKind(kind string) string {
	return kind
}

// FileStructureProcessor 处理文件结构查询的结果
type FileStructureProcessor struct {
	*BaseProcessor
}
