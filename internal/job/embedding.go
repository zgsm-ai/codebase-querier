package job

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
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
	taskHistoryId   int64
	totalFileCnt    int
	successFileCnt  int
	failedFileCnt   int
	ignoreFileCnt   int
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

	t.logger.Infof("start to execute embedding embeddingProcessor %v", t.msg)
	start := time.Now()

	err := func(t *embeddingProcessor) error {

		if err := t.initTaskHistory(); err != nil {
			return err
		}

		// TODO 并发处理任务
		// fileCnt := len(syncFileMap)
		// maxConcurrency := t.svcCtx.Config.IndexTask.EmbeddingTask.MaxConcurrency

		t.totalFileCnt = len(t.syncFileModeMap)
		//TODO ignore cnt
		for k, v := range t.syncFileModeMap {
			select {
			// timeout
			case <-t.ctx.Done():
				t.logger.Errorf("embedding embeddingProcessor %d timeout", t.taskHistoryId)
				return errs.RunTimeout
			default:
				if err := t.processFile(&types.SyncFile{Path: k, Op: types.FileOp(v)}); err != nil {
					t.logger.Errorf("update embedding embeddingProcessor file %s failed: %v", k, err)
					t.failedFileCnt++
				} else {
					t.successFileCnt++
				}
			}

		}

		// update embeddingProcessor when success
		t.updateTaskSuccess()

		return nil
	}(t)

	if t.handleIfTaskFailed(err) {
		// 如果任务整体失败，Process 返回错误
		return err
	}

	t.logger.Infof("embedding embeddingProcessor end successfully, cost: %d s, msg: %v, total: %d, success: %d, failed: %d",
		time.Since(start), t.msg, t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
	return nil // 任务成功，返回 nil
}

func (t *embeddingProcessor) updateTaskSuccess() {
	m := &model.IndexHistory{
		Id:               t.taskHistoryId,
		Status:           types.TaskStatusSuccess,
		Progress:         sql.NullFloat64{Float64: 1, Valid: true},
		EndTime:          sql.NullTime{Time: time.Now(), Valid: true},
		TotalFileCount:   int64(t.totalFileCnt),
		SuccessFileCount: int64(t.successFileCnt),
		FailFileCount:    int64(t.failedFileCnt),
		IgnoreFileCount:  int64(t.ignoreFileCnt),
	}

	// 更新任务
	if err := t.svcCtx.IndexHistoryModel.Update(t.ctx, m); err != nil {
		// 任务已经成功
		t.logger.Errorf("update embedding embeddingProcessor history %d failed: %v, model:%v", t.msg.CodebaseID, err, m)
	}
}

func (t *embeddingProcessor) handleIfTaskFailed(err error) bool {
	if err != nil {
		t.logger.Errorf("failed to process file, err: %v, file:%v ", err, t.msg)
		if errors.Is(err, errs.InsertDatabaseFailed) {
			return true
		}
		status := types.TaskStatusFailed
		if errors.Is(err, errs.RunTimeout) {
			status = types.TaskStatusTimeout
		}
		err = t.svcCtx.IndexHistoryModel.UpdateStatus(t.ctx, t.taskHistoryId, status, err.Error())
		if err != nil {
			t.logger.Errorf("update embedding embeddingProcessor history failed: %v", t.msg.CodebaseID, err)
		}
		return true
	}
	return false
}

func (t *embeddingProcessor) initTaskHistory() error {
	// 插入一条任务记录
	embedTaskHistory := &model.IndexHistory{
		SyncId:       t.msg.SyncID,
		CodebaseId:   t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath,
		TaskType:     taskTypeEmbedding,
		Status:       types.TaskStatusPending,
		Progress:     sql.NullFloat64{Float64: 0, Valid: true},
		StartTime:    sql.NullTime{Time: time.Now(), Valid: true},
	}
	if _, err := t.svcCtx.IndexHistoryModel.Insert(t.ctx, embedTaskHistory); err != nil {
		t.logger.Errorf("insert embeddingProcessor history %v failed: %v", embedTaskHistory, err)
		return errs.InsertDatabaseFailed
	}
	t.taskHistoryId = embedTaskHistory.Id
	return nil
}

func (t *embeddingProcessor) processFile(syncFile *types.SyncFile) error {

	t.logger.Debugf("start process file %v", syncFile)
	switch syncFile.Op {
	case types.FileOpAdd:
	case types.FileOpModify:
		err2 := t.processAddFile(syncFile)
		if err2 != nil {
			return err2
		}
	case types.FileOpDelete:
		t.logger.Debugf("process delete file %v", syncFile)
		return t.processDeleteFile(syncFile)
	default:
		return fmt.Errorf("unknown file op %s", syncFile.Op)
	}
	return nil
}

func (t *embeddingProcessor) processAddFile(syncFile *types.SyncFile) error {
	t.logger.Debugf("process add file %v", syncFile)
	file, err := t.svcCtx.CodebaseStore.Read(t.ctx, t.msg.CodebasePath, syncFile.Path, types.ReadOptions{})
	if err != nil {
		return err
	}
	// 切分文件
	codeChunks, err := t.svcCtx.CodeSplitter.Split(&types.CodeFile{CodebaseId: t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath, CodebaseName: t.msg.CodebaseName, Path: syncFile.Path, Content: file})
	if err != nil {
		return err
	}

	// 保存到向量库
	err = t.svcCtx.VectorStore.UpsertCodeChunks(t.ctx, codeChunks)
	if err != nil {
		return err
	}
	t.logger.Debugf("process add file successfully %v", syncFile)
	return nil
}

func (t *embeddingProcessor) processDeleteFile(file *types.SyncFile) error {
	del := []*types.CodeChunk{
		{
			CodebaseId:   t.msg.CodebaseID,
			CodebasePath: t.msg.CodebasePath,
			CodebaseName: t.msg.CodebaseName,
			FilePath:     file.Path,
		},
	}

	resp, err := t.svcCtx.VectorStore.DeleteCodeChunks(t.ctx, del)
	t.logger.Debugf("process delete file resp:%v", resp)
	return err
}
