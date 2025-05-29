package vector

import (
	"context"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/weaviate"
	goweaviate "github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	http    = "http"
	https   = "https"
	Verbose = "verbose"
	Normal  = "normal"
)

type weaviateWrapper struct {
	ctx context.Context
	weaviate.Store
	client    *goweaviate.Client // langchaingo not supported delete
	className string
}

func New(ctx context.Context, cfg config.VectorStoreConf, embedder embeddings.Embedder) (Store, error) {
	store, err := weaviate.New(
		weaviate.WithHost(cfg.Weaviate.Host),
		weaviate.WithAPIKey(cfg.Weaviate.APIKey),
		weaviate.WithIndexName(cfg.Weaviate.IndexName),
		weaviate.WithNameSpace(cfg.Weaviate.Namespace),
		weaviate.WithEmbedder(embedder),
	)
	if err != nil {
		return nil, err
	}
	// langchaingo not supported delete
	client, err := goweaviate.NewClient(goweaviate.Config{
		Host:       cfg.Weaviate.Host,
		Scheme:     http,
		AuthConfig: auth.ApiKey{Value: cfg.Weaviate.APIKey},
	})

	return &weaviateWrapper{
		Store:     store,
		client:    client,
		ctx:       ctx,
		className: cfg.Weaviate.ClassName,
	}, nil
}

func (v *weaviateWrapper) DeleteCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options ...vectorstores.Option) (any, error) {
	if len(chunks) == 0 {
		return nil, nil // Nothing to delete
	}

	// Build a list of filters, one for each codebaseId and filePath pair
	chunkFilters := make([]*filters.WhereBuilder, len(chunks))
	for i, chunk := range chunks {
		chunkFilters[i] = filters.Where().
			WithOperator(filters.And).
			WithOperands([]*filters.WhereBuilder{
				filters.Where().
					WithPath([]string{types.MetadataCodebaseId}).
					WithOperator(filters.Equal).
					WithValueInt(chunk.CodebaseId),
				filters.Where().
					WithPath([]string{types.MetadataFilePath}).
					WithOperator(filters.Equal).
					WithValueText(chunk.FilePath),
			})
	}

	// Combine all chunk filters with OR to support batch deletion of files
	combinedFilter := filters.Where().
		WithOperator(filters.Or).
		WithOperands(chunkFilters)

	do, err := v.client.Batch().ObjectsBatchDeleter().WithWhere(
		combinedFilter,
	).WithOutput(Verbose).WithClassName(v.className).Do(v.ctx)
	if err != nil {
		return nil, err
	}
	return do, nil
}

func (v *weaviateWrapper) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	return v.Store.AddDocuments(ctx, docs, options...)
}

func (v *weaviateWrapper) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	return v.Store.SimilaritySearch(ctx, query, numDocuments, options...)
}

func (v *weaviateWrapper) Close() {
}

func (v *weaviateWrapper) UpsertCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options ...vectorstores.Option) error {
	// 转换为文档
	documents := make([]schema.Document, len(chunks))
	for i, chunk := range chunks {
		documents[i] = schema.Document{
			PageContent: chunk.Content,
			Metadata: map[string]any{
				types.MetadataFilePath:     chunk.FilePath,
				types.MetadataLanguage:     chunk.Language,
				types.MetadataCodebaseId:   chunk.CodebaseId,
				types.MetadataCodebasePath: chunk.CodebasePath,
				types.MetadataCodebaseName: chunk.CodebaseName,
			},
		}
	}
	_, err := v.AddDocuments(ctx, documents, options...)
	return err
}
