package job

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	redisstore "github.com/zgsm-ai/codebase-indexer/internal/store/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"time"
)

const cleanLockKey = "codebase-indexer:cleaner:lock"
const lockTimteoutSeconds = time.Second * 120

type cleaner struct {
	svcCtx *svc.ServiceContext
	ctx    context.Context
	cron   *cron.Cron
}

func (c *cleaner) Close() {
	//TODO implement me
	// panic("implement me")
}

func (c *cleaner) Start() {
	//TODO implement me
	c.cron.Start() // 启动 Cron
	logx.Infof("cleaner job started")
}

func newCleaner(ctx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	// go cron
	cr := cron.New() // 创建默认 Cron 实例（支持秒级精度）
	// 添加任务（参数：Cron 表达式, 要执行的函数）
	_, err := cr.AddFunc(svcCtx.Config.Cleaner.Cron, func() {
		// aquice lock
		locked, err := svcCtx.DistLock.TryLock(ctx, cleanLockKey, lockTimteoutSeconds)
		if err != nil {
			logx.Errorf("cleaner try lock error: %v", err)
			return
		}
		if !locked {
			logx.Infof("cleaner lock %s is already locked ,return", cleanLockKey)
			return
		}
		defer func(DistLock redisstore.DistributedLock, ctx context.Context, key string) {
			err := DistLock.Unlock(ctx, key)
			if err != nil {
				logx.Errorf("cleaner unlock %s error: %v", cleanLockKey, err)
			}
			logx.Errorf("cleaner unlock successfully.")
		}(svcCtx.DistLock, ctx, cleanLockKey)

		logx.Infof("cleaner get lock %s successfully, start.", cleanLockKey)

		codebases, err := svcCtx.CodebaseModel.FindExpiredCodebase(ctx, svcCtx.Config.Cleaner.CodebaseExpireDays)
		if err != nil {
			logx.Errorf("find expired codebase error: %v", err)
			return
		}
		for _, cb := range codebases {
			logx.Infof("start to clean codebase: %s", cb.Path)
			// todo clean codebase
			err = svcCtx.CodebaseStore.DeleteAll(ctx, cb.Path)
			if err != nil {
				logx.Errorf("drop codebase store %s error: %v", cb.Path, err)
				continue
			}
			// todo clean vector store
			_, err = svcCtx.VectorStore.DeleteCodeChunks(ctx, []*types.CodeChunk{{CodebaseId: cb.Id}})
			if err != nil {
				logx.Errorf("drop codebase store %s error: %v", cb.Path, err)
				continue
			}
			// todo clean graph store
			graphStore, err := codegraph.NewBadgerDBGraph(ctx, codegraph.WithPath(filepath.Join(cb.Path, types.CodebaseIndexDir)))

			err = graphStore.DeleteAll(ctx)
			if err != nil {
				logx.Errorf("drop codebase store %s error: %v", cb.Path, err)
				continue
			}
			// todo update db status
			cb.Status = model.CodebaseStatusExpired
			if err = svcCtx.CodebaseModel.Update(ctx, cb); err != nil {
				logx.Errorf("update codebase %s status expired error: %v", cb.Path, err)
				return
			}
			logx.Infof("clean codebase successfully: %s", cb.Path)
		}
		logx.Infof("clean codebases end, cnt: %d", len(codebases))
	})
	if err != nil {
		return nil, err
	}
	return &cleaner{
		svcCtx: svcCtx,
		ctx:    ctx,
		cron:   cr,
	}, nil
}
