package api_test

import (
	"context"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"path/filepath"
)

// Note: to run these tests, start server manually at first.
const (
	baseURL      = "http://localhost:8888" // 替换为实际的服务地址和端口
	basePath     = "/projects/codebase-indexer"
	zipOutputDir = "/tmp/upload"
)

var clientId = "test-client-123"
var clientPath = "/tmp/test/test-project"

const codebasePath = "\\codebase-store\\11a8180b9a4b034c153f6ce8c48316f2498843f52249a98afbe95b824f815917" // your local repo path
const codebaseID = 2

func InitSvcCtx() *svc.ServiceContext {
	ctx := context.Background()
	var c config.Config
	conf.MustLoad(filepath.Join(basePath, "etc/conf.yaml"), &c, conf.UseEnv())
	c.IndexTask.GraphTask.ConfFile = filepath.Join(basePath, "etc/codegraph.yaml")
	svcCtx, err := svc.NewServiceContext(ctx, c)
	if err != nil {
		panic(err)
	}
	return svcCtx
}
