package lang

import (
	"errors"
	"os"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// CodeParser defines the interface for types.Language-specific code parsing.
type CodeParser interface {
	Split(codeFile *types.CodeFile, maxTokensPerChunk, overlapTokens int) ([]*types.CodeChunk, error)

	Parse(codeFile *types.CodeFile) (*sitter.Node, error)

	Close()
}

func loadQuery(lang *sitter.Language, queryFilePath string) (string, error) {
	// Read the query file
	queryContent, err := os.ReadFile(queryFilePath)
	if err != nil {
		return "", err
	}
	queryStr := string(queryContent)
	// Validate query early
	_, queryErr := sitter.NewQuery(lang, queryStr)
	if queryErr != nil {
		return "", err
	}
	return queryStr, nil
}

func doParse(codeFile *types.CodeFile, p *sitter.Parser) (*sitter.Node, error) {
	if p == nil {
		return nil, errors.New("parser is not properly initialized or has been closed")
	}

	tree := p.Parse([]byte(codeFile.Content), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}

	// We don't close the tree here because the caller is expected to use it.
	// The caller is responsible for closing the returned *sitter.Node which will close the tree.
	return tree.RootNode(), nil
}

// splitIntoChunks applies a sliding window to a large content string.
func splitIntoChunks(content string, filePath string, startLine, endLine int,
	parentFunc, parentClass string, maxTokensPerChunk, overlapTokens int) []*types.CodeChunk {
	var chunks []*types.CodeChunk
	contentBytes := []byte(content)
	contentLen := len(contentBytes)

	// Use character count as a simple token approximation for now
	maxLen := maxTokensPerChunk
	overlapLen := overlapTokens

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
			ParentFunc:   parentFunc,
			ParentClass:  parentClass,
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
func countLines(data []byte) int {
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		lines++
	}
	return lines
}
