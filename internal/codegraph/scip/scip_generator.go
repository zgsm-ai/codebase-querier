package scip

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"path/filepath"
	"strings"

	"github.com/zgsm-ai/codebase-indexer/internal/config"

	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
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

	// Find language config
	for _, lang := range c.config.Languages {
		for _, file := range lang.DetectionFiles {
			if fileInfo, err := c.codebaseStore.Stat(ctx, codebasePath, file); err == nil {
				if fileInfo.IsDir {
					continue
				}
				if len(lang.BuildTools) == 0 {
					return lang.Index, nil, nil
				}
				for _, tool := range lang.BuildTools {
					if utils.SliceContains[string](tool.DetectionFiles, file) {
						return lang.Index, tool, nil
					}
				}
				return lang.Index, nil, nil
			}
		}
	}
	logx.Errorf("infer language and build tool failed for codebase:%s, try other way.", codebasePath)
	// First try to detect Python project
	isPython, err := c.isPythonProject(ctx, codebasePath)
	if err != nil {
		logx.Errorf("isPythonProject walk error: %v", err)
	}
	if isPython {
		logx.Errorf("found .py file in codedebase:%s, assume it as python project.", codebasePath)
		// Find Python config in languages
		for _, lang := range c.config.Languages {
			if lang.Name == "python" {
				return lang.Index, nil, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("no matching language configuration found")
}

// isPythonProject checks if the codebase contains any .py files
func (c *IndexGenerator) isPythonProject(ctx context.Context, codebasePath string) (bool, error) {
	hasPythonFile := false
	err := c.codebaseStore.Walk(ctx, codebasePath, "", func(walkCtx *codebase.WalkContext, reader io.ReadCloser) error {
		if strings.HasSuffix(walkCtx.RelativePath, ".py") {
			hasPythonFile = true
			return errors.New("found python file") // Use error to break the walk early
		}
		return nil
	}, codebase.WalkOptions{
		IgnoreError: true, // Ignore the error we use to break early
	})

	if err != nil && !hasPythonFile {
		return false, fmt.Errorf("failed to walk codebase: %w", err)
	}
	return hasPythonFile, nil
}

// Cleanup removes the output directory and releases any locks
func (e *IndexGenerator) Cleanup() error {
	return nil
}

func indexOutputDir(codebasePath string) string {
	return filepath.Join(codebasePath, types.CodebaseIndexDir)
}
