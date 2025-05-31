package scip

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"path/filepath"
)

const (
	placeholderSourcePath = "__sourcePath__"
	placeholderOutputPath = "__outputPath__"
)

// IndexGenerator represents the SCIP index generator
type IndexGenerator struct {
	codebaseStore codebase.Store
	config        *Config
}

// NewIndexGenerator creates a new SCIP index generator
func NewIndexGenerator(config *Config, codebaseStore codebase.Store) *IndexGenerator {
	return &IndexGenerator{
		config:        config,
		codebaseStore: codebaseStore,
	}
}

// Generate generates a SCIP index for the given codebase
func (g *IndexGenerator) Generate(ctx context.Context, codebasePath string) error {

	if err := g.codebaseStore.MkDirs(ctx, codebasePath, types.CodebaseIndexDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	index, build, err := g.detectLanguageAndTool(ctx, codebasePath)
	if err != nil {
		return err
	}

	placeHolders := map[string]string{
		placeholderSourcePath: codebasePath,
		placeholderOutputPath: indexOutputDir(codebasePath),
	}

	executor, err := newCommandExecutor(ctx, codebasePath, index, build, placeHolders)
	if err != nil {
		return err
	}

	if err = executor.Execute(); err != nil {
		return err
	}

	return nil
}

// detectLanguageAndTool detects the language and tool for a repository
func (c *IndexGenerator) detectLanguageAndTool(ctx context.Context, codebasePath string) (*IndexTool, *BuildTool, error) {
	// Find language config
	for _, lang := range c.config.Languages {
		for _, file := range lang.DetectionFiles {
			if fileInfo, err := c.codebaseStore.Stat(ctx, codebasePath, filepath.Join(codebasePath, file)); err == nil {
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

	return nil, nil, fmt.Errorf("no matching language configuration found")
}

// Cleanup removes the output directory and releases any locks
func (e *IndexGenerator) Cleanup() error {
	return nil
}

func indexOutputDir(codebasePath string) string {
	return filepath.Join(codebasePath, types.CodebaseIndexDir)
}
