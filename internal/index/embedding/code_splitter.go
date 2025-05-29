package embedding

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/tiktoken-go/tokenizer"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/index/parser"
)

// CodeSplitter splits code content into smaller, manageable blocks.
type CodeSplitter interface {
	// Split SplitCode splits the given code string into CodeBlocks based on language-specific parsing rules and sliding window using token count.
	// It infers the language based on the file path.
	Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error)
}

type codeSplitter struct {
	logger            logx.Logger
	registry          *parser.Registry // Store the parser registry
	tokenizer         tokenizer.Codec  // Add tokenizer instance
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

func NewCodeSplitter(ctx context.Context, opts ...SplitOption) (CodeSplitter, error) {
	registry, err := parser.NewParserRegistry()

	if err != nil {
		return nil, err // Or handle the error appropriately
	}

	// Initialize the tokenizer
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, err
	}

	s := &codeSplitter{
		logger:    logx.WithContext(ctx),
		config:    c,
		registry:  registry,
		tokenizer: enc,
	}

	return s, nil
}

func (s *codeSplitter) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {

	codeParser, err := s.registry.TryGet(codeFile)

	if err != nil {
		return nil, err
	}

	s.logger.Debugf("file %s inferred language %v", codeFile)

	// Step 1: Perform syntax-based splitting using the language codeParser
	return codeParser.Split(codeFile, s.config.MaxTokensPerChunk, s.config.OverlapTokens)

}
