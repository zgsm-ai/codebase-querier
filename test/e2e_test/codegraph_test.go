package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
)

func TestCodegraphSummary(t *testing.T) {
	assert.NotPanics(t, func() {
		ctx := context.Background()

		logx.DisableStat()
		codebasePath := "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160"
		graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
		summary, err := graphStore.GetIndexSummary(ctx, 1, codebasePath)
		assert.NoError(t, err)
		assert.True(t, summary.TotalFiles > 0)

	})

}
