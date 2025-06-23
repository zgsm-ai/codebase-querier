package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path/filepath"
	"testing"
)

func TestCodegraphSummary(t *testing.T) {
	assert.NotPanics(t, func() {
		ctx := context.Background()
		logx.DisableStat()
		codebasePath := "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160"
		graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
		summary, err := graphStore.GetIndexSummary(ctx, 1, codebasePath)
		assert.NoError(t, err)
		assert.True(t, summary.TotalFiles > 0)

	})

}

func TestQueryGraphByCodeSnippet(t *testing.T) {
	ctx := context.Background()
	logx.DisableStat()
	codebasePath := "G:\\codebase-store\\7ec27814b60376c6fba936bf1fcaf430f8a84c37eb8f093f91e5664fd26c3160"
	graphStore, err := codegraph.NewBadgerDBGraph(codegraph.WithPath(filepath.Join(codebasePath, types.CodebaseIndexDir)))
	assert.NoError(t, err)
	defer graphStore.Close()
	testCases := []struct {
		Name    string
		req     *types.DefinitionRequest
		wantErr error
	}{
		{
			Name: "go",
			req: &types.DefinitionRequest{
				ClientId:     "1",
				CodebasePath: codebasePath,
				FilePath:     "test.go", // TODO 下面的var config AdmissionConfig 没有解析成功； d := yaml.NewYAMLOrJSONDecoder 这种short_val 前面的variable没有解析成功； arguments没有去掉首尾括号；对象.字段未做解析
				CodeSnippet: `
func NewImagePolicyWebhook(configFile io.Reader) (*Plugin, error) {
	if configFile == nil {
		return nil, fmt.Errorf("no config specified")
	}

	// TODO: move this to a versioned configuration file format
	var config AdmissionConfig 
	d := yaml.NewYAMLOrJSONDecoder(configFile, 4096)
	err := d.Decode(&config)
	if err != nil {
		return nil, err
	}

	whConfig := config.ImagePolicyWebhook
	if err := normalizeWebhookConfig(&whConfig); err != nil {
		return nil, err
	}

	clientConfig, err := webhook.LoadKubeconfig(whConfig.KubeConfigFile, nil)
	if err != nil {
		return nil, err
	}
	retryBackoff := webhook.DefaultRetryBackoffWithInitialDelay(whConfig.RetryBackoff)
	gw, err := webhook.NewGenericWebhook(legacyscheme.Scheme, legacyscheme.Codecs, clientConfig, groupVersions, retryBackoff)
	if err != nil {
		return nil, err
	}
	return &Plugin{
		Handler:       admission.NewHandler(admission.Create, admission.Update),
		webhook:       gw,
		responseCache: cache.NewLRUExpireCache(1024),
		allowTTL:      whConfig.AllowTTL,
		denyTTL:       whConfig.DenyTTL,
		defaultAllow:  whConfig.DefaultAllow,
	}, nil
}
`,
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			definitions, err := graphStore.QueryDefinition(ctx, tt.req)
			assert.NoError(t, err)
			assert.NotEmpty(t, definitions)
		})
	}

}
