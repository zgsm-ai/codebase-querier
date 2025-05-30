package job

import (
	"context"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	taskTypeCodegraph = "codegraph"
)

type codegraphProcessor struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	msg    *types.CodebaseSyncMessage
}

func NewCodegraphProcessor(ctx context.Context, svcCtx *svc.ServiceContext, msg *types.CodebaseSyncMessage) Processor {
	return &codegraphProcessor{
		ctx:    ctx,
		svcCtx: svcCtx,
		msg:    msg,
	}
}

func (t *codegraphProcessor) Process() error {
	//TODO implement me
	panic("implement me")
}
