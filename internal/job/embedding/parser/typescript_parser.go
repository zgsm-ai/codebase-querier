package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const typeScriptQueryFile = "queries/typescript.scm"

type typeScriptTSParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser // Add Tree-sitter parser instance
	queryBytes        []byte         // Store query content as bytes
}

func NewTypeScriptTSParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	// Use the LanguageTypescript() function for pure TypeScript
	lang := sitter.NewLanguage(sitter_typescript.LanguageTypescript())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		// For now, we'll just log and return nil
		fmt.Printf("Error setting TypeScript language for parser: %v\n", err)
		return nil // Or return a nil parser with an error
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", typeScriptQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading TypeScript query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	typeScriptTSParser := &typeScriptTSParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.TypeScript, typeScriptTSParser)
	return typeScriptTSParser
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *typeScriptTSParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *typeScriptTSParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(code), nil)
	if tree == nil {
		// Parsing failed (e.g., timeout, cancellation)
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Use the query content from the struct field, converting to string
	lang := sitter.NewLanguage(sitter_typescript.LanguageTypescript())
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

		// Find the node corresponding to the declaration/definition itself
		// The captures might include the name, but we need the parent node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Check if the captured node's parent is the declaration/definition (common for name nodes)
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == "class_declaration" || capturedNode.Parent().Kind() == "function_declaration" || capturedNode.Parent().Kind() == "method_definition" || capturedNode.Parent().Kind() == "interface_declaration") {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == "class_declaration" || capturedNode.Kind() == "function_declaration" || capturedNode.Kind() == "method_definition" || capturedNode.Kind() == "interface_declaration" {
			// In case the query captured the node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != "class_declaration" && curr.Kind() != "function_declaration" && curr.Kind() != "method_definition" && curr.Kind() != "interface_declaration" {
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
		if definitionNode.Kind() == "method_definition" {
			// Traverse up the tree from the method definition to find the enclosing type declaration
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_declaration", "interface_declaration": // Methods can be in classes or interfaces
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName("name")
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
func (p *typeScriptTSParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".ts"
}
