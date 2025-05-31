package scip

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v2"
)

// Config represents the SCIP configuration
type Config struct {
	Languages []LanguageConfig `yaml:"languages"`
}

// LanguageConfig represents a language configuration
type LanguageConfig struct {
	Name            string       `yaml:"name"`
	DetectionFiles  []string     `yaml:"detection_files"`
	BuildTools      []BuildTool  `yaml:"build_tools"`
	Tools           []ToolConfig `yaml:"tools"`
}

// BuildTool represents a build tool configuration
type BuildTool struct {
	Name           string    `yaml:"name"`
	DetectionFiles []string  `yaml:"detection_files"`
	Priority       int       `yaml:"priority"`
	BuildCommands  []Command `yaml:"build_commands"`
}

// ToolConfig represents a tool configuration
type ToolConfig struct {
	Name     string    `yaml:"name"`
	Commands []Command `yaml:"commands"`
}

// Command represents a command configuration
type Command struct {
	Base string   `yaml:"base"`
	Args []string `yaml:"args"`
}

// LoadConfig loads the SCIP configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, NewError(ErrCodeConfig, "failed to read config file", err)
	}

	// Parse config
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, NewError(ErrCodeConfig, "failed to parse config file", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.Languages) == 0 {
		return NewError(ErrCodeConfig, "no languages configured", nil)
	}

	for _, lang := range c.Languages {
		if err := lang.Validate(); err != nil {
			return NewError(ErrCodeConfig, fmt.Sprintf("invalid language config for %s", lang.Name), err)
		}
	}

	return nil
}

// Validate validates a language configuration
func (l *LanguageConfig) Validate() error {
	if l.Name == "" {
		return fmt.Errorf("language name is required")
	}

	if len(l.DetectionFiles) == 0 {
		return fmt.Errorf("detection files are required")
	}

	if len(l.Tools) == 0 {
		return fmt.Errorf("tools are required")
	}

	for _, tool := range l.Tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("invalid tool config for %s: %v", tool.Name, err)
		}
	}

	for _, buildTool := range l.BuildTools {
		if err := buildTool.Validate(); err != nil {
			return fmt.Errorf("invalid build tool config for %s: %v", buildTool.Name, err)
		}
	}

	return nil
}

// Validate validates a tool configuration
func (t *ToolConfig) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	if len(t.Commands) == 0 {
		return fmt.Errorf("commands are required")
	}

	for _, cmd := range t.Commands {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("invalid command config: %v", err)
		}
	}

	return nil
}

// Validate validates a build tool configuration
func (b *BuildTool) Validate() error {
	if b.Name == "" {
		return fmt.Errorf("build tool name is required")
	}

	if len(b.DetectionFiles) == 0 {
		return fmt.Errorf("detection files are required")
	}

	for _, cmd := range b.BuildCommands {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("invalid build command config: %v", err)
		}
	}

	return nil
}

// Validate validates a command configuration
func (c *Command) Validate() error {
	if c.Base == "" {
		return fmt.Errorf("command base is required")
	}
	return nil
}

// FindLanguageConfig finds the language configuration for a codebase
func (c *Config) FindLanguageConfig(codebasePath string) (*LanguageConfig, error) {
	for _, lang := range c.Languages {
		if found, err := findDetectionFile(codebasePath, lang.DetectionFiles); err == nil && found != "" {
			return &lang, nil
		}
	}
	return nil, NewError(ErrCodeLanguage, "no matching language configuration found", nil)
}

// FindBuildTool finds the build tool configuration for a language
func (l *LanguageConfig) FindBuildTool(codebasePath string) (*BuildTool, error) {
	// Sort build tools by priority
	sort.Slice(l.BuildTools, func(i, j int) bool {
		return l.BuildTools[i].Priority < l.BuildTools[j].Priority
	})

	for _, tool := range l.BuildTools {
		if found, err := findDetectionFile(codebasePath, tool.DetectionFiles); err == nil && found != "" {
			return &tool, nil
		}
	}

	return nil, NewError(ErrCodeBuildTool, "no matching build tool configuration found", nil)
}

// findDetectionFile finds a detection file in the codebase
func findDetectionFile(codebasePath string, detectionFiles []string) (string, error) {
	for _, pattern := range detectionFiles {
		matches, err := filepath.Glob(filepath.Join(codebasePath, pattern))
		if err != nil {
			return "", NewError(ErrCodeResource, "failed to search for detection files", err)
		}
		if len(matches) > 0 {
			return matches[0], nil
		}
	}
	return "", nil
}
