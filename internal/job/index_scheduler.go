package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/mq"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"os"
	"time"
)

const indexNodeEnableVal = "1"
const indexNodeEnv = "INDEX_NODE"
const lockKeyPrefixFmt = "codebase_indexer:lock:%d"
const DistLockTimeout = time.Minute * 3
const msgFailedThreshold = 10

type IndexTaskScheduler struct {
	svcCtx     *svc.ServiceContext
	ctx        context.Context
	enableFlag bool

	messageQueue  mq.MessageQueue
	consumerGroup string // 消费者组名称
}

func NewIndexTaskScheduler(serverCtx context.Context, svcCtx *svc.ServiceContext) (Job, error) {
	s := &IndexTaskScheduler{
		ctx:           serverCtx,
		svcCtx:        svcCtx,
		enableFlag:    os.Getenv(indexNodeEnv) == indexNodeEnableVal,
		messageQueue:  svcCtx.MessageQueue,
		consumerGroup: svcCtx.Config.IndexTask.ConsumerGroup,
	}

	if !s.enableFlag {
		logx.Infof("INDEX_NODE flag is %t, not subscribe message queue", s.enableFlag)
		return s, nil
	}

	return s, nil
}

func (i *IndexTaskScheduler) Start() {
	if !i.enableFlag {
		logx.Infof("INDEX_NODE flag is %t, disable index job.", i.enableFlag)
		return
	}

	logx.Infof("index job started, topic: %s", i.svcCtx.Config.IndexTask.Topic)

	// 轮询消息
	for {
		select {
		case <-i.ctx.Done():
			logx.Info("Context cancelled, exiting Job.")
			return

		default:
			// 消费消息队列
			msg, err := i.messageQueue.Consume(i.ctx, i.svcCtx.Config.IndexTask.Topic,
				i.svcCtx.Config.IndexTask.ConsumerGroup,
				types.ConsumeOptions{})
			if errors.Is(err, errs.ReadTimeout) {
				continue
			}
			if err != nil {
				logx.Errorf("consume index params from mq error:%v", err)
				continue
			}
			// 处理消息
			ctx := context.WithValue(i.ctx, tracer.Key, tracer.MsgTraceId(msg.ID))
			i.processMessage(ctx, msg)
		}
	}
}

// processMessage 处理单条消息的全部流程
func (i *IndexTaskScheduler) processMessage(ctx context.Context, msg *types.Message) {
	syncMsg, err := parseSyncMessage(msg)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("parse sync message failed. ack message, err:%v", err)
		i.ackSilently(ctx, msg)
		return
	}
	if syncMsg == nil {
		tracer.WithTrace(ctx).Error("sync params is nil after parsing with no error. ack message.")
		i.ackSilently(ctx, msg)
		return
	}

	// 获取分布式锁， n分钟超时
	// 在任务中执行结束unlock
	lockKey := IndexJobKey(syncMsg.CodebaseID)
	mux, locked, err := i.svcCtx.DistLock.TryLock(i.ctx, lockKey, i.svcCtx.Config.IndexTask.LockTimeout)
	if err != nil || !locked {
		tracer.WithTrace(ctx).Debugf("failed to acquire lock, nack message %s, err:%v", msg.ID, err)
		i.nackSilently(ctx, msg.Topic, i.consumerGroup, msg.ID, msg.Body)
		return
	}

	tracer.WithTrace(ctx).Debugf("acquire lock successfully, start to process message %s, lockKey: %s", msg.ID, lockKey)

	// 元数据列表
	medataFiles, err := i.svcCtx.CodebaseStore.GetSyncFileListCollapse(i.ctx, syncMsg.CodebasePath)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("index job GetSyncFileListCollapse err:%w", err)
		i.ackSilently(ctx, msg)
		return
	}
	if medataFiles == nil || len(medataFiles.FileModelMap) == 0 {
		tracer.WithTrace(ctx).Errorf("sync file list is nil, not process %v", syncMsg)
		i.ackSilently(ctx, msg)
		return
	}

	// 判断消息是否是最新消息，如果不是最新消息，跳过
	if !i.IsCurrentLatestVersion(ctx, syncMsg) {
		i.ackSilently(ctx, msg)
		return
	}

	// 设置本次任务的trace_id
	traceCtx := context.WithValue(i.ctx, tracer.Key, tracer.TaskTraceId(int(syncMsg.SyncID)))

	// 任务处理失败，uack 消息，重复处理，并记录重复处理次数。

	isEmbedTaskSuccess := syncMsg.IsEmbedTaskSuccess
	isGraphTaskSuccess := syncMsg.IsGraphTaskSuccess
	// 提交失败, nack; TODO处理失败，得看下怎么做，不行放在一个协程池里面处理，
	if isEmbedTaskSuccess && isGraphTaskSuccess {
		tracer.WithTrace(ctx).Debugf("not submit embedding task, just ack ,because params isEmbedTaskSuccess is %t, isGraphTaskSuccess is %t ",
			isEmbedTaskSuccess, isGraphTaskSuccess)
		i.ackSilently(ctx, msg)
		return
	}
	// 失败次数
	if syncMsg.FailedTimes >= msgFailedThreshold {
		tracer.WithTrace(ctx).Debugf("not submit embedding task, just ack ,because params reached failed times limit: %d ",
			msgFailedThreshold)
		i.ackSilently(ctx, msg)
		return
	}

	var submitErr error

	// 嵌入+ 关系图 任务
	submitErr = i.submitIndexTask(traceCtx, mux, msg, syncMsg, medataFiles)
	if submitErr != nil {
		tracer.WithTrace(ctx).Errorf("failed to submit index task submit, submitErr: %v", submitErr)
		// reproduce
		i.nackSilently(ctx, msg.Topic, i.consumerGroup, msg.ID, msg.Body)
		return
	}
	tracer.WithTrace(ctx).Info("submit index task successfully.")

}

