package task

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
)

type cleaner struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	logger logx.Logger
}

func (c *cleaner) Run() {
	//TODO implement me

	// go cron
	cr := cron.New() // 创建默认 Cron 实例（支持秒级精度）

	// 添加任务（参数：Cron 表达式, 要执行的函数）
	_, err := cr.AddFunc(c.svcCtx.Config.Cleaner.Cron, func() {
		// TODO
		// TODO delete expired codebases 、 vectors、db records
	})
	if err != nil {
		panic(err)
	}
	cr.Start() // 启动 Cron
	c.logger.Infof("cleaner task start")
}

func NewCleaner(svcCtx *svc.ServiceContext, ctx context.Context) Task {
	return &cleaner{
		svcCtx: svcCtx,
		ctx:    ctx,
		logger: logx.WithContext(ctx),
	}
}
