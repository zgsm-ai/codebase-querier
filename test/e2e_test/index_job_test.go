package e2e_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/job"
	"github.com/zgsm-ai/codebase-indexer/test/api_test"
	"os"
	"testing"
	"time"
)

func TestIndexJobRun(t *testing.T) {
	// 环境变量 INDEX_NODE 设置为1
	if os.Getenv("INDEX_NODE") != "1" {
		panic("please set env INDEX_NODE=1")
	}
	logx.DisableStat()
	serviceContext := api_test.InitSvcCtx()
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancelFunc()
	indexJob, err := job.NewIndexJob(ctx, serviceContext)
	assert.NoError(t, err)
	// 先执行 upload 上传文件， INDEX_NODE 环境变量设置为0，确保主程序不会执行
	assert.NoError(t, err)
	go indexJob.Start()
	for {
		select {
		case <-ctx.Done():
			return
		}
	}

}
