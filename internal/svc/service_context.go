package svc

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/mq"
	redisstore "github.com/zgsm-ai/codebase-indexer/internal/store/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
)

type ServiceContext struct {
	Config            config.Config
	CodebaseModel     model.CodebaseModel
	IndexHistoryModel model.IndexHistoryModel
	SyncHistoryModel  model.SyncHistoryModel
	CodebaseStore     codebase.Store
	MessageQueue      mq.MessageQueue
	DistLock          redisstore.DistributedLock
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
	client, err := redisstore.NewRedisClient(c.Redis)
	if err != nil {
		return nil, err
	}

	// 1. 创建消息队列实例
	messageQueue, err := mq.NewRedisMQ(ctx, client, c.MessageQueue.ConsumerGroup)
	if err != nil {
		client.Close() // If MQ creation fails, close the redis client
		return nil, err
	}

	// 2. 创建分布式锁实例
	lock, err := redisstore.NewRedisDistributedLock(client)
	if err != nil {
		messageQueue.Close() // If lock creation fails, close the message queue
		client.Close()       // And the redis client
		return nil, err
	}

	codebaseStore, err := codebase.New(ctx, c.CodeBaseStore)
	if err != nil {
		lock.Unlock(ctx, "") // Attempt to unlock if lock was acquired during creation (though NewRedisDistributedLock doesn't acquire)
		messageQueue.Close()
		client.Close()
		return nil, err
	}

	embedder, err := vector.NewEmbedder(ctx, c.VectorStore.Embedder)
	if err != nil {
		// Cleanup previous resources
		// No unlock needed as embedder creation doesn't acquire locks
		codebaseStore.Delete(ctx, "", "") // Assuming Delete can handle empty paths or nil objects safely
		messageQueue.Close()
		client.Close()
		return nil, err
	}
	reranker := vector.NewReranker(c.VectorStore.Reranker)

	vectorStore, err := vector.NewVectorStore(ctx, c.VectorStore, embedder, reranker)
	if err != nil {
		// Cleanup previous resources
		// No unlock needed
		codebaseStore.Delete(ctx, "", "") // Assuming Delete can handle empty paths or nil objects safely
		// No specific cleanup for embedder needed usually
		messageQueue.Close()
		client.Close()
		return nil, err
	}

	splitter, err := embedding.NewCodeSplitter(ctx,
		embedding.WithOverlapTokens(c.IndexTask.EmbeddingTask.OverlapTokens),
		embedding.WithMaxTokensPerChunk(c.IndexTask.EmbeddingTask.MaxTokensPerChunk),
	)
	if err != nil {
		// Cleanup previous resources
		// No unlock needed
		codebaseStore.Delete(ctx, "", "") // Assuming Delete can handle empty paths or nil objects safely
		// No specific cleanup for embedder or vectorStore needed usually
		messageQueue.Close()
		client.Close()
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
	svcCtx.DistLock = lock
	return svcCtx, err
}
