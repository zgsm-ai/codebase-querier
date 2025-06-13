package test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/stretchr/testify/assert"
	goweaviate "github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
	"time"
)

const basePath = "/projects/codebase-indexer"

func TestEmbeddingCodebase(t *testing.T) {
	assert.NotPanics(t, func() {
		msg := &types.CodebaseSyncMessage{
			SyncID:       int32(time.Now().Unix()),
			CodebaseID:   2,
			CodebasePath: "\\codebase-store\\11a8180b9a4b034c153f6ce8c48316f2498843f52249a98afbe95b824f815917",
			SyncTime:     time.Now(),
		}
		ctx := context.Background()
		var c config.Config
		conf.MustLoad(filepath.Join(basePath, "etc/conf.yaml"), &c, conf.UseEnv())
		c.IndexTask.GraphTask.ConfFile = filepath.Join(basePath, "etc/codegraph.yaml")
		svcCtx, err := svc.NewServiceContext(ctx, c)
		assert.NoError(t, err)
		// 本次同步的元数据列表
		syncFileModeMap, _, err := svcCtx.CodebaseStore.GetSyncFileListCollapse(ctx, msg.CodebasePath)
		assert.NoError(t, err)
		assert.True(t, len(syncFileModeMap) > 0)

		processor, err := job.NewEmbeddingProcessor(ctx, svcCtx, msg, syncFileModeMap)
		assert.NoError(t, err)
		err = processor.Process()
		assert.NoError(t, err)
		// assert 查询向量数据库内容
		client, err := goweaviate.NewClient(goweaviate.Config{
			Host:       c.VectorStore.Weaviate.Endpoint,
			Scheme:     "http",
			AuthConfig: auth.ApiKey{Value: c.VectorStore.Weaviate.APIKey},
		})
		tenantName, err := generateTenantName(msg.CodebasePath)
		assert.NoError(t, err)
		fields := []graphql.Field{
			{Name: vector.MetadataCodebaseName},
			{Name: vector.MetadataCodebasePath},
			{Name: vector.MetadataCodebaseId},
			{Name: vector.MetadataSyncId},
			{Name: vector.MetadataFilePath},
			{Name: vector.MetadataRange},
			{Name: vector.MetadataLanguage},
			{Name: vector.MetadataTokenCount},
			{Name: vector.Content},
			{
				Name: "_additional",
				Fields: []graphql.Field{
					{Name: "id"},
					//{Name: "vector"},
					{Name: "certainty"},
					{Name: "distance"},
				},
			},
		}

		whereFilter := filters.Where().WithPath([]string{vector.MetadataSyncId}).WithOperator(filters.Equal).WithValueInt(int64(msg.SyncID))
		resp, err := client.GraphQL().Get().
			WithTenant(tenantName).
			WithWhere(whereFilter).
			WithClassName(c.VectorStore.Weaviate.ClassName).
			WithFields(fields...).
			Do(ctx)
		assert.NoError(t, err)
		assert.True(t, len(resp.Errors) == 0)
		assert.NotNil(t, resp.Data)
		m := resp.Data["Get"].(map[string]interface{})
		assert.True(t, len(m) > 0)

	})
}

func TestDeleteEmbeddings(t *testing.T) {

}

func TestSemanticQuery(t *testing.T) {
	// 先启动服务

}

// generateTenantName 使用 MD5 哈希生成合规租户名（32字符，纯十六进制）
func generateTenantName(codebasePath string) (string, error) {
	if codebasePath == types.EmptyString {
		return "", errors.New("invalid codebase path")
	}
	hash := md5.Sum([]byte(codebasePath))   // 计算 MD5 哈希
	return hex.EncodeToString(hash[:]), nil // 转为32位十六进制字符串
}
