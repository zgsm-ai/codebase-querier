package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type IndexLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IndexLogic {
	return &IndexLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *IndexLogic) DeleteIndex(req *types.DeleteIndexRequest) (resp *types.DeleteIndexResponseData, err error) {
	clientId := req.ClientId
	clientPath := req.CodebasePath
	indexType := req.IndexType

	// 查找代码库记录
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))
	// 根据索引类型删除对应的索引
	switch indexType {
	case string(types.Embedding):
		if err := l.svcCtx.VectorStore.DeleteByCodebase(ctx, codebase.ID, codebase.Path); err != nil {
			return nil, fmt.Errorf("failed to delete embedding index, err:%w", err)
		}
	case string(types.CodeGraph):
		graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebase.Path, types.CodebaseIndexDir)))
		if err != nil {
			return nil, fmt.Errorf("failed to open graph store, err:%w", err)
		}
		defer graphStore.Close()
		if err := graphStore.DeleteByCodebase(ctx, codebase.ID, codebase.Path); err != nil {
			return nil, fmt.Errorf("failed to delete graph index, err:%w", err)
		}
	case string(types.All):
		if err := l.svcCtx.VectorStore.DeleteByCodebase(ctx, codebase.ID, codebase.Path); err != nil {
			return nil, fmt.Errorf("failed to delete embedding index, err:%w", err)
		}
		graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebase.Path, types.CodebaseIndexDir)))
		if err != nil {
			return nil, fmt.Errorf("failed to open graph store, err:%w", err)
		}
		defer graphStore.Close()
		if err := graphStore.DeleteByCodebase(ctx, codebase.ID, codebase.Path); err != nil {
			return nil, fmt.Errorf("failed to delete graph index, err:%w", err)
		}
	default:
		return nil, errs.NewInvalidParamErr("indexType", indexType)
	}

	return &types.DeleteIndexResponseData{}, nil
}
