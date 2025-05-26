package logic

import (
	"context"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SemanticLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSemanticLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SemanticLogic {
	return &SemanticLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SemanticLogic) Semantic(req *types.SemanticRequest) (resp *types.SemanticResponseData, err error) {
	// todo: add your logic here and delete this line

	return
}
