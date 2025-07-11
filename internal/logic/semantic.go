package logic

import (
	"context"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	minPositive = 1
	defaultTopK = 5
	paramQuery  = "query"
)

type SemanticLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSemanticSearchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SemanticLogic {
	return &SemanticLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SemanticLogic) SemanticSearch(req *types.SemanticSearchRequest) (resp *types.SemanticSearchResponseData, err error) {
	panic("implement me")

}
