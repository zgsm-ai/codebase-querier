package vector

import (
	"context"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	goweaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"math"
	"strings"
)

const (
	schemeHttp  = "http"
	schemeHttps = "https"
	Verbose     = "verbose"
	Normal      = "normal"
	responseKey = "text"
)

type weaviateWrapper struct {
	ctx       context.Context
	reranker  Reranker
	embedder  Embedder
	client    *goweaviate.Client // langchaingo not supported delete
	className string
	cfg       config.VectorStoreConf
	logger    logx.Logger
}

func New(ctx context.Context, cfg config.VectorStoreConf, embedder Embedder, reranker Reranker) (Store, error) {
	// langchaingo not supported delete
	client, err := goweaviate.NewClient(goweaviate.Config{
		Host:       cfg.Weaviate.Endpoint,
		Scheme:     schemeHttp,
		AuthConfig: auth.ApiKey{Value: cfg.Weaviate.APIKey},
	})

	store := &weaviateWrapper{
		client:    client,
		ctx:       ctx,
		className: cfg.Weaviate.ClassName,
		embedder:  embedder,
		reranker:  reranker,
		cfg:       cfg,
		logger:    logx.WithContext(ctx),
	}
	if err != nil {
		return nil, err
	}

	// init class
	err = store.createClassWithAutoTenantEnabled(ctx, client)
	return store, err
}

func (r *weaviateWrapper) DeleteCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options Options) (any, error) {
	if len(chunks) == 0 {
		return nil, nil // Nothing to delete
	}
	tenant, err := r.generateTenantName(chunks[0].CodebaseId)
	if err != nil {
		return nil, err
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

	do, err := r.client.Batch().ObjectsBatchDeleter().
		WithTenant(tenant).WithWhere(
		combinedFilter,
	).WithOutput(Verbose).WithClassName(r.className).Do(r.ctx)
	if err != nil {
		return nil, err
	}
	return do, nil
}

func (r *weaviateWrapper) SimilaritySearch(ctx context.Context, query string, numDocuments int, options Options) ([]*types.SemanticFileItem, error) {
	embedQuery, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	tenantName, err := r.generateTenantName(options.CodebaseId)
	if err != nil {
		return nil, err
	}
	res, err := r.client.GraphQL().Get().
		WithNearVector(r.client.GraphQL().
			NearVectorArgBuilder().
			WithVector(embedQuery)).
		WithClassName(r.className).
		WithTenant(tenantName).
		WithLimit(numDocuments).Do(ctx)
	if err != nil {
		return nil, err
	}

	return r.unmarshalResponse(res)
}

func (r *weaviateWrapper) unmarshalResponse(res *models.GraphQLResponse) ([]*types.SemanticFileItem, error) {
	if len(res.Errors) > 0 {
		err := make([]string, 0, len(res.Errors))
		for _, e := range res.Errors {
			err = append(err, e.Message)
		}
		return nil, fmt.Errorf("err response from vector store: %s", strings.Join(err, ", "))
	}

	data, ok := res.Data["Get"].(map[string]any)[r.className]
	if !ok || data == nil {
		return nil, ErrEmptyResponse
	}
	dataList, ok := data.([]any)

	docs := make([]*types.SemanticFileItem, 0, len(dataList))
	if !ok || len(dataList) == 0 {
		return docs, nil
	}
	for _, item := range dataList {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, ErrInvalidResponse
		}
		//TODO the real key
		content, ok := itemMap[responseKey].(string)
		if !ok {
			return nil, ErrInvalidResponse
		}
		var score float64
		if additional, ok := itemMap["_additional"].(map[string]any); ok {
			score, _ = additional["certainty"].(float64)
		}
		delete(itemMap, responseKey)
		docs = append(docs, &types.SemanticFileItem{
			Content:  content,
			Score:    float32(score),
			FilePath: itemMap[types.MetadataFilePath].(string),
		})
	}
	return docs, nil
}

func (r *weaviateWrapper) Close() {
}

func (r *weaviateWrapper) UpsertCodeChunks(ctx context.Context, docs []*types.CodeChunk, options Options) error {
	if len(docs) == 0 {
		return nil
	}
	tenantName, err := r.generateTenantName(docs[0].CodebaseId)
	if err != nil {
		return err
	}
	chunks, err := r.embedder.EmbedCodeChunks(ctx, docs)
	if err != nil {
		return err
	}

	objs := make([]*models.Object, 0, len(chunks))
	for _, c := range chunks {
		objs = append(objs, &models.Object{
			ID:     strfmt.UUID(uuid.New().String()),
			Class:  r.className,
			Tenant: tenantName,
			Vector: c.Embedding,
			Properties: map[string]any{
				types.MetadataFilePath:     c.FilePath,
				types.MetadataLanguage:     c.Language,
				types.MetadataCodebaseId:   c.CodebaseId,
				types.MetadataCodebasePath: c.CodebasePath,
				types.MetadataCodebaseName: c.CodebaseName,
			},
		})
	}
	resp, err := r.client.Batch().ObjectsBatcher().WithObjects(objs...).Do(ctx)
	if err != nil {
		return err
	}
	logx.Debugf("add documents returns %v", resp)
	return nil
}

func (r *weaviateWrapper) Query(ctx context.Context, query string, topK int, options Options) ([]*types.SemanticFileItem, error) {
	documents, err := r.SimilaritySearch(ctx, query, r.cfg.Weaviate.MaxDocuments, options)

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
	return rerankedDocs, nil
}

func (r *weaviateWrapper) createClassWithAutoTenantEnabled(ctx context.Context, client *goweaviate.Client) error {

	logx.Infof("start to create weaviate class %s", r.className)
	res, err := client.Schema().ClassExistenceChecker().WithClassName(r.className).Do(ctx)
	if err != nil {
		logx.Errorf("check weaviate class exists err:%v", err)
	}
	if err == nil && res {
		logx.Infof("weaviate class %s already exists, not create.", r.className)
		return nil
	}
	class := &models.Class{
		Class:      r.className,
		Properties: []*models.Property{},
		// auto create tenant
		MultiTenancyConfig: &models.MultiTenancyConfig{
			Enabled:            true,
			AutoTenantCreation: true,
		},
	}
	logx.Infof("try to create weaviate class %s", r.className)
	err = client.Schema().ClassCreator().WithClass(class).Do(ctx)
	// TODO skip already exists err
	if err != nil && strings.Contains(err.Error(), "already exists") {
		logx.Infof("weaviate class %s already exists, not create.", r.className)
		return nil
	}
	return err
}

func (r *weaviateWrapper) generateTenantName(codebaseId int64) (string, error) {
	if codebaseId == 0 {
		return "", ErrInvalidCodebaseId
	}
	return fmt.Sprintf("tenant-%d", codebaseId), nil
}
