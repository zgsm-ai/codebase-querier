package codebase

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
)

var _ Store = &minioCodebase{}

type minioCodebase struct {
	logger logx.Logger
	cfg    config.CodeBaseStoreConf
}

func newMinioCodebase(ctx context.Context, cfg config.CodeBaseStoreConf) Store {
	return &minioCodebase{
		cfg:    cfg,
		logger: logx.WithContext(ctx),
	}
}

func (m *minioCodebase) Init(ctx context.Context, clientId string, clientCodebasePath string) (types.Codebase, error) {

	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Delete(ctx context.Context, codebasePath string, path string) error {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Stat(ctx context.Context, codebasePath string, path string) (types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Tree(ctx context.Context, codebasePath string, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) Walk(ctx context.Context, codebasePath string, dir string, process func(io.ReadCloser) (bool, error)) error {
	//TODO implement me
	panic("implement me")
}

func (m *minioCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	//TODO implement me
	panic("implement me")
}
