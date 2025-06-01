package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

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
	topK := req.TopK
	if topK < minPositive {
		topK = defaultTopK
	}
	if utils.IsBlank(req.Query) {
		return nil, errs.NewInvalidParamErr(paramQuery, req.Query)
	}
	clientId := req.ClientId
	clientCodebasePath := req.CodebasePath

	codebase, err := l.svcCtx.CodebaseModel.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, model.ErrNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	// TODO  向量库隔离
	documents, err := l.svcCtx.VectorStore.Query(l.ctx, req.Query, topK, vectorstores.WithNameSpace(codebase.LocalPath))
	if err != nil {
		return nil, err
	}
	return &types.SemanticSearchResponseData{
		List: documents,
	}, nil
}
