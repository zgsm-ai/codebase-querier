package svc

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/internal/task/embedding"
)

type ServiceContext struct {
	Config            config.Config
	CodebaseModel     model.CodebaseModel
	IndexHistoryModel model.IndexHistoryModel
	SyncHistoryModel  model.SyncHistoryModel
	CodebaseStore     codebase.Store
	MessageQueue      mq.MessageQueue
	Embedder          vector.Embedder
	VectorStore       vector.Store
	CodeSplitter      embedding.CodeSplitter
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	var err error
	svcCtx := &ServiceContext{
		Config: c,
	}

	sqlConn := postgres.New(
		c.Database.DataSource,
	)

	messageQueue, err := mq.New(ctx, c.MessageQueue)
	if err != nil {
		return nil, err
	}

	codebaseStore, err := codebase.New(ctx, c.CodeBaseStore)
	if err != nil {
		return nil, err
	}

	embedder, err := vector.NewEmbedder(ctx, c.VectorStore.Embedder)
	if err != nil {
		return nil, err
	}
	reranker := vector.NewReranker(c.VectorStore.Reranker)

	vectorStore, err := vector.NewVectorStore(ctx, c.VectorStore, embedder, reranker)
	if err != nil {
		return nil, err
	}

	splitter, err := embedding.NewCodeSplitter(ctx,
		embedding.WithOverlapTokens(c.IndexTask.EmbeddingTask.OverlapTokens),
		embedding.WithMaxTokensPerChunk(c.IndexTask.EmbeddingTask.MaxTokensPerChunk),
	)
	if err != nil {
		return nil, err
	}

	svcCtx.CodebaseModel = model.NewCodebaseModel(sqlConn)
	svcCtx.IndexHistoryModel = model.NewIndexHistoryModel(sqlConn)
	svcCtx.SyncHistoryModel = model.NewSyncHistoryModel(sqlConn)
	svcCtx.CodebaseStore = codebaseStore
	svcCtx.MessageQueue = messageQueue
	svcCtx.VectorStore = vectorStore
	svcCtx.Embedder = embedder
	svcCtx.CodeSplitter = splitter

	return svcCtx, err
}
