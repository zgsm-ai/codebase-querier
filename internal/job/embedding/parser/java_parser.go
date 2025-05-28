package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const javaQueryFile = "queries/java.scm"

type javaParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	queryBytes        []byte // Store query content as bytes
}

func NewJavaParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_java.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		fmt.Printf("Error setting Java language for parser: %v\n", err)
		return nil
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", javaQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading Java query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	javaParser := &javaParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.Java, javaParser)
	return javaParser
}

func (p *javaParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Use the query content from the struct field, converting to string
	query, queryErr := sitter.NewQuery(sitter.NewLanguage(sitter_java.Language()), string(p.queryBytes))
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

		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == "class_declaration" || capturedNode.Parent().Kind() == "interface_declaration" || capturedNode.Parent().Kind() == "enum_declaration" || capturedNode.Parent().Kind() == "method_declaration") {
			declarationNode = capturedNode.Parent()
		} else if capturedNode.Kind() == "class_declaration" || capturedNode.Kind() == "interface_declaration" || capturedNode.Kind() == "enum_declaration" || capturedNode.Kind() == "method_declaration" {
			declarationNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != "class_declaration" && curr.Kind() != "interface_declaration" && curr.Kind() != "enum_declaration" && curr.Kind() != "method_declaration" {
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

		if declarationNode.Kind() == "method_declaration" {
			curr := declarationNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_declaration", "interface_declaration", "enum_declaration":
					nameNode := curr.ChildByFieldName("name")
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
