package svc

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/structure"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/dao/query"
	"github.com/zgsm-ai/codebase-indexer/internal/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/store/cache"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/store/database"
	"github.com/zgsm-ai/codebase-indexer/internal/store/mq"
	redisstore "github.com/zgsm-ai/codebase-indexer/internal/store/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config          config.Config
	CodegraphConf   *config.CodegraphConfig
	db              *gorm.DB
	Querier         *query.Query
	CodebaseStore   codebase.Store
	MessageQueue    mq.MessageQueue
	DistLock        redisstore.DistributedLock
	Embedder        vector.Embedder
	VectorStore     vector.Store
	CodeSplitter    *embedding.CodeSplitter
	Cache           cache.Store[any]
	redisClient     *redis.Client // 保存Redis客户端引用以便关闭
	StructureParser *structure.Parser
}

// Close closes the shared Redis client and database connection
func (s *ServiceContext) Close() error {
	var errs []error
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.db != nil {
		if err := database.CloseDB(s.db); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0] // 返回第一个错误
	}
	return nil
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	var err error
	svcCtx := &ServiceContext{
		Config: c,
	}
	svcCtx.CodegraphConf = config.MustLoadCodegraphConfig(c.IndexTask.GraphTask.ConfFile)

	// 初始化数据库连接
	db, err := database.NewPostgresDB(c.Database)
	if err != nil {
		return nil, err
	}
	svcCtx.db = db

	querier := query.Use(db)
	svcCtx.Querier = querier

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

	embedder, err := vector.NewEmbedder(c.VectorStore.Embedder)
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
	parser, err := structure.NewStructureParser()
	if err != nil {
		return nil, err
	}

	svcCtx.StructureParser = parser
	svcCtx.CodebaseStore = codebaseStore
	svcCtx.MessageQueue = messageQueue
	svcCtx.VectorStore = vectorStore
	svcCtx.Embedder = embedder
	svcCtx.CodeSplitter = splitter
	svcCtx.DistLock = lock
	svcCtx.Cache = cacheStore

	return svcCtx, err
}
