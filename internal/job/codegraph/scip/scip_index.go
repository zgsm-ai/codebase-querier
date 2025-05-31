package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zeromicro/go-zero/core/logx"
)

type IndexGenerator interface {
	Generate(ctx context.Context, codebasePath string) (path string, err error)
}

// SCIPIndexGenerator handles the generation of SCIP indices
type SCIPIndexGenerator struct {
	config *Config
}

// NewSCIPIndexGenerator creates a new SCIPIndexGenerator
func NewSCIPIndexGenerator() (*SCIPIndexGenerator, error) {
	// Load configuration
	config, err := LoadConfig("scripts/scip_commands.yaml")
	if err != nil {
		return nil, NewError(ErrCodeConfig, "failed to load SCIP configuration", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewError(ErrCodeConfig, "invalid SCIP configuration", err)
	}

	return &SCIPIndexGenerator{
		config: config,
	}, nil
}

// Generate generates a SCIP index for the given codebase
func (g *SCIPIndexGenerator) Generate(ctx context.Context, codebasePath string) (string, error) {
	// Verify codebase path exists
	if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
		return "", NewError(ErrCodeResource, "codebase path does not exist", err)
	}

	// Detect language
	langConfig, err := g.config.FindLanguageConfig(codebasePath)
	if err != nil {
		return "", NewError(ErrCodeLanguage, "failed to detect language", err)
	}
	LogIndexInfo("Detected language: %s for codebase: %s", langConfig.Name, codebasePath)

	// Create output directory
	outputPath := filepath.Join(codebasePath, ".codebase_index")
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return "", NewError(ErrCodeResource, "failed to create output directory", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(outputPath)
	if err != nil {
		return "", NewError(ErrCodeResource, "failed to create command executor", err)
	}
	defer executor.Cleanup()

	// Generate index
	if err := executor.GenerateIndex(ctx, codebasePath, langConfig); err != nil {
		return "", err
	}

	// Return the path to the generated index file
	indexPath := filepath.Join(outputPath, "index.scip")
	LogIndexInfo("Successfully generated SCIP index at: %s", indexPath)
	return indexPath, nil
}
