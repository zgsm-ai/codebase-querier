package lang

import (
	"errors"
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_ruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const rubyQueryFile = "queries/ruby.scm"

// Node kinds for Ruby
const (
	rubyClassDeclaration  = "class_declaration"
	rubyModuleDeclaration = "module_declaration"
	rubyMethodDeclaration = "method_declaration"
)

// Field names for Ruby
const (
	rubyNameField = "name"
)

type rubyParser struct {
	maxTokensPerChunk int
	overlapTokens     int
	parser            *sitter.Parser   // Add Tree-sitter parser instance
	language          *sitter.Language // Store language instance
	query             string           // Store query string
}

func NewRubyParser() (CodeParser, error) {
	parser := sitter.NewParser()
	lang := sitter.NewLanguage(sitter_ruby.Language())
	err := parser.SetLanguage(lang)
	if err != nil {
		// Handle error: incompatible language version, etc.
		return nil, fmt.Errorf("error setting Ruby language: %w", err)
	}

	// Read the query file
	queryStr, err := loadQuery(lang, rubyQueryFile)
	if err != nil {
		return nil, fmt.Errorf("error loading Ruby query: %w", err)
	}

	rubyParser := &rubyParser{

		parser:   parser,
		language: lang,
		query:    queryStr,
	}
	return rubyParser, nil
}

// Close releases the Tree-sitter parser resources.
// It's important to call this when the parser is no longer needed.
func (p *rubyParser) Close() {
	if p.parser != nil {
		p.parser.Close()
		p.parser = nil // Avoid double close
	}
}

func (p *rubyParser) Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error) {
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

		// Find the node corresponding to the definition itself
		// The captures might include the name, but we need the parent definition node.
		if len(match.Captures) == 0 {
			continue // Should not happen with the current query, but as a safeguard
		}

		capturedNode := match.Captures[0].Node
		var definitionNode *sitter.Node

		// Traverse up to find the nearest definition ancestor
		// Note: In Ruby, methods can be defined at the top level, within classes, or within modules.
		// The query targets the method node itself, so we traverse up from there.
		curr := capturedNode.Parent()
		for curr != nil && curr.Kind() != rubyClassDeclaration && curr.Kind() != rubyModuleDeclaration && curr.Kind() != rubyMethodDeclaration {
			curr = curr.Parent()
		}
		definitionNode = curr // This should be the method, class, or module node

		if definitionNode == nil {
			// This case should ideally not be reached for valid code with matches,
			// but as a safeguard:
			continue
		}

		// Extract the full content of the definition node
		content := definitionNode.Utf8Text([]byte(codeFile.Content))

		// Get start and end lines of the definition node
		startLine := definitionNode.StartPosition().Row + 1
		endLine := definitionNode.EndPosition().Row + 1

		// Determine parent class/module for method definitions
		if definitionNode.Kind() == rubyMethodDeclaration {
			// Traverse up the tree from the method definition to find the enclosing class or module
			curr := definitionNode.Parent()
			for curr != nil {
				switch curr.Kind() {
				case rubyClassDeclaration, rubyModuleDeclaration: // Methods can be in classes or modules
					// Found the enclosing type definition, try to find its name
					nameNode := curr.ChildByFieldName(rubyNameField)
					if nameNode != nil {
						parentClass = nameNode.Utf8Text([]byte(codeFile.Content))
					}
					goto foundParent // Exit loops once parent is found
				}
				curr = curr.Parent()
			}
		foundParent:
		}
		// For class and module definitions, parentFunc and parentClass remain empty

		blocks = append(blocks, &types.CodeChunk{
			Content:      content,
			FilePath:     codeFile.Path,
			StartLine:    int(startLine),
			EndLine:      int(endLine),
			ParentFunc:   parentFunc,  // Empty for now for top-level structures
			ParentClass:  parentClass, // Populated for methods, empty for top-level types
			OriginalSize: len(content),
			TokenCount:   0,
		})
	}

	return blocks, nil
}

// Parse the code file into a tree-sitter node.
func (p *rubyParser) Parse(codeFile *types.CodeFile) (*sitter.Node, error) {
	return doParse(codeFile, p.parser)
}
