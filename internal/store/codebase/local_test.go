package codebase

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func setupTestLocalCodebase(t *testing.T) (Store, string) {
	// Create a temporary directory for testing
	dir := "/tmp"
	err := os.MkdirAll(dir, defaultLocalFileMode)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: dir,
		},
	}

	codebase, err := NewLocalCodebase(cfg)
	assert.NoError(t, err)
	path, err := codebase.Init(context.Background(), "test-client", "test-path")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	return codebase, path.BasePath
}

func TestLocalCodebase_Init(t *testing.T) {
	tests := []struct {
		name         string
		clientId     string
		codebasePath string
		want         string
		wantErr      bool
	}{
		{
			name:         "successful initialization",
			clientId:     "test-client",
			codebasePath: "test-path",
			want:         filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), filepathSlash),
			wantErr:      false,
		},
		{
			name:         "empty client id",
			clientId:     "",
			codebasePath: "test-path",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "empty codebase path",
			clientId:     "test-client",
			codebasePath: "",
			want:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := "/tmp"
			err := os.MkdirAll(tempDir, defaultLocalFileMode)
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}

			cfg := config.CodeBaseStoreConf{
				Local: config.LocalStoreConf{
					BasePath: tempDir,
				},
			}

			codebase, err := NewLocalCodebase(cfg)
			assert.NoError(t, err)
			got, err := codebase.Init(context.Background(), tt.clientId, tt.codebasePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got.BasePath)
			}
		})
	}
}

func TestLocalCodebase_Add(t *testing.T) {
	tests := []struct {
		name         string
		clientId     string
		codebasePath string
		target       string
		wantErr      bool
	}{
		{
			name:         "successful add",
			clientId:     "test-client",
			codebasePath: "test-path",
			target:       "test.txt",
			wantErr:      false,
		},
		{
			name:         "empty codebase path",
			clientId:     "test-client",
			codebasePath: "",
			target:       "test.txt",
			wantErr:      true,
		},
		{
			name:         "empty target",
			clientId:     "test-client",
			codebasePath: "test-path",
			target:       "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, _ := setupTestLocalCodebase(t)

			var fullPath string
			if !tt.wantErr {
				// 先初始化路径
				result, err := codebase.Init(context.Background(), tt.clientId, tt.codebasePath)
				assert.NoError(t, err)
				fullPath = result.BasePath
			} else {
				fullPath = tt.codebasePath
			}

			err := codebase.Add(context.Background(), fullPath, strings.NewReader("test content"), tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLocalCodebase_Delete(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		path         string
		wantErr      bool
	}{
		{
			name:         "successful delete",
			codebasePath: filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			path:         "test.txt",
			wantErr:      false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			path:         "test.txt",
			wantErr:      true,
		},
		{
			name:         "empty path",
			codebasePath: filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			path:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, _ := setupTestLocalCodebase(t)
			// Create a test file first
			if !tt.wantErr {
				err := codebase.Add(context.Background(), tt.codebasePath, strings.NewReader("test content"), tt.path)
				assert.NoError(t, err)
			}

			err := codebase.Delete(context.Background(), tt.codebasePath, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLocalCodebase_List(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		option       types.ListOptions
		setupFiles   func(string) error
		want         []*types.FileInfo
		wantErr      bool
	}{
		{
			name:         "successful list",
			codebasePath: filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			dir:          "",
			option:       types.ListOptions{},
			setupFiles: func(basePath string) error {
				// 清理目录
				err := os.RemoveAll(basePath)
				if err != nil {
					return err
				}
				err = os.MkdirAll(basePath, defaultLocalDirMode)
				if err != nil {
					return err
				}

				// 只创建一个测试文件
				err = os.WriteFile(filepath.Join(basePath, "test.txt"), []byte("test content"), defaultLocalFileMode)
				if err != nil {
					return err
				}
				return nil
			},
			want: []*types.FileInfo{
				{
					Name:    "test.txt",
					Path:    "test.txt",
					Size:    12,
					IsDir:   false,
					ModTime: time.Time{},
					Mode:    defaultFileMode,
				},
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			option:       types.ListOptions{},
			setupFiles:   func(string) error { return nil },
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, _ := setupTestLocalCodebase(t)
			if tt.setupFiles != nil {
				err := tt.setupFiles(tt.codebasePath)
				assert.NoError(t, err)
			}

			got, err := codebase.List(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.want), len(got))
				if len(got) > 0 {
					assert.Equal(t, tt.want[0].Name, got[0].Name)
					assert.Equal(t, tt.want[0].Size, got[0].Size)
					assert.Equal(t, tt.want[0].IsDir, got[0].IsDir)
				}
			}
		})
	}
}

func TestLocalCodebase_Tree(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		option       types.TreeOptions
		setupFiles   func(string) error
		want         []*types.TreeNode
		wantErr      bool
	}{
		{
			name:         "successful tree",
			codebasePath: filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			dir:          "",
			option:       types.TreeOptions{},
			setupFiles: func(basePath string) error {
				// 清理目录
				err := os.RemoveAll(basePath)
				if err != nil {
					return err
				}
				err = os.MkdirAll(basePath, defaultLocalDirMode)
				if err != nil {
					return err
				}

				// 只创建一个测试文件
				err = os.WriteFile(filepath.Join(basePath, "test.txt"), []byte("test content"), defaultLocalFileMode)
				if err != nil {
					return err
				}
				return nil
			},
			want: []*types.TreeNode{
				{
					FileInfo: types.FileInfo{
						Name:    "test.txt",
						Path:    "test.txt",
						Size:    12,
						IsDir:   false,
						ModTime: time.Time{},
						Mode:    defaultFileMode,
					},
					Children: []*types.TreeNode{},
				},
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			option:       types.TreeOptions{},
			setupFiles:   func(string) error { return nil },
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, _ := setupTestLocalCodebase(t)
			if tt.setupFiles != nil {
				err := tt.setupFiles(tt.codebasePath)
				assert.NoError(t, err)
			}

			got, err := codebase.Tree(context.Background(), tt.codebasePath, tt.dir, tt.option)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.want), len(got))
				if len(got) > 0 {
					assert.Equal(t, tt.want[0].Name, got[0].Name)
					assert.Equal(t, tt.want[0].Path, got[0].Path)
					assert.Equal(t, tt.want[0].Size, got[0].Size)
					assert.Equal(t, tt.want[0].IsDir, got[0].IsDir)
				}
			}
		})
	}
}

