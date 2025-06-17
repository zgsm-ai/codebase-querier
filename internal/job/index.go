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
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const indexNodeEnableVal = "1"
const indexNodeEnv = "INDEX_NODE"
const lockKeyPrefixFmt = "codebase_indexer:lock:%d"
const distLockTimeout = time.Minute * 5
const msgIdTraceKey = "msg_id"

type indexJob struct {
	svcCtx                *svc.ServiceContext
	ctx                   context.Context
	enableFlag            bool
	embeddingTaskPool     *ants.Pool
	graphTaskPool         *ants.Pool
	messageQueue          mq.MessageQueue
	consumerGroup         string   // 消费者组名称
	syncMetaFileCountDown sync.Map // 清理同步元数据文件计数器,key为msgId,value为计数，每完成一个任务，计数-1，当计数为0时，删除文件列表
}

type cleanSyncMetaFile struct {
	CodebasePath string       `json:"codebasePath"` // 代码库路径
	Paths        []string     `json:"paths"`        // 需要删除的文件路径
	counter      atomic.Int32 // 原子计数器
}

func NewIndexJob(serverCtx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	s := &indexJob{
		ctx:           serverCtx,
		svcCtx:        svcCtx,
		enableFlag:    os.Getenv(indexNodeEnv) == indexNodeEnableVal,
		messageQueue:  svcCtx.MessageQueue,
		consumerGroup: svcCtx.Config.MessageQueue.ConsumerGroup,
	}

	if !s.enableFlag {
		logx.Infof("INDEX_NODE flag is %t, not subscribe message queue", s.enableFlag)
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

func (i *indexJob) Start() {
	if !i.enableFlag {
		logx.Infof("INDEX_NODE flag is %t, disable index job.", i.enableFlag)
		return
	}

	logx.Infof("index job started, topic: %s", i.svcCtx.Config.IndexTask.Topic)

	// 启动一个协程，去清理同步元数据文件
	go i.cleanProcessedMetadataFile()

	// 轮询消息
	for {
		select {
		case <-i.ctx.Done():
			logx.Info("Context cancelled, exiting Job.")
			return

		default:
			// 消费消息队列
			msg, err := i.messageQueue.Consume(i.ctx, i.svcCtx.Config.IndexTask.Topic, types.ConsumeOptions{})
			if errors.Is(err, errs.ReadTimeout) {
				continue
			}
			if err != nil {
				logx.Errorf("consume index msg from mq error:%v", err)
				continue
			}
			// 处理消息
			i.processMessage(msg)
		}
	}
}

func (i *indexJob) cleanProcessedMetadataFile() {
	func() {
		for {
			select {
			case <-i.ctx.Done():
				logx.Info("context cancelled, exiting meta data clean Job.")
				return
			default:
				i.syncMetaFileCountDown.Range(func(key, value any) bool {
					// 如果value 为0 ，批量删除文件
					meta := value.(*cleanSyncMetaFile)
					if meta.counter.Load() <= 0 {
						logx.Infof("clean sync meta file, codebasePath:%s, paths:%v", meta.CodebasePath, meta.Paths)
						// TODO 当调用链和嵌入任务都成功时，清理元数据文件。改为移动到另一个隐藏文件夹中，每天定时清理，便于排查问题。
						if err := i.svcCtx.CodebaseStore.BatchDelete(i.ctx, meta.CodebasePath, meta.Paths); err != nil {
							logx.Errorf("failed to delete codebase %s metadata : %v, err: %v", meta.CodebasePath, meta.Paths, err)
						}
						// 删除计数器
						i.syncMetaFileCountDown.Delete(key)
					}
					return true
				})
			}
		}
	}()
}

// processMessage 处理单条消息的全部流程
func (i *indexJob) processMessage(msg *types.Message) {
	logger := logx.WithContext(i.ctx).WithFields(logx.Field(msgIdTraceKey, msg.ID))
	syncMsg, err := parseSyncMessage(msg)
	if err != nil {
		logger.Errorf("parse sync message failed. ack message, err:%v", err)
		err = i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			logger.Errorf("failed to Nack invalid message: %v", err)
		}
		return
	}
	if syncMsg == nil {
		logger.Error("sync msg is nil after parsing with no error. ack message.")
		err = i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			logger.Errorf("failed to Nack nil syncMsg message err: %v", err)
		}
		return
	}
	// 获取分布式锁， 5分钟超时
	locked, err := i.svcCtx.DistLock.TryLock(i.ctx, indexJobKey(syncMsg.CodebaseID), distLockTimeout)
	if err != nil || !locked {
		logger.Debugf("failed to acquire lock, nack message %s, err:%v", msg.ID, err)
		logger.Debugf("start to nack message %s", msg.ID)
		if ackErr := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
			logger.Errorf("failed to nack message %s from stream %s, group %s, err: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
			// TODO: Handle ACK failure - this is rare, but might require logging or alerting
		}
		logger.Debugf("nack message %s successfully.", msg.ID)
		return
	}
	defer i.svcCtx.DistLock.Unlock(i.ctx, indexJobKey(syncMsg.CodebaseID))
	logger.Debugf("acquire lock successfully, start to process message %s", msg.ID)
	// ack
	defer func() {
		logger.Debugf("start to ack message %s", msg.ID)
		if ackErr := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
			logger.Errorf("failed to Ack message %s from stream %s, group %s after processing attempt: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
			// TODO: Handle ACK failure - this is rare, but might require logging or alerting
		}
		logger.Debugf("ack message %s successfully.", msg.ID)
	}()

	// 本次同步的元数据列表
	syncFileModeMap, medataFileList, err := i.svcCtx.CodebaseStore.GetSyncFileListCollapse(i.ctx, syncMsg.CodebasePath)
	if err != nil {
		logger.Errorf("index job GetSyncFileListCollapse err:%w", err)
		return
	}
	if len(syncFileModeMap) == 0 {
		logger.Errorf("sync file list is nil, not process %v", syncMsg)
		return
	}

	// 判断消息是否是最新消息，如果不是最新消息，跳过
	if !i.IsCurrentLatestVersion(logger, syncMsg) {
		return
	}

	meta := &cleanSyncMetaFile{
		CodebasePath: syncMsg.CodebasePath,
		Paths:        medataFileList,
	}
	// index job ; graph job
	i.syncMetaFileCountDown.Store(syncMsg.SyncID, meta)

	// 设置本次任务的trace_id
	traceCtx := context.WithValue(i.ctx, tracer.Key, tracer.TaskTraceId(int(syncMsg.SyncID)))

	// 嵌入任务
	i.submitEmbeddingTask(traceCtx, msg, syncMsg, syncFileModeMap)

	// 关系图任务
	i.submitGraphTask(traceCtx, msg, syncMsg, syncFileModeMap)

}

