package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"path/filepath"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SummaryLogic {
	return &SummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SummaryLogic) Summary(req *types.IndexSummaryRequest) (resp *types.IndexSummaryResonseData, err error) {
	clientId := req.ClientId
	clientPath := req.CodebasePath

	// 查找代码库记录
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	if err != nil {
		return nil, err
	}

	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	// 获取向量索引状态
	embeddingSummary, err := l.svcCtx.VectorStore.GetIndexSummary(ctx, codebase.ID, codebase.Path)
	if err != nil {
		return nil, err
	}

	// 获取图索引状态
	graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebase.Path, types.CodebaseIndexDir)))
	if err != nil {
		return nil, fmt.Errorf("failed to open graph store, err:%w", err)
	}
	defer graphStore.Close()

	codegraphSummary, err := graphStore.GetIndexSummary(ctx, codebase.ID, codebase.Path)
	if err != nil {
		return nil, err
	}

	// 获取文件总数
	codegraphStatus, embeddingStatus := types.TaskStatusPending, types.TaskStatusPending
	embeddingIndexTask, err := l.svcCtx.Querier.IndexHistory.GetLatestTaskHistory(ctx, codebase.ID, types.TaskTypeEmbedding)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("failed to get latest codegraph index task, err:%v", err)
	}
	codegraphIndexTask, err := l.svcCtx.Querier.IndexHistory.GetLatestTaskHistory(ctx, codebase.ID, types.TaskTypeCodegraph)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("failed to get latest codegraph index task, err:%v", err)
	}
	if embeddingIndexTask != nil {
		embeddingStatus = convertStatus(embeddingIndexTask.Status)
	} else if embeddingSummary.TotalChunks > 0 {
		embeddingStatus = types.TaskStatusSuccess
	}

	if codegraphIndexTask != nil {
		codegraphStatus = convertStatus(codegraphIndexTask.Status)
	} else if codegraphSummary.TotalFiles > 0 {
		codegraphStatus = types.TaskStatusSuccess
	}

	resp = &types.IndexSummaryResonseData{
		TotalFiles: int(codebase.FileCount),
		Embedding: types.EmbeddingSummary{
			Status:      embeddingStatus,
			TotalFiles:  embeddingSummary.TotalFiles,
			TotalChunks: embeddingSummary.TotalChunks,
		},
		CodeGraph: types.CodeGraphSummary{
			Status:     codegraphStatus,
			TotalFiles: codegraphSummary.TotalFiles,
		},
	}

	return resp, nil
}

func convertStatus(status string) string {
	var embeddingStatus string
	switch status {
	case types.TaskStatusSuccess:
		embeddingStatus = types.TaskStatusSuccess
	case types.TaskStatusRunning:
		embeddingStatus = types.TaskStatusRunning
	case types.TaskStatusPending:
		embeddingStatus = types.TaskStatusPending
	default:
		embeddingStatus = types.TaskStatusFailed
	}
	return embeddingStatus
}
