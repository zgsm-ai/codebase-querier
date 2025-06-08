package svc

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/postgres"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/job/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/store/cache"
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
	CodeSplitter      *embedding.CodeSplitter
	Cache             cache.Store[any]
	redisClient       *redis.Client // 保存Redis客户端引用以便关闭
}

// Close closes the shared Redis client
func (s *ServiceContext) Close() error {
	if s.redisClient != nil {
		return s.redisClient.Close()
	}
	return nil
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	var err error
	svcCtx := &ServiceContext{
		Config: c,
	}

	sqlConn := postgres.New(
		c.Database.DataSource,
	)

	// 创建Redis客户端
	client, err := redisstore.NewRedisClient(c.Redis)
	if err != nil {
		return nil, err
	}
	svcCtx.redisClient = client

	// 创建各个组件，共用Redis客户端
	messageQueue, err := mq.NewRedisMQ(ctx, client, c.MessageQueue.ConsumerGroup)
	if err != nil {
		return nil, err
	}

	lock, err := redisstore.NewRedisDistributedLock(client)
	if err != nil {
		return nil, err
	}

	cacheStore := cache.NewRedisStore[any](client)

	codebaseStore, err := codebase.NewLocalCodebase(ctx, c.CodeBaseStore)
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

	splitter, err := embedding.NewCodeSplitter(embedding.SplitOptions{
		MaxTokensPerChunk:          c.IndexTask.EmbeddingTask.MaxTokensPerChunk,
		SlidingWindowOverlapTokens: c.IndexTask.EmbeddingTask.OverlapTokens})
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
	svcCtx.DistLock = lock
	svcCtx.Cache = cacheStore
	return svcCtx, err
}
