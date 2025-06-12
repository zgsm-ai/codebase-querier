package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYaml  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			configYaml: `
languages:
  - name: go
    detection_files:
      - go.mod
      - go.sum
    index:
      name: scip-go
      commands:
        - base: scip-go
          args: ["index"]
`,
			wantErr: false,
		},
		{
			name:        "non-existent file",
			configYaml:  "",
			wantErr:     true,
			errContains: "failed to read config file",
		},
		{
			name: "invalid yaml",
			configYaml: `
languages:
  - name: go
    detection_files: invalid
`,
			wantErr:     true,
			errContains: "failed to parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if tt.configYaml != "" {
				err := os.WriteFile(configPath, []byte(tt.configYaml), 0644)
				assert.NoError(t, err)
			}

			config := MustLoadCodegraphConfig(configPath)
			if tt.wantErr {
				assert.Nil(t, config)
			} else {
				assert.NotNil(t, config)
				assert.NotEmpty(t, config.Languages)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *CodegraphConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &CodegraphConfig{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
						Index: &IndexTool{
							Name: "scip-go",
							Commands: []*Command{
								{
									Base: "scip-go",
									Args: []string{"index"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "empty languages",
			config:      &CodegraphConfig{Languages: []*LanguageConfig{}},
			wantErr:     true,
			errContains: "no languages configured",
		},
		{
			name: "missing language name",
			config: &CodegraphConfig{
				Languages: []*LanguageConfig{
					{
						DetectionFiles: []string{"go.mod"},
						Index: &IndexTool{
							Name: "scip-go",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "language name is required",
		},
		{
			name: "missing detection files",
			config: &CodegraphConfig{
				Languages: []*LanguageConfig{
					{
						Name: "go",
						Index: &IndexTool{
							Name: "scip-go",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "detection files are required",
		},
		{
			name: "missing index tool",
			config: &CodegraphConfig{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
					},
				},
			},
			wantErr:     true,
			errContains: "index are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
