package logic

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxReadLine = 5000

type StructureLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFileDefinitionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StructureLogic {
	return &StructureLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StructureLogic) ParseFileDefinitions(req *types.FileDefinitionParseRequest) (resp *types.FileDefinitionResponseData, err error) {
	panic("implement me")

}
