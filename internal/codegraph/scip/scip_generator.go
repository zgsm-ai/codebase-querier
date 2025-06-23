package scip

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	placeholderSourcePath = "__sourcePath__"
	placeholderOutputPath = "__outputPath__"
	indexFileName         = "index.scip"
	// 基础限制
	defaultMaxFiles   = 500 // 默认最大文件数
	minFilesToAnalyze = 50  // 最少需要分析的文件数
)

var maxFileReached = errors.New("max files reached")

// IndexGenerator represents the SCIP index generator
type IndexGenerator struct {
	codebaseStore codebase.Store
	config        *config.CodegraphConfig
	cmdLogger     *tracer.CmdLogger
}

// NewIndexGenerator creates a new SCIP index generator
func NewIndexGenerator(cmdLogger *tracer.CmdLogger, config *config.CodegraphConfig, codebaseStore codebase.Store) *IndexGenerator {
	return &IndexGenerator{
		cmdLogger:     cmdLogger,
		config:        config,
		codebaseStore: codebaseStore,
	}
}

// Generate generates a SCIP index for the given codebase
func (g *IndexGenerator) Generate(ctx context.Context, codebasePath string) error {

	if err := g.codebaseStore.MkDirs(ctx, codebasePath, types.CodebaseIndexDir); err != nil {
		return fmt.Errorf("failed to create codebase index directory: %w", err)
	}

	executor, err := g.InitCommandExecutor(ctx, g.cmdLogger, codebasePath)
	if err != nil {
		return err
	}
	defer executor.Close()
	if err = executor.Execute(ctx); err != nil {
		return err
	}

	return nil
}

func (g *IndexGenerator) InitCommandExecutor(ctx context.Context, cmdLogger *tracer.CmdLogger, codebasePath string) (*CommandExecutor, error) {
	start := time.Now()
	index, build, err := g.DetectLanguageAndTool(ctx, codebasePath)
	if err != nil {
		return nil, fmt.Errorf("scip_generator failed to detect [%s] launguage index tool, err: %w", codebasePath, err)
	}
	buildToolName := types.EmptyString
	if build != nil {
		buildToolName = build.Name
	}
	tracer.WithTrace(ctx).Infof("scip_generator detect language successfully, cost %d ms. index tool is [%s], build tool is [%s]",
		time.Since(start).Milliseconds(), index.Name, buildToolName)

	placeHolders := map[string]string{
		placeholderSourcePath: codebasePath,
		placeholderOutputPath: indexOutputDir(codebasePath),
	}
	if len(g.config.Variables) > 0 {
		for k, v := range g.config.Variables {
			placeHolders[k] = v
		}
	}

	return newCommandExecutor(cmdLogger, codebasePath, index, build, placeHolders)
}

// DetectLanguageAndTool detects the language and tool for a repository
func (c *IndexGenerator) DetectLanguageAndTool(ctx context.Context, codebasePath string) (*config.IndexTool, *config.BuildTool, error) {
	// 通过Walk统计文件频率
	languageStats := make(map[string]int)
	analyzedFiles := 0
	start := time.Now()
	err := c.codebaseStore.Walk(ctx, codebasePath, types.EmptyString, func(walkCtx *codebase.WalkContext, reader io.ReadCloser) error {

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
	tracer.WithTrace(ctx).Infof("scip_generator detect language analyzed %d files, cost %d ms",
		analyzedFiles, time.Since(start).Milliseconds())
	if err != nil && !errors.Is(err, maxFileReached) {
		tracer.WithTrace(ctx).Errorf("failed to analyze codebase: %v", err)
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