func (i *indexJob) submitGraphTask(ctx context.Context, msg *types.Message, syncMsg *types.CodebaseSyncMessage, syncFileModeMap map[string]string) {
	tracer.WithTrace(ctx).Infof("start to submit codegraph task for msg:%s", msg.ID)
	if !i.svcCtx.Config.IndexTask.GraphTask.Enabled {
		tracer.WithTrace(ctx).Info("graph task is disabled, not process msg")
		return
	}

	// task 计数 + 1 graph task TODO, graph task now is relied on scip, which failed easily.
	// i.taskCounterIncr(syncMsg.SyncID)

	var err error
	// codegraph job
	codegraphCtx, graphTimeoutCancel := context.WithTimeout(ctx, i.svcCtx.Config.IndexTask.GraphTask.Timeout)
	codegraphProcessor, err := NewCodegraphProcessor(i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("failed to create codegraph task for message %s, err: %v", msg.ID, err)
		graphTimeoutCancel()
	} else {
		errGraphSubmit := i.graphTaskPool.Submit(func() {
			defer graphTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := codegraphProcessor.Process(codegraphCtx)
			if processErr != nil {
				tracer.WithTrace(ctx).Errorf("codegraph task failed, err: %v", processErr)
			} else {
				tracer.WithTrace(ctx).Infof("codegraph task successfully.")
				// TODO 让计数-1
				//value, ok := i.syncMetaFileCountDown.Load(syncMsg.SyncID)
				//if !ok {
				//	i.Logger.Errorf("sync meta file count down not found, syncID:%d", syncMsg.SyncID)
				//	return
				//}
				//value.(*cleanSyncMetaFile).counter.Add(-1)
			}
		})

		if errGraphSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			tracer.WithTrace(ctx).Errorf("failed to submit codegraph task, nack message, err: %v.", errGraphSubmit)
			err = i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
			if err != nil {
				tracer.WithTrace(ctx).Errorf("failed to nack message after graph submit failure: %v", err)
			}
		} else {
			tracer.WithTrace(ctx).Info("submit codegraph task successfully.")
		}
	}
}

