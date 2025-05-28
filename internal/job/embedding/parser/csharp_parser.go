package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const csharpQueryFile = "queries/csharp.scm"

// Node kinds
const (
	csharpClassDeclaration     = "class_declaration"
	csharpStructDeclaration    = "struct_declaration"
	csharpInterfaceDeclaration = "interface_declaration"
	csharpEnumDeclaration      = "enum_declaration"
	csharpMethodDeclaration    = "method_declaration"
	csharpPropertyDeclaration  = "property_declaration"
	csharpEventDeclaration     = "event_declaration"
	csharpFieldDeclaration     = "field_declaration"
)

// Field names
const (
	csharpIdentifierField = "identifier"
	csharpNameField       = "name"
)

type cSharpParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewCSharpParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_csharp.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, fmt.Errorf("error setting C# language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, csharpQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading C# query: %w", err)
	}

	cSharpParser := &cSharpParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return cSharpParser, nil
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *cSharpParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *cSharpParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil || p.language == nil || p.query == "" {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		// Parsing failed (e.g., timeout, cancellation)
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

		// Find the node corresponding to the declaration itself
		// The captures might include the name, but we need the parent declaration node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var declarationNode *sitter.Node

		// Check if the captured node's parent is the declaration (common for identifier nodes)
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == csharpClassDeclaration || capturedNode.Parent().Kind() == csharpStructDeclaration || capturedNode.Parent().Kind() == csharpInterfaceDeclaration || capturedNode.Parent().Kind() == csharpEnumDeclaration || capturedNode.Parent().Kind() == csharpMethodDeclaration || capturedNode.Parent().Kind() == csharpPropertyDeclaration || capturedNode.Parent().Kind() == csharpEventDeclaration || capturedNode.Parent().Kind() == csharpFieldDeclaration) {
			declarationNode = capturedNode.Parent()
		} else if capturedNode.Kind() == csharpClassDeclaration || capturedNode.Kind() == csharpStructDeclaration || capturedNode.Kind() == csharpInterfaceDeclaration || capturedNode.Kind() == csharpEnumDeclaration || capturedNode.Kind() == csharpMethodDeclaration || capturedNode.Kind() == csharpPropertyDeclaration || capturedNode.Kind() == csharpEventDeclaration || capturedNode.Kind() == csharpFieldDeclaration {
			// In case the query captured the declaration node directly
			declarationNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest declaration ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != csharpClassDeclaration && curr.Kind() != csharpStructDeclaration && curr.Kind() != csharpInterfaceDeclaration && curr.Kind() != csharpEnumDeclaration && curr.Kind() != csharpMethodDeclaration && curr.Kind() != csharpPropertyDeclaration && curr.Kind() != csharpEventDeclaration && curr.Kind() != csharpFieldDeclaration {
				curr = curr.Parent()
			}
			declarationNode = curr
		}

		if declarationNode == nil {
			// Should not happen if query is correct and structure is valid, but as a safeguard
			continue
		}

		// Extract the full content of the declaration node
		content := declarationNode.Utf8Text([]byte(code))

		// Get start and end lines of the declaration node
		startLine := declarationNode.StartPosition().Row + 1
		endLine := declarationNode.EndPosition().Row + 1

		// Determine parent class/struct/interface/enum for method and property declarations
		switch declarationNode.Kind() {
		case csharpMethodDeclaration, csharpPropertyDeclaration:
			// Traverse up the tree to find the enclosing type declaration
			curr := declarationNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case csharpClassDeclaration, csharpStructDeclaration, csharpInterfaceDeclaration, csharpEnumDeclaration:
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName(csharpNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
		}

		// For other top-level declarations, parentFunc and parentClass remain empty

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,  // Empty for now for top-level structures
			ParentClass:  parentClass, // Populated for methods/properties, empty for top-level types
			OriginalSize: len(content),
			TokenCount:   0,
		})
	}

	return blocks, nil
}

// InferLanguage checks if the file extension is .cs
func (p *cSharpParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".cs"
}
