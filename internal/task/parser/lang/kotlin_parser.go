package lang

import (
	"errors"
	"fmt"

	sitter_kotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const kotlinQueryFile = "queries/kotlin.scm"

// Node kinds for Kotlin
const (
	kotlinClassDeclaration  = "class_declaration"
	kotlinObjectDeclaration = "object_declaration"
	kotlinFunDeclaration    = "fun_declaration"
)

// Field names for Kotlin
const (
	kotlinNameField = "name"
)

type kotlinParser struct {
	maxTokensPerChunk int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewKotlinParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_kotlin.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting Kotlin language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, kotlinQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Kotlin query: %w", err)
	}

	kotlinParser := &kotlinParser{

		parser:   parser,
		language: lang,
		query:    queryStr,
	}

	return kotlinParser, nil
}

func (p *kotlinParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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

	matches := qc.Matches(query, root, []byte(codeFile.Content)) // Corrected from qc.Exec

	var blocks []*types.CodeChunk

	captureNames := query.CaptureNames() // Corrected from query.CaptureNameForId

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		parentFunc := ""
		parentClass := ""

		var definitionNode *sitter.Node
		for _, capture := range match.Captures {
			if capture.Index < uint32(len(captureNames)) && captureNames[capture.Index] == kotlinNameField { // Corrected capture name check
				curr := capture.Node.Parent()
				for curr != nil {
					switch curr.Kind() {
					case kotlinClassDeclaration, kotlinObjectDeclaration, kotlinFunDeclaration:
						definitionNode = curr
						goto foundDefinition
					}
					curr = curr.Parent()
				}
				switch capture.Node.Kind() {
				case kotlinClassDeclaration, kotlinObjectDeclaration, kotlinFunDeclaration:
					definitionNode = &capture.Node
					goto foundDefinition
				}
			}
		}

	foundDefinition:

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(codeFile.Content)) // Corrected from definitionNode.Content

		startPoint := definitionNode.StartPosition() // Corrected from definitionNode.StartPoint
		endPoint := definitionNode.EndPosition()     // Corrected from definitionNode.EndPoint()

		startLine := startPoint.Row + 1
		endLine := endPoint.Row + 1

		switch definitionNode.Kind() {
		case kotlinFunDeclaration:
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case kotlinClassDeclaration, kotlinObjectDeclaration:
					nameNode := curr.ChildByFieldName(kotlinNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(codeFile.Content)) // Corrected from nameNode.Content
					}
					goto foundParent
				}
				curr = curr.Parent()
			}
		foundParent:
		}

		blocks = append(blocks, &types.CodeChunk{
			Content:      content,
			FilePath:     codeFile.Path,
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

// Parse the code file into a tree-sitter node.
func (p *kotlinParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *kotlinParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}
