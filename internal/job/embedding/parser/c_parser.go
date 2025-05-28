package parser

import (
	"errors"
	"fmt"
	"path/filepath"

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
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language
	query             string
}

func NewCParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
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

	cParser := &cParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return cParser, nil
}

func (p *cParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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

		if len(match.Captures) == 0 {
			continue
		}

		declarationNode := match.Captures[0].Node

		content := declarationNode.Utf8Text([]byte(code))

		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		var parentFunc string
		var parentClass string

		switch declarationNode.Kind() {
		case structSpecifier, unionSpecifier, enumSpecifier:
			nameNode := declarationNode.ChildByFieldName(cNameField)
			if nameNode != nil {
				parentClass = nameNode.Utf8Text([]byte(code))
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
			TokenCount:   0,
		})
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

// InferLanguage checks if the file extension is .c or .h
func (p *cParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".c" || ext == ".h"
}
