package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/handler"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/migrations"
	"net/http"
)

var configFile = flag.String("f", "etc/conf.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	logx.MustSetup(c.Log)

	if err := migrations.AutoMigrate(c.Database); err != nil {
		panic(err)
	}

	server := rest.MustNewServer(c.RestConf, rest.WithFileServer("/static/", http.Dir("api/static/")))
	defer server.Stop()

	serverCtx, cancelFunc := context.WithCancel(context.Background())
	svcCtx, err := svc.NewServiceContext(serverCtx, c)
	if err != nil {
		panic(err)
	}
	defer cancelFunc()
	// start index job
	jobScheduler, err := job.NewScheduler(serverCtx, svcCtx)
	if err != nil {
		panic(err)
	}
	jobScheduler.Schedule()
	defer jobScheduler.Close()

	handler.RegisterHandlers(server, svcCtx)

	fmt.Printf("Started server at %s:%d\n", c.Host, c.Port)
	server.Start()
}
