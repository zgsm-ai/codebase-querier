package codegraph

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/index"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type task struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	msg    *types.CodebaseSyncMessage
}

func NewTask(ctx context.Context, svcCtx *svc.ServiceContext, msg *types.CodebaseSyncMessage) index.Task {
	return &task{
		ctx:    ctx,
		svcCtx: svcCtx,
		msg:    msg,
	}
}

func (t *task) Execute() {
	//TODO implement me
	panic("implement me")
}
