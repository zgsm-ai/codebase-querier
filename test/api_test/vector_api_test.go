package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func setup(syncId int32) error {

	ctx := context.Background()
	svcCtx := InitSvcCtx(ctx, nil)
	msg := &types.CodebaseSyncMessage{
		SyncID:       syncId,
		CodebaseID:   codebaseID,
		CodebasePath: codebasePath,
		SyncTime:     time.Now(),
	}
	// 本次同步的元数据列表
	syncFiles, err := svcCtx.CodebaseStore.GetSyncFileListCollapse(ctx, msg.CodebasePath)
	if err != nil {
		return err
	}
	if len(syncFiles.FileModelMap) == 0 {
		return errors.New("metadata file list is empty, cannot continue")
	}

	params := &job.IndexTaskParams{
		CodebaseID:           msg.CodebaseID,
		CodebasePath:         msg.CodebasePath,
		CodebaseName:         msg.CodebaseName,
		SyncMetaFiles:        &types.CollapseSyncMetaFile{},
		EnableCodeGraphBuild: true,
		EnableEmbeddingBuild: true,
	}

	processor, err := job.NewEmbeddingProcessor(svcCtx, params, syncFiles.FileModelMap)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, tracer.Key, tracer.TaskTraceId(codebaseID))
	return processor.Process(ctx)

}

func TestSemanticQuery(t *testing.T) {

	// init data
	syncId := int32(time.Now().Unix())
	err := setup(syncId)
	if err != nil {
		panic(err)
	}
	// Prepare test data
	req := types.SemanticSearchRequest{
		ClientId:     clientId,
		CodebasePath: clientPath,
		Query:        "codebase目录树",
		TopK:         5,
	}

	// Send request to local service
	reqUrl := fmt.Sprintf("%s/codebase-indexer/api/v1/search/semantic?clientId=%s&codebasePath=%s&query=%s&topK=%d",
		baseURL, req.ClientId, url.QueryEscape(req.CodebasePath), url.QueryEscape(req.Query), req.TopK)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(reqUrl)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	// Parse response
	var result response.Response[types.SemanticSearchResponseData]
	err = json.Unmarshal(body, &result)
	assert.NoError(t, err)

	t.Logf("resp:%+v", string(body))

	// Verify response structure
	assert.Equal(t, 0, result.Code) // 0 indicates success
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		// Verify the structure of the first result
		firstResult := result.Data.List[0]
		assert.NotEmpty(t, firstResult.Content)
		assert.NotEmpty(t, firstResult.FilePath)
		assert.Greater(t, firstResult.Score, float32(0))
	}
}
