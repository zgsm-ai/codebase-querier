package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const javaScriptQueryFile = "queries/javascript.scm"

// Node kinds
const (
	jsClassDeclaration     = "class_declaration"
	jsFunctionDeclaration  = "function_declaration"
	jsMethodDefinition     = "method_definition"
	jsInterfaceDeclaration = "interface_declaration" // Assuming JavaScript/TSX might use this from TS bindings
)

// Field names
const (
	jsNameField = "name"
)

type javaScriptParser struct {
	parser   *sitter.Parser
	language *sitter.Language
	query    string
}

func NewJavaScriptParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_javascript.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting JavaScript language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, javaScriptQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading JavaScript query: %w", err)
	}

	javaScriptParser := &javaScriptParser{
		parser:   parser,
		language: lang,
		query:    queryStr,
	}
	return javaScriptParser, nil
}

func (p *javaScriptParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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
			continue
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == jsClassDeclaration || capturedNode.Parent().Kind() == jsFunctionDeclaration || capturedNode.Parent().Kind() == jsMethodDefinition) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == jsClassDeclaration || capturedNode.Kind() == jsFunctionDeclaration || capturedNode.Kind() == jsMethodDefinition {
			definitionNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != jsClassDeclaration && curr.Kind() != jsFunctionDeclaration && curr.Kind() != jsMethodDefinition {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(codeFile.Content))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		if definitionNode.Kind() == jsMethodDefinition {
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case jsClassDeclaration: // Methods can be in classes
					nameNode := curr.ChildByFieldName(jsNameField)
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

func (p *javaScriptParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// Parse the code file into a tree-sitter node.
func (p *javaScriptParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
