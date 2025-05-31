package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zeromicro/go-zero/core/logx"
)

// Indexer handles the generation of SCIP indices for codebases
type Indexer struct {
	configPath string
}

// NewIndexer creates a new Indexer instance
func NewIndexer(configPath string) *Indexer {
	return &Indexer{
		configPath: configPath,
	}
}

// GenerateIndex generates a SCIP index for the specified repository
func (i *Indexer) GenerateIndex(ctx context.Context, repoPath string) error {
	// Ensure the repository path exists
	if _, err := os.Stat(repoPath); err != nil {
		logx.Errorf("Repository path does not exist: %v", err)
		return fmt.Errorf("repository path does not exist: %w", err)
	}

	// Create command executor
	executor, err := NewCommandExecutor(repoPath, i.configPath)
	if err != nil {
		logx.Errorf("Failed to create command executor: %v", err)
		return fmt.Errorf("failed to create command executor: %w", err)
	}

	// Generate the index
	logx.Infof("Generating index for repository: %s", repoPath)
	if err := executor.GenerateIndex(ctx); err != nil {
		logx.Errorf("Failed to generate index: %v", err)
		return fmt.Errorf("failed to generate index: %w", err)
	}

	logx.Infof("Index generated successfully")
	return nil
}

// GetIndexPath returns the path where the index file should be located
func (i *Indexer) GetIndexPath(repoPath string) string {
	return filepath.Join(repoPath, ".codebase_index", "index.scip")
}

// Cleanup removes any temporary files created during indexing
func (i *Indexer) Cleanup(repoPath string) error {
	outputDir := filepath.Join(repoPath, ".codebase_index")
	if _, err := os.Stat(outputDir); err == nil {
		logx.Infof("Cleaning up output directory: %s", outputDir)
		if err := os.RemoveAll(outputDir); err != nil {
			logx.Errorf("Failed to clean up output directory: %v", err)
			return fmt.Errorf("failed to clean up output directory: %w", err)
		}
	}
	return nil
} 