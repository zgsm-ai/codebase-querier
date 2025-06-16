package scip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zgsm-ai/codebase-indexer/internal/config"

	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	placeholderSourcePath = "__sourcePath__"
	placeholderOutputPath = "__outputPath__"
	indexFileName         = "index.scip"
	// 基础限制
	defaultMaxFiles   = 100             // 默认最大文件数
	minFilesToAnalyze = 20              // 最少需要分析的文件数
	maxAnalysisTime   = 2 * time.Second // 最大分析时间
)

var maxFileReached = errors.New("max files reached")

// IndexGenerator represents the SCIP index generator
type IndexGenerator struct {
	codebaseStore codebase.Store
	config        *config.CodegraphConfig
}

// NewIndexGenerator creates a new SCIP index generator
func NewIndexGenerator(config *config.CodegraphConfig, codebaseStore codebase.Store) *IndexGenerator {
	return &IndexGenerator{
		config:        config,
		codebaseStore: codebaseStore,
	}
}

// Generate generates a SCIP index for the given codebase
func (g *IndexGenerator) Generate(ctx context.Context, codebasePath string) error {

	if err := g.codebaseStore.MkDirs(ctx, codebasePath, types.CodebaseIndexDir); err != nil {
		return fmt.Errorf("failed to create codebase index directory: %w", err)
	}

	index, build, err := g.detectLanguageAndTool(ctx, codebasePath)
	if err != nil {
		return err
	}

	placeHolders := map[string]string{
		placeholderSourcePath: codebasePath,
		placeholderOutputPath: indexOutputDir(codebasePath),
	}
	if len(g.config.Variables) > 0 {
		for k, v := range g.config.Variables {
			placeHolders[k] = v
		}
	}

	executor, err := newCommandExecutor(ctx,
		codebasePath,
		index,
		build,
		g.config.LogDir,
		placeHolders)
	if err != nil {
		return err
	}

	if err = executor.Execute(); err != nil {
		return err
	}

	return nil
}

// detectLanguageAndTool detects the language and tool for a repository
func (c *IndexGenerator) detectLanguageAndTool(ctx context.Context, codebasePath string) (*config.IndexTool, *config.BuildTool, error) {

	// 通过Walk统计文件频率
	languageStats := make(map[string]int)
	analyzedFiles := 0
	startTime := time.Now()

	err := c.codebaseStore.Walk(ctx, codebasePath, types.EmptyString, func(walkCtx *codebase.WalkContext, reader io.ReadCloser) error {
		// 检查是否超时
		if time.Since(startTime) > maxAnalysisTime {
			return maxFileReached
		}

		// 如果已经分析了足够多的文件，且某个语言占比超过60%，可以提前结束
		if analyzedFiles >= minFilesToAnalyze {
			totalFiles := 0
			maxCount := 0
			for _, count := range languageStats {
				totalFiles += count
				if count > maxCount {
					maxCount = count
				}
			}
			if float64(maxCount)/float64(totalFiles) > 0.6 {
				return maxFileReached
			}
		}

		if analyzedFiles >= defaultMaxFiles {
			return maxFileReached
		}

		ext := filepath.Ext(walkCtx.RelativePath)
		if ext == "" {
			return nil
		}

		// 使用parser包中的语言配置
		langConfig, _ := parser.GetLangConfigByFilePath(walkCtx.RelativePath)
		if langConfig != nil {
			languageStats[string(langConfig.Language)]++
		} else {
			return nil
		}

		analyzedFiles++
		return nil
	}, codebase.WalkOptions{
		IgnoreError: true,
	})

	if err != nil && !errors.Is(err, maxFileReached) {
		logx.Errorf("failed to analyze codebase: %v", err)
	}

	// 5. 选择出现频率最高的语言
	var dominantLanguage string
	maxCount := 0
	for lang, count := range languageStats {
		if count > maxCount {
			maxCount = count
			dominantLanguage = lang
		}
	}

	var langConfig *config.LanguageConfig
	if dominantLanguage != "" {
		// 查找对应的语言配置
		for _, lang := range c.config.Languages {
			if lang.Name == dominantLanguage {
				langConfig = lang
			}
		}
	}
	if langConfig == nil {
		return nil, nil, fmt.Errorf("no matching language configuration found")
	}
	if len(langConfig.BuildTools) == 0 {
		return langConfig.Index, nil, nil
	}

	// 按优先级排序构建工具
	sort.Slice(langConfig.BuildTools, func(i, j int) bool {
		return langConfig.BuildTools[i].Priority < langConfig.BuildTools[j].Priority
	})

	for _, tool := range langConfig.BuildTools {
		for _, detectFile := range tool.DetectionFiles {
			if _, err := c.codebaseStore.Stat(ctx, codebasePath, detectFile); err == nil {
				return langConfig.Index, tool, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no matching language configuration found")
}

// Cleanup removes the output directory and releases any locks
func (e *IndexGenerator) Cleanup() error {
	return nil
}

func indexOutputDir(codebasePath string) string {
	return filepath.Join(codebasePath, types.CodebaseIndexDir)
}
