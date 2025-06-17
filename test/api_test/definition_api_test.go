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

func TestQueryDefinition(t *testing.T) {
	// Prepare test data
	req := types.RelationRequest{
		ClientId:     "test-client-123",
		CodebasePath: "G:\\tmp\\projects\\go\\kubernetes",
		FilePath:     "pkg/auth/authorizer/abac/abac.go",
		StartLine:    59,
		EndLine:      119,
	}

	// Send request to local service
	reqUrl := fmt.Sprintf("%s/codebase-indexer/api/v1/search/definition?clientId=%s&codebasePath=%s&filePath=%s&startLine=%d&endLine=%d",
		baseURL,
		req.ClientId,
		url.QueryEscape(req.CodebasePath),
		req.FilePath,
		req.StartLine,
		req.EndLine,
	)

	client := &http.Client{
		Timeout: time.Second * 30,
	}
	resp, err := client.Get(reqUrl)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	// Parse response
	var result response.Response[types.DefinitionResponseData]
	err = json.Unmarshal(body, &result)
	assert.NoError(t, err)

	//t.Logf("resp:%+v", string(body))

	// Verify response structure
	assert.Equal(t, 0, result.Code) // 0 indicates success
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		// Verify the structure of the first result
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotNil(t, firstNode.Position)
		assert.NotNil(t, firstNode.Content)
		assert.NotEmpty(t, firstNode.Content)

	}
}
