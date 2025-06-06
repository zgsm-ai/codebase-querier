package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

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
	codebase, err := l.svcCtx.CodebaseModel.FindByClientIdAndPath(l.ctx, clientId, clientCodebasePath)
	if errors.Is(err, model.ErrNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientCodebasePath: %s", clientId, clientCodebasePath))
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path
	if utils.IsBlank(codebasePath) {
		return nil, errors.New("codebase path is empty")
	}
	return l.svcCtx.CodebaseStore.Read(l.ctx, codebasePath, relativePath, types.ReadOptions{})
}
