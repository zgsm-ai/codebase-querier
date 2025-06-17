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
	"sync"
	"sync/atomic"
	"time"
)

const indexNodeEnableVal = "1"
const indexNodeEnv = "INDEX_NODE"
const lockKeyPrefixFmt = "codebase_indexer:lock:%d"
const distLockTimeout = time.Minute * 5

type indexJob struct {
	logx.Logger
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
		Logger:        logx.WithContext(serverCtx),
		svcCtx:        svcCtx,
		enableFlag:    os.Getenv(indexNodeEnv) == indexNodeEnableVal,
		messageQueue:  svcCtx.MessageQueue,
		consumerGroup: svcCtx.Config.MessageQueue.ConsumerGroup,
	}

	if !s.enableFlag {
		s.Logger.Infof("INDEX_NODE flag is %t, not subscribe message queue", s.enableFlag)
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
		i.Logger.Infof("INDEX_NODE flag is %t, disable index job.", i.enableFlag)
		return
	}

	i.Logger.Infof("index job started, topic: %s", i.svcCtx.Config.IndexTask.Topic)

	// 启动一个协程，去清理同步元数据文件
	go i.cleanProcessedMetadataFile()

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

func (i *indexJob) cleanProcessedMetadataFile() {
	func() {
		for {
			select {
			case <-i.ctx.Done():
				i.Logger.Info("context cancelled, exiting meta data clean Job.")
				return
			default:
				i.syncMetaFileCountDown.Range(func(key, value any) bool {
					// 如果value 为0 ，批量删除文件
					meta := value.(*cleanSyncMetaFile)
					if meta.counter.Load() <= 0 {
						i.Logger.Infof("clean sync meta file, codebasePath:%s, paths:%v", meta.CodebasePath, meta.Paths)
						// TODO 当调用链和嵌入任务都成功时，清理元数据文件。
						if err := i.svcCtx.CodebaseStore.BatchDelete(i.ctx, meta.CodebasePath, meta.Paths); err != nil {
							i.Logger.Errorf("failed to delete codebase %s metadata : %v, err: %v", meta.CodebasePath, meta.Paths, err)
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

	syncMsg, err := parseSyncMessage(msg)
	if err != nil {
		i.Logger.Errorf("parse sync message failed for message %s: %v. ack message.", msg.ID, err)
		err := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			i.Logger.Errorf("failed to Nack invalid message %s: %v", msg.ID, err)
		}
		return
	}
	if syncMsg == nil {
		i.Logger.Error("sync msg is nil after parsing with no error for message %s. ack message.", msg.ID)
		err := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
		if err != nil {
			i.Logger.Errorf("failed to Nack nil syncMsg message %s: %v", msg.ID, err)
		}
		return
	}
	// 获取分布式锁， 5分钟超时
	locked, err := i.svcCtx.DistLock.TryLock(i.ctx, indexJobKey(syncMsg.CodebaseID), distLockTimeout)
	if err != nil || !locked {
		i.Logger.Errorf("failed to acquire lock, nack message %s, err:%v", msg.ID, err)
		i.Logger.Debugf("start to nack message %s", msg.ID)
		if ackErr := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
			i.Logger.Errorf("failed to nack message %s from stream %s, group %s after processing attempt: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
			// TODO: Handle ACK failure - this is rare, but might require logging or alerting
		}
		i.Logger.Debugf("nack message %s successfully.", msg.ID)
		return
	}

	defer i.svcCtx.DistLock.Unlock(i.ctx, indexJobKey(syncMsg.CodebaseID))

	// ack
	defer func() {
		i.Logger.Debugf("start ack message %s", msg.ID)
		if ackErr := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
			i.Logger.Errorf("failed to Ack message %s from stream %s, group %s after processing attempt: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
			// TODO: Handle ACK failure - this is rare, but might require logging or alerting
		}
		i.Logger.Debugf("ack message %s successfully.", msg.ID)
	}()

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

	// 判断消息是否是最新消息，如果不是最新消息，跳过
	if !i.IsCurrentLatestVersion(err, syncMsg) {
		return
	}

	meta := &cleanSyncMetaFile{
		CodebasePath: syncMsg.CodebasePath,
		Paths:        medataFileList,
	}
	// index job ; graph job
	i.syncMetaFileCountDown.Store(syncMsg.SyncID, meta)

	// 嵌入任务
	i.submitEmbeddingTask(msg, syncMsg, syncFileModeMap)

	// 关系图任务
	i.submitGraphTask(msg, syncMsg, syncFileModeMap)

}

func (i *indexJob) submitGraphTask(msg *types.Message, syncMsg *types.CodebaseSyncMessage, syncFileModeMap map[string]string) {
	if !i.svcCtx.Config.IndexTask.GraphTask.Enabled {
		i.Logger.Infof("graph task is disabled, not process msg:%+v", syncMsg)
		return
	}

	// task 计数 + 1 graph task TODO, graph task now is relied on scip, which failed easily.
	// i.taskCounterIncr(syncMsg.SyncID)

	var err error
	// codegraph job
	codegraphCtx, graphTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.GraphTask.Timeout)
	codegraphProcessor, err := NewCodegraphProcessor(codegraphCtx, i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		i.Logger.Errorf("failed to create codegraph processor for message %s, err: %v", msg.ID, err)
		graphTimeoutCancel()
	} else {
		errGraphSubmit := i.graphTaskPool.Submit(func() {
			defer graphTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := codegraphProcessor.Process()
			if processErr != nil {
				i.Logger.Errorf("codegraph processor failed for message %s: err: %v", msg.ID, processErr)
			} else {
				// TODO 让计数-1
				value, ok := i.syncMetaFileCountDown.Load(syncMsg.SyncID)
				if !ok {
					i.Logger.Errorf("sync meta file count down not found, syncID:%d", syncMsg.SyncID)
					return
				}
				value.(*cleanSyncMetaFile).counter.Add(-1)
			}
		})

		if errGraphSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			i.Logger.Errorf("failed to submit codegraph processor task for message %s, nack message, err: %v.", msg.ID, errGraphSubmit)
			produceErr := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
			if produceErr != nil {
				i.Logger.Errorf("failed to nack message %s after graph submit failure: %v", msg.ID, produceErr)
			}
		}
	}
}

func (i *indexJob) submitEmbeddingTask(msg *types.Message, syncMsg *types.CodebaseSyncMessage, syncFileModeMap map[string]string) {
	if !i.svcCtx.Config.IndexTask.EmbeddingTask.Enabled {
		i.Logger.Infof("embedding task is disabled, not process msg: %+v", syncMsg)
		return
	}
	// task 计数 + 1
	i.taskCounterIncr(syncMsg.SyncID)

	var err error
	// embedding job
	embeddingCtx, embeddingTimeoutCancel := context.WithTimeout(i.ctx, i.svcCtx.Config.IndexTask.EmbeddingTask.Timeout)
	embeddingProcessor, err := NewEmbeddingProcessor(embeddingCtx, i.svcCtx, syncMsg, syncFileModeMap)
	if err != nil {
		i.Logger.Errorf("failed to create embedding processor for message %d: %v", syncMsg, err)
		embeddingTimeoutCancel()
	} else {
		errEmbeddingSubmit := i.embeddingTaskPool.Submit(func() {
			defer embeddingTimeoutCancel() // Cancel context when the goroutine finishes
			processErr := embeddingProcessor.Process()
			if processErr != nil {
				// Embedding task failed, log and re-queue the original message body
				i.Logger.Errorf("embedding processor failed for message %s, err:%v.", msg.ID, processErr)
			} else {
				value, ok := i.syncMetaFileCountDown.Load(syncMsg.SyncID)
				if !ok {
					i.Logger.Errorf("sync meta file count down not found, syncID:%d", syncMsg.SyncID)
					return
				}
				value.(*cleanSyncMetaFile).counter.Add(-1)
			}
		})

		if errEmbeddingSubmit != nil {
			// Submission failed (pool full or closed), log and re-queue the original message body
			i.Logger.Errorf("failed to submit embedding processor task for message %s, err: %v, nack message", msg.ID, errEmbeddingSubmit)
			produceErr := i.messageQueue.Nack(i.ctx, msg.Topic, i.consumerGroup, msg.ID)
			if produceErr != nil {
				i.Logger.Errorf("failed to nack message %s after embedding submit, err: %v", msg.ID, produceErr)
			}
		}
	}
}

func (i *indexJob) taskCounterIncr(syncId int32) {
	// 计数 +1
	value, ok := i.syncMetaFileCountDown.Load(syncId)
	if !ok {
		i.Logger.Errorf("sync meta file count down not found, syncID:%d", syncId)
	} else {
		value.(*cleanSyncMetaFile).counter.Add(1)
	}
}

// IsCurrentLatestVersion 判断消息是否是最新消息
func (i *indexJob) IsCurrentLatestVersion(err error, syncMsg *types.CodebaseSyncMessage) bool {
	latestVersion, err := i.svcCtx.Cache.GetLatestVersion(i.ctx, types.SyncVersionKey(syncMsg.CodebaseID))
	if err != nil {
		i.Logger.Errorf("index job GetLatestVersion err:%v", err)
	}
	if latestVersion > 0 && latestVersion > int64(syncMsg.SyncID) {
		logx.Infof("message %+v version less than the latest version %d, ack and skip it.", syncMsg, latestVersion)
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

	i.Logger.Info("indexJob closed successfully.")
}

func indexJobKey(codebaseId int32) string {
	return fmt.Sprintf(lockKeyPrefixFmt, codebaseId)
}
