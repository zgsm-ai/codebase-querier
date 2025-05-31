package scip

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Load config from local file
	configPath := filepath.Join("scripts", "scip_commands.yaml")
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify basic config structure
	require.NotEmpty(t, config.Languages)

	// Verify some key languages are present
	var foundGo, foundJava, foundPython bool
	for _, lang := range config.Languages {
		switch lang.Name {
		case "go":
			foundGo = true
			require.Equal(t, []string{"go.mod"}, lang.DetectionFiles)
			require.Len(t, lang.Tools, 1)
			require.Equal(t, "scip-go", lang.Tools[0].Name)
		case "java":
			foundJava = true
			require.Equal(t, []string{"pom.xml", "build.gradle"}, lang.DetectionFiles)
			require.Len(t, lang.Tools, 1)
			require.Equal(t, "scip-java", lang.Tools[0].Name)
		case "python":
			foundPython = true
		}
	}

	require.True(t, foundGo, "Go language config not found")
	require.True(t, foundJava, "Java language config not found")
	require.True(t, foundPython, "Python language config not found")

	// Validate config
	err = config.Validate()
	require.NoError(t, err)
}

func TestConfig_Validate(t *testing.T) {
	// Load config from local file
	configPath := filepath.Join("scripts", "scip_commands.yaml")
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Test valid config
	err = config.Validate()
	require.NoError(t, err)

	// Test invalid configs
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "empty languages",
			config: &Config{
				Languages: []*LanguageConfig{},
			},
			wantErr: true,
		},
		{
			name: "missing language name",
			config: &Config{
				Languages: []*LanguageConfig{
					{
						DetectionFiles: []string{"go.mod"},
						Tools: []*ToolConfig{
							{
								Name: "scip-go",
								Commands: []*Command{
									{
										Base: "scip-go",
										Args: []string{"--project-root", "__sourcePath__"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_DetectLanguageAndTool(t *testing.T) {
	// Load config from local file
	configPath := filepath.Join("scripts", "scip_commands.yaml")
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Test with non-existent directory
	_, _, err = config.DetectLanguageAndTool("non-existent-repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "codebase does not exist")
}

func TestConfig_GenerateCommand(t *testing.T) {
	// Load config from local file
	configPath := filepath.Join("scripts", "scip_commands.yaml")
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Test command generation
	cmd, err := config.GenerateCommand("/path/to/source", "go", "scip-go")
	require.NoError(t, err)
	assert.Equal(t, "scip-go", cmd.Base)
	assert.Contains(t, cmd.Args, "--project-root")
	assert.Contains(t, cmd.Args, "/path/to/source")
	assert.Contains(t, cmd.Args, "--output")
	assert.Contains(t, cmd.Args, "/path/to/source/.codebase_index/index.scip")

	// Test with invalid language
	_, err = config.GenerateCommand("/path/to/source", "invalid", "scip-go")
	assert.Error(t, err)

	// Test with invalid tool
	_, err = config.GenerateCommand("/path/to/source", "go", "invalid")
	assert.Error(t, err)
}
