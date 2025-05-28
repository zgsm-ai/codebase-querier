package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const goQueryFile = "queries/go.scm"

type goParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	queryBytes        []byte // Store query content as bytes
}

func NewGoParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_go.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		fmt.Printf("Error setting Go language for parser: %v\n", err)
		return nil
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", goQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading Go query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	goParser := &goParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.Go, goParser)
	return goParser
}

func (p *goParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

func (p *goParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Use the query content from the struct field
	lang := sitter.NewLanguage(sitter_go.Language())
	query, queryErr := sitter.NewQuery(lang, string(p.queryBytes))
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
		var declarationNode *sitter.Node

		if capturedNode.Kind() == "function_declaration" || capturedNode.Kind() == "method_declaration" {
			declarationNode = &capturedNode
		} else {
			declarationNode = capturedNode.Parent()
		}

		for declarationNode != nil && declarationNode.Kind() != "function_declaration" && declarationNode.Kind() != "method_declaration" {
			declarationNode = declarationNode.Parent()
		}

		if declarationNode == nil {
			continue
		}

		content := declarationNode.Utf8Text([]byte(code))

		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		if declarationNode.Kind() == "method_declaration" {
			receiverNode := declarationNode.Child(0)
			if receiverNode != nil {
				parentClass = receiverNode.Utf8Text([]byte(code))
			}
		}

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,
			ParentClass:  parentClass,
			OriginalSize: len(content),
		})
	}

	return blocks, nil
}

// InferLanguage checks if the file extension is .go
func (p *goParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".go"
}
