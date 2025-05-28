package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const pythonQueryFile = "queries/python.scm"

// Node kinds for Python
const (
	pythonClassDefinition    = "class_definition"
	pythonFunctionDefinition = "function_definition"
)

// Field names for Python
const (
	pythonNameField = "name"
)

type pythonParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewPythonParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_python.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting Python language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, pythonQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Python query: %w", err)
	}

	pythonParser := &pythonParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return pythonParser, nil
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *pythonParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *pythonParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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

		// Find the node corresponding to the definition itself
		// The captures might include the name, but we need the parent definition node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Check if the captured node's parent is the definition (common for name nodes)
		// Use constants for node kinds
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == pythonClassDefinition || capturedNode.Parent().Kind() == pythonFunctionDefinition) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == pythonClassDefinition || capturedNode.Kind() == pythonFunctionDefinition {
			// In case the query captured the definition node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest definition ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != pythonClassDefinition && curr.Kind() != pythonFunctionDefinition {
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

		// Determine parent class for function definitions within classes
		// Use constant for node kind and field name
		if definitionNode.Kind() == pythonFunctionDefinition {
			// Traverse up the tree from the function definition to find the enclosing class definition
			curr := definitionNode.Parent()
			for curr != nil {
				if curr.Kind() == pythonClassDefinition {
					// Found the enclosing class definition, try to find its name
					nameNode := curr.ChildByFieldName(pythonNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
			// parentFunc remains empty for methods within classes
		}

		// For class definitions, parentFunc and parentClass remain empty

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

// InferLanguage checks if the file extension is .py
func (p *pythonParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	return ext == ".py"
}
