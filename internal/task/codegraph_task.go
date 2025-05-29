package task

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type codegraphTask struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	msg    *types.CodebaseSyncMessage
}

func NewCodegraphTask(ctx context.Context, svcCtx *svc.ServiceContext, msg *types.CodebaseSyncMessage) Task {
	return &codegraphTask{
		ctx:    ctx,
		svcCtx: svcCtx,
		msg:    msg,
	}
}

func (t *codegraphTask) Run() {
	//TODO implement me
	panic("implement me")
}
