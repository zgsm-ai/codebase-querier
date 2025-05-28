package vector

import (
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

type Retriever interface {
	schema.Retriever
}

type retriever struct {
	vectorstores.Retriever
}

func NewRetriever(store VectorStore, embedder Embedder, cfg config.RetrieverConf) Retriever {
	return &retriever{
		Retriever: vectorstores.ToRetriever(store,
			cfg.NumDocuments,
			vectorstores.WithEmbedder(embedder),
			vectorstores.WithNameSpace(cfg.Namespace),
			vectorstores.WithScoreThreshold(cfg.ScoreThreshold),
			//vectorstores.WithDeduplicater(func(ctx context.Context, doc schema.Document) bool {
			//	return false
			//}),
			//vectorstores.WithFilters(func() {
			//
			//}),
		),
	}
}
