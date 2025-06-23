package job

import (
	"context"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/definition"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	graphstore "github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type codegraphProcessor struct {
	baseProcessor
	indexGenerator *scip.IndexGenerator
	indexParser    *scip.IndexParser
	graphStore     graphstore.GraphStore
}

func NewCodegraphProcessor(
	svcCtx *svc.ServiceContext,
	params *IndexTaskParams,
	syncFileModeMap map[string]string) (Processor, error) {

	graphStore, err := graphstore.NewBadgerDBGraph(graphstore.WithPath(filepath.Join(params.CodebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}

	graphBuilder := scip.NewIndexGenerator(svcCtx.CmdLogger, svcCtx.CodegraphConf, svcCtx.CodebaseStore)
	graphParser := scip.NewIndexParser(svcCtx.CodebaseStore, graphStore)

	return &codegraphProcessor{
		baseProcessor: baseProcessor{
			svcCtx:          svcCtx,
			params:          params,
			syncFileModeMap: syncFileModeMap,
		},
		indexGenerator: graphBuilder,
		indexParser:    graphParser,
		graphStore:     graphStore,
	}, nil
}

func (t *codegraphProcessor) Process(ctx context.Context) error {
	tracer.WithTrace(ctx).Infof("start to execute codegraph task for params: %+v", t.params)
	defer t.graphStore.Close()
	var wait sync.WaitGroup
	// 启动一个协程去将所有文件的结构提取处理
	go func() {
		wait.Add(1)
		defer wait.Done()
		t.parseCodeStructure(ctx)
	}()

	start := time.Now()

	err := func(t *codegraphProcessor) error {
		wait.Add(1)
		defer wait.Done()
		if err := t.initTaskHistory(ctx, types.TaskTypeCodegraph); err != nil {
			return err
		}

		defer t.indexGenerator.Cleanup()

		// TODO scip是整个项目一起解析，后面看能否换成tree-sitter统一做
		// 构建代码图
		err := t.indexGenerator.Generate(ctx, t.params.CodebasePath)
		if err != nil {
			return fmt.Errorf("codegraph task failed to generate %s index file: %w", t.params.CodebasePath, err)
		}

		// 解析并保存
		if err = t.indexParser.ProcessScipIndexFile(ctx, t.params.CodebasePath, scip.DefaultIndexFilePath()); err != nil {
			return fmt.Errorf("codegraph task failed to parse & save code graph: %w", err)
		}

		// 更新任务状态为成功
		if err = t.updateTaskSuccess(ctx); err != nil {
			tracer.WithTrace(ctx).Errorf("codegraph task failed to update status success , err:%v", err)
		}

		return nil
	}(t)

	wait.Wait()

	if t.handleIfTaskFailed(ctx, err) {
		return fmt.Errorf("codegraph task failed to update status:%w", err)
	}

	tracer.WithTrace(ctx).Infof("codegraph processor end successfully, cost: %d ms, params:%+v", time.Since(start).Milliseconds(), t.params)
	return nil
}

func (t *codegraphProcessor) parseCodeStructure(ctx context.Context) {
	tracer.WithTrace(ctx).Info("start to execute code structure task.")
	start := time.Now()

	t.totalFileCnt = int32(len(t.syncFileModeMap))
	var (
		structureData = make([]*codegraphpb.CodeDefinition, 0, t.totalFileCnt)
		deleteFiles   = make([]string, 0, t.totalFileCnt)
		mu            sync.Mutex // 保护 structureData 和 deleteFiles
	)

	// 处理单个文件的函数
	processFile := func(path string, op types.FileOp) error {
		select {
		case <-ctx.Done():
			return errs.RunTimeout
		default:
			switch op {
			case types.FileOpAdd, types.FileOpModify:
				content, err := t.svcCtx.CodebaseStore.Read(ctx, t.params.CodebasePath, path, types.ReadOptions{})
				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("code structure task read file failed: %w", err)
				}

				structureParser, err := definition.NeDefinitionParser()
				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("code structure task init parser failed: %w", err)
				}

				parsedData, err := structureParser.Parse(ctx, &types.SourceFile{
					Path:         path,
					CodebasePath: t.params.CodebasePath,
					Name:         filepath.Base(path),
					Content:      content,
				}, definition.ParseOptions{IncludeContent: false})

				if parser.IsNotSupportedFileError(err) {
					atomic.AddInt32(&t.ignoreFileCnt, 1)
					return nil
				}

				if err != nil {
					atomic.AddInt32(&t.failedFileCnt, 1)
					return fmt.Errorf("code structure task parse file failed: %w", err)
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
				return fmt.Errorf("code structure task unknown file op %s", op)
			}
		}
		return nil
	}

	// 使用基础结构的并发处理方法
	if err := t.processFilesConcurrently(ctx, processFile, t.svcCtx.Config.IndexTask.GraphTask.MaxConcurrency); err != nil {
		tracer.WithTrace(ctx).Errorf("code structure task failed: %v", err)
		return
	}

	// 批量处理结果
	dataSize := int32(len(structureData))
	if len(structureData) > 0 {
		if err := t.graphStore.BatchWriteCodeStructures(ctx, structureData); err != nil {
			tracer.WithTrace(ctx).Errorf("save code structure data failed: %v", err)
			t.failedFileCnt += dataSize
		} else {
			t.successFileCnt += dataSize
		}
	}

	deleteSize := int32(len(deleteFiles))
	if len(deleteFiles) > 0 {
		if err := t.graphStore.Delete(ctx, deleteFiles); err != nil {
			tracer.WithTrace(ctx).Errorf("delete code structures failed: %v", err)
			t.failedFileCnt += deleteSize
		} else {
			t.successFileCnt += deleteSize
		}
	}

	tracer.WithTrace(ctx).Infof("code structure end successfully, cost: %d ms, total: %d, success: %d, failed: %d",
		time.Since(start).Milliseconds(), t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
}
