package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type TaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TaskLogic {
	return &TaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TaskLogic) Task(req *types.IndexTaskRequest) (resp *types.IndexTaskResponseData, err error) {
	clientId := req.ClientId
	clientPath := req.CodebasePath
	indexType := req.IndexType

	// 查找代码库记录
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	if err != nil {
		return nil, err
	}

	// 创建索引任务
	// 查询最新的同步
	latestSync, err := l.svcCtx.Querier.SyncHistory.FindLatest(l.ctx, codebase.ID)
	if err != nil {
		return nil, errs.NewRecordNotFoundErr(types.NameSyncHistory, fmt.Sprintf("codebase_id: %d", codebase.ID))
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	// 获取同步锁，避免重复处理
	// 获取分布式锁， n分钟超时
	lockKey := job.IndexJobKey(codebase.ID)
	locked, err := l.svcCtx.DistLock.TryLock(ctx, lockKey, job.DistLockTimeout)
	if err != nil || !locked {
		return nil, fmt.Errorf("failed to acquire lock to sumit index task, err:%w", err)
	}

	tracer.WithTrace(ctx).Infof("acquire lock successfully, start to submit index task.")

	// 元数据列表
	medataFiles, err := l.svcCtx.CodebaseStore.GetSyncFileListCollapse(ctx, codebase.Path)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("index job GetSyncFileListCollapse err:%w", err)
		return
	}

	if medataFiles == nil || len(medataFiles.FileModelMap) == 0 {
		return nil, errors.New("sync file list is nil, cannot submit index task")
	}

	var enableEmbeddingBuild, enableCodeGraphBuild bool
	switch indexType {
	case string(types.Embedding):
		enableEmbeddingBuild = true
	case string(types.CodeGraph):
		enableCodeGraphBuild = true
	case string(types.All):
		enableEmbeddingBuild = true
		enableCodeGraphBuild = true
	default:
		return nil, fmt.Errorf("invalid index task type:%s", indexType)
	}
	task := &job.IndexTask{
		SvcCtx:  l.svcCtx,
		LockKey: lockKey,
		Params: &job.IndexTaskParams{
			SyncID:               latestSync.ID,
			CodebaseID:           codebase.ID,
			CodebasePath:         codebase.Path,
			CodebaseName:         codebase.Name,
			SyncMetaFiles:        medataFiles,
			EnableEmbeddingBuild: enableEmbeddingBuild,
			EnableCodeGraphBuild: enableCodeGraphBuild,
		},
	}

	err = l.svcCtx.TaskPool.Submit(func() {
		tracer.WithTrace(ctx).Infof("start to run index task.")
		embedOk, graphOk := task.Run(ctx)
		tracer.WithTrace(ctx).Infof("index task run end, embedding success? %t, codegraph success? %t.", embedOk, graphOk)
	})

	if err != nil {
		return nil, fmt.Errorf("index task submit failed, err:%w", err)
	}
	tracer.WithTrace(ctx).Infof("index task submit successfully.")

	return &types.IndexTaskResponseData{}, nil
}
