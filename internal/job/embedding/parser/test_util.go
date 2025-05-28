package parser

import (
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"strings"
)

// Helper function to find a block by a substring in its content
func findBlockByContentSubstring(blocks []types.CodeBlock, substring string) *types.CodeBlock {
	for _, block := range blocks {
		if strings.Contains(block.Content, substring) {
			return &block
		}
	}
	return nil
}

// Helper function to count blocks by content substring
func countBlocksByContentSubstring(blocks []types.CodeBlock, substring string) int {
	count := 0
	for _, block := range blocks {
		if strings.Contains(block.Content, substring) {
			count++
		}
	}
	return count
}
