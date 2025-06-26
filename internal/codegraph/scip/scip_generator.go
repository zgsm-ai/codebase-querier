package scip

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"sort"
	"time"
)

const (
	placeholderSourcePath = "__sourcePath__"
	placeholderOutputPath = "__outputPath__"
	indexFileName         = "index.scip"
)

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
	if build.Name != types.EmptyString {
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
func (c *IndexGenerator) DetectLanguageAndTool(ctx context.Context, codebasePath string) (config.IndexTool, config.BuildTool, error) {
	// 通过Walk统计文件频率
	dominantLanguage, err := c.codebaseStore.InferLanguage(ctx, codebasePath)
	if err != nil {
		return config.IndexTool{}, config.BuildTool{}, fmt.Errorf("scip generator infer language error:%w", err)
	}
	if dominantLanguage == types.EmptyString {
		return config.IndexTool{}, config.BuildTool{}, fmt.Errorf("scip generator infer language is empty, codebase path is %s", codebasePath)
	}
	indexLanguage := languageIndexToolMapping(dominantLanguage)
	tracer.WithTrace(ctx).Infof("scip_generator inferred language is %s, mapping to %s", dominantLanguage, indexLanguage)
	var index config.ScipIndexConfig
	// 查找对应的语言配置
	for _, conf := range c.config.Languages {
		if conf.Name == string(indexLanguage) {
			index = conf
		}
	}
	if index.Name == types.EmptyString {
		return config.IndexTool{}, config.BuildTool{}, fmt.Errorf("no matching language index config found")
	}
	if len(index.BuildTools) == 0 {
		return index.Index, config.BuildTool{}, nil
	}

	// 按优先级排序构建工具
	sort.Slice(index.BuildTools, func(i, j int) bool {
		return index.BuildTools[i].Priority < index.BuildTools[j].Priority
	})

	for _, tool := range index.BuildTools {
		for _, detectFile := range tool.DetectionFiles {
			if _, err := c.codebaseStore.Stat(ctx, codebasePath, detectFile); err == nil {
				return index.Index, tool, nil
			}
		}
	}

	return config.IndexTool{}, config.BuildTool{}, fmt.Errorf("no matching language index config found")
}

// Cleanup removes the output directory and releases any locks
func (e *IndexGenerator) Cleanup() error {
	return nil
}

func indexOutputDir(codebasePath string) string {
	return filepath.Join(codebasePath, types.CodebaseIndexDir)
}

func languageIndexToolMapping(lang parser.Language) parser.Language {
	switch lang {
	case parser.TSX:
		return parser.TypeScript
	default:
		return lang
	}
}
