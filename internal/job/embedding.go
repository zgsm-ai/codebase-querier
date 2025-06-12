package job

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/model"
	"github.com/zgsm-ai/codebase-indexer/internal/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	taskTypeEmbedding = "embedding"
)

type embeddingProcessor struct {
	ctx             context.Context
	svcCtx          *svc.ServiceContext
	msg             *types.CodebaseSyncMessage
	logger          logx.Logger
	syncFileModeMap map[string]string
	taskHistoryId   int32
	totalFileCnt    int32
	successFileCnt  int32
	failedFileCnt   int32
	ignoreFileCnt   int32
}

func NewEmbeddingProcessor(ctx context.Context,
	svcCtx *svc.ServiceContext,
	msg *types.CodebaseSyncMessage,
	syncFileModeMap map[string]string,
) (Processor, error) {
	return &embeddingProcessor{
		ctx:             ctx,
		svcCtx:          svcCtx,
		msg:             msg,
		syncFileModeMap: syncFileModeMap,
		logger:          logx.WithContext(ctx),
	}, nil
}

func (t *embeddingProcessor) Process() error {

	t.logger.Infof("start to execute embedding embeddingProcessor %+v", t.msg)
	start := time.Now()

	err := func(t *embeddingProcessor) []error {
		var runErrs []error
		if err := t.initTaskHistory(); err != nil {
			return []error{err}
		}

		// TODO 并发处理任务
		// fileCnt := len(syncFileMap)
		// maxConcurrency := t.svcCtx.Config.IndexTask.EmbeddingTask.MaxConcurrency

		t.totalFileCnt = int32(len(t.syncFileModeMap))
		// 批量写入
		var addChunks []*types.CodeChunk
		var deleteFilePaths map[string]struct{}
		// 遍历syncFileMap
		//TODO ignore cnt
		for k, v := range t.syncFileModeMap {
			op := types.FileOp(v)
			select {
			// timeout
			case <-t.ctx.Done():
				t.logger.Errorf("embeddingProcessor %d cancelled", t.taskHistoryId)
				return []error{errs.RunTimeout}
			default:
				switch op {
				case types.FileOpAdd:
					fallthrough
				case types.FileOpModify:
					chunks, err := t.splitFile(&types.SyncFile{Path: k})
					if errors.Is(err, embedding.ErrUnSupportedFileExt) {
						t.ignoreFileCnt++
						continue
					}
					if err != nil {
						t.failedFileCnt++
						continue
					}
					addChunks = append(addChunks, chunks...)
					t.successFileCnt++
				case types.FileOpDelete:
					t.logger.Debugf("process delete file %v", k)
					deleteFilePaths[k] = struct{}{}
				default:
					return []error{fmt.Errorf("unknown file op %s", v)}
				}
			}

		}
		if len(addChunks) > 0 {
			// 批量写入、删除
			// 保存到向量库
			err := t.svcCtx.VectorStore.UpsertCodeChunks(t.ctx, addChunks, vector.Options{
				CodebaseId:   t.msg.CodebaseID,
				CodebasePath: t.msg.CodebasePath,
				CodebaseName: t.msg.CodebaseName,
				SyncId:       t.msg.SyncID})
			if err != nil {
				logx.Errorf("embeddingProcessor upsert code chunks err:%v", err)
				t.failedFileCnt = t.successFileCnt
				t.successFileCnt = 0
				runErrs = append(runErrs, err)
			}
		}
		t.logger.Debugf("embeddingProcessor process add files end, chunk count:%d", len(addChunks))

		var deleteChunks []*types.CodeChunk
		if len(deleteFilePaths) > 0 {
			for k := range deleteFilePaths {
				deleteChunks = append(deleteChunks, &types.CodeChunk{
					CodebaseId:   t.msg.CodebaseID,
					CodebasePath: t.msg.CodebasePath,
					CodebaseName: t.msg.CodebaseName,
					FilePath:     k,
				})
			}
			resp, err := t.svcCtx.VectorStore.DeleteCodeChunks(t.ctx, deleteChunks, vector.Options{})
			if err != nil {
				logx.Errorf("embeddingProcessor delete code chunks err:%v", err)
				t.failedFileCnt += int32(len(deleteFilePaths))
				runErrs = append(runErrs, err)
			} else {
				t.successFileCnt += int32(len(deleteFilePaths))
			}
			t.logger.Debugf("embeddingProcessor process delete file resp:%v, err:%v", resp, err)
		}
		t.logger.Debugf("embeddingProcessor process delete files end, chunk count:%d", len(deleteChunks))

		return runErrs
	}(t)

	t.logger.Infof("embeddingProcessor end, msg:%v, total:%d, success:%d, failed:%d ", t.msg,
		t.totalFileCnt, t.successFileCnt, t.failedFileCnt)

	if t.handleIfTaskFinish(err) {
		// 如果任务整体失败，Process 返回错误
		return errors.Errorf("embeddingProcessor err:%s", utils.JoinErrors(err))
	}
	t.logger.Infof("embedding embeddingProcessor end successfully, cost: %d ms, msg: %+v, total: %d, success: %d, failed: %d",
		time.Since(start).Milliseconds(), t.msg, t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
	return nil // 任务成功，返回 nil
}

func (t *embeddingProcessor) updateTaskSuccess() {
	progress := float64(1)
	m := &model.IndexHistory{
		ID:                t.taskHistoryId,
		Status:            types.TaskStatusSuccess,
		Progress:          &progress,
		EndTime:           utils.CurrentTime(),
		TotalFileCount:    &t.totalFileCnt,
		TotalSuccessCount: &t.successFileCnt,
		TotalFailCount:    &t.failedFileCnt,
		TotalIgnoreCount:  &t.ignoreFileCnt,
	}

	// 更新任务
	if _, err := t.svcCtx.Querier.IndexHistory.WithContext(t.ctx).
		Where(t.svcCtx.Querier.IndexHistory.ID.Eq(m.ID)).
		Updates(m); err != nil {
		// 任务已经成功
		t.logger.Errorf("update embedding embeddingProcessor history %d failed: %v, model:%+v", t.msg.CodebaseID, err, m)
	}
}

func (t *embeddingProcessor) handleIfTaskFinish(runErrs []error) bool {
	if len(runErrs) > 0 {
		t.logger.Errorf("embeddingProcessor failed, err: %v, msg:%+v ", runErrs, t.msg)
		if errors.Is(runErrs[0], errs.InsertDatabaseFailed) {
			return true
		}
		status := types.TaskStatusFailed
		if errors.Is(runErrs[0], errs.RunTimeout) {
			status = types.TaskStatusTimeout
		}
		_, err := t.svcCtx.Querier.IndexHistory.WithContext(t.ctx).
			Where(t.svcCtx.Querier.IndexHistory.ID.Eq(t.taskHistoryId)).
			UpdateColumnSimple(t.svcCtx.Querier.IndexHistory.Status.Value(status),
				t.svcCtx.Querier.IndexHistory.ErrorMessage.Value(utils.JoinErrors(runErrs).Error()))
		if err != nil {
			t.logger.Errorf("update embedding embeddingProcessor history failed: %v", t.msg.CodebaseID, err)
		}
		return true
	} else {
		t.updateTaskSuccess()
		t.logger.Errorf("embeddingProcessor successfully, msg:%+v ", t.msg)
		return false
	}

}

func (t *embeddingProcessor) initTaskHistory() error {
	// 插入一条任务记录
	embedTaskHistory := &model.IndexHistory{
		SyncID:       t.msg.SyncID,
		CodebaseID:   t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath,
		TaskType:     taskTypeEmbedding,
		Status:       types.TaskStatusPending,
		StartTime:    utils.CurrentTime(),
	}
	if err := t.svcCtx.Querier.IndexHistory.WithContext(t.ctx).Save(embedTaskHistory); err != nil {
		t.logger.Errorf("insert embeddingProcessor history %v failed: %v", embedTaskHistory, err)
		return errs.InsertDatabaseFailed
	}
	t.taskHistoryId = embedTaskHistory.ID
	return nil
}

func (t *embeddingProcessor) splitFile(syncFile *types.SyncFile) ([]*types.CodeChunk, error) {
	t.logger.Debugf("process add file %v", syncFile)
	file, err := t.svcCtx.CodebaseStore.Read(t.ctx, t.msg.CodebasePath, syncFile.Path, types.ReadOptions{})
	if err != nil {
		return nil, err
	}
	// 切分文件
	return t.svcCtx.CodeSplitter.Split(&types.CodeFile{CodebaseId: t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath, CodebaseName: t.msg.CodebaseName, Path: syncFile.Path, Content: file})
}
