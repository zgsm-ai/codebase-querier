package scip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SCIPIndexGenerator represents the SCIP index generator
type SCIPIndexGenerator struct {
	config *Config
}

// NewSCIPIndexGenerator creates a new SCIP index generator
func NewSCIPIndexGenerator(config *Config) *SCIPIndexGenerator {
	return &SCIPIndexGenerator{config: config}
}

// Generate generates a SCIP index for the given codebase
func (g *SCIPIndexGenerator) Generate(ctx context.Context, codebasePath string) (string, error) {
	if err := g.validateCodebase(codebasePath); err != nil {
		return "", err
	}

	language, tool, err := g.config.DetectLanguageAndTool(codebasePath)
	if err != nil {
		return "", err
	}

	cmd, err := g.config.GenerateCommand(codebasePath, language, tool)
	if err != nil {
		return "", err
	}

	outputPath := filepath.Join(codebasePath, ".codebase_index")
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	executor, err := NewCommandExecutor(outputPath)
	if err != nil {
		return "", err
	}

	if err := executor.Execute(ctx, cmd); err != nil {
		return "", err
	}

	indexPath := filepath.Join(outputPath, "index.scip")
	return indexPath, nil
}

// GenerateConcurrent generates SCIP indices for multiple codebases concurrently
func (g *SCIPIndexGenerator) GenerateConcurrent(ctx context.Context, codebasePaths []string) error {
	if len(codebasePaths) == 0 {
		return fmt.Errorf("at least one codebase path is required")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(codebasePaths))

	for _, path := range codebasePaths {
		wg.Add(1)
		go func(codebasePath string) {
			defer wg.Done()
			if _, err := g.Generate(ctx, codebasePath); err != nil {
				errChan <- err
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to generate indices: %v", errors)
	}

	return nil
}

// validateCodebase validates the codebase path
func (g *SCIPIndexGenerator) validateCodebase(codebasePath string) error {
	if codebasePath == "" {
		return fmt.Errorf("codebase path is required")
	}

	if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
		return fmt.Errorf("codebase does not exist: %w", err)
	}

	return nil
}
