package logic

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DefinitionQueryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDefinitionQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DefinitionQueryLogic {
	return &DefinitionQueryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DefinitionQueryLogic) QueryDefinition(req *types.DefinitionRequest) (resp *types.DefinitionResponseData, err error) {
	panic("implement me")
}
