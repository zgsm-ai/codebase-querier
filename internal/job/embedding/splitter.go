package embedding

import (
	"fmt"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/tiktoken-go/tokenizer"
	"github.com/zeromicro/go-zero/core/logx"
)

type CodeSplitter struct {
	registry          *ParserRegistry // Store the parser registry
	overlapTokens     int
	maxTokensPerChunk int
}

type SplitOption func(*CodeSplitter)

// NewCodeSplitter creates a new CodeSplitter instance.
// It initializes the parser registry and the tokenizer.
func NewCodeSplitter(maxTokensPerChunk, overlapTokens int) (*CodeSplitter, error) {

	// Initialize the tokenizer	// Initialize the tokenizer
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	registry, err := NewParserRegistry(enc, maxTokensPerChunk, overlapTokens) // NewParserRegistry now returns *Registry

	if err != nil {
		return nil, fmt.Errorf("failed to create parser registry: %w", err) // Added fmt.Errorf for better error context
	}
	return &CodeSplitter{

		registry: registry,
	}, nil

}

// Split splits the code file into chunks using the appropriate language parser.
func (s *CodeSplitter) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {
	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		return nil, fmt.Errorf("file %s has no extension, cannot determine language", codeFile.Path)
	}

	// Get the parser by file extension
	parser, ok := s.registry.GetParserByFileExtension(ext)
	if !ok {
		return nil, fmt.Errorf("no language processor found for file %s with extension %s", codeFile.Path, ext)
	}

	logx.Debugf("file %s using processor for language %s (extension %s)",
		codeFile.Path, parser.config.Language, ext)

	// The parser is managed by the registry, so we don't need to close it here
	chunks, err := parser.SplitWithParse(codeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to process file %s using %s processor: %w",
			codeFile.Path, parser.config.Language, err)
	}

	return chunks, nil
}
