package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const javaQueryFile = "queries/java.scm"

// Node kinds for Java
const (
	javaClassDeclaration     = "class_declaration"
	javaInterfaceDeclaration = "interface_declaration"
	javaEnumDeclaration      = "enum_declaration"
	javaMethodDeclaration    = "method_declaration"
)

// Field names for Java
const (
	javaNameField = "name"
)

type javaParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewJavaParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_java.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting Java language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, javaQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Java query: %w", err)
	}

	javaParser := &javaParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,     // Store language instance
		query:             queryStr, // Store query string
	}
	return javaParser, nil
}

func (p *javaParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
		var declarationNode *sitter.Node

		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == javaClassDeclaration || capturedNode.Parent().Kind() == javaInterfaceDeclaration || capturedNode.Parent().Kind() == javaEnumDeclaration || capturedNode.Parent().Kind() == javaMethodDeclaration) {
			declarationNode = capturedNode.Parent()
		} else if capturedNode.Kind() == javaClassDeclaration || capturedNode.Kind() == javaInterfaceDeclaration || capturedNode.Kind() == javaEnumDeclaration || capturedNode.Kind() == javaMethodDeclaration {
			declarationNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != javaClassDeclaration && curr.Kind() != javaInterfaceDeclaration && curr.Kind() != javaEnumDeclaration && curr.Kind() != javaMethodDeclaration {
				curr = curr.Parent()
			}
			declarationNode = curr
		}

		if declarationNode == nil {
			continue
		}

		content := declarationNode.Utf8Text([]byte(code))

		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		// Use constants for node kinds and field names
		if declarationNode.Kind() == javaMethodDeclaration {
			curr := declarationNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case javaClassDeclaration, javaInterfaceDeclaration, javaEnumDeclaration:
					nameNode := curr.ChildByFieldName(javaNameField)
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

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *javaParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .java
func (p *javaParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".java"
}