func (i *indexJob) submitEmbeddingTask(ctx context.Context, msg *types.Message, syncMsg *types.CodebaseSyncMessage, syncFileModeMap map[string]string) {
	tracer.WithTrace(ctx).Infof("start to submit embedding task for msg:%s", msg.ID)
	if !i.svcCtx.Config.IndexTask.EmbeddingTask.Enabled {
		tracer.WithTrace(ctx).Infof("embedding task is disabled, not process msg")
		return
	}
	// task 计数 + 1
	i.taskCounterIncr(syncMsg.SyncID)

	var err error
	// embedding job
	embeddingCtx, embeddingTimeoutCancel := context.WithTimeout(ctx, i.svcCtx.Config.IndexTask.EmbeddingTask.Timeout)
	embeddingProcessor, err := NewEmbeddingProcessor(i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("failed to create embedding task, err: %v", err)
		embeddingTimeoutCancel()
	} else {
		errEmbeddingSubmit := i.embeddingTaskPool.Submit(func() {
			defer embeddingTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := embeddingProcessor.Process(embeddingCtx)
			if processErr != nil {
				// Embedding task failed, log and re-queue the original message body
				tracer.WithTrace(ctx).Errorf("embedding task failed, err:%v.", processErr)
			} else {
				tracer.WithTrace(ctx).Infof("embedding task successfully.")
				value, ok := i.syncMetaFileCountDown.Load(syncMsg.SyncID)
				if !ok {
					tracer.WithTrace(ctx).Errorf("sync meta file count down not found")
					return
				}
				value.(*cleanSyncMetaFile).counter.Add(-1)
			}
		})

		if errEmbeddingSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			tracer.WithTrace(ctx).Errorf("failed to submit embedding task, err: %v, nack message", msg.ID, errEmbeddingSubmit)
			err = i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
			if err != nil {
				tracer.WithTrace(ctx).Errorf("failed to nack message after embedding submit failed, err: %v", err)
			}
		} else {
			tracer.WithTrace(ctx).Info("submit embedding task successfully.")
		}
	}
}

func (i *indexJob) taskCounterIncr(syncId int32) {
	// 计数 +1
	value, ok := i.syncMetaFileCountDown.Load(syncId)
	if !ok {
		logx.Errorf("sync meta file count down not found, syncID:%d", syncId)
	} else {
		value.(*cleanSyncMetaFile).counter.Add(1)
	}
}

// IsCurrentLatestVersion 判断消息是否是最新消息
func (i *indexJob) IsCurrentLatestVersion(logger logx.Logger, syncMsg *types.CodebaseSyncMessage) bool {
	latestVersion, err := i.svcCtx.Cache.GetLatestVersion(i.ctx, types.SyncVersionKey(syncMsg.CodebaseID))
	if err != nil {
		logger.Errorf("index job GetLatestVersion for sync %d err:%v", syncMsg.SyncID, err)
	}
	if latestVersion > 0 && latestVersion > int64(syncMsg.SyncID) {
		logger.Infof("message version %d less than the latest version %d, ack and skip it.", syncMsg.SyncID, latestVersion)
		return false
	}
	return true
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
	if i.graphTaskPool != nil {
		i.graphTaskPool.Release()
	}
	if i.embeddingTaskPool != nil {
		i.embeddingTaskPool.Release()
	}

	logx.Info("indexJob closed successfully.")
}

func indexJobKey(codebaseId int32) string {
	return fmt.Sprintf(lockKeyPrefixFmt, codebaseId)
}
