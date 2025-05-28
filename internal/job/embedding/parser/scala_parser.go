package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_scala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const scalaQueryFile = "queries/scala.scm"

type scalaParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	queryBytes        []byte // Store query content as bytes
}

func NewScalaParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_scala.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		fmt.Printf("Error setting Scala language for parser: %v\n", err)
		return nil
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", scalaQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading Scala query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	scalaParser := &scalaParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.Scala, scalaParser)
	return scalaParser
}

func (p *scalaParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
	lang := sitter.NewLanguage(sitter_scala.Language())
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
		var definitionNode *sitter.Node

		curr := capturedNode.Parent()
		for curr != nil && curr.Kind() != "class_definition" && curr.Kind() != "object_definition" && curr.Kind() != "trait_definition" && curr.Kind() != "method_declaration" && curr.Kind() != "function_definition" {
			curr = curr.Parent()
		}
		definitionNode = curr

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(code))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		switch definitionNode.Kind() {
		case "method_declaration", "function_definition":
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_definition", "object_definition", "trait_definition":
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

func (p *scalaParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .scala or .sc
func (p *scalaParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".scala" || ext == ".sc"
}
