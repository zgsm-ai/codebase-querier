package embedding

import (
	"errors"
	"fmt"
	"github.com/tiktoken-go/tokenizer"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/lang"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
)

type CodeSplitter struct {
	languages    []*lang.LanguageConfig // Language-specific configuration
	tokenizer    tokenizer.Codec
	splitOptions SplitOptions
}

type SplitOptions struct {
	// when exceed MaxTokensPerChunk, split it with sliding window. it is also the windows size.
	MaxTokensPerChunk int
	// take effects when exceed MaxTokensPerChunk
	SlidingWindowOverlapTokens int
}

// NewCodeSplitter creates a new generic parser with the given config.
func NewCodeSplitter(splitOptions SplitOptions) (*CodeSplitter, error) {
	// Initialize the codec	// Initialize the codec
	codec, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec: %w", err)
	}

	languages, err := lang.GetLanguageConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to get languages config: %w", err)
	}

	return &CodeSplitter{
		languages:    languages,
		tokenizer:    codec,
		splitOptions: splitOptions,
	}, nil
}

// Split splits the code content into chunks based on the LanguageConfig.
func (p *CodeSplitter) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {
	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		return nil, fmt.Errorf("file %s has no extension, cannot determine language", codeFile.Path)
	}
	language := lang.GetLanguageConfigByExt(p.languages, ext)
	if language == nil {
		return nil, fmt.Errorf("cannot find language config by ext %s", ext)
	}

	parser := sitter.NewParser()
	if err := parser.SetLanguage(language.SitterLanguage); err != nil {
		return nil, fmt.Errorf("cannot init tree-sitter parser: %w", err)
	}
	defer parser.Close()

	tree := parser.Parse([]byte(codeFile.Content), nil)
	if tree == nil {
		return nil, errors.New("failed to parse code")
	}
	defer tree.Close()

	root := tree.RootNode()

	// Create Tree-sitter query from the config's query string
	query, queryErr := sitter.NewQuery(language.SitterLanguage, language.ChunkQuery)
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
		defInfos, err := language.Processor.ProcessMatch(match, root, contentBytes)
		if err != nil {
			continue
		}
		for _, defInfo := range defInfos {
			if defInfo == nil || defInfo.Node == nil {
				continue
			}
			content := defInfo.Node.Utf8Text(contentBytes)
			startLine := int(defInfo.Node.StartPosition().Row)
			endLine := int(defInfo.Node.EndPosition().Row)
			tokenCount := p.countToken(content)
			if tokenCount > p.splitOptions.MaxTokensPerChunk {
				subChunks := p.splitFuncWithSlidingWindow(content, codeFile.Path, startLine, defInfo.ParentFunc, defInfo.ParentClass)
				allChunks = append(allChunks, subChunks...)
			} else {
				chunk := &types.CodeChunk{
					Name:         defInfo.Name,
					Content:      content,
					FilePath:     codeFile.Path,
					StartLine:    startLine,
					EndLine:      endLine,
					OriginalSize: len(content),
					TokenCount:   tokenCount,
					ParentFunc:   defInfo.ParentFunc,
					ParentClass:  defInfo.ParentClass,
				}
				allChunks = append(allChunks, chunk)
			}
		}
	}
	return allChunks, nil
}

func (p *CodeSplitter) countToken(content string) int {
	tokenCount, err := p.tokenizer.Count(content)
	if err != nil {
		tokenCount = len(content)
	}
	return tokenCount
}

func (p *CodeSplitter) splitFuncWithSlidingWindow(
	content string,
	filePath string,
	funcStartLine int,
	parentFunc, parentClass string,
) []*types.CodeChunk {
	maxTokens := p.splitOptions.MaxTokensPerChunk
	overlapTokens := p.splitOptions.SlidingWindowOverlapTokens

	if maxTokens <= 0 || overlapTokens < 0 || overlapTokens >= maxTokens {
		return nil
	}

	_, tokens, err := p.tokenizer.Encode(content)
	if err != nil {
		return nil
	}
	totalTokens := len(tokens)
	if totalTokens == 0 {
		return nil
	}

	byteOffsets := make([]int, len(tokens)+1)
	currentOffset := 0
	for i, token := range tokens {
		byteOffsets[i] = currentOffset
		currentOffset += len(token)
	}
	byteOffsets[len(tokens)] = currentOffset

	chunks := make([]*types.CodeChunk, 0)
	startTokenIdx := 0
	chunkCount := 0

	for startTokenIdx < totalTokens {
		// 计算当前块结束位置（正常情况）
		endTokenIdx := startTokenIdx + maxTokens
		if endTokenIdx > totalTokens {
			endTokenIdx = totalTokens
		}
		currentTokens := endTokenIdx - startTokenIdx
		chunkCount++

		// 提取代码块
		startByte := byteOffsets[startTokenIdx]
		endByte := byteOffsets[endTokenIdx] - 1
		if endByte >= len(content) {
			endByte = len(content) - 1
		}
		chunkContent := content[startByte : endByte+1]
		startLine := funcStartLine + countLines(content[:startByte])
		endLine := startLine + countLines(chunkContent) - 1

		chunks = append(chunks, &types.CodeChunk{
			Content:    chunkContent,
			FilePath:   filePath,
			StartLine:  startLine,
			EndLine:    endLine,
			TokenCount: currentTokens,
		})

		if endTokenIdx >= totalTokens {
			break
		}

		// **优化最后一块逻辑**
		if chunkCount < (totalTokens+maxTokens-1)/maxTokens-1 {
			// 非最后一块，使用固定重叠
			startTokenIdx = endTokenIdx - overlapTokens
			if startTokenIdx < chunkCount*(maxTokens-overlapTokens) { // 防止回退
				startTokenIdx = chunkCount * (maxTokens - overlapTokens)
			}
		} else {
			// 最后一块，动态调整重叠
			remainingTokens := totalTokens - endTokenIdx
			if remainingTokens > 0 {
				// 允许重叠量减少为 maxTokens - remainingTokens，但至少为0
				newOverlap := maxTokens - remainingTokens
				if newOverlap < 0 {
					newOverlap = 0
				}
				startTokenIdx = endTokenIdx - newOverlap
			} else {
				startTokenIdx = endTokenIdx
			}
		}

		// 防止索引越界
		if startTokenIdx < 0 {
			startTokenIdx = 0
		}
		if startTokenIdx >= endTokenIdx {
			startTokenIdx = endTokenIdx
		}
	}

	return chunks
}

// 辅助函数：计算字符串中的行数
func countLines(s string) int {
	if len(s) == 0 {
		return 0
	}
	count := 0
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count + 1 // 最后一行可能没有换行符
}
