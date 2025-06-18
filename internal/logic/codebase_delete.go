package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type Delete_codebaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCodebaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *Delete_codebaseLogic {
	return &Delete_codebaseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *Delete_codebaseLogic) DeleteCodebase(req *types.DeleteCodebaseRequest) (resp *types.DeleteCodebaseResponseData, err error) {
	clientId := req.ClientId
	clientPath := req.CodebasePath

	// 查找代码库记录
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))
	// 删除代码库存储
	if err := l.svcCtx.CodebaseStore.DeleteAll(ctx, codebase.Path); err != nil {
		return nil, err
	}

	// 删除数据库记录
	if _, err := l.svcCtx.Querier.WithContext(ctx).Codebase.Delete(codebase); err != nil {
		return nil, err
	}

	return &types.DeleteCodebaseResponseData{}, nil
}
