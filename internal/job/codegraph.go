package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/job/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/job/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	graphstore "github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"io"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"time"
)

const (
	taskTypeCodegraph = "codegraph"
)

type codegraphProcessor struct {
	ctx            context.Context
	svcCtx         *svc.ServiceContext
	msg            *types.CodebaseSyncMessage
	indexGenerator *scip.IndexGenerator
	indexParser    *scip.IndexParser
	graphStore     graphstore.GraphStore
	logger         logx.Logger
	taskHistoryId  int64
}

func NewCodegraphProcessor(ctx context.Context, svcCtx *svc.ServiceContext, msg *types.CodebaseSyncMessage) (Processor, error) {
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
		ctx:            ctx,
		svcCtx:         svcCtx,
		msg:            msg,
		indexGenerator: graphBuilder,
		indexParser:    graphParser,
		graphStore:     graphStore,
		logger:         logx.WithContext(ctx),
	}, nil
}

func (t *codegraphProcessor) Process() error {
	t.logger.Infof("start to execute codegraph processor %v", t.msg)

	// 启动一个协程去将所有文件的结构提取处理
	go func(t *codegraphProcessor) {
		t.logger.Infof("start to parse code structure %v", t.msg)
		start := time.Now()
		var data []*codegraphpb.CodeStructure
		err := t.svcCtx.CodebaseStore.Walk(t.ctx, t.msg.CodebasePath, "", func(walkCtx *codebase.WalkContext, reader io.ReadCloser) error {

			// 1. 每次回调开始时检查 context 是否已取消
			if t.ctx.Err() != nil {
				t.logger.Info("code structure parse, context cancelled, exit.")
				return t.ctx.Err() // 返回 context 错误以终止遍历
			}

			defer reader.Close()
			bytes, err := io.ReadAll(reader)
			if err != nil {
				t.logger.Errorf("code structure parse, read err:%w", err)
				// 继续处理
				return nil
			}
			structureParser, err := codegraph.NewStructureParser()
			if err != nil {
				t.logger.Errorf("init code structure parser err:%w", err)
				// 继续处理
				return nil
			}
			parsedData, err := structureParser.Parse(&types.CodeFile{
				Path:         walkCtx.RelativePath,
				CodebasePath: t.msg.CodebasePath,
				Name:         walkCtx.Info.Name,
				Content:      bytes,
			})
			if err != nil {
				t.logger.Errorf("code structure parse err:%w", err)
				// 继续处理
				return nil
			}
			data = append(data, parsedData)
			return nil
		})
		t.logger.Infof("code structure parsed successfully, cost: %d s, msg: %v, err:%v", time.Since(start), t.msg, err)

		if len(data) != 0 {
			if err = t.graphStore.BatchWriteCodeStructures(t.ctx, data); err != nil {
				t.logger.Errorf("code structure parsed data write err:%w", err)
			}
		}
		t.logger.Infof("code structure saved successfully, cost: %d s, msg: %v, err:%v", time.Since(start), t.msg, err)
	}(t)

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

	t.logger.Infof("codegraph processor end successfully, cost: %d s, : %v", time.Since(start), t.msg)
	return nil
}

func (t *codegraphProcessor) updateTaskSuccess() {
	m := &model.IndexHistory{
		Id:       t.taskHistoryId,
		Status:   types.TaskStatusSuccess,
		Progress: sql.NullFloat64{Float64: 1, Valid: true},
		EndTime:  sql.NullTime{Time: time.Now(), Valid: true},
	}

	// 更新任务
	if err := t.svcCtx.IndexHistoryModel.Update(t.ctx, m); err != nil {
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
		err = t.svcCtx.IndexHistoryModel.UpdateStatus(t.ctx, t.taskHistoryId, status, err.Error())
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
