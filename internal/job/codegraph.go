package job

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/structure"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	graphstore "github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	taskTypeCodegraph = "codegraph"
)

type codegraphProcessor struct {
	baseProcessor
	indexGenerator *scip.IndexGenerator
	indexParser    *scip.IndexParser
	graphStore     graphstore.GraphStore
}

func NewCodegraphProcessor(ctx context.Context,
	svcCtx *svc.ServiceContext,
	msg *types.CodebaseSyncMessage,
	syncFileModeMap map[string]string) (Processor, error) {

	graphStore, err := graphstore.NewBadgerDBGraph(ctx, graphstore.WithPath(filepath.Join(msg.CodebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}

	graphBuilder := scip.NewIndexGenerator(svcCtx.CodegraphConf, svcCtx.CodebaseStore)
	graphParser := scip.NewIndexParser(ctx, svcCtx.CodebaseStore, graphStore)

	return &codegraphProcessor{
		baseProcessor: baseProcessor{
			ctx:             ctx,
			svcCtx:          svcCtx,
			msg:             msg,
			syncFileModeMap: syncFileModeMap,
			logger:          logx.WithContext(ctx),
		},
		indexGenerator: graphBuilder,
		indexParser:    graphParser,
		graphStore:     graphStore,
	}, nil
}

func (t *codegraphProcessor) Process() error {
	t.logger.Infof("start to execute codegraph processor %v", t.msg)
	defer t.graphStore.Close()
	var wait sync.WaitGroup
	// 启动一个协程去将所有文件的结构提取处理
	go func() {
		wait.Add(1)
		defer wait.Done()
		t.parseCodeStructure()
	}()

	start := time.Now()

	err := func(t *codegraphProcessor) error {
		wait.Add(1)
		defer wait.Done()
		if err := t.initTaskHistory(taskTypeCodegraph); err != nil {
			return err
		}

		defer t.indexGenerator.Cleanup()

		// TODO scip是整个项目一起解析，后面看能否换成tree-sitter统一做
		// 构建代码图
		err := t.indexGenerator.Generate(t.ctx, t.msg.CodebasePath)
		if err != nil {
			return fmt.Errorf("failed to generate %s  index file: %w", t.msg.CodebasePath, err)
		}

		// 解析并保存
		if err = t.indexParser.ParseSCIPFile(t.ctx, t.msg.CodebasePath, scip.DefaultIndexFilePath()); err != nil {
			return fmt.Errorf("failed to save code graph: %w", err)
		}

		// 更新任务状态为成功
		t.updateTaskSuccess()

		return nil
	}(t)

	wait.Wait()

	if t.handleIfTaskFailed(err) {
		return err
	}

	t.logger.Infof("codegraph processor end successfully, cost: %d s, msg: %v", time.Since(start), t.msg)
	return nil
}

type fileStructureResult struct {
	data *codegraphpb.CodeStructure
	err  error
	path string
	op   types.FileOp
}

func (t *codegraphProcessor) parseCodeStructure() {
	t.logger.Infof("start to parse code structure %v", t.msg)
	start := time.Now()

	t.totalFileCnt = int32(len(t.syncFileModeMap))
	var (
		structureData = make([]*codegraphpb.CodeStructure, 0, t.totalFileCnt)
		deleteFiles   = make([]string, 0, t.totalFileCnt)
		mu            sync.Mutex // 保护 structureData 和 deleteFiles
	)

	// 处理单个文件的函数
	processFile := func(path string, op types.FileOp) error {
		select {
		case <-t.ctx.Done():
			return errs.RunTimeout
		default:
			switch op {
			case types.FileOpAdd, types.FileOpModify:
				content, err := t.svcCtx.CodebaseStore.Read(t.ctx, t.msg.CodebasePath, path, types.ReadOptions{})
				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("read file failed: %w", err)
				}

				structureParser, err := structure.NewStructureParser()
				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("init parser failed: %w", err)
				}

				parsedData, err := structureParser.Parse(&types.CodeFile{
					Path:         path,
					CodebasePath: t.msg.CodebasePath,
					Name:         filepath.Base(path),
					Content:      content,
				}, structure.ParseOptions{IncludeContent: false})

				if parser.IsNotSupportedFileError(err) {
					atomic.AddInt32(&t.ignoreFileCnt, 1)
					return nil
				}

				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("parse file failed: %w", err)
				}

				mu.Lock()
				structureData = append(structureData, parsedData)
				mu.Unlock()
				atomic.AddInt32(&t.successFileCnt, 1)

			case types.FileOpDelete:
				mu.Lock()
				deleteFiles = append(deleteFiles, path)
				mu.Unlock()
				atomic.AddInt32(&t.successFileCnt, 1)

			default:
				return fmt.Errorf("unknown file op %s", op)
			}
		}
		return nil
	}

	// 使用基础结构的并发处理方法
	if err := t.processFilesConcurrently(processFile, t.svcCtx.Config.IndexTask.GraphTask.MaxConcurrency); err != nil {
		t.logger.Errorf("process files failed: %v", err)
		return
	}

	// 批量处理结果
	dataSize := int32(len(structureData))
	if len(structureData) > 0 {
		if err := t.graphStore.BatchWriteCodeStructures(t.ctx, structureData); err != nil {
			t.logger.Errorf("write code structures failed: %v", err)
			t.failedFileCnt += dataSize
		} else {
			t.successFileCnt += dataSize
		}
	}

	deleteSize := int32(len(deleteFiles))
	if len(deleteFiles) > 0 {
		if err := t.graphStore.Delete(t.ctx, deleteFiles); err != nil {
			t.logger.Errorf("delete code structures failed: %v", err)
			t.failedFileCnt += deleteSize
		} else {
			t.successFileCnt += deleteSize
		}
	}

	t.logger.Infof("code structure saved successfully, cost: %d s, msg: %v, total:%d, success:%d, failed:%d",
		time.Since(start).Seconds(), t.msg, t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
}
