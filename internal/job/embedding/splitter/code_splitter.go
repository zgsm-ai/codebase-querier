package splitter

import (
	"context"
	"fmt"

	"github.com/tiktoken-go/tokenizer"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// CodeSplitter splits code content into smaller, manageable blocks.
type CodeSplitter interface {
	// SplitCode splits the given code string into CodeBlocks based on language-specific parsing rules and sliding window using token count.
	// It infers the language based on the file path.
	SplitCode(code, filePath string) ([]types.CodeBlock, error)
	// Close releases any resources used by the splitter and its underlying parsers and tokenizer.
	Close()
}

// CodeBlock represents a chunk of code with associated metadata.
// Moved to internal/types/types.go

type codeSplitter struct {
	logger    logx.Logger
	config    config.CodeSplitterConf // Store config to access MaxTokensPerBlock and OverlapTokens
	parsers   map[types.Language]parser.CodeParser
	tokenizer tokenizer.Codec // Add tokenizer instance
}

func NewCodeSplitter(ctx context.Context, c config.CodeSplitterConf) CodeSplitter {
	// Ensure all parsers are registered before creating the splitter.
	// This is now handled by InitRegisteredParsers in the parser package.
	parser.InitRegisteredParsers(c.MaxTokensPerBlock, c.OverlapTokens)

	// Initialize the tokenizer
	enc, err := tokenizer.Get(tokenizer.Cl100kBase) // Using cl100k_base encoding
	if err != nil {
		// Handle error: encoding not found, etc.
		// For now, we'll log and panic or return nil
		// In a real app, you'd want more robust error handling
		logx.WithContext(ctx).Errorf("Failed to get tokenizer encoding cl100k_base: %v", err)
		// Depending on error handling strategy, might return nil or a splitter that can't tokenize
		return nil
	}

	s := &codeSplitter{
		logger:    logx.WithContext(ctx),
		config:    c, // Store config
		parsers:   make(map[types.Language]parser.CodeParser),
		tokenizer: enc, // Store the initialized tokenizer
	}

	// Retrieve registered parsers and add them to the splitter's map.
	for lang, p := range parser.RegisteredParsers {
		s.parsers[lang] = p
	}

	return s
}

func (s *codeSplitter) SplitCode(code, filePath string) ([]types.CodeBlock, error) {
	var parserToUse parser.CodeParser
	var inferredLanguage = types.Unknown

	// Infer language and find the appropriate parser
	for lang, p := range s.parsers {
		if p.InferLanguage(filePath) {
			parserToUse = p
			inferredLanguage = lang
			break
		}
	}
	s.logger.Debugf("file %s inferred language %v", filePath, inferredLanguage)

	// Step 1: Perform syntax-based splitting using the language parser
	initialBlocks, err := parserToUse.Parse(code, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code for %s: %w", filePath, err)
	}

	var finalBlocks []types.CodeBlock

	// Step 2: Calculate token count for initial blocks and apply sliding window to large blocks
	for _, block := range initialBlocks {
		// Calculate token count for the block content
		tokens, _, err := s.tokenizer.Encode(block.Content)
		if err != nil {
			s.logger.Errorf("Failed to encode block content to tokens for %s (lines %d-%d): %v. Using byte count as fallback size.", block.FilePath, block.StartLine, block.EndLine, err)
			// Fallback to byte count if tokenization fails
			block.TokenCount = len(block.Content)
		} else {
			block.TokenCount = len(tokens) // Store the token count
		}

		// Use TokenCount for size check
		if block.TokenCount > s.config.MaxTokensPerBlock {
			// Apply sliding window to this large block based on tokens
			s.logger.Infof("Applying sliding window to large block in %s (lines %d-%d), token size %d", block.FilePath, block.StartLine, block.EndLine, block.TokenCount)

			chunkSize := s.config.MaxTokensPerBlock
			overlapTokens := s.config.OverlapTokens

			// Ensure chunk size is positive and not zero
			if chunkSize <= 0 {
				s.logger.Errorf("Invalid chunk size (%d) for sliding window in %s. Skipping sliding window for this block.", chunkSize, block.FilePath)
				// Fallback: add the original large block if chunk size is invalid
				finalBlocks = append(finalBlocks, block)
				continue
			}

			// Get token IDs for the block content again for slicing
			blockTokens, _, encodeErr := s.tokenizer.Encode(block.Content)
			if encodeErr != nil {
				s.logger.Errorf("Failed to re-encode block content for slicing for %s (lines %d-%d): %v. Skipping sliding window for this block.", block.FilePath, block.StartLine, block.EndLine, encodeErr)
				finalBlocks = append(finalBlocks, block)
				continue
			}

			tokensLen := len(blockTokens)

			for i := 0; i < tokensLen; {
				// Calculate the end token index for the current chunk
				endTokenIndex := i + chunkSize
				if endTokenIndex > tokensLen {
					endTokenIndex = tokensLen
				}

				// Get the token IDs for the chunk
				chunkTokens := blockTokens[i:endTokenIndex]

				// Decode the token IDs back to string content
				chunkContent, decodeErr := s.tokenizer.Decode(chunkTokens)
				if decodeErr != nil {
					s.logger.Errorf("Failed to decode chunk tokens for %s (original lines %d-%d): %v. Skipping this chunk.", block.FilePath, block.StartLine, block.EndLine, decodeErr)
					// Move to the next potential chunk start if decoding fails
					i = endTokenIndex
					continue
				}

				// Calculate the start and end line numbers for the chunk (approximate based on token ratio)
				// This is a very simplified approach and can be inaccurate.
				// A better approach would involve mapping token indices back to byte offsets and then to line numbers.
				totalTokensInBlock := float64(block.TokenCount) // Use the potentially fallback token count
				linesInBlock := float64(block.EndLine - block.StartLine + 1)
				tokensPerLine := totalTokensInBlock / linesInBlock

				var chunkStartLineOffset, chunkEndLineOffset int
				if tokensPerLine > 0 {
					// Approximate line offset based on token indices
					chunkStartLineOffset = int(float64(i) / tokensPerLine)
					chunkEndLineOffset = int(float64(endTokenIndex-1) / tokensPerLine)
				} else {
					// Handle blocks with zero tokens or lines
					chunkStartLineOffset = 0
					chunkEndLineOffset = 0
				}

				chunkStartLine := block.StartLine + chunkStartLineOffset
				chunkEndLine := block.StartLine + chunkEndLineOffset

				// Adjust line numbers to stay within the original block's lines
				if chunkStartLine < block.StartLine {
					chunkStartLine = block.StartLine
				}
				if chunkEndLine > block.EndLine {
					chunkEndLine = block.EndLine
				}
				// Ensure start line does not exceed end line
				if chunkStartLine > chunkEndLine {
					chunkStartLine = chunkEndLine
				}

				// Ensure chunk has content and positive size before adding (especially important with tokenization issues)
				if len(chunkContent) == 0 || len(chunkTokens) == 0 {
					// If at the very end and no content, might be unavoidable. Otherwise skip.
					if endTokenIndex < tokensLen {
						i = endTokenIndex // Move past this empty chunk
						continue
					}
				}

				finalBlocks = append(finalBlocks, types.CodeBlock{
					Content:      chunkContent,
					FilePath:     block.FilePath,
					StartLine:    chunkStartLine,
					EndLine:      chunkEndLine,
					ParentFunc:   block.ParentFunc,
					ParentClass:  block.ParentClass,
					OriginalSize: len(chunkContent), // Size of the chunk in bytes
					TokenCount:   len(chunkTokens),  // Token count of the chunk
				})

				// Move to the next window position based on tokens
				if endTokenIndex == tokensLen {
					break // Reached the end of the content
				}

				// Calculate the start of the next chunk with overlap
				nextStartTokenIndex := endTokenIndex - overlapTokens
				if nextStartTokenIndex < i {
					nextStartTokenIndex = i // Safeguard to prevent moving backwards
				}
				i = nextStartTokenIndex
			}
		} else {
			// Block is within token size limits, add as is
			// Ensure TokenCount is set even if no sliding window is applied
			if block.TokenCount == 0 && len(block.Content) > 0 { // If token count wasn't set due to error earlier
				tokens, _, err := s.tokenizer.Encode(block.Content)
				if err == nil {
					block.TokenCount = len(tokens)
				} else {
					// Log error but proceed with byte count as token count proxy if needed elsewhere
					s.logger.Errorf("Failed to encode block content to tokens for %s (lines %d-%d) for final block: %v", block.FilePath, block.StartLine, block.EndLine, err)
					block.TokenCount = len(block.Content) // Still use byte count as a very rough fallback
				}
			}
			finalBlocks = append(finalBlocks, block)
		}
	}

	return finalBlocks, nil
}

// Close releases resources held by the CodeSplitter and its parsers.
func (s *codeSplitter) Close() {
	for _, p := range s.parsers {
		p.Close()
	}
	// Tokenizer does not have a Close method based on current interface,
	// but if it did, we would call it here.
}
