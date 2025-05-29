package lang

import (
	"errors"
	"fmt"

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
	parser   *sitter.Parser
	language *sitter.Language // Store language instance
	query    string           // Store query string
}

func NewRustParser() (CodeParser, error) {
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
		parser:   parser,
		language: lang,
		query:    queryStr,
	}
	return rustParser, nil
}

func (p *rustParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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

		content := itemNode.Utf8Text([]byte(codeFile.Content))

		startLine := itemNode.StartPosition().Row + 1
		endLine := itemNode.EndPosition().Row + 1

		// Use constants for node kinds and field names
		if itemNode.Kind() == rustFunctionItem {
			curr := itemNode.Parent()
			for curr != nil {
				if curr.Kind() == rustStructItem {
					nameNode := curr.ChildByFieldName(rustNameField)
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

func (p *rustParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// Parse the code file into a tree-sitter node.
func (p *rustParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
