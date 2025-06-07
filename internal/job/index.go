package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
)

const indexNodeEnableVal = "1"
const isIndexNodeEnv = "IS_INDEX_NODE"

type indexJob struct {
	logx.Logger
	svcCtx            *svc.ServiceContext
	ctx               context.Context
	enableFlag        bool
	embeddingTaskPool *ants.Pool
	graphTaskPool     *ants.Pool
	messageQueue      mq.MessageQueue
	consumerGroup     string // 消费者组名称

}

func newIndexJob(serverCtx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	s := &indexJob{
		ctx:           serverCtx,
		Logger:        logx.WithContext(serverCtx),
		svcCtx:        svcCtx,
		enableFlag:    os.Getenv(isIndexNodeEnv) == indexNodeEnableVal,
		messageQueue:  svcCtx.MessageQueue,
		consumerGroup: svcCtx.Config.MessageQueue.ConsumerGroup,
	}

	if !s.enableFlag {
		s.Logger.Infof("Job is disabled, IS_INDEX_NODE flag is %d, not subscribe message queue", s.enableFlag)
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

	// TODO: Implement ReclaimPendingMessages logic on startup if needed

	return s, nil
}

func (i *indexJob) Start() {
	if !i.enableFlag {
		i.Logger.Infof("index job is disabled, IS_INDEX_NODE flag is %d", i.enableFlag)
		return
	}

	i.Logger.Info("index job started.")
	// TODO 多消息合并，避免重复处理，尤其是代码图构建，间隔一定时间再触发下次构建。

	// 轮询消息
	for {
		select {
		case <-i.ctx.Done():
			i.Logger.Info("Context cancelled, exiting Job.")
			return

		default:
			// 消费消息队列
			msg, err := i.messageQueue.Consume(i.ctx, i.svcCtx.Config.IndexTask.Topic, types.ConsumeOptions{})
			if errors.Is(err, errs.ReadTimeout) {
				continue
			}
			if err != nil {
				i.Logger.Errorf("consume index msg from mq error:%v", err)
				continue
			}
			// 处理消息
			i.processMessage(msg)
		}
	}
}

// processMessage 处理单条消息的全部流程
func (i *indexJob) processMessage(msg *types.Message) {
	syncMsg, err := parseSyncMessage(msg)
	if err != nil {
		i.Logger.Errorf("parse sync message failed for message %s: %v. nack message.", msg.ID, err)
		err := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			i.Logger.Errorf("failed to Nack invalid message %s: %v", msg.ID, err)
		}
		return
	}
	if syncMsg == nil {

		i.Logger.Error("sync msg is nil after parsing with no error for message %s. Nacking message.", msg.ID)
		err := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			i.Logger.Errorf("failed to Nack nil syncMsg message %s: %v", msg.ID, err)
		}
		return
	}

	// 本次同步的元数据列表
	syncFileModeMap, medataFileList, err := i.svcCtx.CodebaseStore.GetSyncFileListCollapse(i.ctx, syncMsg.CodebasePath)
	if err != nil {
		i.Logger.Errorf("index job GetSyncFileListCollapse err:%w", err)
		return
	}
	if len(syncFileModeMap) == 0 {
		i.Logger.Errorf("sync file list is nil, not process %v", syncMsg)
		return
	}

	embeddingCtx, embeddingTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.EmbeddingTask.Timeout)
	codegraphCtx, graphTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.GraphTask.Timeout)
	embeddingProcessor, err := NewEmbeddingProcessor(embeddingCtx, i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		i.Logger.Errorf("failed to create embedding processor for message %s: %v", syncMsg, err)
		embeddingTimeoutCancel()
	} else {
		errEmbeddingSubmit := i.embeddingTaskPool.Submit(func() {
			defer embeddingTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := embeddingProcessor.Process()
			if processErr != nil {
				// Embedding task failed, log and re-queue the original message body
				i.Logger.Errorf("embedding processor failed for message %s: %v. Re-queueing message.", msg.ID, processErr)
				produceErr := i.messageQueue.Produce(context.Background(), msg.Topic, msg.Body, types.ProduceOptions{})
				if produceErr != nil {
					i.Logger.Errorf("failed to re-queue message %s after embedding failure: %v", msg.ID, produceErr)
				}
			}
		})

		if errEmbeddingSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			i.Logger.Errorf("failed to submit embedding processor task for message %s: %v. Re-queueing message.", msg.ID, errEmbeddingSubmit)
			produceErr := i.messageQueue.Produce(context.Background(), msg.Topic, msg.Body, types.ProduceOptions{})
			if produceErr != nil {
				i.Logger.Errorf("failed to re-queue message %s after embedding submission failure: %v", msg.ID, produceErr)
			}
		}
	}

	codegraphProcessor, err := NewCodegraphProcessor(codegraphCtx, i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		i.Logger.Errorf("failed to create codegraph processor for message %s: %v", msg.ID, err)
		graphTimeoutCancel()
	} else {
		errGraphSubmit := i.graphTaskPool.Submit(func() {
			defer graphTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := codegraphProcessor.Process()
			if processErr != nil {
				// Graph task failed, log and re-queue the original message body
				i.Logger.Errorf("codegraph processor failed for message %s: %v. Re-queueing message.", msg.ID, processErr)
				produceErr := i.messageQueue.Produce(context.Background(), msg.Topic, msg.Body, types.ProduceOptions{})
				if produceErr != nil {
					i.Logger.Errorf("failed to re-queue message %s after graph failure: %v", msg.ID, produceErr)
				}
			}
		})

		if errGraphSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			i.Logger.Errorf("failed to submit codegraph processor task for message %s: %v. Re-queueing message.", msg.ID, errGraphSubmit)
			produceErr := i.messageQueue.Produce(context.Background(), msg.Topic, msg.Body, types.ProduceOptions{})
			if produceErr != nil {
				i.Logger.Errorf("failed to re-queue message %s after graph submission failure: %v", msg.ID, produceErr)
			}
		}
	}

	if ackErr := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
		i.Logger.Errorf("failed to Ack message %s from stream %s, group %s after processing attempt: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
		// TODO: Handle ACK failure - this is rare, but might require logging or alerting
	}
	// 当调用链和嵌入任务都成功时，清理元数据文件。
	if err = i.svcCtx.CodebaseStore.BatchDelete(i.ctx, syncMsg.CodebasePath, medataFileList); err != nil {
		i.Logger.Errorf("failed to delete codebase %s metadata : %v, err: %v", syncMsg.CodebasePath, medataFileList, err)
	}
}

// parseSyncMessage
func parseSyncMessage(m *types.Message) (*types.CodebaseSyncMessage, error) {
	if m == nil {
		return nil, errors.New("sync message is nil")
	}
	var msg types.CodebaseSyncMessage // Use a concrete type here to avoid potential nil pointer issues after Unmarshal
	if err := json.Unmarshal(m.Body, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal sync message failed: %w", err)
	}
	return &msg, nil
}

func (i *indexJob) Close() {
	i.graphTaskPool.Release()
	i.embeddingTaskPool.Release()
	// 关闭消息队列连接
	err := i.messageQueue.Close()
	if err != nil {
		i.Logger.Errorf("close message queue failed: %v", err)
	}
	i.Logger.Info("indexJob closed successfully.")
}
