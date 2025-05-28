package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const typeScriptQueryFile = "queries/typescript.scm"

// Node kinds for TypeScript
const (
	tsClassDeclaration     = "class_declaration"
	tsFunctionDeclaration  = "function_declaration"
	tsMethodDefinition     = "method_definition"
	tsInterfaceDeclaration = "interface_declaration"
)

// Field names for TypeScript
const (
	tsNameField = "name"
)

type typeScriptParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewTypeScriptTSParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	// Use the Language() function for standard TypeScript
	lang := sitter.NewLanguage(sitter_typescript.LanguageTypescript())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, err
	}

	// Read the query file
	queryStr, err := loadQuery(lang, typeScriptQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading TypeScript query: %w", err)
	}

	typeScriptParser := &typeScriptParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return typeScriptParser, nil
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *typeScriptParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *typeScriptParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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

		// Find the node corresponding to the declaration/definition itself
		// The captures might include the name, but we need the parent node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Check if the captured node's parent is the declaration/definition (common for name nodes)
		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == tsClassDeclaration || capturedNode.Parent().Kind() == tsFunctionDeclaration || capturedNode.Parent().Kind() == tsMethodDefinition || capturedNode.Parent().Kind() == tsInterfaceDeclaration) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == tsClassDeclaration || capturedNode.Kind() == tsFunctionDeclaration || capturedNode.Kind() == tsMethodDefinition || capturedNode.Kind() == tsInterfaceDeclaration {
			// In case the query captured the node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != tsClassDeclaration && curr.Kind() != tsFunctionDeclaration && curr.Kind() != tsMethodDefinition && curr.Kind() != tsInterfaceDeclaration {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			// Should not happen if query is correct and structure is valid, but as a safeguard
			continue
		}

		// Extract the full content of the definition node
		content := definitionNode.Utf8Text([]byte(code))

		// Get start and end lines of the definition node
		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		// Determine parent class/interface for method definitions
		// Use constants for node kinds and field names
		if definitionNode.Kind() == tsMethodDefinition {
			// Traverse up the tree from the method definition to find the enclosing type declaration
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case tsClassDeclaration, tsInterfaceDeclaration: // Methods can be in classes or interfaces
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName(tsNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
			// parentFunc remains empty for members within types
		}

		// For class, interface, and function declarations, parentFunc and parentClass remain empty

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,   // Empty for now for top-level structures
			ParentClass:  parentClass,  // Populated for methods, empty for top-level types
			OriginalSize: len(content), // Size in bytes
			TokenCount:   0,            // Will be calculated by CodeSplitter
		})
	}

	return blocks, nil
}

// InferLanguage checks if the file extension is .ts
func (p *typeScriptParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".ts"
}
