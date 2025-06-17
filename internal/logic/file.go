package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileContentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileContentLogic {
	return &GetFileContentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFileContentLogic) GetFileContent(req *types.FileContentRequest) ([]byte, error) {
	// 读取文件
	relativePath := req.FilePath
	clientCodebasePath := req.CodebasePath
	clientId := req.ClientId
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path
	if utils.IsBlank(codebasePath) {
		return nil, errors.New("codebase path is empty")
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))
	return l.svcCtx.CodebaseStore.Read(ctx, codebasePath, relativePath, types.ReadOptions{StartLine: req.StartLine, EndLine: req.EndLine})
}
