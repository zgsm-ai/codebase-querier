package scip

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the SCIP configuration
type Config struct {
	Languages []*LanguageConfig `yaml:"languages"`
}

// LanguageConfig represents a language configuration
type LanguageConfig struct {
	Name           string        `yaml:"name"`
	DetectionFiles []string      `yaml:"detection_files"`
	BuildTools     []*BuildTool  `yaml:"build_tools,omitempty"`
	Tools          []*ToolConfig `yaml:"tools"`
}

// BuildTool represents a build tool configuration
type BuildTool struct {
	Name           string     `yaml:"name"`
	DetectionFiles []string   `yaml:"detection_files"`
	Priority       int        `yaml:"priority"`
	BuildCommands  []*Command `yaml:"build_commands"`
}

// ToolConfig represents a tool configuration
type ToolConfig struct {
	Name     string     `yaml:"name"`
	Commands []*Command `yaml:"commands"`
}

// Command represents a command configuration
type Command struct {
	Base string   `yaml:"base"`
	Args []string `yaml:"args"`
}

// LoadConfig loads the SCIP configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("CONFIG_ERROR: failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("CONFIG_ERROR: failed to parse config file: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Languages) == 0 {
		return fmt.Errorf("CONFIG_ERROR: no languages configured")
	}

	for _, lang := range c.Languages {
		if lang.Name == "" {
			return fmt.Errorf("CONFIG_ERROR: language name is required")
		}
		if len(lang.DetectionFiles) == 0 {
			return fmt.Errorf("CONFIG_ERROR: detection files are required for language %s", lang.Name)
		}
		if len(lang.Tools) == 0 {
			return fmt.Errorf("CONFIG_ERROR: tools are required for language %s", lang.Name)
		}
	}

	return nil
}

// DetectLanguageAndTool detects the language and tool for a repository
func (c *Config) DetectLanguageAndTool(repoPath string) (string, string, error) {
	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("codebase does not exist: %w", err)
	}

	// Find language config
	for _, lang := range c.Languages {
		for _, file := range lang.DetectionFiles {
			if _, err := os.Stat(filepath.Join(repoPath, file)); err == nil {
				return lang.Name, lang.Tools[0].Name, nil
			}
		}
	}

	return "", "", fmt.Errorf("no matching language configuration found")
}

// GenerateCommand generates a command for a language and tool
func (c *Config) GenerateCommand(sourcePath, language, tool string) (*Command, error) {
	// Find language config
	var langConfig *LanguageConfig
	for _, lang := range c.Languages {
		if lang.Name == language {
			langConfig = lang
			break
		}
	}
	if langConfig == nil {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Find tool config
	var toolConfig *ToolConfig
	for _, t := range langConfig.Tools {
		if t.Name == tool {
			toolConfig = t
			break
		}
	}
	if toolConfig == nil {
		return nil, fmt.Errorf("unsupported tool: %s", tool)
	}

	// Generate command
	cmd := toolConfig.Commands[0]
	args := make([]string, len(cmd.Args))
	for i, arg := range cmd.Args {
		// Trim spaces from arguments
		arg = strings.TrimSpace(arg)
		args[i] = strings.ReplaceAll(arg, "__sourcePath__", sourcePath)
		args[i] = strings.ReplaceAll(args[i], "__outputPath__", filepath.Join(sourcePath, ".codebase_index"))
	}

	return &Command{
		Base: strings.TrimSpace(cmd.Base),
		Args: args,
	}, nil
}
