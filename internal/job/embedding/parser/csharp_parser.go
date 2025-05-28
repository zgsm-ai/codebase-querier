package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_csharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const csharpQueryFile = "queries/csharp.scm"

type cSharpParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser // Add Tree-sitter parser instance
	queryBytes        []byte         // Store query content as bytes
}

func NewCSharpParser(maxTokensPerBlock, overlapTokens int) CodeParser {
	parser := sitter.NewParser()
	// Use the standard Language() function from the official binding
	lang := sitter.NewLanguage(sitter_csharp.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		// For now, we'll just log and return nil
		fmt.Printf("Error setting C# language for parser: %v\n", err)
		return nil // Or return a nil parser with an error
	}

	// Read the query file
	queryPath := filepath.Join("internal/job/embedding/splitter/parser", csharpQueryFile)
	queryContent, err := os.ReadFile(queryPath)
	if err != nil {
		fmt.Printf("Error reading C# query file %s: %v\n", queryPath, err)
		return nil // Or handle error appropriately
	}

	cSharpParser := &cSharpParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		queryBytes:        queryContent, // Store query content
	}
	registerParser(types.CSharp, cSharpParser)
	return cSharpParser
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
	lang := sitter.NewLanguage(sitter_csharp.Language())
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

		// Find the node corresponding to the declaration itself
		// The captures might include the name, but we need the parent declaration node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var declarationNode *sitter.Node

		// Check if the captured node's parent is the declaration (common for identifier nodes)
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == "class_declaration" || capturedNode.Parent().Kind() == "struct_declaration" || capturedNode.Parent().Kind() == "interface_declaration" || capturedNode.Parent().Kind() == "enum_declaration" || capturedNode.Parent().Kind() == "method_declaration" || capturedNode.Parent().Kind() == "property_declaration") {
			declarationNode = capturedNode.Parent()
		} else if capturedNode.Kind() == "class_declaration" || capturedNode.Kind() == "struct_declaration" || capturedNode.Kind() == "interface_declaration" || capturedNode.Kind() == "enum_declaration" || capturedNode.Kind() == "method_declaration" || capturedNode.Kind() == "property_declaration" {
			// In case the query captured the declaration node directly
			declarationNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest declaration ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != "class_declaration" && curr.Kind() != "struct_declaration" && curr.Kind() != "interface_declaration" && curr.Kind() != "enum_declaration" && curr.Kind() != "method_declaration" && curr.Kind() != "property_declaration" {
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
		case "method_declaration", "property_declaration":
			// Traverse up the tree to find the enclosing type declaration
			curr := declarationNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case "class_declaration", "struct_declaration", "interface_declaration", "enum_declaration":
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName("identifier")
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

		// For other top-level declarations, parentFunc and parentClass remain empty

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,   // Empty for now for top-level structures
			ParentClass:  parentClass,  // Populated for methods/properties, empty for top-level types
			OriginalSize: len(content), // Size in bytes
			TokenCount:   0,            // Will be calculated by CodeSplitter
		})
	}

	return blocks, nil
}

// InferLanguage checks if the file extension is .cs
func (p *cSharpParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".cs"
}
