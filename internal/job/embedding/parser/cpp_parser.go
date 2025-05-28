package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_cpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const cppQueryFile = "queries/cpp.scm"

// Node kinds
const (
	cppFunctionDefinition = "function_definition"
	cppMethodDefinition   = "method_definition"
	cppFunctionDeclarator = "function_declarator"
	cppClassSpecifier     = "class_specifier"
	cppStructSpecifier    = "struct_specifier"
)

// Field names
const (
	cppNameField = "name"
)

type cppParser struct {
	maxTokensPerBlock int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewCPPParser(maxTokensPerBlock, overlapTokens int) (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_cpp.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting C++ language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, cppQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading C++ query: %w", err)
	}

	cppParser := &cppParser{
		maxTokensPerBlock: maxTokensPerBlock,
		overlapTokens:     overlapTokens,
		parser:            parser,
		language:          lang,
		query:             queryStr,
	}
	return cppParser, nil
}

func (p *cppParser) Parse(code string, filePath string) ([]types.CodeBlock, error) {
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
		// In the query, we capture the function/method name, so we need to go up to the parent node
		// which is the function_declarator, and then up again to the function_definition or method_definition.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		nameNode := match.Captures[0].Node // This is the identifier node (the name)
		// Traverse up to find the nearest function_definition or method_definition ancestor
		definitionNode := nameNode.Parent()
		if definitionNode != nil && (definitionNode.Kind() == cppFunctionDeclarator) {
			// Go up one more level to the actual definition node
			definitionNode = definitionNode.Parent()
		}

		if definitionNode == nil || (definitionNode.Kind() != cppFunctionDefinition && definitionNode.Kind() != cppMethodDefinition) {
			// If we couldn't find the definition node (should not happen with this query)
			// or it's not the expected type, skip.
			continue
		}

		// Determine parent class for method definitions
		if definitionNode.Kind() == cppMethodDefinition {
			// Traverse up the tree to find the enclosing class_specifier or struct_specifier
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case cppClassSpecifier, cppStructSpecifier:
					// Found the enclosing type definition, try to find its name
					nameNode := curr.ChildByFieldName(cppNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(code))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
		}
		// For function_definition, parentFunc and parentClass remain empty

		// Extract the full content of the definition node
		content := definitionNode.Utf8Text([]byte(code))

		// Get start and end lines of the definition node
		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		blocks = append(blocks, types.CodeBlock{
			Content:      content,
			FilePath:     filePath,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,  // Empty for now for C++ functions/methods themselves
			ParentClass:  parentClass, // Populated for methods, empty for functions
			OriginalSize: len(content),
			TokenCount:   0,
		})
	}

	return blocks, nil
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *cppParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

// InferLanguage checks if the file extension is .cpp, .cc, .cxx, or .hpp
func (p *cppParser) InferLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".cpp", ".cc", ".cxx", ".hpp":
		return true
	default:
		return false
	}
}
