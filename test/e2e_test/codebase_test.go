package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/test/api_test"
	"path/filepath"
	"testing"
	"time"
)

const currentBasePath = "G:\\projects\\codebase-indexer"

func TestTree(t *testing.T) {

	testcases := []struct {
		Name         string
		codebasePath string
		wantErr      error
		subDir       string
		treeOptions  types.TreeOptions
		assertResult func(t *testing.T, tree []*types.TreeNode)
	}{

		{
			Name:         "no subdir",
			codebasePath: filepath.Join(currentBasePath, "test/"),
			wantErr:      nil,
			subDir:       "",
			treeOptions: types.TreeOptions{
				MaxDepth: 1,
			},
			assertResult: func(t *testing.T, tree []*types.TreeNode) {
				assert.Equal(t, 3, len(tree))
			},
		},
		{
			Name:         "no subdir layer 2",
			codebasePath: filepath.Join(currentBasePath, "test/"),
			wantErr:      nil,
			subDir:       "",
			treeOptions: types.TreeOptions{
				MaxDepth: 3,
			},
			assertResult: func(t *testing.T, tree []*types.TreeNode) {
				assert.Equal(t, 3, len(tree))
			},
		},
		{
			Name:         "has subdir",
			codebasePath: filepath.Join(currentBasePath, "test/"),
			wantErr:      nil,
			subDir:       "e2e_test/conf",
			treeOptions: types.TreeOptions{
				MaxDepth: 1,
			},
			assertResult: func(t *testing.T, tree []*types.TreeNode) {
				assert.Equal(t, 1, len(tree))
				assert.True(t, tree[0].FileInfo.Name == "codegraph.yaml")
			},
		},
	}

	ctx := context.Background()

	svcCtx := api.InitSvcCtx(ctx, nil)

	localCodebase, err := codebase.NewLocalCodebase(svcCtx.Config.CodeBaseStore)
	assert.NoError(t, err)
	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			tree, err := localCodebase.Tree(ctx, tt.codebasePath, tt.subDir, tt.treeOptions)
			assert.ErrorIs(t, err, tt.wantErr)
			tt.assertResult(t, tree)
		})
	}

}

func TestReadFile(t *testing.T) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Second*60)
	defer cancelFunc()
	svcCtx := api.InitSvcCtx(context.Background(), nil)
	localCodebase, err := codebase.NewLocalCodebase(svcCtx.Config.CodeBaseStore)
	assert.NoError(t, err)
	testcases := []struct {
		Name         string
		codebasePath string
		filePath     string
		wantErr      error
	}{
		{
			Name:         "big file",
			codebasePath: "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160\\.shenma_sync",
			filePath:     "1749903492799",
			wantErr:      nil,
		},
	}
	for _, tt := range testcases {
		read, err := localCodebase.Read(timeout, tt.codebasePath, tt.filePath, types.ReadOptions{})
		assert.NoError(t, err)
		assert.True(t, len(read) > 0)
	}

}

func TestInferLanguage(t *testing.T) {
	ctx := context.Background()
	svcCtx := api.InitSvcCtx(ctx, nil)
	testcases := []struct {
		Name         string
		codebasePath string
		wantErr      error
		wantResult   parser.Language
	}{
		{
			Name:         "typescript",
			codebasePath: "G:\\projects\\zgsm",
			wantErr:      nil,
			wantResult:   parser.TSX,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			language, err := svcCtx.CodebaseStore.InferLanguage(ctx, tt.codebasePath)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantResult, language)
		})
	}
}
