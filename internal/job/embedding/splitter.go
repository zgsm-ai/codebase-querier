package embedding

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/lang"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	"github.com/tiktoken-go/tokenizer"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
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

func (p *CodeSplitter) getLanguageConfigByExt(ext string) *lang.LanguageConfig {
	for _, c := range p.languages {
		if slices.Contains(c.SupportedExts, ext) {
			return c
		}
	}
	return nil
}

// Split splits the code content into chunks based on the LanguageConfig.
func (p *CodeSplitter) Split(codeFile *types.CodeFile) ([]*types.CodeChunk, error) {
	// Extract file extension
	ext := filepath.Ext(codeFile.Path)
	if ext == "" {
		return nil, fmt.Errorf("file %s has no extension, cannot determine language", codeFile.Path)
	}
	language := p.getLanguageConfigByExt(ext)
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
	query, queryErr := sitter.NewQuery(language.SitterLanguage, language.Query)
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

// splitFuncWithSlidingWindow: 对超长函数体做滑动窗口切分，chunk包含完整函数头，token计数准确，重叠token数准确
func (p *CodeSplitter) splitFuncWithSlidingWindow(
	content string,
	filePath string,
	funcStartLine int,
	parentFunc, parentClass string,
) []*types.CodeChunk {
	var chunks []*types.CodeChunk
	maxTokens := p.splitOptions.MaxTokensPerChunk
	overlapTokens := p.splitOptions.SlidingWindowOverlapTokens

	// 按行切分，保证每个chunk都包含完整的函数头
	lines := utils.SplitLines(content)
	// 计算每行的token数
	lineTokens := make([]int, len(lines))
	for i, line := range lines {
		lineTokens[i] = p.countToken(line)
	}
	// 滑动窗口切分
	for start := 0; start < len(lines); {
		tokens := 0
		end := start
		for end < len(lines) && tokens+lineTokens[end] <= maxTokens {
			tokens += lineTokens[end]
			end++
		}
		if end == start {
			// 单行就超限，强制切分
			end = start + 1
			tokens = lineTokens[start]
		}
		chunkLines := lines[start:end]
		chunkContent := utils.JoinLines(chunkLines)
		chunk := &types.CodeChunk{
			Content:      chunkContent,
			FilePath:     filePath,
			StartLine:    funcStartLine + start,
			EndLine:      funcStartLine + end - 1,
			ParentFunc:   parentFunc,
			ParentClass:  parentClass,
			OriginalSize: len(chunkContent),
			TokenCount:   p.countToken(chunkContent),
		}
		chunks = append(chunks, chunk)
		if end == len(lines) {
			break
		}
		// 下一个窗口起点，重叠overlapTokens
		// 计算重叠行数
		overlap := 0
		for i := end - 1; i >= start; i-- {
			overlap += lineTokens[i]
			if overlap >= overlapTokens {
				start = i
				break
			}
			if i == start {
				start = end - 1 // 至少重叠一行
			}
		}
	}
	return chunks
}
