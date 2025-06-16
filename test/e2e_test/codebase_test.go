package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/test/api_test"
	"path/filepath"
	"testing"
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

	svcCtx := api_test.InitSvcCtx(ctx, nil)

	localCodebase, err := codebase.NewLocalCodebase(ctx, svcCtx.Config.CodeBaseStore)
	assert.NoError(t, err)
	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			tree, err := localCodebase.Tree(ctx, tt.codebasePath, tt.subDir, tt.treeOptions)
			assert.ErrorIs(t, err, tt.wantErr)
			tt.assertResult(t, tree)
		})
	}

}
