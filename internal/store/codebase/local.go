package codebase

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
)

var _ Store = &localCodebase{}

type localCodebase struct {
	logger logx.Logger
	cfg    config.CodeBaseStoreConf
}

func newLocalCodeBase(ctx context.Context, cfg config.CodeBaseStoreConf) Store {
	return &localCodebase{
		cfg:    cfg,
		logger: logx.WithContext(ctx),
	}
}

func (l *localCodebase) Init(ctx context.Context, codebase types.Codebase) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Delete(ctx context.Context, codebasePath string, path string) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Stat(ctx context.Context, codebasePath string, path string) (types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) Walk(ctx context.Context, codebasePath string, dir string, process func(io.ReadCloser) (bool, error)) error {
	//TODO implement me
	panic("implement me")
}

func (l *localCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	//TODO implement me
	panic("implement me")
}
