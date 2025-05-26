package logic

import (
	"context"

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

func (l *GetFileContentLogic) GetFileContent(req *types.FileContentRequest) error {
	// todo: add your logic here and delete this line

	return nil
}
