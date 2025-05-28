package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const goQueryFile = "queries/go.scm"

// Node kinds for Go
const (
	goFunctionDeclaration = "function_declaration"
	goMethodDeclaration   = "method_declaration"
	goTypeDeclaration     = "type_declaration"
	goStructType          = "struct_type"
	goInterfaceType       = "interface_type"
)

// Field names for Go
const (
	goNameField     = "name"
	goTypeField     = "type"
	goReceiverField = "receiver"
)

type goParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewGoParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_go.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting Go language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, goQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Go query: %w", err)
	}

	goParser := &goParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return goParser, nil
}

func (p *goParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

func (p *goParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil || p.language == nil || p.query == "" {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Use the stored language and query string
	query, queryErr := sitter.NewQuery(p.language, p.query)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to create query for %s: %v", filePath, queryErr)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, root, []byte(code))

	var blocks []types.CodeBlock

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		parentFunc := ""
		parentClass := ""

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Check if the captured node's parent is the declaration/definition (common for name nodes)
		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == goFunctionDeclaration || capturedNode.Parent().Kind() == goMethodDeclaration || capturedNode.Parent().Kind() == goTypeDeclaration) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == goFunctionDeclaration || capturedNode.Kind() == goMethodDeclaration || capturedNode.Kind() == goTypeDeclaration {
			// In case the query captured the node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != goFunctionDeclaration && curr.Kind() != goMethodDeclaration && curr.Kind() != goTypeDeclaration {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(code))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		// Determine parent class/interface for methods
		// Use constants for node kinds and field names
		if definitionNode.Kind() == goMethodDeclaration {
			// Traverse up from the method declaration to find the type declaration it belongs to.
			// The receiver field node's parent is the method_declaration.
			// The method_declaration is a child of the source_file or type_declaration.
			// So we need to go up to the parent and check if it's a type_declaration.
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case goTypeDeclaration: // Methods are associated with type declarations
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName(goNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
		}

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,
			ParentClass:  parentClass,
			OriginalSize: len(content),
			TokenCount:   0,
		})
	}

	return blocks, nil
}

// InferLanguage checks if the file extension is .go
func (p *goParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".go"
}
