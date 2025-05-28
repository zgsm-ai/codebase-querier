package vector

import (
	"github.com/tmc/langchaingo/vectorstores/weaviate"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

func newWeaviate(cfg config.VectorStoreConf, embedder Embedder) (VectorStore, error) {
	return weaviate.New(
		weaviate.WithHost(cfg.Weaviate.Host),
		weaviate.WithAPIKey(cfg.Weaviate.APIKey),
		weaviate.WithIndexName(cfg.Weaviate.IndexName),
		weaviate.WithNameSpace(cfg.Weaviate.Namespace),
		weaviate.WithEmbedder(embedder),
	)
}
