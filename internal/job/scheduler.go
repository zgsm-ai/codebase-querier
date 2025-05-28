package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"time"
)

const indexNodeEnableVal = 1

type IndexJobScheduler interface {
	Start()
}

type indexJobScheduler struct {
	logx.Logger
	svcCtx     *svc.ServiceContext
	ctx        context.Context
	enableFlag bool
}

func NewIndexJobScheduler(svcCtx *svc.ServiceContext, serverCtx context.Context) IndexJobScheduler {
	return &indexJobScheduler{
		ctx:        serverCtx,
		Logger:     logx.WithContext(serverCtx),
		svcCtx:     svcCtx,
		enableFlag: svcCtx.Config.IndexJob.EnableFlag == indexNodeEnableVal,
	}
}
func (i *indexJobScheduler) Start() {
	if !i.enableFlag {
		i.Logger.Infof("IndexJobScheduler is disabled, IS_INDEX_NODE flag is %d", i.enableFlag)
		return
	}

	i.Logger.Info("IndexJobScheduler started with worker pool")
	defer i.Logger.Info("IndexJobScheduler stopped")

	// 创建固定大小的协程池
	workerPool := make(chan struct{}, i.svcCtx.Config.IndexJob.PoolSize)
	stopCh := make(chan struct{})

	// 订阅消息队列
	ch, err := i.svcCtx.MessageQueue.Subscribe(i.ctx, i.svcCtx.Config.IndexJob.Topic)
	if err != nil {
		i.Logger.Errorf("Failed to subscribe topic: %v", err)
		return
	}

	// 启动消息分发协程
	go func() {
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					i.Logger.Infof("Message channel closed, exiting dispatcher")
					close(stopCh)
					return
				}

				// 获取worker槽位（阻塞直到有可用槽位）
				workerPool <- struct{}{}

				// 启动工作协程处理消息
				go func(m mq.Message) {
					defer func() { <-workerPool }() // 释放worker槽位

					// 创建带超时的子上下文
					ctx, cancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexJob.Timeout)
					defer cancel()

					// 处理消息
					err := i.processIndexJobWithContext(ctx, m)
					if err != nil {
						if errors.Is(err, context.DeadlineExceeded) {
							i.Logger.Errorf("Job timeout after %v: %v", i.svcCtx.Config.IndexJob.Timeout, err)
						} else {
							i.Logger.Errorf("Failed to process job: %v", err)
						}
					}
				}(msg)

			case <-i.ctx.Done():
				i.Logger.Info("Context cancelled, exiting dispatcher")
				close(stopCh)
				return
			}
		}
	}()

	// 等待停止信号
	<-stopCh

	// 关闭消息队列连接
	err = i.svcCtx.MessageQueue.Close()
	if err != nil {
		i.Logger.Info("Context cancelled, exiting dispatcher")
	}

	// 等待所有工作协程完成（通过workerPool缓冲区实现）
	i.Logger.Info("Waiting for all workers to complete...")
	for len(workerPool) > 0 {
		time.Sleep(100 * time.Millisecond)
	}
}

// processIndexJobWithContext 带上下文的消息处理
func (i *indexJobScheduler) processIndexJobWithContext(ctx context.Context, msg mq.Message) error {
	var jobIndexTask IndexTask
	if err := json.Unmarshal(msg.Body, &jobIndexTask); err != nil {
		return fmt.Errorf("unmarshal message failed: %w", err)
	}

	// 更新任务状态为处理中
	if err := i.svcCtx.IndexHistoryModel.UpdateStatus(ctx, jobIndexTask.JobId, "running"); err != nil {
		return fmt.Errorf("update job status failed: %w", err)
	}

	// 执行索引任务（支持上下文取消）
	err := i.executeIndexJobWithContext(ctx, jobIndexTask)
	if err != nil {
		// 更新任务状态为失败
		i.svcCtx.IndexHistoryModel.UpdateStatus(ctx, jobIndexTask.JobId, "failed")
		return err
	}

	// 更新任务状态为成功
	return i.svcCtx.IndexHistoryModel.UpdateStatus(ctx, jobIndexTask.JobId, "success")
}

// executeIndexJobWithContext 执行索引任务（支持超时取消）
func (i *indexJobScheduler) executeIndexJobWithContext(ctx context.Context, task types.IndexTask) error {
	// 模拟耗时操作，支持上下文取消
	select {
	case <-time.After(5 * time.Second): // 模拟5秒处理时间
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
