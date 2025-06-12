package vector

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	goweaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
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
	client    *goweaviate.Client
	className string
	cfg       config.VectorStoreConf
	logger    logx.Logger
}

func New(ctx context.Context, cfg config.VectorStoreConf, embedder Embedder, reranker Reranker) (Store, error) {
	var authConf auth.Config
	if cfg.Weaviate.APIKey != types.EmptyString {
		authConf = auth.ApiKey{Value: cfg.Weaviate.APIKey}
	}
	client, err := goweaviate.NewClient(goweaviate.Config{
		Host:       cfg.Weaviate.Endpoint,
		Scheme:     schemeHttp,
		AuthConfig: authConf,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Weaviate client: %w", err)
	}

	store := &weaviateWrapper{
		client:    client,
		ctx:       ctx,
		className: cfg.Weaviate.ClassName,
		embedder:  embedder,
		reranker:  reranker,
		cfg:       cfg,
		logger:    logx.WithContext(ctx),
	}

	// init class
	err = store.createClassWithAutoTenantEnabled(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create class: %w", err)
	}

	return store, nil
}

func (r *weaviateWrapper) DeleteCodeChunks(ctx context.Context, chunks []*types.CodeChunk, options Options) (any, error) {
	if len(chunks) == 0 {
		return nil, nil // Nothing to delete
	}
	tenant, err := r.generateTenantName(chunks[0].CodebasePath)
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
					WithValueInt(int64(chunk.CodebaseId)),
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
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	tenantName, err := r.generateTenantName(options.CodebasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tenant name: %w", err)
	}

	// Define GraphQL fields using proper Field type
	fields := []graphql.Field{
		{Name: responseKey},
		{Name: types.MetadataFilePath},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "certainty"},
			{Name: "distance"},
			{Name: "id"},
		}},
	}

	// Build GraphQL query with proper tenant filter
	nearVector := r.client.GraphQL().NearVectorArgBuilder().
		WithVector(embedQuery)

	res, err := r.client.GraphQL().Get().
		WithClassName(r.className).
		WithFields(fields...).
		WithNearVector(nearVector).
		WithLimit(numDocuments).
		WithTenant(tenantName).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to execute similarity search: %w", err)
	}

	// Improved error handling for response validation
	if res == nil || res.Data == nil {
		return nil, fmt.Errorf("received empty response from Weaviate")
	}

	items, err := r.unmarshalResponse(res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return items, nil
}

func (r *weaviateWrapper) unmarshalResponse(res *models.GraphQLResponse) ([]*types.SemanticFileItem, error) {
	// Get the data for our class
	data, ok := res.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: 'Get' field not found or has wrong type")
	}

	results, ok := data[r.className].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format: class data not found or has wrong type")
	}

	items := make([]*types.SemanticFileItem, 0, len(results))
	for _, result := range results {
		obj, ok := result.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract additional properties
		additional, ok := obj["_additional"].(map[string]interface{})
		if !ok {
			continue
		}

		// Create SemanticFileItem with proper fields
		item := &types.SemanticFileItem{
			Content:  getStringValue(obj, responseKey),
			FilePath: getStringValue(obj, types.MetadataFilePath),
			Score:    float32(getFloatValue(additional, "certainty")), // Convert float64 to float32
		}

		items = append(items, item)
	}

	return items, nil
}

// Helper functions for safe type conversion
func getStringValue(obj map[string]interface{}, key string) string {
	if val, ok := obj[key].(string); ok {
		return val
	}
	return ""
}

func getFloatValue(obj map[string]interface{}, key string) float64 {
	if val, ok := obj[key].(float64); ok {
		return val
	}
	return 0
}

func (r *weaviateWrapper) Close() {
}

func (r *weaviateWrapper) UpsertCodeChunks(ctx context.Context, docs []*types.CodeChunk, options Options) error {
	if len(docs) == 0 {
		return nil
	}
	tenantName, err := r.generateTenantName(docs[0].CodebasePath)
	if err != nil {
		return err
	}
	chunks, err := r.embedder.EmbedCodeChunks(ctx, docs)
	if err != nil {
		return err
	}
	logx.Infof("embedded %d chunks for codebase %s successfully", len(docs), docs[0].CodebaseName)
	objs := make([]*models.Object, len(chunks), len(chunks))
	for i, c := range chunks {
		objs[i] = &models.Object{
			ID:     strfmt.UUID(uuid.New().String()),
			Class:  r.className,
			Tenant: tenantName,
			Vector: c.Embedding,
			Properties: map[string]any{
				types.MetadataFilePath:     c.FilePath,
				types.MetadataLanguage:     c.Language,
				types.MetadataCodebaseId:   c.CodebaseId,
				types.MetadataCodebasePath: c.CodebasePath,
				types.MetadataSyncId:       options.SyncId,
				types.MetadataCodebaseName: c.CodebaseName,
			},
		}
	}
	logx.Infof("start to save %d chunks for codebase %s successfully", len(docs), docs[0].CodebaseName)
	if _, err = r.client.Batch().ObjectsBatcher().WithObjects(objs...).Do(ctx); err != nil {
		return err
	}
	logx.Infof("save %d chunks for codebase %s successfully", len(docs), docs[0].CodebaseName)
	return nil
}

func (r *weaviateWrapper) Query(ctx context.Context, query string, topK int, options Options) ([]*types.SemanticFileItem, error) {
	documents, err := r.SimilaritySearch(ctx, query, r.cfg.Weaviate.MaxDocuments, options)

	if err != nil {
		return nil, err
	}
	//  调用reranker模型进行重排
	rerankedDocs, err := r.reranker.Rerank(ctx, query, documents)
	if err != nil {
		r.logger.Errorf("failed customReranker docs: %v", err)
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

// generateTenantName 使用 MD5 哈希生成合规租户名（32字符，纯十六进制）
func (r *weaviateWrapper) generateTenantName(codebasePath string) (string, error) {
	if codebasePath == types.EmptyString {
		return "", ErrInvalidCodebasePath
	}
	hash := md5.Sum([]byte(codebasePath))   // 计算 MD5 哈希
	return hex.EncodeToString(hash[:]), nil // 转为32位十六进制字符串
}
