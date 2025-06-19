package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// CodegraphConfig represents the SCIP configuration
type CodegraphConfig struct {
	LogDir        string            `yaml:"log_dir"`
	RetentionDays int               `yaml:"retention_days"`
	Variables     map[string]string `yaml:"variables"`
	Languages     []*LanguageConfig `yaml:"languages"`
}

// LanguageConfig represents a language configuration
type LanguageConfig struct {
	Name           string       `yaml:"name"`
	DetectionFiles []string     `yaml:"detection_files"`
	BuildTools     []*BuildTool `yaml:"build_tools,omitempty"`
	Index          *IndexTool   `yaml:"index"`
}

// BuildTool represents a build tool configuration
type BuildTool struct {
	Name           string     `yaml:"name"`
	DetectionFiles []string   `yaml:"detection_files"`
	Priority       int        `yaml:"priority"`
	Commands       []*Command `yaml:"build_commands"`
}

// IndexTool represents a tool configuration
type IndexTool struct {
	Name     string     `yaml:"name"`
	Commands []*Command `yaml:"commands"`
}

// Command represents a command configuration
type Command struct {
	Base string   `yaml:"base"`
	Args []string `yaml:"args"`
	Env  []string `yaml:"env"`
}

// MustLoadCodegraphConfig loads the SCIP configuration from a file
func MustLoadCodegraphConfig(path string) *CodegraphConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("CONFIG_ERROR: failed to read config file: %v", err))
	}

	var config CodegraphConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(fmt.Sprintf("CONFIG_ERROR: failed to parse config file: %v", err))
	}

	return &config
}

// Validate validates the configuration
func (c *CodegraphConfig) Validate() error {
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
		if lang.Index == nil {
			return fmt.Errorf("CONFIG_ERROR: index are required for language %s", lang.Name)
		}
	}

	return nil
}
