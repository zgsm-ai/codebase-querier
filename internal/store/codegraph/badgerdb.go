package codegraph

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type badgerDBGraph struct {
	path string
}

func NewBadgerDBGraph(opts ...GraphOption) GraphStore {
	b := &badgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

type GraphOption func(*badgerDBGraph)

func WithPath(basePath string) GraphOption {
	return func(b *badgerDBGraph) {
		b.path = basePath
	}
}
func (b badgerDBGraph) Save(ctx context.Context, nodes []*types.GraphNode) error {
	//TODO implement me
	panic("implement me")
}

func (b badgerDBGraph) Query(ctx context.Context, req *types.RelationQueryOptions) ([]*types.GraphNode, error) {
	//TODO implement me
	panic("implement me")
}

func (b badgerDBGraph) DeleteAll(ctx context.Context, codebaseId int64, codebasePath string) error {
	//TODO implement me
	panic("implement me")
}
