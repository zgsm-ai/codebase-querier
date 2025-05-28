package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter_kotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const kotlinQueryFile = "queries/kotlin.scm"

type kotlinParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	queryBytes        []byte // Store query content as bytes
}

func NewKotlinParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	language := sitter.NewLanguage(sitter_kotlin.Language())
	err := parser.SetLanguage(language)
	if err != nil {
		fmt.Printf("Error setting Kotlin language for parser: %v\n", err)
		return nil
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", javaScriptQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading JavaScript query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	kotlinParser := &kotlinParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent,
	}
	registerParser(types.Kotlin, kotlinParser)
	return kotlinParser
}

func (p *kotlinParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	lang := sitter_kotlin.Language()
	query, queryErr := sitter.NewQuery(lang, string(p.queryBytes))
	if queryErr != nil {
		var qe *sitter.QueryError
		if errors.As(queryErr, &qe) {
			return nil, fmt.Errorf("failed to create query: %w at offset %d, kind %s", queryErr, qe.Offset, qe.Kind)
		} else {
			return nil, fmt.Errorf("failed to create query: %w", queryErr)
		}
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Exec(query, root)

	var blocks []types.CodeBlock

	for {
		match, ok := matches.NextMatch()
		if !ok {
			break
		}

		parentFunc := ""
		parentClass := ""

		var definitionNode *sitter.Node
		for _, capture := range match.Captures {
			if query.CaptureNameForId(capture.Index) == "name" {
				curr := capture.Node.Parent()
				for curr != nil {
					switch curr.Kind() {
					case "class_declaration", "object_declaration", "interface_declaration", "function_declaration", "property_declaration":
						definitionNode = curr
						goto foundDefinition
					}
					curr = curr.Parent()
				}
				switch capture.Node.Kind() {
				case "class_declaration", "object_declaration", "interface_declaration", "function_declaration", "property_declaration":
					definitionNode = capture.Node
					goto foundDefinition
				}
			}
		}

	foundDefinition:

		if definitionNode == nil {
			continue
		}

		contentBytes := definitionNode.Content([]byte(code))
		content := string(contentBytes)

		startLine := definitionNode.StartPoint().Row + 1
		endLine := definitionNode.EndPoint().Row + 1

		switch definitionNode.Kind() {
		case "function_declaration", "property_declaration":
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_declaration", "object_declaration", "interface_declaration":
					nameNode := curr.ChildByFieldName("name")
					if nameNode != nil {
						parentClassBytes := nameNode.Content([]byte(code))
						parentClass = string(parentClassBytes)
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

func (p *kotlinParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .kt or .kts
func (p *kotlinParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".kt" || ext == ".kts"
}