func (i *IndexTaskScheduler) nackSilently(ctx context.Context, topic string, consumerGroup string, msgId string, body []byte) {
	if ackErr := i.messageQueue.Nack(i.ctx, topic, consumerGroup, msgId, body, types.NackOptions{}); ackErr != nil {
		tracer.WithTrace(ctx).Errorf("failed to nack message %s from stream %s, group %s, err: %v", msgId, topic, i.consumerGroup, ackErr)
		// TODO: Handle ACK failure - this is rare, but might require logging or alerting
	}
	tracer.WithTrace(ctx).Debugf("nack message %s successfully.", msgId)
}

func (i *IndexTaskScheduler) ackSilently(ctx context.Context, msg *types.Message) {
	tracer.WithTrace(ctx).Debugf("start to ack message %s", msg.ID)
	if ackErr := i.messageQueue.Ack(i.ctx, msg.Topic, i.consumerGroup, msg.ID); ackErr != nil {
		tracer.WithTrace(ctx).Errorf("failed to ack message %s from stream %s, group %s, err: %v", msg.ID, msg.Topic, i.consumerGroup, ackErr)
	}
	tracer.WithTrace(ctx).Debugf("ack message %s successfully.", msg.ID)
}

func (i *IndexTaskScheduler) Submit(ctx context.Context, taskRun func()) error {
	return i.svcCtx.TaskPool.Submit(taskRun)
}

func (i *IndexTaskScheduler) submitIndexTask(ctx context.Context, lockMux *redsync.Mutex, msg *types.Message, syncMsg *types.CodebaseSyncMessage,
	syncMetaFiles *types.CollapseSyncMetaFile) error {
	tracer.WithTrace(ctx).Infof("start to submit task for params:%s", msg.ID)
	taskRun := func() {
		task := &IndexTask{
			SvcCtx:  i.svcCtx,
			LockMux: lockMux,
			Params: &IndexTaskParams{
				CodebaseID:           syncMsg.CodebaseID,
				CodebasePath:         syncMsg.CodebasePath,
				CodebaseName:         syncMsg.CodebaseName,
				SyncMetaFiles:        syncMetaFiles,
				EnableCodeGraphBuild: i.svcCtx.Config.IndexTask.GraphTask.Enabled,
				EnableEmbeddingBuild: i.svcCtx.Config.IndexTask.EmbeddingTask.Enabled,
			},
		}

		embedOk, graphOk := task.Run(ctx)
		if embedOk && graphOk {
			tracer.WithTrace(ctx).Infof("index task Run successfully.")
			return
		}

		syncMsg.FailedTimes++

		tracer.WithTrace(ctx).Errorf("index task failed, embedOk:%t, graphOk: %t, failedTimes: %d reproduce params.",
			embedOk, graphOk, syncMsg.FailedTimes)

		if embedOk {
			syncMsg.IsEmbedTaskSuccess = true
		}
		if graphOk {
			syncMsg.IsGraphTaskSuccess = true
		}

		// reproduce
		bytes, err := json.Marshal(syncMsg)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("index task failed, reproduce params, marshal err:%v", err)
			return
		}
		i.nackSilently(ctx, msg.Topic, i.consumerGroup, msg.ID, bytes)
	}

	return i.Submit(ctx, taskRun)
}

// IsCurrentLatestVersion 判断消息是否是最新消息
func (i *IndexTaskScheduler) IsCurrentLatestVersion(ctx context.Context, syncMsg *types.CodebaseSyncMessage) bool {
	latestVersion, err := i.svcCtx.Cache.GetLatestVersion(i.ctx, types.SyncVersionKey(syncMsg.CodebaseID))
	if err != nil {
		tracer.WithTrace(ctx).Errorf("index job GetLatestVersion for sync %d err:%v", syncMsg.SyncID, err)
	}
	if latestVersion > 0 && latestVersion > int64(syncMsg.SyncID) {
		tracer.WithTrace(ctx).Infof("message version %d less than the latest version %d, ack and skip it.", syncMsg.SyncID, latestVersion)
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

func (i *IndexTaskScheduler) Close() {
	logx.Info("IndexTaskScheduler closed successfully.")
}

func IndexJobKey(codebaseId int32) string {
	return fmt.Sprintf(lockKeyPrefixFmt, codebaseId)
}
