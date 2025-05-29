package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const typeScriptTSXQueryFile = "queries/typescript_tsx.scm"

// Node kinds for TypeScript TSX
const (
	tsTSXClassDeclaration             = "class_declaration"
	typeScriptTSXFunctionDeclaration  = "function_declaration"
	typeScriptTSXMethodDefinition     = "method_definition"
	typeScriptTSXInterfaceDeclaration = "interface_declaration"
)

// Field names for TypeScript TSX
const (
	typeScriptTSXNameField = "name"
)

type typeScriptTSXParser struct {
	parser   *sitter.Parser   // Add Tree-sitter parser instance
	language *sitter.Language // Store language instance
	query    string           // Store query string
}

func NewTypeScriptTSXParser() (CodeParser, error) {
	parser := sitter.NewParser()
	// Use the LanguageTSX() function for TSX
	lang := sitter.NewLanguage(sitter_typescript.LanguageTSX())
	err := parser.SetLanguage(lang)
	if err != nil {
		return nil, err
	}

	// Read the query file
	queryStr, err := loadQuery(lang, typeScriptTSXQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading TSX query: %w", err)
	}

	typeScriptTSXParser := &typeScriptTSXParser{
		parser:   parser,
		language: lang,
		query:    queryStr,
	}

	return typeScriptTSXParser, err
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *typeScriptTSXParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *typeScriptTSXParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
	if p.parser == nil || p.language == nil || p.query == "" {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(codeFile.Content), nil)
	if tree == nil {
		// Parsing failed (e.g., timeout, cancellation)
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

		// Find the node corresponding to the declaration/definition itself
		// The captures might include the name, but we need the parent node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Check if the captured node's parent is the declaration/definition (common for name nodes)
		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == tsTSXClassDeclaration || capturedNode.Parent().Kind() == typeScriptTSXFunctionDeclaration || capturedNode.Parent().Kind() == typeScriptTSXMethodDefinition || capturedNode.Parent().Kind() == typeScriptTSXInterfaceDeclaration) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == tsTSXClassDeclaration || capturedNode.Kind() == typeScriptTSXFunctionDeclaration || capturedNode.Kind() == typeScriptTSXMethodDefinition || capturedNode.Kind() == typeScriptTSXInterfaceDeclaration {
			// In case the query captured the node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != tsTSXClassDeclaration && curr.Kind() != typeScriptTSXFunctionDeclaration && curr.Kind() != typeScriptTSXMethodDefinition && curr.Kind() != typeScriptTSXInterfaceDeclaration {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			// Should not happen if query is correct and structure is valid, but as a safeguard
			continue
		}

		// Extract the full content of the definition node
		content := definitionNode.Utf8Text([]byte(codeFile.Content))

		// Get start and end lines of the definition node
		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		// Determine parent class/interface for method definitions
		// Use constants for node kinds and field names
		if definitionNode.Kind() == typeScriptTSXMethodDefinition {
			// Traverse up the tree from the method definition to find the enclosing type declaration
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case tsTSXClassDeclaration, typeScriptTSXInterfaceDeclaration: // Methods can be in classes or interfaces
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName(typeScriptTSXNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(codeFile.Content))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
			// parentFunc remains empty for members within types
		}

		// For class, interface, and function declarations, parentFunc and parentClass remain empty

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

// Parse the code file into a tree-sitter node.
func (p *typeScriptTSXParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
