package scip

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase/mocks"
	"path/filepath"
	"testing"

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
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), "/tmp/test", types.CodebaseIndexDir).
					Return(nil)
				m.EXPECT().
					Stat(gomock.Any(), "/tmp/test", "/tmp/test/go.mod").
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
									Base: "echo",
									Args: []string{"scip-go"},
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
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), "/tmp/test", types.CodebaseIndexDir).
					Return(assert.AnError)
			},
			config: &Config{
				Languages: []*LanguageConfig{},
			},
			wantErr:     true,
			errContains: "failed to create output directory",
		},
		{
			name:         "no matching language configuration",
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					MkDirs(gomock.Any(), "/tmp/test", types.CodebaseIndexDir).
					Return(nil)
				m.EXPECT().
					Stat(gomock.Any(), "/tmp/test", "/tmp/test/go.mod").
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
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), "/tmp/test", "/tmp/test/go.mod").
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
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), "/tmp/test", "/tmp/test/go.mod").
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
			codebasePath: "/tmp/test",
			setupMock: func(m *mocks.MockStore) {
				m.EXPECT().
					Stat(gomock.Any(), "/tmp/test", "/tmp/test/go.mod").
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
			index, build, err := generator.detectLanguageAndTool(context.Background(), tt.codebasePath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, index)
				assert.Nil(t, build)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIndex.Name, index.Name)
				if tt.wantBuild != nil {
					assert.Equal(t, tt.wantBuild.Name, build.Name)
				} else {
					assert.Nil(t, build)
				}
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
			codebasePath: "/tmp/test",
			expected:     filepath.Join("/tmp/test", types.CodebaseIndexDir),
		},
		{
			name:         "nested path",
			codebasePath: "/tmp/test/nested/path",
			expected:     filepath.Join("/tmp/test/nested/path", types.CodebaseIndexDir),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOutputDir(tt.codebasePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
