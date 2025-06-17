package job

import (
	"context"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/model"
	redisstore "github.com/zgsm-ai/codebase-indexer/internal/store/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"time"
)

const cleanLockKey = "codebase_indexer:cleaner:lock"
const lockTimeout = time.Second * 120

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
	c.cron.Start() // 启动 Cron
	logx.Infof("cleaner job started")
}

func NewCleaner(ctx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	// go cron
	cr := cron.New() // 创建默认 Cron 实例（支持秒级精度）
	// 添加任务（参数：Cron 表达式, 要执行的函数）
	_, err := cr.AddFunc(svcCtx.Config.Cleaner.Cron, func() {
		// aquice lock
		locked, err := svcCtx.DistLock.TryLock(ctx, cleanLockKey, lockTimeout)
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

		expireDays := time.Duration(svcCtx.Config.Cleaner.CodebaseExpireDays) * 24 * time.Hour
		expiredDate := time.Now().Add(-expireDays)

		codebases, err := findExpiredCodebases(ctx, svcCtx, expiredDate)
		if err != nil {
			logx.Errorf("find expired codebase error: %v", err)
			return
		}
		for _, cb := range codebases {
			logx.Infof("start to clean codebase: %s", cb.Path)
			// todo clean codebase
			err = svcCtx.CodebaseStore.DeleteAll(ctx, cb.Path)
			if err != nil {
				logx.Errorf("cleaner drop codebase store %s error: %v", cb.Path, err)
			}
			// todo clean vector store
			err = svcCtx.VectorStore.DeleteCodeChunks(ctx, []*types.CodeChunk{{CodebaseId: cb.ID}}, vector.Options{})
			if err != nil {
				logx.Errorf("cleaner drop codebase store %s error: %v", cb.Path, err)
			}
			// todo clean graph store ， clean codebase alerady delete all files， now graph store is in codebase store.
			//graphStore, err := codegraph.NewBadgerDBGraph(ctx, codegraph.WithPath(filepath.Join(cb.FilePath, types.CodebaseIndexDir)))
			//
			//err = graphStore.DeleteAll(ctx)
			//if err != nil {
			//	logx.Errorf("drop codebase store %s error: %v", cb.FilePath, err)
			//	continue
			//}
			// 清理redis cache
			if err = svcCtx.Cache.CleanExpiredVersions(ctx, utils.FormatInt(int64(cb.ID))); err != nil {
				logx.Errorf("cleaner clean codebase store %s error: %v", cb.Path, err)
			}

			// todo update db status
			cb.Status = string(model.CodebaseStatusExpired)
			if _, err = svcCtx.Querier.Codebase.WithContext(ctx).
				Where(svcCtx.Querier.Codebase.ID.Eq(cb.ID)).
				Updates(cb); err != nil {
				logx.Errorf("cleaner update codebase %s status expired error: %v", cb.Path, err)
				return
			}
			logx.Infof("cleaner clean codebase successfully: %s", cb.Path)
		}
		logx.Infof("cleaner clean codebases end, cnt: %d", len(codebases))
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

func findExpiredCodebases(ctx context.Context, svcCtx *svc.ServiceContext, expiredDate time.Time) ([]*model.Codebase, error) {
	codebases, err := svcCtx.Querier.Codebase.WithContext(ctx).
		Where(svcCtx.Querier.Codebase.CreatedAt.Lt(expiredDate)).
		Where(svcCtx.Querier.Codebase.Status.Eq(string(model.CodebaseStatusActive))).
		Find()
	return codebases, err
}
