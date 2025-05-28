package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const rustQueryFile = "queries/rust.scm"

// Node kinds for Rust
const (
	rustFunctionItem = "function_item"
	rustStructItem   = "struct_item"
)

// Field names for Rust
const (
	rustNameField = "name"
)

type rustParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewRustParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_rust.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting Rust language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, rustQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Rust query: %w", err)
	}

	rustParser := &rustParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return rustParser, nil
}

func (p *rustParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
		var itemNode *sitter.Node

		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == rustFunctionItem || capturedNode.Parent().Kind() == rustStructItem) {
			itemNode = capturedNode.Parent()
		} else if capturedNode.Kind() == rustFunctionItem || capturedNode.Kind() == rustStructItem {
			itemNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != rustFunctionItem && curr.Kind() != rustStructItem {
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

		// Use constants for node kinds and field names
		if itemNode.Kind() == rustFunctionItem {
			curr := itemNode.Parent()
			for curr != nil {
				if curr.Kind() == rustStructItem {
					nameNode := curr.ChildByFieldName(rustNameField)
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
