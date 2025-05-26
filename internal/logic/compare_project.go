package logic

import (
	"context"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CompareProjectsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompareProjectsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompareProjectsLogic {
	return &CompareProjectsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CompareProjectsLogic) CompareProjects(req *types.ProjectComparisonRequest) (resp *types.ComparisonResponseData, err error) {
	// todo: add your logic here and delete this line

	return
}
