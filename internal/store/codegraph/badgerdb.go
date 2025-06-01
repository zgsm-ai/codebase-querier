package codegraph

import (
	"context"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
)

type badgerDBGraph struct {
	path string
	db   *badger.DB
}

func (b badgerDBGraph) Close() error {
	return b.db.Close()
}

func NewBadgerDBGraph(opts ...GraphOption) (GraphStore, error) {
	b := &badgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}
	// 打开数据库
	// 如果需要在内存中运行，可以使用: opts := badger.DefaultOptions("").WithInMemory(true)
	baderDB, err := badger.Open(badger.DefaultOptions(filepath.Join(b.path + "codegraph.db")))
	if err != nil {
		return nil, err
	}

	b.db = baderDB
	return b, nil
}

type GraphOption func(*badgerDBGraph)

func WithPath(basePath string) GraphOption {
	return func(b *badgerDBGraph) {
		b.path = basePath
	}
}
func (b badgerDBGraph) Save(ctx context.Context, codebaseId int64, codebasePath string, nodes []*types.GraphNode) error {
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
