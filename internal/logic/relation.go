package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RelationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRelationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RelationLogic {
	return &RelationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RelationLogic) Relation(req *types.RelationQueryOptions) (resp *types.RelationResponseData, err error) {
	clientId := req.ClientId
	clientCodebasePath := req.CodebasePath
	codebase, err := l.svcCtx.CodebaseModel.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, model.ErrNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path

	graphStore, err := codegraph.NewBadgerDBGraph(l.ctx, codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}
	nodes, err := graphStore.Query(l.ctx, req)
	if err != nil {
		return nil, err
	}
	return &types.RelationResponseData{
		List: nodes,
	}, nil
}
