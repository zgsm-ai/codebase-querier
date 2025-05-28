package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const rustQueryFile = "queries/rust.scm"

type rustParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	queryBytes        []byte // Store query content as bytes
}

func NewRustParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_rust.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		fmt.Printf("Error setting Rust language for parser: %v\n", err)
		return nil
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", rustQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading Rust query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	rustParser := &rustParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.Rust, rustParser)
	return rustParser
}

func (p *rustParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
	lang := sitter.NewLanguage(sitter_rust.Language())
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

		if len(match.Captures) == 0 {
			continue
		}

		capturedNode := match.Captures[0].Node
		var itemNode *sitter.Node

		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == "function_item" || capturedNode.Parent().Kind() == "struct_item") {
			itemNode = capturedNode.Parent()
		} else if capturedNode.Kind() == "function_item" || capturedNode.Kind() == "struct_item" {
			itemNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != "function_item" && curr.Kind() != "struct_item" {
				curr = curr.Parent()
			}
			itemNode = curr
		}

		if itemNode == nil {
			continue
		}

		content := itemNode.Utf8Text([]byte(code))

		startLine := itemNode.StartPosition().Row + 1
		endLine := itemNode.EndPosition().Row + 1

		if itemNode.Kind() == "function_item" {
			curr := itemNode.Parent()
			for curr != nil {
				if curr.Kind() == "struct_item" {
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

func (p *rustParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .rs
func (p *rustParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".rs"
}
