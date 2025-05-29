package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/handler"
	"github.com/zgsm-ai/codebase-indexer/internal/model"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/task"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/conf.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	logx.MustSetup(c.Log)

	if err := model.AutoMigrate(c.Database); err != nil {
		panic(err)
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	serverCtx, cancelFunc := context.WithCancel(context.Background())

	svcCtx, err := svc.NewServiceContext(serverCtx, c)

	if err != nil {
		panic(err)
	}

	defer cancelFunc()
	// start index job
	indexJobScheduler, err := task.NewIndexJobScheduler(svcCtx, serverCtx)
	if err != nil {
		panic(err)
	}

	go indexJobScheduler.Schedule()
	defer indexJobScheduler.Close()

	cleaner := task.NewCleaner(svcCtx, serverCtx)
	cleaner.Run()

	if err != nil {
		panic(err)
	}

	handler.RegisterHandlers(server, svcCtx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
