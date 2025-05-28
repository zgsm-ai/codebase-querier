package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const javaScriptQueryFile = "queries/javascript.scm"

type javaScriptParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser
	language          *sitter.Language
	query             string // Store query content as bytes
}

func NewJavaScriptParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_javascript.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil
	}

	// Read the query file
	queryContent, err := os.ReadFile(javaScriptQueryFile)
	if err != nil {
		return nil
	}

	javaScriptParser := &javaScriptParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		language:          lang,
		parser:            parser,
		query:             string(queryContent),
	}
	registerParser(types.JavaScript, javaScriptParser)
	return javaScriptParser
}

func (p *javaScriptParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
	query, queryErr := sitter.NewQuery(p.language, string(p.query))
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

		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == "class_declaration" || capturedNode.Parent().Kind() == "function_declaration" || capturedNode.Parent().Kind() == "method_definition") {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == "class_declaration" || capturedNode.Kind() == "function_declaration" || capturedNode.Kind() == "method_definition" {
			definitionNode = &capturedNode
		} else {
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != "class_declaration" && curr.Kind() != "function_declaration" && curr.Kind() != "method_definition" {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(code))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		if definitionNode.Kind() == "method_definition" {
			curr := definitionNode.Parent()
			for curr != nil {
				if curr.Kind() == "class_declaration" {
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

func (p *javaScriptParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}

// InferLanguage checks if the file extension is .js or .jsx
func (p *javaScriptParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".js" || ext == ".jsx"
}
