package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestStructureQuery(t *testing.T) {
	var result response.Response[types.StructureResponseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/files/structure", map[string]string{
		"clientId":     clientId,
		"codebasePath": codebasePath,
		"filePath":     "pkg/auth/authorizer/abac/abac.go",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstItem := result.Data.List[0]
		assert.NotEmpty(t, firstItem.Name)
		assert.NotEmpty(t, firstItem.ItemType)
		assert.NotEmpty(t, firstItem.Content)
		assert.NotNil(t, firstItem.Position)
	}
}
