package vector

import (
	"context"
	"errors"
	"fmt"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

type Store interface {
	vectorstores.VectorStore
	UpsertCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options ...vectorstores.Option) error
	DeleteCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options ...vectorstores.Option) (any, error)
	Query(ctx context.Context, query string, topK int, options ...vectorstores.Option) ([]types.SemanticFileItem, error)
	Close()
}

const vectorWeaviate = "weaviate"

func NewVectorStore(ctx context.Context, cfg config.VectorStoreConf, embedder Embedder, reranker Reranker) (Store, error) {
	var vectorStoreImpl Store
	var err error
	switch cfg.Type {
	case vectorWeaviate:
		if cfg.Weaviate.Endpoint == types.EmptyString {
			return nil, errors.New("vector conf weaviate is required for weaviate type")
		}
		vectorStoreImpl, err = New(ctx, cfg, embedder, reranker)
	default:
		err = fmt.Errorf("unsupported vector type: %s", cfg.Type)
	}

	if err != nil {
		return nil, err
	}
	return vectorStoreImpl, nil
}
