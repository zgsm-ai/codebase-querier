package scip

// CommandArg represents a single argument for a command.
type CommandArg string

// Command represents a single command with a base and arguments.
type Command struct {
	Base string       `yaml:"base"`
	Args []CommandArg `yaml:"args"`
}

// BuildToolConfig represents the configuration for a specific build tool.
type BuildToolConfig struct {
	Name           string    `yaml:"name"`
	DetectionFiles []string  `yaml:"detection_files"`
	Priority       int       `yaml:"priority"` // Lower number means higher priority
	BuildCommands  []Command `yaml:"build_commands"`
}

// ToolConfig represents the configuration for a specific indexing tool.
type ToolConfig struct {
	Name     string    `yaml:"name"`
	Commands []Command `yaml:"commands"`
}

// LanguageConfig represents the configuration for a specific programming language.
type LanguageConfig struct {
	Name           string            `yaml:"name"`
	DetectionFiles []string          `yaml:"detection_files"`
	BuildTools     []BuildToolConfig `yaml:"build_tools"`
	Tools          []ToolConfig      `yaml:"tools"`
}

// SCIPConfig represents the root structure of the SCIP commands configuration.
type SCIPConfig struct {
	Languages []LanguageConfig `yaml:"languages"`
}
