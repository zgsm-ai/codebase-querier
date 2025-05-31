package codebase

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func setupTestLocalCodebase(t *testing.T) (Store, string) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-codebase-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cfg := config.CodeBaseStoreConf{
		Local: config.LocalStoreConf{
			BasePath: tempDir,
		},
	}

	codebase := NewLocalCodebase(context.Background(), cfg)
	path, err := codebase.Init(context.Background(), "test-client", "test-path")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	return codebase, path.FullPath
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
			want:         filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
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
			tempDir, err := os.MkdirTemp("", "test-codebase-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			cfg := config.CodeBaseStoreConf{
				Local: config.LocalStoreConf{
					BasePath: tempDir,
				},
			}

			codebase := NewLocalCodebase(context.Background(), cfg)
			got, err := codebase.Init(context.Background(), tt.clientId, tt.codebasePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, strings.HasPrefix(got.FullPath, tempDir))
				assert.True(t, strings.HasSuffix(got.FullPath, filepath.Join(tt.clientId, tt.codebasePath)))
			}
		})
	}
}

func TestLocalCodebase_Add(t *testing.T) {
	tests := []struct {
		name         string
		codebasePath string
		target       string
		content      string
		wantErr      bool
	}{
		{
			name:         "successful add",
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
			target:       "test.txt",
			content:      "test content",
			wantErr:      false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			target:       "test.txt",
			content:      "test content",
			wantErr:      true,
		},
		{
			name:         "empty target",
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
			target:       "",
			content:      "test content",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, path := setupTestLocalCodebase(t)
			err := codebase.Add(context.Background(), path, strings.NewReader(tt.content), tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify file exists and content is correct
				content, err := os.ReadFile(filepath.Join(path, tt.target))
				assert.NoError(t, err)
				assert.Equal(t, tt.content, string(content))
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
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
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
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
			path:         "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, path := setupTestLocalCodebase(t)
			// Create a test file first
			if !tt.wantErr {
				err := codebase.Add(context.Background(), path, strings.NewReader("test content"), tt.path)
				assert.NoError(t, err)
			}

			err := codebase.Delete(context.Background(), path, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify file is deleted
				_, err := os.Stat(filepath.Join(path, tt.path))
				assert.True(t, os.IsNotExist(err))
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
		want         []*types.FileInfo
		wantErr      bool
	}{
		{
			name:         "successful list",
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
			dir:          "",
			option:       types.ListOptions{},
			want: []*types.FileInfo{
				{
					Name:    "test.txt",
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
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, path := setupTestLocalCodebase(t)
			// Create a test file first
			if !tt.wantErr {
				err := codebase.Add(context.Background(), path, strings.NewReader("test content"), "test.txt")
				assert.NoError(t, err)
			}

			got, err := codebase.List(context.Background(), path, tt.dir, tt.option)
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
		want         []*types.TreeNode
		wantErr      bool
	}{
		{
			name:         "successful tree",
			codebasePath: filepath.Join(os.TempDir(), "test-codebase-*", "test-client", "test-path"),
			dir:          "",
			option:       types.TreeOptions{},
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
					Children: []types.TreeNode{},
				},
			},
			wantErr: false,
		},
		{
			name:         "empty codebase path",
			codebasePath: "",
			dir:          "",
			option:       types.TreeOptions{},
			want:         nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codebase, path := setupTestLocalCodebase(t)
			// Create a test file first
			if !tt.wantErr {
				err := codebase.Add(context.Background(), path, strings.NewReader("test content"), "test.txt")
				assert.NoError(t, err)
			}

			got, err := codebase.Tree(context.Background(), path, tt.dir, tt.option)
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
