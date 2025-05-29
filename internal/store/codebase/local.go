package codebase

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
)

var _ Store = &localCodebaseStore{}

type localCodebaseStore struct {
	logger logx.Logger
	config config.CodeBaseStoreConf
}

func (l localCodebaseStore) Init(ctx context.Context, codebase types.Codebase) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Add(ctx context.Context, id int64, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Unzip(ctx context.Context, id int64, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Delete(ctx context.Context, id int64, path string) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Mkdirs(ctx context.Context, id int64, path string) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Exists(ctx context.Context, id int64, path string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Stat(ctx context.Context, id int64, path string) (types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) List(ctx context.Context, id int64, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Tree(ctx context.Context, id int64, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Read(ctx context.Context, id int64, filePath string, option types.ReadOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) Walk(ctx context.Context, id int64, dir string, process func(io.ReadCloser) (bool, error)) error {
	//TODO implement me
	panic("implement me")
}

func (l localCodebaseStore) BatchDelete(ctx context.Context, id int64, paths []string) error {
	//TODO implement me
	panic("implement me")
}

func newLocalCodeBase(c config.CodeBaseStoreConf) Store {
	return &localCodebaseStore{
		config: c,
		logger: logx.WithContext(context.Background()),
	}
}
