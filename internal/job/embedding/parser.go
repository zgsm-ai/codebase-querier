package embedding

import (
	"errors"
	"fmt"
	"github.com/tiktoken-go/tokenizer"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// CodeParser defines the interface for language-specific code parsing.
// The genericParser will implement this interface.
type CodeParser interface {
	Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error)

	Parse(codeFile *types.CodeFile) (*sitter.Node, error)

	Close()
}

// DefinitionKind represents the type of code definition (e.g., function, class)
type DefinitionKind string

const (
	FuncKind    DefinitionKind = "function"
	ClassKind   DefinitionKind = "class"
	TypeKind    DefinitionKind = "type"
	MethodKind  DefinitionKind = "method"
	UnknownKind DefinitionKind = "unknown"
)

// DefinitionNodeInfo holds information about a parsed definition node.
type DefinitionNodeInfo struct {
	Node        *sitter.Node
	Kind        DefinitionKind
	Name        string
	ParentFunc  string // Populated if this is a method inside a function
	ParentClass string // Populated if this is a method inside a class or a nested function
}

// ProcessMatchFunc is a function type that takes a Tree-sitter query match,
// the root node, and the code content, and returns a slice of DefinitionNodeInfo
// identified from the match. This function encapsulates language-specific logic.
type ProcessMatchFunc func(match *sitter.QueryMatch, root *sitter.Node, content []byte) ([]*DefinitionNodeInfo, error)

// LanguageConfig holds language-specific configuration and logic function.
type LanguageConfig struct {
	Language       Language         // sitterLanguage name (e.g., "Go", "Python")
	sitterLanguage *sitter.Language // The Tree-sitter language instance
	Query          string           // The Tree-sitter query string for finding initial match nodes
	SupportedExts  []string         // The file extensions supported by this config

	// sitterLanguage-specific function to process a query match and extract definition info.
	ProcessMatch ProcessMatchFunc
}

// GenericParser  is a generic implementation of the CodeParser interface
// that uses a LanguageConfig to handle language-specific details.
type GenericParser struct {
	parser            *sitter.Parser  // Tree-sitter parser instance
	config            *LanguageConfig // Language-specific configuration
	tokenizer         tokenizer.Codec
	maxTokensPerChunk int
	overlapTokens     int
}

// NewGenericParser creates a new generic parser with the given config.
func NewGenericParser(config LanguageConfig, tokenizer tokenizer.Codec, maxTokensPerChunk, overlapTokens int) (*GenericParser, error) {
	parser := sitter.NewParser()
	err := parser.SetLanguage(config.sitterLanguage)

	if err != nil {
		// Close parser on error to prevent resource leak
		parser.Close()
		return nil, fmt.Errorf("error setting language: %w", err)
	}

	return &GenericParser{
		parser:            parser,
		config:            &config,
		tokenizer:         tokenizer,
		maxTokensPerChunk: maxTokensPerChunk,
		overlapTokens:     overlapTokens,
	}, nil
}

