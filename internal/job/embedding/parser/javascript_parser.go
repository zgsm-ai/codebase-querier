package parser

import (
	"errors"
	"fmt"
	"path/filepath"

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
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language
	query             string
}

func NewJavaScriptParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
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
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return javaScriptParser, nil
}

func (p *javaScriptParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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

		content := definitionNode.Utf8Text([]byte(code))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		if definitionNode.Kind() == jsMethodDefinition {
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case jsClassDeclaration: // Methods can be in classes
					nameNode := curr.ChildByFieldName(jsNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent
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

func (p *javaScriptParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .js or .jsx
func (p *javaScriptParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".js" || ext == ".jsx"
}
