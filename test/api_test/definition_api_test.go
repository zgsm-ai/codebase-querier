package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestQueryDefinition(t *testing.T) {
	var result response.Response[types.DefinitionResponseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/search/definition", map[string]string{
		"clientId":     clientId,
		"codebasePath": codebasePath,
		"filePath":     "pkg/auth/authorizer/abac/abac.go",
		"startLine":    "59",
		"endLine":      "119",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotNil(t, firstNode.Position)
		assert.NotNil(t, firstNode.Content)
		assert.NotEmpty(t, firstNode.Content)
	}
}
