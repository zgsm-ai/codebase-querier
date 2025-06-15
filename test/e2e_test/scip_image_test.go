package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/test/api_test"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func getSvcCtx(ctx context.Context) *svc.ServiceContext {
	var c config.Config
	projectPath := "/root/projects/codebase-indexer"
	configPath := filepath.Join(projectPath, "etc/conf.yaml")
	conf.MustLoad(configPath, &c, conf.UseEnv())
	c.IndexTask.GraphTask.ConfFile = filepath.Join(projectPath, "test/e2e_test/codegraph.yaml")
	return api_test.InitSvcCtx(ctx, &c)
}
func TestScipBaseImage_WithOpenSourceProjects(t *testing.T) {
	// 运行 ../fetch_test_projects.sh 拉取开源项目用于测试
	// 运行docker, 设置环境变量 IMAGE=zgsm/scip-base:latest
	if os.Getenv("IMAGE") == "=" {
		panic("please set env IMAGE=")
	}
	logx.DisableStat()

	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	svcCtx := getSvcCtx(timeout)

	codegraphConfig := config.MustLoadCodegraphConfig("./config/")
	generator := scip.NewIndexGenerator(codegraphConfig, svcCtx.CodebaseStore)
	defer cancelFunc()
	basePath := "/tmp/projects/"
	testcases := []struct {
		Language string
		Project  string
		wantErr  error
	}{
		{
			Language: "go",
			Project:  "kubernetes",
			wantErr:  nil,
		},
	}
	for _, tc := range testcases {
		codebasePath := filepath.Join(basePath, tc.Language, tc.Project)
		t.Logf("start to testing %s codebase %s", tc.Language, codebasePath)
		err := generator.Generate(timeout, codebasePath)
		assert.ErrorIs(t, err, tc.wantErr)
		assert.FileExists(t, filepath.Join(codebasePath, ".shenma", "index.scip"), "file index.scip not found")
		t.Logf("testing %s codebase %s done", tc.Language, codebasePath)
		t.Log("---------------------------------------------------------------")
	}
}
