package embedding

import (
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding/lang"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"path/filepath"
	"slices"

	"github.com/tiktoken-go/tokenizer"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type CodeSplitter struct {
	languages         []*lang.LanguageConfig // Language-specific configuration
	tokenizer         tokenizer.Codec
	maxTokensPerChunk int
	overlapTokens     int
}

type SplitOption func(*CodeSplitter)

// NewCodeSplitter creates a new generic parser with the given config.
func NewCodeSplitter(maxTokensPerChunk, overlapTokens int) (*CodeSplitter, error) {
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
		languages:         languages,
		tokenizer:         codec,
		maxTokensPerChunk: maxTokensPerChunk,
		overlapTokens:     overlapTokens,
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
		// Use the language-specific processor to process the match
		defInfos, err := language.Processor.ProcessMatch(match, root, contentBytes)
		if err != nil {
			// Log the error and continue processing other matches
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

			if tokenCount > p.maxTokensPerChunk {
				subChunks := p.splitIntoChunks(content, codeFile.Path, startLine,
					endLine, defInfo.ParentFunc, defInfo.ParentClass)
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

// splitIntoChunks applies a sliding window to a large content string.
// Updated to populate ParentFunc and ParentClass.
func (p *CodeSplitter) splitIntoChunks(
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
		chunkStartLine := startLine + utils.CountLines(contentBytes[:i])
		chunkEndLine := chunkStartLine + utils.CountLines([]byte(chunkContent)) - 1
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
