package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	api "github.com/zgsm-ai/codebase-indexer/test/api_test"
	"testing"
)

func TestLanguageDetect(t *testing.T) {
	testcases := []struct {
		Name          string
		CodebasePath  string
		wantErr       error
		wantIndexTool string
		wantBuildTool string
	}{
		{
			Name:          "typescript",
			CodebasePath:  "G:\\tmp\\projects\\typescript\\vue-next",
			wantErr:       nil,
			wantIndexTool: "scip-typescript",
			wantBuildTool: "",
		},
		{
			Name:          "typescript",
			CodebasePath:  "G:\\tmp\\projects\\typescript\\svelte",
			wantErr:       nil,
			wantIndexTool: "scip-typescript",
			wantBuildTool: "",
		},
		{
			Name:          "typescript",
			CodebasePath:  "G:\\tmp\\projects\\typescript\\TypeScript",
			wantErr:       nil,
			wantIndexTool: "scip-typescript",
			wantBuildTool: "",
		},
		{
			Name:          "c",
			CodebasePath:  "G:\\tmp\\projects\\c\\redis",
			wantErr:       nil,
			wantIndexTool: "scip-clang",
			wantBuildTool: "make",
		},
		{
			Name:          "go",
			CodebasePath:  "G:\\tmp\\projects\\go\\docker-ce",
			wantErr:       nil,
			wantIndexTool: "scip-go",
			wantBuildTool: "",
		},
		{
			Name:          "go",
			CodebasePath:  "G:\\tmp\\projects\\go\\go",
			wantErr:       nil,
			wantIndexTool: "scip-go",
			wantBuildTool: "",
		},
		{
			Name:          "go",
			CodebasePath:  "G:\\tmp\\projects\\go\\kubernetes",
			wantErr:       nil,
			wantIndexTool: "scip-go",
			wantBuildTool: "",
		},
		{
			Name:          "rust",
			CodebasePath:  "G:\\tmp\\projects\\rust\\rust",
			wantErr:       nil,
			wantIndexTool: "rust-analyzer",
			wantBuildTool: "",
		},
		{
			Name:          "rust",
			CodebasePath:  "G:\\tmp\\projects\\go\\starship",
			wantErr:       nil,
			wantIndexTool: "rust-analyzer",
			wantBuildTool: "",
		},
		{
			Name:          "go",
			CodebasePath:  "G:\\tmp\\projects\\go\\docker-ce",
			wantErr:       nil,
			wantIndexTool: "scip-go",
			wantBuildTool: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			logx.DisableStat()
			ctx := context.Background()
			serviceContext := api.InitSvcCtx(ctx, nil)
			conf := config.MustLoadCodegraphConfig("G:\\projects\\codebase-indexer\\test\\e2e_test\\conf\\codegraph.yaml")
			generator := scip.NewIndexGenerator(serviceContext.CmdLogger, conf, serviceContext.CodebaseStore)
			indexTool, buildTool, err := generator.DetectLanguageAndTool(ctx, tc.CodebasePath)
			var buildToolName string
			if buildTool == nil {
				buildToolName = ""
			} else {
				buildToolName = buildTool.Name
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantIndexTool, indexTool.Name)
			assert.Equal(t, tc.wantBuildTool, buildToolName)
		})
	}

}
