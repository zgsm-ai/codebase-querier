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
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"math"
)

const (
	schemeHttp  = "http"
	schemeHttps = "https"
	Verbose     = "verbose"
	Normal      = "normal"
)

type weaviateWrapper struct {
	ctx context.Context
	weaviate.Store
	reranker  Reranker
	client    *goweaviate.Client // langchaingo not supported delete
	className string
	cfg       config.VectorStoreConf
	logger    logx.Logger
}

func New(ctx context.Context, cfg config.VectorStoreConf, embedder embeddings.Embedder, reranker Reranker) (Store, error) {
	store, err := weaviate.New(
		weaviate.WithHost(cfg.Weaviate.Endpoint),
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
		Host:       cfg.Weaviate.Endpoint,
		Scheme:     schemeHttp,
		AuthConfig: auth.ApiKey{Value: cfg.Weaviate.APIKey},
	})

	return &weaviateWrapper{
		Store:     store,
		client:    client,
		ctx:       ctx,
		className: cfg.Weaviate.ClassName,
		reranker:  reranker,
		cfg:       cfg,
		logger:    logx.WithContext(ctx),
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
			PageContent: string(chunk.Content),
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

func (r *weaviateWrapper) Query(ctx context.Context, query string, topK int, options ...vectorstores.Option) ([]types.SemanticFileItem, error) {
	documents, err := r.Store.SimilaritySearch(ctx, query, r.cfg.Weaviate.MaxDocuments, options...)
	if err != nil {
		return nil, err
	}
	// TODO 调用reranker模型进行重排
	rerankedDocs, err := r.reranker.Rerank(ctx, query, documents)
	if err != nil {
		r.logger.Errorf("failed rerank docs: %v", err)
	}
	if len(rerankedDocs) == 0 {
		rerankedDocs = documents
	}
	// topK
	rerankedDocs = rerankedDocs[:int(math.Min(float64(topK), float64(len(rerankedDocs))))]
	res := make([]types.SemanticFileItem, len(rerankedDocs))
	for i, doc := range rerankedDocs {
		res[i] = types.SemanticFileItem{
			Content:  doc.PageContent,
			FilePath: doc.Metadata[types.MetadataFilePath].(string),
			Score:    float64(doc.Score),
		}
	}
	return res, nil
}