func TestLocalCodebase_Walk(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		dir          string
		setupFiles   func(string) error
		wantPaths    []string
		wantErr      bool
	}{
		{
			name:         "successful walk",
			codebasePath: filepath.Join("/tmp", generateUniquePath("test-client", "test-path"), "test-path", filepathSlash),
			dir:          "",
			setupFiles: func(basePath string) error {
				// 清理目录
				err := os.RemoveAll(basePath)
				if err != nil {
					return err
				}
				err = os.MkdirAll(basePath, defaultLocalDirMode)
				if err != nil {
					return err
				}

				// Create test directory structure
				files := map[string]string{
					"file1.txt":           "content1",
					"dir1/file2.txt":      "content2",
					"dir1/dir2/file3.txt": "content3",
					".hidden/file4.txt":   "content4",
				}

				for path, content := range files {
					fullPath := filepath.Join(basePath, path)
					if err := os.MkdirAll(filepath.Dir(fullPath), defaultLocalDirMode); err != nil {
						return err
					}
					if err := os.WriteFile(fullPath, []byte(content), defaultLocalFileMode); err != nil {
						return err
					}
				}
				return nil
			},
			wantPaths: []string{
				"file1.txt",
				"dir1/file2.txt",
				"dir1/dir2/file3.txt",
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			setupFiles:   func(string) error { return nil },
			wantPaths:    nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, _ := setupTestLocalCodebase(t)
			if err := tt.setupFiles(tt.codebasePath); err != nil {
				t.Fatalf("Failed to setup test files: %v", err)
			}

			var visitedPaths []string
			err := codebase.Walk(context.Background(), tt.codebasePath, tt.dir, func(walkCtx *WalkContext, reader io.ReadCloser) error {
				// Skip hidden files and directories
				if strings.HasPrefix(walkCtx.Info.Name, ".") {
					if walkCtx.Info.IsDir {
						return SkipDir
					}
					return nil
				}

				// Only record file paths
				if !walkCtx.Info.IsDir {
					visitedPaths = append(visitedPaths, walkCtx.RelativePath)
				}
				return nil
			}, WalkOptions{})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				sort.Strings(visitedPaths)
				sort.Strings(tt.wantPaths)
				assert.Equal(t, tt.wantPaths, visitedPaths)
			}
		})
	}
}
