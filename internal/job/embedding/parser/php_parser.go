package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type phpParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
}

func NewPhpParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_php.LanguagePHP())
	err := parser.SetLanguage(lang)
	if err != nil {
		fmt.Printf("Error setting PHP language for parser: %v\n", err)
		return nil
	}

	phpParser := &phpParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
	}
	registerParser(types.PHP, phpParser)
	return phpParser
}

func (p *phpParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	queryStr := `
		(class_declaration name: (name) @name)
		(interface_declaration name: (name) @name)
		(trait_declaration name: (name) @name)
		(function_definition name: (name) @name)
	`

	lang := sitter.NewLanguage(sitter_php.LanguagePHP())
	query, queryErr := sitter.NewQuery(lang, queryStr)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to create query at offset %d, kind %d: %s", queryErr.Offset, queryErr.Kind, queryErr.Message)
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

		curr := capturedNode.Parent()
		for curr != nil && curr.Kind() != "class_declaration" && curr.Kind() != "interface_declaration" && curr.Kind() != "trait_declaration" && curr.Kind() != "function_definition" {
			curr = curr.Parent()
		}
		definitionNode = curr

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(code))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		if definitionNode.Kind() == "function_definition" {
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_declaration", "trait_declaration":
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

func (p *phpParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .php or .phtml
func (p *phpParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".php" || ext == ".phtml"
}
