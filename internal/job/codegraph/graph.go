package codegraph

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type GraphBuilder interface {
	Build(ctx context.Context, codebasePath string) ([]*types.GraphNode, error)
}
