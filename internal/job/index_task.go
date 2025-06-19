package job

import (
	"context"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type IndexTask struct {
	SvcCtx  *svc.ServiceContext
	LockMux *redsync.Mutex
	Params  *IndexTaskParams
}

type IndexTaskParams struct {
	SyncID               int32  // 同步操作ID
	CodebaseID           int32  // 代码库ID
	CodebasePath         string // 代码库路径
	CodebaseName         string // 代码库名字
	SyncMetaFiles        *types.CollapseSyncMetaFile
	EnableEmbeddingBuild bool
	EnableCodeGraphBuild bool
}

func (i *IndexTask) Run(ctx context.Context) (embedTaskOk bool, graphTaskOk bool) {
	// 解锁
	defer func() {
		if err := i.SvcCtx.DistLock.Unlock(ctx, i.LockMux); err != nil {
			tracer.WithTrace(ctx).Errorf("index task unlock failed, key %s, err:%v", i.LockMux.Name(), err)
		}
	}()

	embedErr := i.buildEmbedding(ctx)
	if embedErr != nil {
		tracer.WithTrace(ctx).Errorf("embedding task failed:%v", embedErr)
	}
	graphErr := i.buildCodeGraph(ctx)
	if graphErr != nil {
		tracer.WithTrace(ctx).Errorf("graph task failed:%v", graphErr)
	}

	embedTaskOk, graphTaskOk = embedErr == nil, graphErr == nil

	if embedTaskOk && graphTaskOk {
		i.cleanProcessedMetadataFile(ctx)
	}
	return
}

func (i *IndexTask) buildCodeGraph(ctx context.Context) error {
	if !i.Params.EnableCodeGraphBuild {
		tracer.WithTrace(ctx).Infof("codegraph build is disabled, not process.")
		return nil
	}
	codegraphTimeout, graphTimeoutCancel := context.WithTimeout(ctx, i.SvcCtx.Config.IndexTask.GraphTask.Timeout)
	defer graphTimeoutCancel()

	gProcessor, err := NewCodegraphProcessor(i.SvcCtx, i.Params, i.Params.SyncMetaFiles.FileModelMap)
	if err != nil {
		return fmt.Errorf("failed to create codegraph processor for message %d, err: %w", i.Params.SyncID, err)
	}

	if err = gProcessor.Process(codegraphTimeout); err != nil {
		return fmt.Errorf("codegraph task failed, err:%w", err)
	}
	tracer.WithTrace(ctx).Infof("codegraph task successfully.")
	return nil
}

func (i *IndexTask) buildEmbedding(ctx context.Context) error {

	if !i.Params.EnableEmbeddingBuild {
		tracer.WithTrace(ctx).Infof("embedding build is disabled, not process.")
		return nil
	}

	embeddingTimeout, embeddingTimeoutCancel := context.WithTimeout(ctx, i.SvcCtx.Config.IndexTask.EmbeddingTask.Timeout)
	defer embeddingTimeoutCancel()
	eProcessor, err := NewEmbeddingProcessor(i.SvcCtx, i.Params, i.Params.SyncMetaFiles.FileModelMap)
	if err != nil {
		return fmt.Errorf("failed to create embedding task processor for message: %d, err: %w", i.Params.SyncID, err)
	}
	err = eProcessor.Process(embeddingTimeout)
	if err != nil {
		return fmt.Errorf("embedding task failed, err:%w", err)
	}
	tracer.WithTrace(ctx).Infof("embedding task successfully.")
	return nil
}

func (i *IndexTask) cleanProcessedMetadataFile(ctx context.Context) {
	if len(i.Params.SyncMetaFiles.MetaFilePaths) == 0 {
		tracer.WithTrace(ctx).Infof("sync meta file list is empty, not clean.")
		return
	}
	tracer.WithTrace(ctx).Infof("start to clean sync meta file, codebasePath:%s, paths:%v", i.Params.CodebasePath, i.Params.SyncMetaFiles.MetaFilePaths)
	// TODO 当调用链和嵌入任务都成功时，清理元数据文件。改为移动到另一个隐藏文件夹中，每天定时清理，便于排查问题。
	if err := i.SvcCtx.CodebaseStore.BatchDelete(ctx, i.Params.CodebasePath, i.Params.SyncMetaFiles.MetaFilePaths); err != nil {
		tracer.WithTrace(ctx).Errorf("failed to delete codebase %s metadata : %v, err: %v", i.Params.CodebasePath, i.Params.SyncMetaFiles.MetaFilePaths, err)
	}
	tracer.WithTrace(ctx).Infof("clean %s sync meta files successfully.", i.Params.CodebasePath)
}
