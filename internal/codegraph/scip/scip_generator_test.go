package scip

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestNewIndexGenerator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	config := &Config{}

	generator := NewIndexGenerator(config, store)
	assert.NotNil(t, generator)
	assert.Equal(t, config, generator.config)
	assert.Equal(t, store, generator.codebaseStore)
}

func TestIndexGenerator_Generate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create test directory under /tmp
	testDir := filepath.Join("/tmp", "test")
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Determine the echo command based on OS
	var echoCmd string
	var echoArgs []string
	if runtime.GOOS == "windows" {
		echoCmd = "cmd"
		echoArgs = []string{"/c", "echo", "scip-go"}
	} else {
		echoCmd = "echo"
		echoArgs = []string{"scip-go"}
	}

	tests := []struct {
		name         string
		codebasePath string
		setupMock    func(*mocks.MockStore)
		config       *Config
		wantErr      bool
		errContains  string
	}{
		{
			name:         "successful generation",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), testDir, types.CodebaseIndexDir).
					Return(nil)
				m.EXPECT().
					Stat(gomock.Any(), testDir, "go.mod").
					Return(&types.FileInfo{IsDir: false}, nil)
			},
			config: &Config{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
						Index: &IndexTool{
							Name: "scip-go",
							Commands: []*Command{
								{
									Base: echoCmd,
									Args: echoArgs,
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "failed to create output directory",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), testDir, types.CodebaseIndexDir).
					Return(assert.AnError)
			},
			config: &Config{
				Languages: []*LanguageConfig{},
			},
			wantErr:     true,
			errContains: "failed to create codebase index directory",
		},
		{
			name:         "no matching language configuration",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), testDir, types.CodebaseIndexDir).
					Return(nil)
				m.EXPECT().
					Stat(gomock.Any(), testDir, "go.mod").
					Return(nil, assert.AnError)
			},
			config: &Config{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
					},
				},
			},
			wantErr:     true,
			errContains: "no matching language configuration found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewMockStore(ctrl)
			tt.setupMock(store)

			generator := NewIndexGenerator(tt.config, store)
			err := generator.Generate(context.Background(), tt.codebasePath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIndexGenerator_DetectLanguageAndTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create test directory under /tmp
	testDir := filepath.Join("/tmp", "test")
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name         string
		codebasePath string
		setupMock    func(*mocks.MockStore)
		config       *Config
		wantIndex    *IndexTool
		wantBuild    *BuildTool
		wantErr      bool
		errContains  string
	}{
		{
			name:         "language found without build tool",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), testDir, "go.mod").
					Return(&types.FileInfo{IsDir: false}, nil)
			},
			config: &Config{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
						Index: &IndexTool{
							Name: "scip-go",
						},
					},
				},
			},
			wantIndex: &IndexTool{Name: "scip-go"},
			wantBuild: nil,
			wantErr:   false,
		},
		{
			name:         "language found with build tool",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), testDir, "go.mod").
					Return(&types.FileInfo{IsDir: false}, nil)
			},
			config: &Config{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
						BuildTools: []*BuildTool{
							{
								Name:           "go-build",
								DetectionFiles: []string{"go.mod"},
							},
						},
						Index: &IndexTool{
							Name: "scip-go",
						},
					},
				},
			},
			wantIndex: &IndexTool{Name: "scip-go"},
			wantBuild: &BuildTool{
				Name:           "go-build",
				DetectionFiles: []string{"go.mod"},
			},
			wantErr: false,
		},
		{
			name:         "no language found",
			codebasePath: testDir,
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), testDir, "go.mod").
					Return(nil, assert.AnError)
			},
			config: &Config{
				Languages: []*LanguageConfig{
					{
						Name:           "go",
						DetectionFiles: []string{"go.mod"},
					},
				},
			},
			wantIndex:   nil,
			wantBuild:   nil,
			wantErr:     true,
			errContains: "no matching language configuration found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := mocks.NewMockStore(ctrl)
			tt.setupMock(store)

			generator := NewIndexGenerator(tt.config, store)
			indexTool, buildTool, err := generator.detectLanguageAndTool(context.Background(), tt.codebasePath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, indexTool)
				assert.Nil(t, buildTool)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIndex, indexTool)
				assert.Equal(t, tt.wantBuild, buildTool)
			}
		})
	}
}

func TestIndexOutputDir(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		expected     string
	}{
		{
			name:         "simple path",
			codebasePath: filepath.Join("/tmp", "test"),
			expected:     filepath.Join("/tmp", "test", types.CodebaseIndexDir),
		},
		{
			name:         "nested path",
			codebasePath: filepath.Join("/tmp", "test", "nested", "path"),
			expected:     filepath.Join("/tmp", "test", "nested", "path", types.CodebaseIndexDir),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOutputDir(tt.codebasePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