// Close releases the Tree-sitter parser resources.
func (p *GenericParser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// GetLanguage returns the Tree-sitter Language associated with this parser.
func (p *GenericParser) GetLanguage() *sitter.Language {
	if p.config != nil {
		return p.config.sitterLanguage
	}
	return nil
}

// Parse the code file into a tree-sitter node.
// Reuses the existing doParse helper.
func (p *GenericParser) Parse(content string) (*sitter.Tree, error) {
	if p.parser == nil {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}
	if p == nil {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.parser.Parse([]byte(content), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	return tree, nil
}

// Split splits the code content into chunks based on the LanguageConfig.
func (p *GenericParser) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {
	if p.parser == nil || p.config == nil || p.config.ProcessMatch == nil {
		return nil, errors.New("parser is not properly initialized or has been closed or missing config/ProcessMatch function")
	}

	tree, err := p.Parse(codeFile.Content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()

	// Create Tree-sitter query from the config's query string
	query, queryErr := sitter.NewQuery(p.config.sitterLanguage, p.config.Query)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to create query for %s: %v", codeFile.Path, queryErr)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	contentBytes := []byte(codeFile.Content)
	matches := qc.Matches(query, root, contentBytes)

	var allChunks []*types.CodeChunk

	for match := matches.Next(); match != nil; match = matches.Next() {

		// Use the language-specific ProcessMatch function from the config
		defInfos, err := p.config.ProcessMatch(match, root, contentBytes)
		if err != nil {
			// Log the error and continue processing other matches, or return the error
			// fmt.Printf("Error processing match for file %s: %v\n", codeFile.Path, err) // Use logger if available
			continue // Example: continue on error for a single match
		}

		for _, defInfo := range defInfos {
			if defInfo == nil || defInfo.Node == nil {
				continue // Skip invalid results from ProcessMatch
			}

			content := defInfo.Node.Utf8Text(contentBytes)
			// Convert uint line numbers to int
			startLine := int(defInfo.Node.StartPosition().Row)
			endLine := int(defInfo.Node.EndPosition().Row)

			tokenCount := p.countToken(content)

			// Check if the content size exceeds the maximum chunk size (still using char count as proxy)
			if tokenCount > p.maxTokensPerChunk { // Using character count as a proxy for tokens
				// Split the large block using the sliding window approach
				// Pass ParentFunc and ParentClass to splitIntoChunks
				subChunks := p.splitIntoChunks(content, codeFile.Path, startLine,
					endLine, defInfo.ParentFunc, defInfo.ParentClass)
				allChunks = append(allChunks, subChunks...)
			} else {
				// Add the whole block as a single chunk
				chunk := &types.CodeChunk{
					Name:         defInfo.Name,
					Content:      content,
					FilePath:     codeFile.Path,
					StartLine:    startLine,
					EndLine:      endLine,
					OriginalSize: len(content),
					TokenCount:   tokenCount,          // Approximation
					ParentFunc:   defInfo.ParentFunc,  // Populate new fields
					ParentClass:  defInfo.ParentClass, // Populate new fields
				}
				allChunks = append(allChunks, chunk)
			}
		}
	}

	// TODO: Potentially add logic here or within ProcessMatchFunc to capture
	// top-level code that isn't part of a specific definition match, if needed.
	// This might involve iterating through the root node's children and processing
	// nodes that weren't covered by the definition queries/matches.

	return allChunks, nil
}

func (p *GenericParser) countToken(content string) int {
	tokenCount, err := p.tokenizer.Count(content)
	if err != nil {
		tokenCount = len(content)
	}
	return tokenCount
}

// splitIntoChunks applies a sliding window to a large content string.
// Updated to populate ParentFunc and ParentClass.
func (p *GenericParser) splitIntoChunks(
	content string,
	filePath string,
	startLine,
	endLine int,
	parentFunc,
	parentClass string) []*types.CodeChunk {
	var chunks []*types.CodeChunk
	contentBytes := []byte(content)
	contentLen := len(contentBytes)

	// Use character count as a simple token approximation for now
	maxLen := p.maxTokensPerChunk
	overlapLen := p.overlapTokens

	for i := 0; i < contentLen; {
		end := i + maxLen
		if end > contentLen {
			end = contentLen
		}

		chunkContent := string(contentBytes[i:end])

		// Approximate start and end lines for the chunk within the original content's line range.
		// This is a rough approximation and might not be perfectly accurate for multi-line content.
		// A more accurate approach would involve iterating lines within the content string.
		chunkStartLine := startLine + countLines(contentBytes[:i])
		chunkEndLine := chunkStartLine + countLines([]byte(chunkContent)) - 1
		if chunkEndLine < chunkStartLine { // Handle single line chunks
			chunkEndLine = chunkStartLine
		}

		chunks = append(chunks, &types.CodeChunk{
			Content:      chunkContent,
			FilePath:     filePath,
			StartLine:    chunkStartLine,
			EndLine:      chunkEndLine,
			ParentFunc:   parentFunc,  // Populate the field
			ParentClass:  parentClass, // Populate the field
			OriginalSize: len(chunkContent),
			TokenCount:   len(chunkContent), // Approximation
		})

		if end == contentLen {
			break
		}

		i = end - overlapLen
		if i < 0 {
			i = 0
		}
	}

	return chunks
}

// countLines is a helper to count lines in a byte slice.
// Reuses the existing logic.
func countLines(data []byte) int {
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	// Add one for the last line if it doesn't end with a newline
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	// If the content is empty, there are 0 lines. If it's not empty but has no newline, it's 1 line.
	if len(data) == 0 {
		return 0
	}
	if lines == 0 && len(data) > 0 {
		return 1
	} // Fix: handle non-empty single line
	if lines == 0 && len(data) == 0 {
		return 0
	} // Explicitly handle empty
	return lines
}

// GetDefinitionKindFromNodeKind adds a simple helper to determine DefinitionKind from a node kind string
// This is a basic mapping and might need refinement per language's AST.
func GetDefinitionKindFromNodeKind(nodeKind string) DefinitionKind {
	switch nodeKind {
	case "function_declaration", "function_definition", "method_declaration", "func_declaration", "func_definition": // Common names across languages
		// Further logic might be needed in ProcessMatchFunc to differentiate between function and method
		return FuncKind
	case "class_definition", "struct_type", "interface_type", "enum_declaration": // Common names across languages
		return ClassKind // Or TypeKind depending on language nuance
	case "type_declaration", "interface_declaration": // Go, TypeScript specific
		return TypeKind
	// Add more cases for other language-specific definition kinds if necessary
	default:
		return UnknownKind
	}
}
