package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestRelationQuery(t *testing.T) {
	var result response.Response[types.RelationResponseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/search/relation", map[string]string{
		"clientId":       clientId,
		"codebasePath":   codebasePath,
		"filePath":       "pkg/auth/authorizer/abac/abac.go",
		"startLine":      "59",
		"startColumn":    "1",
		"endLine":        "59",
		"endColumn":      "50",
		"symbolName":     "NewFromFile",
		"includeContent": "1",
		"maxLayer":       "2",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotEmpty(t, firstNode.SymbolName)
		assert.NotEmpty(t, firstNode.NodeType)
		assert.NotNil(t, firstNode.Position)

		// If includeContent is 1, verify content is present
		assert.NotEmpty(t, firstNode.Content)
	}
}
