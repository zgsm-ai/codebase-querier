package job

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
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

func NewEmbeddingProcessor(
	svcCtx *svc.ServiceContext,
	msg *types.CodebaseSyncMessage,
	syncFileModeMap map[string]string,
) (Processor, error) {
	return &embeddingProcessor{
		baseProcessor: baseProcessor{
			svcCtx:          svcCtx,
			msg:             msg,
			syncFileModeMap: syncFileModeMap,
		},
	}, nil
}

type fileProcessResult struct {
	chunks []*types.CodeChunk
	err    error
	path   string
	op     types.FileOp
}

func (t *embeddingProcessor) Process(ctx context.Context) error {
	tracer.WithTrace(ctx).Infof("start to execute embedding task, msg: %+v", t.msg)
	start := time.Now()

	err := func(t *embeddingProcessor) error {
		if err := t.initTaskHistory(ctx, taskTypeEmbedding); err != nil {
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
			case <-ctx.Done():
				return errs.RunTimeout
			default:
				switch op {
				case types.FileOpAdd, types.FileOpModify:
					chunks, err := t.splitFile(ctx, &types.SyncFile{Path: path})
					if err != nil {
						if parser.IsNotSupportedFileError(err) {
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
					return fmt.Errorf("embedding task unknown file op %s", op)
				}
			}
			return nil
		}

		// 使用基础结构的并发处理方法
		if err := t.processFilesConcurrently(ctx, processFile, t.svcCtx.Config.IndexTask.EmbeddingTask.MaxConcurrency); err != nil {
			return err
		}

		// 批量处理结果
		if len(addChunks) > 0 {
			err := t.svcCtx.VectorStore.UpsertCodeChunks(ctx, addChunks, vector.Options{
				CodebaseId:   t.msg.CodebaseID,
				CodebasePath: t.msg.CodebasePath,
				CodebaseName: t.msg.CodebaseName,
				SyncId:       t.msg.SyncID,
			})
			if err != nil {
				tracer.WithTrace(ctx).Errorf("embedding task upsert code chunks failed: %v", err)
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
			err := t.svcCtx.VectorStore.DeleteCodeChunks(ctx, deleteChunks, vector.Options{})
			if err != nil {
				tracer.WithTrace(ctx).Errorf("embedding task delete code chunks failed: %v", err)
				t.failedFileCnt += int32(len(deleteFilePaths))
				return err
			}
		}

		t.updateTaskSuccess(ctx)
		return nil
	}(t)

	if t.handleIfTaskFailed(ctx, err) {
		return errors.Errorf("embedding task failed to update status:%v", err)
	}

	tracer.WithTrace(ctx).Infof("embedding task end successfully, cost: %d ms, total: %d, success: %d, failed: %d,msg:%+v",
		time.Since(start).Milliseconds(), t.totalFileCnt, t.successFileCnt, t.failedFileCnt, t.msg)
	return nil
}

func (t *embeddingProcessor) splitFile(ctx context.Context, syncFile *types.SyncFile) ([]*types.CodeChunk, error) {
	file, err := t.svcCtx.CodebaseStore.Read(ctx, t.msg.CodebasePath, syncFile.Path, types.ReadOptions{})
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
