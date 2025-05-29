package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_php "github.com/tree-sitter/tree-sitter-php/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const phpQueryFile = "queries/php.scm"

// Node kinds for PHP
const (
	phpClassDeclaration     = "class_declaration"
	phpInterfaceDeclaration = "interface_declaration"
	phpTraitDeclaration     = "trait_declaration"
	phpFunctionDeclaration  = "function_definition"
	phpMethodDeclaration    = "method_declaration"
)

// Field names for PHP
const (
	phpNameField = "name"
)

type phpParser struct {
	maxTokensPerChunk int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewPhpParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_php.LanguagePHP())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting PHP language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, phpQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading PHP query: %w", err)
	}

	phpParser := &phpParser{

		parser:   parser,
		language: lang,
		query:    queryStr,
	}

	return phpParser, nil
}

func (p *phpParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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
		if capturedNode.Parent() != nil && (capturedNode.Parent().Kind() == phpClassDeclaration || capturedNode.Parent().Kind() == phpInterfaceDeclaration || capturedNode.Parent().Kind() == phpTraitDeclaration || capturedNode.Parent().Kind() == phpFunctionDeclaration || capturedNode.Parent().Kind() == phpMethodDeclaration) {
			definitionNode = capturedNode.Parent()
		} else if capturedNode.Kind() == phpClassDeclaration || capturedNode.Kind() == phpInterfaceDeclaration || capturedNode.Kind() == phpTraitDeclaration || capturedNode.Kind() == phpFunctionDeclaration || capturedNode.Kind() == phpMethodDeclaration {
			// In case the query captured the node directly
			definitionNode = &capturedNode // Take the address
		} else {
			// If not immediately obvious, traverse up to find the nearest ancestor
			curr := capturedNode.Parent()
			for curr != nil && curr.Kind() != phpClassDeclaration && curr.Kind() != phpInterfaceDeclaration && curr.Kind() != phpTraitDeclaration && curr.Kind() != phpFunctionDeclaration && curr.Kind() != phpMethodDeclaration {
				curr = curr.Parent()
			}
			definitionNode = curr
		}

		if definitionNode == nil {
			continue
		}

		content := definitionNode.Utf8Text([]byte(codeFile.Content))

		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		// Determine parent class/trait/interface for method definitions
		// Use constants for node kinds and field names
		if definitionNode.Kind() == phpMethodDeclaration {
			// Traverse up the tree from the method definition to find the enclosing type declaration
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case phpClassDeclaration, phpInterfaceDeclaration, phpTraitDeclaration: // Methods can be in classes, interfaces or traits
					// Found the enclosing type declaration, try to find its name
					nameNode := curr.ChildByFieldName(phpNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(codeFile.Content))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
		}

		blocks = append(blocks, types.CodeChunk{
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
func (p *phpParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *phpParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil
	}
}
