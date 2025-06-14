package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestRelationQuery(t *testing.T) {
	// init data
	syncId := int32(time.Now().Unix())
	err := setup(syncId)
	if err != nil {
		panic(err)
	}

	// Prepare test data
	req := types.RelationQueryOptions{
		ClientId:       clientId,
		CodebasePath:   clientPath,
		FilePath:       "internal/logic/relation.go",
		StartLine:      1,
		StartColumn:    1,
		EndLine:        10,
		EndColumn:      50,
		SymbolName:     "RelationLogic",
		IncludeContent: 1,
		MaxLayer:       2,
	}

	// Send request to local service
	reqUrl := fmt.Sprintf("%s/codebase-indexer/api/v1/codegraph/relation?clientId=%s&codebasePath=%s&filePath=%s&startLine=%d&startColumn=%d&endLine=%d&endColumn=%d&symbolName=%s&includeContent=%d&maxLayer=%d",
		baseURL,
		req.ClientId,
		url.QueryEscape(req.CodebasePath),
		req.FilePath,
		req.StartLine,
		req.StartColumn,
		req.EndLine,
		req.EndColumn,
		url.QueryEscape(req.SymbolName),
		req.IncludeContent,
		req.MaxLayer,
	)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(reqUrl)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	// Parse response
	var result response.Response[types.RelationResponseData]
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
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotEmpty(t, firstNode.SymbolName)
		assert.NotEmpty(t, firstNode.NodeType)
		assert.NotNil(t, firstNode.Position)

		// If includeContent is 1, verify content is present
		if req.IncludeContent == 1 {
			assert.NotEmpty(t, firstNode.Content)
		}
	}
}
