package codebase

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"io"
)

var _ CodebaseStore = &localCodebaseStore{}

type localCodebaseStore struct {
	logger logx.Logger
	config config.CodeBaseStoreConf
}

func newLocalCodeBase(c config.CodeBaseStoreConf) CodebaseStore {
	return &localCodebaseStore{
		config: c,
		logger: logx.WithContext(context.Background()),
	}
}

func (c localCodebaseStore) Add(ctx context.Context, source io.Reader, target string) error {

	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Unzip(ctx context.Context, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Delete(ctx context.Context, path string) error {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Exists(ctx context.Context, path string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Stat(ctx context.Context, path string) (FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) List(ctx context.Context, dir string, opts ...ListOption) ([]*FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Tree(ctx context.Context, dir string, opts ...TreeOption) ([]*TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Read(ctx context.Context, filePath string) (io.ReadCloser, error) {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) Walk(ctx context.Context, dir string, process func(io.ReadCloser) error) error {
	//TODO implement me
	panic("implement me")
}

func (c localCodebaseStore) BatchDelete(ctx context.Context, paths []string) error {
	//TODO implement me
	panic("implement me")
}
