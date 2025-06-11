package job

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/structure"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/model"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	graphstore "github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"time"
)

const (
	taskTypeCodegraph = "codegraph"
)

type codegraphProcessor struct {
	ctx             context.Context
	svcCtx          *svc.ServiceContext
	msg             *types.CodebaseSyncMessage
	indexGenerator  *scip.IndexGenerator
	indexParser     *scip.IndexParser
	graphStore      graphstore.GraphStore
	logger          logx.Logger
	syncFileModeMap map[string]string
	taskHistoryId   int32
	totalFileCnt    int32
	successFileCnt  int32
	failedFileCnt   int32
	ignoreFileCnt   int32
}

func NewCodegraphProcessor(ctx context.Context,
	svcCtx *svc.ServiceContext,
	msg *types.CodebaseSyncMessage,
	syncFileModeMap map[string]string) (Processor, error) {
	config, err := scip.LoadConfig(svcCtx.Config.IndexTask.GraphTask.ConfFile)
	if err != nil {
		return nil, err
	}
	graphStore, err := graphstore.NewBadgerDBGraph(ctx, graphstore.WithPath(filepath.Join(msg.CodebasePath, types.CodebaseIndexDir)))
	if err != nil {
		return nil, err
	}

	graphBuilder := scip.NewIndexGenerator(config, svcCtx.CodebaseStore)
	graphParser := scip.NewIndexParser(ctx, svcCtx.CodebaseStore, graphStore)

	return &codegraphProcessor{
		ctx:             ctx,
		svcCtx:          svcCtx,
		msg:             msg,
		indexGenerator:  graphBuilder,
		indexParser:     graphParser,
		graphStore:      graphStore,
		logger:          logx.WithContext(ctx),
		syncFileModeMap: syncFileModeMap,
	}, nil
}

