package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_scala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const scalaQueryFile = "queries/scala.scm"

// Node kinds for Scala
const (
	scalaClassDeclaration  = "class_declaration"
	scalaObjectDeclaration = "object_declaration"
	scalaTraitDeclaration  = "trait_declaration"
	scalaDefDeclaration    = "def_declaration"
	scalaValDeclaration    = "val_declaration"
	scalaVarDeclaration    = "var_declaration"
	scalaTypeDeclaration   = "type_declaration"
)

// Field names for Scala
const (
	scalaNameField = "name"
)

type scalaParser struct {
	parser   *sitter.Parser   // Add Tree-sitter parser instance
	language *sitter.Language // Store language instance
	query    string           // Store query string
}

func NewScalaParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_scala.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting Scala language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, scalaQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Scala query: %w", err)
	}

	scalaParser := &scalaParser{
		parser:   parser,
		language: lang,
		query:    queryStr,
	}
	return scalaParser, nil
}

func (p *scalaParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
	if p.parser == nil || p.language == nil || p.query == "" {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(codeFile.Content), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Use the stored language and query string
	query, queryErr := sitter.NewQuery(p.language, p.query)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to create query for %s: %v", codeFile.Path, queryErr)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, root, []byte(codeFile.Content))

	var blocks []*types.CodeChunk

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		parentFunc := ""
		parentClass := ""

		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var declarationNode *sitter.Node

		// Check if the captured node's parent is the declaration (common for name nodes)
		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == scalaClassDeclaration || capturedNode.Parent().Kind() == scalaObjectDeclaration || capturedNode.Parent().Kind() == scalaTraitDeclaration || capturedNode.Parent().Kind() == scalaDefDeclaration || capturedNode.Parent().Kind() == scalaValDeclaration || capturedNode.Parent().Kind() == scalaVarDeclaration || capturedNode.Parent().Kind() == scalaTypeDeclaration) {
			declarationNode = capturedNode.Parent()
		} else if capturedNode.Kind() == scalaClassDeclaration || capturedNode.Kind() == scalaObjectDeclaration || capturedNode.Kind() == scalaTraitDeclaration || capturedNode.Kind() == scalaDefDeclaration || capturedNode.Kind() == scalaValDeclaration || capturedNode.Kind() == scalaVarDeclaration || capturedNode.Kind() == scalaTypeDeclaration {
			// In case the query captured the node directly
			declarationNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != scalaClassDeclaration && curr.Kind() != scalaObjectDeclaration && curr.Kind() != scalaTraitDeclaration && curr.Kind() != scalaDefDeclaration && curr.Kind() != scalaValDeclaration && curr.Kind() != scalaVarDeclaration && curr.Kind() != scalaTypeDeclaration {
				curr = curr.Parent()
			}
			declarationNode = curr
		}

		if declarationNode == nil {
			continue
		}

		content := declarationNode.Utf8Text([]byte(codeFile.Content))

		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		switch declarationNode.Kind() {
		case scalaClassDeclaration, scalaObjectDeclaration, scalaTraitDeclaration, scalaDefDeclaration, scalaValDeclaration, scalaVarDeclaration, scalaTypeDeclaration:
			curr := declarationNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case scalaClassDeclaration, scalaObjectDeclaration, scalaTraitDeclaration:
					nameNode := curr.ChildByFieldName(scalaNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(codeFile.Content))
					}
					goto foundParent
				}
				curr = curr.Parent()
			}
		foundParent:
		}

		// Check if the content size exceeds the maximum chunk size
		if len(content) > maxTokensPerChunk { // Using character count as a proxy for tokens
			// Split the large block using the sliding window approach
			subChunks := splitIntoChunks(content, codeFile.Path, int(startLine), int(endLine), parentFunc, parentClass, maxTokensPerChunk, overlapTokens)
			blocks = append(blocks, subChunks...)
		} else {
			// Add the whole block as a single chunk
			blocks = append(blocks, &types.CodeChunk{
				Content:      content,
				FilePath:     codeFile.Path,
				StartLine:    int(startLine),
				EndLine:      int(endLine),
				ParentFunc:   parentFunc,
				ParentClass:  parentClass,
				OriginalSize: len(content),
				TokenCount:   len(content), // Approximation
			})
		}
	}

	return blocks, nil
}

func (p *scalaParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// Parse the code file into a tree-sitter node.
func (p *scalaParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
