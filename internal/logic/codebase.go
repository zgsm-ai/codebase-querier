package logic

import (
	"context"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CompareCodebasesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompareCodebaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompareCodebasesLogic {
	return &CompareCodebasesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CompareCodebasesLogic) CompareCodebase(req *types.CodebaseComparisonRequest) (resp *types.ComparisonResponseData, err error) {
	// todo: add your logic here and delete this line

	return
}
