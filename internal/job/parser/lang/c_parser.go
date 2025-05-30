package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// 定义常量
const (
	cQueryFile      = "queries/c.scm"
	structSpecifier = "struct_specifier"
	unionSpecifier  = "union_specifier"
	enumSpecifier   = "enum_specifier"
	cNameField      = "name"
)

type cParser struct {
	parser   *sitter.Parser
	language *sitter.Language
	query    string
}

func NewCParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_c.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting C language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, cQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading C query: %w", err)
	}

	return &cParser{
		parser:   parser,
		language: lang,
		query:    queryStr,
	}, nil
}

func (p *cParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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

		if len(match.Captures) == 0 {
			continue
		}

		declarationNode := match.Captures[0].Node

		content := declarationNode.Utf8Text([]byte(codeFile.Content))

		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		var parentFunc string
		var parentClass string

		switch declarationNode.Kind() {
		case structSpecifier, unionSpecifier, enumSpecifier:
			nameNode := declarationNode.ChildByFieldName(cNameField)
			if nameNode != nil {
				parentClass = nameNode.Utf8Text([]byte(codeFile.Content))
			}
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

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *cParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// Parse the code file into a tree-sitter node.
func (p *cParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
