package e2e

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	api "github.com/zgsm-ai/codebase-indexer/test/api_test"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func getSvcCtx(ctx context.Context) *svc.ServiceContext {
	var c config.Config
	projectPath := "/root/projects/codebase-indexer"
	configPath := filepath.Join(projectPath, "etc/conf.yaml")
	conf.MustLoad(configPath, &c, conf.UseEnv())
	c.IndexTask.GraphTask.ConfFile = filepath.Join(projectPath, "test/e2e_test/conf/codegraph.yaml")
	return api.InitSvcCtx(ctx, &c)
}
func TestScipBaseImage_WithOpenSourceProjects(t *testing.T) {
	// 运行 ../fetch_test_projects.sh 拉取开源项目用于测试
	logx.DisableStat()

	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	svcCtx := getSvcCtx(timeout)

	codegraphConfig := config.MustLoadCodegraphConfig(svcCtx.Config.IndexTask.GraphTask.ConfFile)
	generator := scip.NewIndexGenerator(svcCtx.CmdLogger, codegraphConfig, svcCtx.CodebaseStore)
	defer cancelFunc()
	basePath := "/test/tmp/projects/"
	testcases := []struct {
		Name     string
		Language string
		Project  string
		wantErr  error
	}{
		//{
		//	Name:     "typescript",
		//	Language: "typescript",
		//	CodebasePath:  "vue-next",
		//	wantErr:  nil,
		//},
		{
			Name:     "javascript",
			Language: "javascript",
			Project:  "vue",
			wantErr:  nil,
		},
		//{
		//	Name:     "go",
		//	Language: "go",
		//	CodebasePath:  "kubernetes",
		//	wantErr:  nil,
		//},
		{
			Name:     "java maven",
			Language: "java",
			Project:  "hadoop",
			wantErr:  nil,
		},
		{
			Name:     "java gradle",
			Language: "java",
			Project:  "spring-boot",
			wantErr:  nil,
		},
		{
			Name:     "python",
			Language: "python",
			Project:  "django",
			wantErr:  nil,
		},
		{
			Name:     "ruby",
			Language: "ruby",
			Project:  "vagrant",
			wantErr:  nil,
		},
		{
			Name:     "csharp",
			Language: "csharp",
			Project:  "mono",
			wantErr:  nil,
		},
		{
			Name:     "c cmake",
			Language: "c",
			Project:  "netdata",
			wantErr:  nil,
		},
		{
			Name:     "cpp",
			Language: "cpp",
			Project:  "opencv",
			wantErr:  nil,
		},
		{
			Name:     "rust",
			Language: "rust",
			Project:  "rust",
			wantErr:  nil,
		},
	}
	for _, tc := range testcases {

		t.Run(tc.Name, func(t *testing.T) {
			codebasePath := filepath.Join(basePath, tc.Language, tc.Project)
			t.Logf("start to testing %s codebase %s", tc.Name, codebasePath)
			err := generator.Generate(timeout, codebasePath)
			assert.ErrorIs(t, err, tc.wantErr)
			indexFilePath := filepath.Join(codebasePath, ".shenma", "index.scip")
			assert.FileExists(t, indexFilePath, "file index.scip not found")
			fileInfo, err := os.Stat(indexFilePath)
			assert.NoError(t, err, "file index.scip not found")
			assert.True(t, fileInfo.Size() > 100*1024, "file index.scip is empty") // 一般大项目index至少大于 100KB
			t.Logf("index file %s size: %d, modTime: %s", indexFilePath, fileInfo.Size(), fileInfo.ModTime().Format("2006-01-02 15:04:05"))
			t.Logf("testing %s codebase %s done", tc.Name, codebasePath)
			t.Log("---------------------------------------------------------------")
		})

	}
}

func TestCodegraphConfig(t *testing.T) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), time.Minute*10)
	svcCtx := getSvcCtx(timeout)

	codegraphConfig := config.MustLoadCodegraphConfig(svcCtx.Config.IndexTask.GraphTask.ConfFile)
	generator := scip.NewIndexGenerator(svcCtx.CmdLogger, codegraphConfig, svcCtx.CodebaseStore)
	defer cancelFunc()
	basePath := "/test/tmp/projects/"
	testcases := []struct {
		Name     string
		Language string
		Project  string
		wantErr  error
	}{
		{
			Name:     "typescript",
			Language: "typescript",
			Project:  "vue-next",
			wantErr:  nil,
		},
		{
			Name:     "javascript",
			Language: "javascript",
			Project:  "vue",
			wantErr:  nil,
		},
		{
			Name:     "go",
			Language: "go",
			Project:  "kubernetes",
			wantErr:  nil,
		},
		{
			Name:     "java maven",
			Language: "java",
			Project:  "hadoop",
			wantErr:  nil,
		},
		{
			Name:     "java gradle",
			Language: "java",
			Project:  "spring-boot",
			wantErr:  nil,
		},
		{
			Name:     "python",
			Language: "python",
			Project:  "django",
			wantErr:  nil,
		},
		{
			Name:     "ruby",
			Language: "ruby",
			Project:  "vagrant",
			wantErr:  nil,
		},
		{
			Name:     "csharp",
			Language: "csharp",
			Project:  "mono",
			wantErr:  nil,
		},
		{
			Name:     "c cmake",
			Language: "c",
			Project:  "netdata",
			wantErr:  nil,
		},
		{
			Name:     "cpp",
			Language: "cpp",
			Project:  "opencv",
			wantErr:  nil,
		},
		{
			Name:     "rust",
			Language: "rust",
			Project:  "rust",
			wantErr:  nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			codebasePath := filepath.Join(basePath, tc.Language, tc.Project)
			t.Logf("start to testing %s codebase %s", tc.Name, codebasePath)
			executor, err := generator.InitCommandExecutor(timeout, svcCtx.CmdLogger, codebasePath)
			assert.NoError(t, err)
			buildCmds, indexCmds := executor.BuildCmds, executor.IndexCmds
			var build []string
			for _, c := range buildCmds {
				build = append(build, extractCmd(c))
			}
			var index []string
			for _, c := range indexCmds {
				index = append(index, extractCmd(c))
			}
			t.Logf("language %s, project %s, build: %s", tc.Name, tc.Project, build)
			t.Logf("language %s, project %s, index: %s", tc.Name, tc.Project, index)
			t.Log("---------------------------------------------------------------")
		})
	}

}

func extractCmd(c *exec.Cmd) string {
	var s strings.Builder
	if c.Dir != "" {
		s.WriteString("cd ")
		s.WriteString(c.Dir)
		s.WriteString(" && ")
	}
	for _, e := range c.Env {
		s.WriteString("export" + e + " && ")
	}

	s.WriteString(c.Path + " ")
	for _, a := range c.Args {
		s.WriteString(a + " ")
	}
	return s.String()
}
