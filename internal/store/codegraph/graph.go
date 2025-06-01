package codegraph

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type GraphStore interface {
	Save(ctx context.Context, nodes []*types.GraphNode) error
	Query(ctx context.Context, req *types.RelationQueryOptions) ([]*types.GraphNode, error)
	DeleteAll(ctx context.Context, codebaseId int64, codebasePath string) error
}
