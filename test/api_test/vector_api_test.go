package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
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

	var result response.Response[types.SemanticSearchResponseData]
	err = doRequest(http.MethodGet, "/codebase-indexer/api/v1/search/semantic", map[string]string{
		"clientId":     clientId,
		"codebasePath": clientPath,
		"query":        "codebase目录树",
		"topK":         "5",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstResult := result.Data.List[0]
		assert.NotEmpty(t, firstResult.Content)
		assert.NotEmpty(t, firstResult.FilePath)
		assert.Greater(t, firstResult.Score, float32(0))
	}
}
