package scip

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
)

// MockStore 模拟codebase.Store接口
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Walk(ctx context.Context, codebasePath string, prefix string, walkFn func(*codebase.WalkContext, io.ReadCloser) error, opts codebase.WalkOptions) error {
	args := m.Called(ctx, codebasePath, prefix, walkFn, opts)
	return args.Error(0)
}

func (m *MockStore) Stat(ctx context.Context, codebasePath string, path string) (bool, error) {
	args := m.Called(ctx, codebasePath, path)
	return args.Bool(0), args.Error(1)
}

func (m *MockStore) MkDirs(ctx context.Context, codebasePath string, dirs ...string) error {
	args := m.Called(ctx, codebasePath, dirs)
	return args.Error(0)
}

func TestDetectLanguageAndTool(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*MockStore)
		expectedIndex *config.IndexTool
		expectedTool  *config.BuildTool
		expectedError error
	}{
		{
			name: "构建工具优先级排序",
			setupMock: func(m *MockStore) {
				// 模拟文件遍历
				m.On("Walk", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						walkFn := args.Get(2).(func(*codebase.WalkContext, io.ReadCloser) error)
						walkCtx := &codebase.WalkContext{
							RelativePath: "Main.java",
						}
						walkFn(walkCtx, nil)
					}).
					Return(nil)

				// 模拟检测到build.gradle和pom.xml
				m.On("Stat", mock.Anything, mock.Anything, "build.gradle").Return(true, nil)
				m.On("Stat", mock.Anything, mock.Anything, "pom.xml").Return(true, nil)
			},
			expectedIndex: &config.IndexTool{
				Name: "scip-java",
				Commands: []config.Command{
					{
						Base: "scip-java",
						Args: []string{
							"index",
							"--cwd",
							"__sourcePath__",
							"--targetroot",
							"__outputPath__/build",
							"--output",
							"__outputPath__/index.scip",
							"--",
							"__buildArgs__",
						},
					},
				},
			},
			expectedTool: &config.BuildTool{
				Name:           "maven",
				DetectionFiles: []string{"pom.xml"},
				Priority:       10,
				BuildCommands: []config.Command{
					{
						Base: "mvn",
						Args: []string{
							"verify",
							"--batch-mode",
							"--fail-never",
							"-DskipTests",
							"--offline",
							"-T",
							"8",
						},
					},
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建mock store
			mockStore := new(MockStore)
			tt.setupMock(mockStore)

			// 创建测试配置
			cfg := &config.CodegraphConfig{
				Languages: []*config.LanguageConfig{
					{
						Name:           "java",
						DetectionFiles: []string{"pom.xml", "build.gradle"},
						BuildTools: []*config.BuildTool{
							{
								Name:           "gradle",
								DetectionFiles: []string{"build.gradle"},
								Priority:       20,
								BuildCommands: []config.Command{
									{
										Base: "gradle",
										Args: []string{
											"--offline",
											"--continue",
											"--no-tests",
											"--parallel",
											"--max-workers",
											"8",
											"--no-interactive",
										},
									},
								},
							},
							{
								Name:           "maven",
								DetectionFiles: []string{"pom.xml"},
								Priority:       10,
								BuildCommands: []config.Command{
									{
										Base: "mvn",
										Args: []string{
											"verify",
											"--batch-mode",
											"--fail-never",
											"-DskipTests",
											"--offline",
											"-T",
											"8",
										},
									},
								},
							},
						},
						Index: &config.IndexTool{
							Name: "scip-java",
							Commands: []config.Command{
								{
									Base: "scip-java",
									Args: []string{
										"index",
										"--cwd",
										"__sourcePath__",
										"--targetroot",
										"__outputPath__/build",
										"--output",
										"__outputPath__/index.scip",
										"--",
										"__buildArgs__",
									},
								},
							},
						},
					},
				},
			}

			// 创建IndexGenerator实例
			generator := NewIndexGenerator(cfg, mockStore)

			// 执行测试
			index, tool, err := generator.detectLanguageAndTool(context.Background(), "/test/path")

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIndex, index)
				assert.Equal(t, tt.expectedTool, tool)
			}

			// 验证mock调用
			mockStore.AssertExpectations(t)
		})
	}
}
