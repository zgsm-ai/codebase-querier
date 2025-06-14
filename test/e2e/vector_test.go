package e2e

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"
	"time"

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
)

const basePath = "/projects/codebase-indexer"

var c config.Config
var svcCtx *svc.ServiceContext

var syncFileModeMap map[string]string

var clientId = "test-client-123"
var clientPath = "/tmp/test/test-project"

const codebasePath = "\\codebase-store\\11a8180b9a4b034c153f6ce8c48316f2498843f52249a98afbe95b824f815917" // your local repo path
const codebaseID = 2

func setup(syncId int32) error {
	msg := &types.CodebaseSyncMessage{
		SyncID:       syncId,
		CodebaseID:   codebaseID,
		CodebasePath: codebasePath,
		SyncTime:     time.Now(),
	}
	ctx := context.Background()
	var err error
	conf.MustLoad(filepath.Join(basePath, "etc/conf.yaml"), &c, conf.UseEnv())
	c.IndexTask.GraphTask.ConfFile = filepath.Join(basePath, "etc/codegraph.yaml")
	svcCtx, err = svc.NewServiceContext(ctx, c)
	if err != nil {
		return err
	}
	// 本次同步的元数据列表
	syncFileModeMap, _, err = svcCtx.CodebaseStore.GetSyncFileListCollapse(ctx, msg.CodebasePath)
	if err != nil {
		return err
	}
	if len(syncFileModeMap) == 0 {
		return errors.New("metadata file list is empty, cannot continue")
	}

	processor, err := job.NewEmbeddingProcessor(ctx, svcCtx, msg, syncFileModeMap)
	if err != nil {
		return err
	}
	return processor.Process()

}

func TestEmbeddingCodebase(t *testing.T) {
	assert.NotPanics(t, func() {
		syncId := int32(time.Now().Unix())
		err := setup(syncId)
		if err != nil {
			panic(err)
		}
		// assert 查询向量数据库内容
		client, err := goweaviate.NewClient(goweaviate.Config{
			Host:       c.VectorStore.Weaviate.Endpoint,
			Scheme:     "http",
			AuthConfig: auth.ApiKey{Value: c.VectorStore.Weaviate.APIKey},
		})
		tenantName, err := generateTenantName(codebasePath)
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

		whereFilter := filters.Where().WithPath([]string{vector.MetadataSyncId}).WithOperator(filters.Equal).WithValueInt(int64(syncId))
		resp, err := client.GraphQL().Get().
			WithTenant(tenantName).
			WithWhere(whereFilter).
			WithClassName(c.VectorStore.Weaviate.ClassName).
			WithFields(fields...).
			Do(context.Background())
		assert.NoError(t, err)
		assert.True(t, len(resp.Errors) == 0)
		assert.NotNil(t, resp.Data)
		m := resp.Data["Get"].(map[string]interface{})
		assert.True(t, len(m) > 0)

	})
}

func TestDeleteEmbeddings(t *testing.T) {
	assert.NotPanics(t, func() {
		syncId := int32(time.Now().Unix())
		err := setup(syncId)
		if err != nil {
			panic(err)
		}
		// assert 查询向量数据库内容
		client, err := goweaviate.NewClient(goweaviate.Config{
			Host:       c.VectorStore.Weaviate.Endpoint,
			Scheme:     "http",
			AuthConfig: auth.ApiKey{Value: c.VectorStore.Weaviate.APIKey},
		})
		tenantName, err := generateTenantName(codebasePath)
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

		whereFilter := filters.Where().WithPath([]string{vector.MetadataSyncId}).WithOperator(filters.Equal).WithValueInt(int64(syncId))
		resp, err := client.GraphQL().Get().
			WithTenant(tenantName).
			WithWhere(whereFilter).
			WithClassName(c.VectorStore.Weaviate.ClassName).
			WithFields(fields...).
			Do(context.Background())
		assert.NoError(t, err)
		assert.True(t, len(resp.Errors) == 0)
		assert.NotNil(t, resp.Data)
		m := resp.Data["Get"].(map[string]interface{})
		assert.True(t, len(m) > 0)

		var filePaths []string
		for k := range syncFileModeMap {
			filePaths = append(filePaths, k)
		}

		var deleteChunks []*types.CodeChunk
		// 删除
		for _, pa := range filePaths {
			deleteChunks = append(deleteChunks, &types.CodeChunk{
				CodebaseId:   codebaseID,
				CodebasePath: codebasePath,
				FilePath:     pa,
			})
		}

		err = svcCtx.VectorStore.DeleteCodeChunks(context.Background(), deleteChunks, vector.Options{})
		assert.NoError(t, err)
		// 再次查询
		resp, err = client.GraphQL().Get().
			WithTenant(tenantName).
			WithWhere(whereFilter).
			WithClassName(c.VectorStore.Weaviate.ClassName).
			WithFields(fields...).
			Do(context.Background())
		assert.NoError(t, err)
		assert.True(t, len(resp.Errors) == 0)
		assert.NotNil(t, resp.Data)
		m = resp.Data["Get"].(map[string]interface{})
		assert.True(t, len(m[c.VectorStore.Weaviate.ClassName].([]interface{})) == 0)
	})
}

// generateTenantName 使用 MD5 哈希生成合规租户名（32字符，纯十六进制）
func generateTenantName(codebasePath string) (string, error) {
	if codebasePath == types.EmptyString {
		return "", errors.New("invalid codebase path")
	}
	hash := md5.Sum([]byte(codebasePath))   // 计算 MD5 哈希
	return hex.EncodeToString(hash[:]), nil // 转为32位十六进制字符串
}
