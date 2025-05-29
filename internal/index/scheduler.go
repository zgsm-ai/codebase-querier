package index

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/panjf2000/ants/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/index/codegraph"
	"github.com/zgsm-ai/codebase-indexer/internal/index/embedding"
	"github.com/zgsm-ai/codebase-indexer/internal/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
)

const indexNodeEnableVal = "1"
const isIndexNodeEnv = "IS_INDEX_NODE"

type IndexTaskScheduler interface {
	Schedule()
	Close()
}

type indexTaskScheduler struct {
	logx.Logger
	svcCtx            *svc.ServiceContext
	ctx               context.Context
	enableFlag        bool
	embeddingTaskPool *ants.Pool
	graphTaskPool     *ants.Pool
	messageQueue      mq.MessageQueue
}

func NewIndexJobScheduler(svcCtx *svc.ServiceContext, serverCtx context.Context) (IndexTaskScheduler, error) {
	s := &indexTaskScheduler{
		ctx:          serverCtx,
		Logger:       logx.WithContext(serverCtx),
		svcCtx:       svcCtx,
		enableFlag:   os.Getenv(isIndexNodeEnv) == indexNodeEnableVal,
		messageQueue: svcCtx.MessageQueue,
	}

	if !s.enableFlag {
		s.Logger.Infof("IndexTaskScheduler is disabled, IS_INDEX_NODE flag is %d, not subscribe message queue", s.enableFlag)
		return s, nil
	}

	// 初始化协程池
	embeddingTaskPool, err := ants.NewPool(svcCtx.Config.IndexTask.EmbeddingTask.PoolSize, ants.WithOptions(
		ants.Options{
			MaxBlockingTasks: 1, // max queue tasks, if queue is full, will block
			Nonblocking:      false,
		},
	))
	if err != nil {
		return nil, err
	}
	graphTaskPool, err := ants.NewPool(svcCtx.Config.IndexTask.GraphTask.PoolSize, ants.WithOptions(
		ants.Options{
			MaxBlockingTasks: 1, // max queue tasks, if queue is full, will block
			Nonblocking:      false,
		},
	))
	if err != nil {
		return nil, err
	}
	s.embeddingTaskPool = embeddingTaskPool
	s.graphTaskPool = graphTaskPool

	return s, nil
}
func (i *indexTaskScheduler) Schedule() {
	if !i.enableFlag {
		i.Logger.Infof("IndexTaskScheduler is disabled, IS_INDEX_NODE flag is %d", i.enableFlag)
		return
	}

	i.Logger.Info("IndexTaskScheduler started.")

	// 轮询消息
	for {
		select {
		case <-i.ctx.Done():
			i.Logger.Info("Context cancelled, exiting IndexTaskScheduler.")
			return

		default:

			// 消费消息队列
			msg, err := i.messageQueue.Consume(i.ctx, i.svcCtx.Config.IndexTask.Topic, types.ConsumeOptions{})
			if errors.Is(err, types.ErrReadTimeout) {
				continue
			}
			if err != nil {
				i.Logger.Errorf("consume index task msg from mq error:%v", err)
				continue
			}

			i.Logger.Debugf("received sync message %v", msg)

			syncMsg, err := parseSyncMessage(msg)
			if err != nil {
				i.Logger.Errorf("parse sync message failed: %v", err)
				continue
			}
			if syncMsg == nil {
				i.Logger.Error("sync msg is nil")
				continue
			}

			// 嵌入任务
			embeddingCtx, embeddingTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.EmbeddingTask.Timeout)
			embeddingTask := embedding.NewTask(embeddingCtx, i.svcCtx, syncMsg)

			// 代码关系图任务
			graphCtx, graphTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.GraphTask.Timeout)
			codegraphTask := codegraph.NewTask(graphCtx, i.svcCtx, syncMsg)

			// 超时终止
			embeddingTimeoutCancel()
			graphTimeoutCancel()

			// TODO 队列满了，会阻塞任务提交，两个池彼此间会受影响
			err = i.embeddingTaskPool.Submit(embeddingTask.Execute)
			if err != nil {
				i.Logger.Errorf("submit embedding task failed: %v", err)
			}

			err = i.graphTaskPool.Submit(codegraphTask.Execute)
			if err != nil {
				i.Logger.Errorf("submit graph task failed: %v", err)
			}
		}
	}

}

// parseSyncMessage
func parseSyncMessage(m *types.Message) (*types.CodebaseSyncMessage, error) {
	if m == nil {
		return nil, errors.New("sync message is nil")
	}
	var msg *types.CodebaseSyncMessage
	if err := json.Unmarshal(m.Body, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (i *indexTaskScheduler) Close() {
	i.graphTaskPool.Release()
	i.embeddingTaskPool.Release()
	// 关闭消息队列连接
	err := i.svcCtx.MessageQueue.Close()
	if err != nil {
		i.Logger.Errorf("close message queue failed: %v", err)
	}
	i.Logger.Info("indexTaskScheduler closed successfully.")
}
