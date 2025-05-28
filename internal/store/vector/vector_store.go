package vector

import (
	"context"
	"errors"
	"fmt"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

type VectorStore interface {
	vectorstores.VectorStore
}

const vectorWeaviate = "weaviate"

type vectorStore struct {
	logger          logx.Logger
	config          config.VectorStoreConf
	vectorStoreImpl vectorstores.VectorStore
}

func NewVectorStore(ctx context.Context, cfg config.VectorStoreConf, embed Embedder) (VectorStore, error) {
	var vectorStoreImpl vectorstores.VectorStore
	var err error
	switch cfg.Type {
	case vectorWeaviate:
		if cfg.Weaviate == nil {
			return nil, errors.New("vector conf weaviate is required for weaviate type")
		}
		vectorStoreImpl, err = newWeaviate(cfg, embed)
	default:
		err = fmt.Errorf("unsupported vector type: %s", cfg.Type)
	}

	if err != nil {
		return nil, err
	}
	return &vectorStore{
		logger:          logx.WithContext(ctx),
		config:          cfg,
		vectorStoreImpl: vectorStoreImpl,
	}, nil
}

func (v *vectorStore) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	return v.AddDocuments(ctx, docs, options...)
}

func (v *vectorStore) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	return v.SimilaritySearch(ctx, query, numDocuments, options...)
}