func (t *codegraphProcessor) Process() error {
	t.logger.Infof("start to execute codegraph processor %v", t.msg)

	// 启动一个协程去将所有文件的结构提取处理 TODO ,循环启协程并发处理， + ignorePatterns ignoreExts 去跳过部分文件/目录
	go t.parseCodeStructure()

	start := time.Now()

	err := func(t *codegraphProcessor) error {
		if err := t.initTaskHistory(); err != nil {
			return err
		}

		// 初始化 SCIP 索引生成器
		config, err := scip.LoadConfig(t.svcCtx.Config.IndexTask.GraphTask.ConfFile)
		if err != nil {
			return fmt.Errorf("failed to load SCIP config: %w", err)
		}
		indexGenerator := scip.NewIndexGenerator(config, t.svcCtx.CodebaseStore)

		// 生成 SCIP 索引
		if err := indexGenerator.Generate(t.ctx, t.msg.CodebasePath); err != nil {
			return fmt.Errorf("failed to generate SCIP index: %w", err)
		}
		defer indexGenerator.Cleanup()

		// TODO scip是整个项目一起解析，后面看能否换成tree-sitter统一做
		// 构建代码图
		err = t.indexGenerator.Generate(t.ctx, t.msg.CodebasePath)
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

	if t.handleIfTaskFailed(err) {
		return err
	}

	t.logger.Infof("codegraph processor end successfully, cost: %d s, msg: %v", time.Since(start), t.msg)
	return nil
}

func (t *codegraphProcessor) parseCodeStructure() {
	t.logger.Infof("start to parse code structure %v", t.msg)
	start := time.Now()
	var data []*codegraphpb.CodeStructure

	t.totalFileCnt = int32(len(t.syncFileModeMap))

	// 新增、修改的，重新解析； 删除的，直接删除
	var deleteFiles []string
	for path, mode := range t.syncFileModeMap {
		// 1. 每次回调开始时检查 context 是否已取消
		if mode == types.FileOpDelete {
			deleteFiles = append(deleteFiles, path)
			continue
		}
		select {
		// timeout
		case <-t.ctx.Done():
			t.logger.Errorf("embedding embeddingProcessor %d timeout", t.taskHistoryId)
			return
		default:
			content, err := t.svcCtx.CodebaseStore.Read(t.ctx, t.msg.CodebasePath, path, types.ReadOptions{})
			if err != nil {
				continue
			}
			structureParser, err := structure.NewStructureParser()
			if err != nil {
				t.logger.Errorf("init code structure parser err:%w", err)
				// 继续处理
				continue
			}
			parsedData, err := structureParser.Parse(&types.CodeFile{
				Path:         path,
				CodebasePath: t.msg.CodebasePath,
				Name:         filepath.Base(path),
				Content:      content,
			}, structure.ParseOptions{IncludeContent: false})
			if err != nil {
				t.logger.Errorf("code structure parse err:%w", err)
				// 继续处理
				continue
			}
			data = append(data, parsedData)
		}
	}

	t.logger.Infof("code structure parsed successfully, cost: %d s, msg: %v", time.Since(start), t.msg)

	dataSize := int32(len(data))
	if len(data) > 0 {
		if err := t.graphStore.BatchWriteCodeStructures(t.ctx, data); err != nil {
			t.logger.Errorf("code structure parsed data write err:%w", err)
			//
			t.failedFileCnt += dataSize
		} else {
			t.successFileCnt += dataSize
		}
	}
	deleteSize := int32(len(deleteFiles))
	if len(deleteFiles) > 0 {
		if err := t.graphStore.Delete(t.ctx, deleteFiles); err != nil {
			t.logger.Errorf("code structure parse delete docs error:%w", err)
			t.failedFileCnt += deleteSize
		} else {
			t.successFileCnt += deleteSize
		}
	}
	t.logger.Infof("code structure saved successfully, cost: %d s, msg: %v, total:%d, success:%d, failed:%d", time.Since(start),
		t.msg, t.totalFileCnt, t.successFileCnt, t.failedFileCnt)
}

func (t *codegraphProcessor) updateTaskSuccess() {
	progress := float64(1)
	m := &model.IndexHistory{
		ID:                int32(t.taskHistoryId),
		Status:            types.TaskStatusSuccess,
		Progress:          &progress,
		EndTime:           utils.CurrentTime(),
		TotalFileCount:    &t.totalFileCnt,
		TotalSuccessCount: &t.successFileCnt,
		TotalFailCount:    &t.failedFileCnt,
		TotalIgnoreCount:  &t.ignoreFileCnt,
	}

	// 更新任务
	if _, err := t.svcCtx.Querier.IndexHistory.WithContext(t.ctx).Updates(m); err != nil {
		// 任务已经成功
		t.logger.Errorf("update embedding embeddingProcessor history %d failed: %v, model:%v", t.msg.CodebaseID, err, m)
	}
}

func (t *codegraphProcessor) handleIfTaskFailed(err error) bool {
	if err != nil {
		t.logger.Errorf("failed to process file, err: %v, file:%v ", err, t.msg)
		if errors.Is(err, errs.InsertDatabaseFailed) {
			return true
		}
		status := types.TaskStatusFailed
		if errors.Is(err, errs.RunTimeout) {
			status = types.TaskStatusTimeout
		}
		_, err = t.svcCtx.Querier.IndexHistory.WithContext(t.ctx).
			Where(t.svcCtx.Querier.IndexHistory.ID.Eq(int32(t.taskHistoryId))).
			UpdateColumnSimple(t.svcCtx.Querier.IndexHistory.Status.Value(status),
				t.svcCtx.Querier.IndexHistory.ErrorMessage.Value(err.Error()))
		if err != nil {
			t.logger.Errorf("update embedding embeddingProcessor history failed: %v", t.msg.CodebaseID, err)
		}
		return true
	}
	return false
}

func (t *codegraphProcessor) initTaskHistory() error {
	// 插入一条任务记录
	embedTaskHistory := &model.IndexHistory{
		SyncID:       t.msg.SyncID,
		CodebaseID:   t.msg.CodebaseID,
		CodebasePath: t.msg.CodebasePath,
		TaskType:     taskTypeCodegraph,
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
