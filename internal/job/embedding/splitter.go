package embedding

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/tiktoken-go/tokenizer"
	"github.com/zeromicro/go-zero/core/logx"
)

// CodeSplitter splits code content into smaller, manageable blocks.
type CodeSplitter interface {
	// Split SplitCode splits the given code string into CodeBlocks based on language-specific parsing rules and sliding window using token count.
	// It infers the language based on the file path.
	Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error)
}

type codeSplitter struct {
	logger            logx.Logger
	registry          *ParserRegistry // Store the parser registry
	tokenizer         tokenizer.Codec // Add tokenizer instance
	overlapTokens     int
	maxTokensPerChunk int
}

type SplitOption func(*codeSplitter)

func WithOverlapTokens(OverlapTokens int) SplitOption {
	return func(s *codeSplitter) {
		s.overlapTokens = OverlapTokens
	}
}

func WithMaxTokensPerChunk(MaxTokensPerChunk int) SplitOption {
	return func(s *codeSplitter) {
		s.maxTokensPerChunk = MaxTokensPerChunk
	}
}

// NewCodeSplitter creates a new CodeSplitter instance.
// It initializes the parser registry and the tokenizer.
func NewCodeSplitter(ctx context.Context, opts ...SplitOption) (CodeSplitter, error) {
	registry, err := NewParserRegistry() // NewParserRegistry now returns *Registry

	if err != nil {
		return nil, fmt.Errorf("failed to create parser registry: %w", err) // Added fmt.Errorf for better error context
	}

	// Initialize the tokenizer	// Initialize the tokenizer
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		// Close registry on error to prevent resource leak
		registry.Close()
		return nil, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	s := &codeSplitter{
		logger:    logx.WithContext(ctx),
		registry:  registry,
		tokenizer: enc,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Split splits the code file into chunks using the appropriate language parser.
func (s *codeSplitter) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {

	// Get the language configuration from the registry
	// Now we directly get the parser instance based on file extension

	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		// Handle files without extension, maybe infer based on shebang or content if needed
		// For now, return error or handle as plain text
		return nil, fmt.Errorf("file %s has no extension, cannot determine language", codeFile.Path)
	}

	// Get the parser by file extension
	parser, ok := s.registry.GetParserByFileExtension(ext)
	if !ok {
		return nil, fmt.Errorf("no language parser found for file %s with extension %s", codeFile.Path, ext)
	}

	s.logger.Debugf("file %s found parser for extension %s", codeFile.Path, ext) // Log parser found

	// No need to create a new parser instance here, registry provides a ready one.
	// We might need to handle if the parser needs context-specific state, but for now, assume stateless or context-managed by registry.

	// The obtained parser is a *GenericParser
	defer parser.Close() // Ensure the parser is closed after use if GetParserByFileExtension returns a new instance each time. If registry caches, Close() might be on registry.
	// Let's assume GetParserByFileExtension returns a potentially cached parser, but Close() is safe to call on it.

	// Step 1: Perform splitting using the generic parser's Split method
	// Note: The Split method on GenericParser takes codeFile, maxTokens, overlapTokens
	chunks, err := parser.Split(codeFile, s.maxTokensPerChunk, s.overlapTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to split file %s using parser for %s: %w", codeFile.Path, ext, err)
	}

	// TODO: Optionally, use the tokenizer here if needed for more accurate token counting
	// The current lang/parser.go splitIntoChunks uses character count as a proxy.
	// If accurate token counts are required, integrate the tokenizer here or within lang/parser.go.

	return chunks, nil
}
