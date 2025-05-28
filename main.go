package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/handler"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/codebaseindexer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	logx.MustSetup(c.Log)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	serverCtx, cancelFunc := context.WithCancel(context.Background())

	svcCtx, err := svc.NewServiceContext(serverCtx, c)

	defer cancelFunc()
	// start index job
	indexJobScheduler := job.NewIndexJobScheduler(svcCtx, serverCtx)
	go indexJobScheduler.Start()

	if err != nil {
		panic(err)
	}

	handler.RegisterHandlers(server, svcCtx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
