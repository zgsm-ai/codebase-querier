package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/handler"
	"github.com/zgsm-ai/codebase-indexer/internal/index"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/conf.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	logx.MustSetup(c.Log)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	serverCtx, cancelFunc := context.WithCancel(context.Background())

	svcCtx, err := svc.NewServiceContext(serverCtx, c)

	if err != nil {
		panic(err)
	}

	defer cancelFunc()
	// start index job
	indexJobScheduler, err := index.NewIndexJobScheduler(svcCtx, serverCtx)
	if err != nil {
		panic(err)
	}

	go indexJobScheduler.Schedule()

	defer indexJobScheduler.Close()
	if err != nil {
		panic(err)
	}

	handler.RegisterHandlers(server, svcCtx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
