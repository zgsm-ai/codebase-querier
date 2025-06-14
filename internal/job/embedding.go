package job

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	taskTypeEmbedding = "embedding"
)

type embeddingProcessor struct {
	baseProcessor
}

func NewEmbeddingProcessor(ctx context.Context,
	svcCtx *svc.ServiceContext,
	msg *types.CodebaseSyncMessage,
	syncFileModeMap map[string]string,
) (Processor, error) {
	return &embeddingProcessor{
		baseProcessor: baseProcessor{
			ctx:             ctx,
			svcCtx:          svcCtx,
			msg:             msg,
			syncFileModeMap: syncFileModeMap,
			logger:          logx.WithContext(ctx),
		},
	}, nil
}

type fileProcessResult struct {
	chunks []*types.CodeChunk
	err    error
	path   string
	op     types.FileOp
}

func (t *embeddingProcessor) Process() error {
	t.logger.Infof("start to execute embedding processor %+v", t.msg)
	start := time.Now()

	err := func(t *embeddingProcessor) error {
		if err := t.initTaskHistory(taskTypeEmbedding); err != nil {
			return err
		}

		t.totalFileCnt = int32(len(t.syncFileModeMap))
		var (
			addChunks       = make([]*types.CodeChunk, 0, t.totalFileCnt)
			deleteFilePaths = make(map[string]struct{})
			mu              sync.Mutex // 保护 addChunks
		)

		// 处理单个文件的函数
		processFile := func(path string, op types.FileOp) error {
			var result fileProcessResult
			result.path = path
			result.op = op

			select {
			case <-t.ctx.Done():
				return errs.RunTimeout
			default:
				switch op {
				case types.FileOpAdd, types.FileOpModify:
					chunks, err := t.splitFile(&types.SyncFile{Path: path})
					if err != nil {
						if errors.Is(err, embedding.ErrUnSupportedFileExt) {
							atomic.AddInt32(&t.ignoreFileCnt, 1)
							return nil
						}
						atomic.AddInt32(&t.failedFileCnt, 1)
						return err
					}
					mu.Lock()
					addChunks = append(addChunks, chunks...)
					mu.Unlock()
					atomic.AddInt32(&t.successFileCnt, 1)

				case types.FileOpDelete:
					mu.Lock()
					deleteFilePaths[path] = struct{}{}
					mu.Unlock()
					atomic.AddInt32(&t.successFileCnt, 1)

				default:
					return fmt.Errorf("unknown file op %s", op)
				}
			}
			return nil
		}

		// 使用基础结构的并发处理方法
		if err := t.processFilesConcurrently(processFile, t.svcCtx.Config.IndexTask.EmbeddingTask.MaxConcurrency); err != nil {
			return err
		}

		// 批量处理结果
		if len(addChunks) > 0 {
			err := t.svcCtx.VectorStore.UpsertCodeChunks(t.ctx, addChunks, vector.Options{
				CodebaseId:   t.msg.CodebaseID,
				CodebasePath: t.msg.CodebasePath,
				CodebaseName: t.msg.CodebaseName,
				SyncId:       t.msg.SyncID,
			})
			if err != nil {
				t.logger.Errorf("upsert code chunks failed: %v", err)
				t.failedFileCnt = t.successFileCnt
				t.successFileCnt = 0
				return err
			}
		}

		if len(deleteFilePaths) > 0 {
			var deleteChunks []*types.CodeChunk
			for path := range deleteFilePaths {
				deleteChunks = append(deleteChunks, &types.CodeChunk{
					CodebaseId:   t.msg.CodebaseID,
					CodebasePath: t.msg.CodebasePath,
					CodebaseName: t.msg.CodebaseName,
					FilePath:     path,
				})
			}
			err := t.svcCtx.VectorStore.DeleteCodeChunks(t.ctx, deleteChunks, vector.Options{})
			if err != nil {
				t.logger.Errorf("delete code chunks failed: %v", err)
				t.failedFileCnt += int32(len(deleteFilePaths))
				return err
			}
		}

		t.updateTaskSuccess()
		return nil
	}(t)

	if t.handleIfTaskFailed(err) {
		return errors.Errorf("embedding processor err:%v", err)
	}

	t.logger.Infof("embedding processor end successfully, cost: %d ms, msg: %+v, total: %d, success: %d, failed: %d",
		time.Since(start).Milliseconds(), t.msg, t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
	return nil
}

func (t *embeddingProcessor) splitFile(syncFile *types.SyncFile) ([]*types.CodeChunk, error) {
	t.logger.Debugf("process add file %v", syncFile)
	file, err := t.svcCtx.CodebaseStore.Read(t.ctx, t.msg.CodebasePath, syncFile.Path, types.ReadOptions{})
	if err != nil {
		return nil, err
	}
	// 切分文件
	return t.svcCtx.CodeSplitter.Split(&types.CodeFile{
		CodebaseId:   t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath,
		CodebaseName: t.msg.CodebaseName,
		Path:         syncFile.Path,
		Content:      file,
	})
}
