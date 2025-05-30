package job

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
	cron   *cron.Cron
}

func (c *cleaner) Close() {
	//TODO implement me
	// panic("implement me")
}

func (c *cleaner) Start() {
	//TODO implement me
	c.cron.Start() // 启动 Cron
	c.logger.Infof("cleaner task started")
}

func newCleaner(ctx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	// go cron
	cr := cron.New() // 创建默认 Cron 实例（支持秒级精度）
	// 添加任务（参数：Cron 表达式, 要执行的函数）
	_, err := cr.AddFunc(svcCtx.Config.Cleaner.Cron, func() {
		// TODO
		// TODO delete expired codebases 、 vectors、db records
	})
	if err != nil {
		return nil, err
	}
	return &cleaner{
		svcCtx: svcCtx,
		ctx:    ctx,
		logger: logx.WithContext(ctx),
	}, nil
}
