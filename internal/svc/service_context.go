package svc

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/index/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
)

type ServiceContext struct {
	Config            config.Config
	CodebaseModel     model.CodebaseModel
	IndexHistoryModel model.IndexHistoryModel
	SyncHistoryModel  model.SyncHistoryModel
	CodebaseStore     codebase.Store
	MessageQueue      mq.MessageQueue
	Retriever         vector.Retriever
	Embedder          vector.Embedder
	VectorStore       vector.Store
	CodeSplitter      embedding.CodeSplitter
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	var err error
	sqlConn := postgres.New(
		c.DB.DataSource,
	)

	messageQueue, err := mq.New(c.MessageQueue)
	if err != nil {
		return nil, err
	}

	codebaseStore, err := codebase.New(c.CodeBaseStore)
	if err != nil {
		return nil, err
	}

	embedder, err := vector.NewEmbedder(ctx, c.VectorStore.Embedder)
	if err != nil {
		return nil, err
	}

	vectorStore, err := vector.NewVectorStore(context.Background(), c.VectorStore, embedder)
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

	return &ServiceContext{
		Config:            c,
		CodebaseModel:     model.NewCodebaseModel(sqlConn),
		IndexHistoryModel: model.NewIndexHistoryModel(sqlConn),
		SyncHistoryModel:  model.NewSyncHistoryModel(sqlConn),
		CodebaseStore:     codebaseStore,
		MessageQueue:      messageQueue,
		VectorStore:       vectorStore,
		Retriever:         vector.NewRetriever(vectorStore, embedder, c.VectorStore.Retriever),
		Embedder:          embedder,
		CodeSplitter:      splitter,
	}, err
}
