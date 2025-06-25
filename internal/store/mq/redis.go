package mq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"

	"github.com/redis/go-redis/v9"

	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const topicField = "topic"
const timestampField = "timestamp"
const bodyField = "body"

// redisMQ Redis消息队列实现（基于原生redis/go-redis/v9 和 Streams）
type redisMQ struct {
	client *redis.Client // 原生Redis客户端
	conf   config.MessageQueueConf
}

// NewRedisMQ 创建Redis消息队列实例
// consumerGroup 参数指定消费者组的名称
func NewRedisMQ(ctx context.Context, client *redis.Client, conf config.MessageQueueConf) (MessageQueue, error) {
	mq := &redisMQ{
		client: client,
		conf:   conf,
	}
	go mq.deadLetterConsumer(ctx, conf)
	return mq, nil
}

func (mq *redisMQ) deadLetterConsumer(ctx context.Context, conf config.MessageQueueConf) {
	logx.Infof("dead_letter_consumer started, dead_letter_topic: %s, internal: %f min", conf.DeadLetterTopic, mq.conf.DeadLetterInterval.Minutes())
	ticker := time.NewTicker(mq.conf.DeadLetterInterval)
	defer ticker.Stop()
	deadLetterTopic := conf.DeadLetterTopic
	consumerGroup := "codebase-indexer-dead-letter-consumer"
	processedCount := 0
	for {
		select {
		case <-ctx.Done():
			logx.Infof("dead_letter_consumer exited, processed %d messages", processedCount)
			logx.Info("dead_letter_consumer exited.")
			return
		case <-ticker.C:
			logx.Info("dead_letter_consumer starting batch processing")
			msg, err := mq.Consume(ctx, deadLetterTopic, consumerGroup, types.ConsumeOptions{
				ReadTimeout: time.Second, // 1秒超时，避免阻塞
			})

			if err != nil {
				if errors.Is(err, errs.ReadTimeout) {
					break // 没有更多消息
				}
				logx.Errorf("dead_letter_consumer read dead letter topic failed: %v", err)
				break
			}

			// 原始topic
			originTopic := msg.Topic // 假设topic是字符串键

			if originTopic == deadLetterTopic || originTopic == types.EmptyString {
				logx.Errorf("dead_letter_consumer invalid original topic: %v, message ID: %s", originTopic, msg.ID)
				if ackErr := mq.Ack(ctx, deadLetterTopic, consumerGroup, msg.ID); ackErr != nil {
					logx.Errorf("dead_letter_consumer failed to ack message with invalid topic: %v", ackErr)
				}
				continue
			}

			// 复用 Produce 方法重投递
			err = mq.Produce(ctx, originTopic, msg.Body, types.ProduceOptions{})
			if err != nil {
				logx.Errorf("dead_letter_consumer redeliver to %s failed: %v, message ID: %s", originTopic, err, msg.ID)
				// 重投递失败，不确认消息，让消息继续留在队列中
				continue
			}

			if err = mq.Ack(ctx, deadLetterTopic, consumerGroup, msg.ID); err != nil {
				logx.Errorf("dead_letter_consumer failed to ack redelivered message: %v, message ID: %s", err, msg.ID)
			} else {
				processedCount++
				logx.Infof("dead_letter_consumer redelivered to topic %s successfully, message ID: %s", originTopic, msg.ID)
			}
		}
	}
}

func (mq *redisMQ) CreateTopic(ctx context.Context, topic string, opts types.TopicOptions) error {
	// Redis Streams 在第一次XAdd时自动创建，无需显式创建
	// 可以在这里选择预先创建消费者组，以便在Consume时少一步检查
	// 为了简化，我们仍然在Consume时按需创建消费者组
	return nil
}

func (mq *redisMQ) DeleteTopic(ctx context.Context, topic string) error {
	// 删除Stream使用DEL命令
	_, err := mq.client.Del(ctx, topic).Result()
	return err
}

// Produce 实现MessageQueue接口的Publish方法
// 使用 XAdd 命令将消息添加到 Stream
func (mq *redisMQ) Produce(ctx context.Context, topic string, message []byte, opts types.ProduceOptions) error {
	// 将消息体、topic和时间戳作为 Streams entry 的字段存储
	values := map[string]interface{}{
		"body":      message,
		"topic":     topic,
		"timestamp": time.Now().UnixNano(),
	}

	// 使用 XAdd 添加消息到 Stream
	// "*" 表示让 Redis 自动生成消息 ID
	_, err := mq.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: values,
		MaxLen: mq.conf.SingleMsgQueueMaxLen,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to XAdd message to stream %s: %w", topic, err)
	}
	return nil
}

// Consume 实现MessageQueue接口的Subscribe方法
// 使用 XReadGroup 命令从 Streams 消费者组中读取消息
// 并使用 XAck 确认消息
func (mq *redisMQ) Consume(ctx context.Context, topic string, consumerGroup string, opts types.ConsumeOptions) (*types.Message, error) {
	if consumerGroup == types.EmptyString {
		return nil, fmt.Errorf("consumerGroup cannot be empty")
	}
	// 确保消费者组存在，如果不存在则创建
	// 使用 MKSTREAM 选项可以在 Stream 不存在时同时创建 Stream
	err := mq.client.XGroupCreateMkStream(ctx, topic, consumerGroup, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		// 忽略 BUSYGROUP 错误，表示组已存在
		return nil, fmt.Errorf("failed to create consumer group %s for stream %s: %w", consumerGroup, topic, err)
	}

	// 使用 XReadGroup 从消费者组读取消息
	// ">" 表示从该消费者组上次读取的位置之后开始读
	// Count 设置每次最多读取一条消息
	// Block 设置阻塞时间，与 opts.ReadTimeout 对应
	streams, err := mq.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: "consumer-" + mq.client.ClientID(ctx).String(), // 使用客户端ID作为消费者名称
		Streams:  []string{topic, ">"},
		Count:    1,
		Block:    opts.ReadTimeout,
		NoAck:    false, // 需要手动 ACK
	}).Result()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, redis.Nil) || err.Error() == "redis: nil" {
			// 超时或无消息
			return nil, errs.ReadTimeout
		}
		return nil, fmt.Errorf("failed to XReadGroup from stream %s, group %s: %w", topic, consumerGroup, err)
	}

	// 检查是否读取到消息
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, errs.ReadTimeout // 没有新消息
	}

	// 取出第一条消息（因为 Count=1）
	msg := streams[0].Messages[0]

	// 解析消息体 ("body" 字段)
	body, ok := msg.Values[bodyField].(string)
	if !ok {
		// TODO: Handle malformed message - maybe move to a dead-letter stream?
		// For now, we continue without acknowledging.
		tracer.WithTrace(ctx).Errorf("received message %s with invalid body format from stream %s, group %s: %v", msg.ID, topic, consumerGroup, msg.Values[bodyField])
		return nil, fmt.Errorf("received message %s with invalid body format", msg.ID)
	}

	// 解析消息体 ("topic" 字段)
	msgTopic, ok := msg.Values[topicField].(string)
	if !ok {
		// TODO: Handle malformed message - maybe move to a dead-letter stream?
		// For now, we continue without acknowledging.
		tracer.WithTrace(ctx).Errorf("received message %s with invalid msgTopic format from stream %s, group %s: %v", msg.ID, topic, consumerGroup, msg.Values[topicField])
		return nil, fmt.Errorf("received message %s with invalid msgTopic format", msg.ID)
	}

	// 解析时间戳 ("timestamp" 字段)
	timestampVal, ok := msg.Values[timestampField].(string)
	if !ok {
		// 如果时间戳字段不存在或格式错误，记录错误并使用当前时间
		tracer.WithTrace(ctx).Errorf("received message %s without valid timestamp field from stream %s, group %s", msg.ID, topic, consumerGroup)
		// 继续处理消息，Timestamp 使用当前时间
		return &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     msgTopic,
			Timestamp: time.Now(),
		}, nil
	}

	nanoTimestamp, err := strconv.ParseInt(timestampVal, 10, 64)
	if err != nil {
		// 如果时间戳解析失败，记录错误并使用当前时间
		tracer.WithTrace(ctx).Errorf("Failed to parse timestamp for message %s from stream %s, group %s: %v", msg.ID, topic, consumerGroup, err)
		// 继续处理消息，Timestamp 使用当前时间
		return &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     msgTopic,
			Timestamp: time.Now(),
		}, nil
	}

	return &types.Message{
		ID:        msg.ID, // Streams 消息有唯一的 ID
		Body:      []byte(body),
		Topic:     msgTopic,
		Timestamp: time.Unix(0, nanoTimestamp), // 从纳秒时间戳转换
	}, nil
}

// Ack 实现MessageQueue接口的Ack方法
// 使用 XAck 命令确认 Streams 消息
func (mq *redisMQ) Ack(ctx context.Context, stream, group string, id string) error {
	_, err := mq.client.XAck(ctx, stream, group, id).Result()
	if err != nil {
		return fmt.Errorf("failed to XAck message %s in stream %s, group %s: %w", id, stream, group, err)
	}
	return nil
}

// Nack 实现MessageQueue接口的Nack方法
// 发送到死信队列，避免消息循环
func (mq *redisMQ) Nack(ctx context.Context, topic, consumerGroup string, msgId string, message []byte, opts types.NackOptions) error {
	values := map[string]interface{}{
		"body":      message,
		"topic":     topic, // 新增topic字段，便于死信队列重发
		"timestamp": time.Now().UnixNano(),
	}

	// 使用 XAdd 添加消息到 Stream
	// "*" 表示让 Redis 自动生成消息 ID
	_, err := mq.client.XAdd(ctx, &redis.XAddArgs{
		Stream: mq.conf.DeadLetterTopic,
		Values: values,
		MaxLen: mq.conf.SingleMsgQueueMaxLen,
	}).Result()
	if err != nil {
		return fmt.Errorf("nack failed to readd message to stream %s: %w", topic, err)
	}

	// 删除消息
	if err := mq.client.XDel(ctx, topic, msgId).Err(); err != nil {
		tracer.WithTrace(ctx).Errorf("nack failed to del original message %s: %v", msgId, err)
		return err
	}

	tracer.WithTrace(ctx).Debugf("nack message %s in stream %s, group %s successfully.", msgId, topic, consumerGroup)
	return nil
}

func (mq *redisMQ) Close(ctx context.Context) error {
	logx.Infof("redis message queue exited.")
	return nil
}

// TODO
// reclaimPendingMessages 可以用于处理消费者崩溃后未 ACK 的消息
// 通常需要在消费者启动时调用，或者通过一个独立的监控进程来处理
func (mq *redisMQ) reclaimPendingMessages(ctx context.Context, topic string, consumerGroup string, idleTime time.Duration, count int) ([]*types.Message, error) {
	// 查找处于 Pending 상태且超时未处理的消息
	// 使用 XPENDING 命令获取 Pending 消息的详细信息
	// 注意：XPENDING 不直接返回消息体，需要通过 XCLAIM 或 XAUTOCLAIM 获取

	// 转移这些 Pending 消息的所有权给当前消费者实例，并获取消息体
	claimedStreams, _, err := mq.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   topic,
		Group:    consumerGroup,
		MinIdle:  idleTime, // 使用 MinIdle 字段
		Start:    "0-0",    // 从 Streams 的开始位置查找
		Count:    int64(count),
		Consumer: "consumer-" + mq.client.ClientID(ctx).String(), // 将消息转移给当前消费者
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to XAutoClaim messages for stream %s, group %s: %w", topic, consumerGroup, err)
	}

	var messages []*types.Message
	for _, msg := range claimedStreams {
		// 解析消息体 ("body" 字段)
		body, ok := msg.Values["body"].(string)
		if !ok {
			// TODO: Handle malformed reclaimed message - maybe move to a dead-letter stream?
			// For now, we continue without acknowledging.
			// ACK 这条无法解析的消息，避免重复处理 (Optional, depends on policy)
			// ackErr := r.client.XAck(ctx, stream, r.consumerGroup, msg.ID).Err()
			// if ackErr != nil {
			// tracer.WithTrace(ctx).Errorf("Failed to XAck malformed reclaimed message %s in stream %s, group %s: %v", msg.ID, stream, r.consumerGroup, ackErr)
			// }
			tracer.WithTrace(ctx).Errorf("received reclaimed message %s with invalid body format from stream %s, group %s: %v", msg.ID, topic, consumerGroup, msg.Values["body"])
			continue
		}

		// 解析时间戳 ("timestamp" 字段)
		timestampVal, ok := msg.Values["timestamp"].(string)
		if !ok {
			tracer.WithTrace(ctx).Errorf("received reclaimed message %s without valid timestamp field from stream %s, group %s", msg.ID, topic, consumerGroup)
			messages = append(messages, &types.Message{
				ID:        msg.ID,
				Body:      []byte(body),
				Topic:     topic,
				Timestamp: time.Now(),
			})
			continue
		}

		nanoTimestamp, err := strconv.ParseInt(timestampVal, 10, 64)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("failed to parse timestamp for reclaimed message %s from stream %s, group %s: %v", msg.ID, topic, consumerGroup, err)
			messages = append(messages, &types.Message{
				ID:        msg.ID,
				Body:      []byte(body),
				Topic:     topic,
				Timestamp: time.Now(),
			})
			continue
		}

		messages = append(messages, &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     topic,
			Timestamp: time.Unix(0, nanoTimestamp),
		})
	}

	// Note: The reclaimed messages are now owned by the current consumer instance.
	// The consumer needs to process these messages and then call XAck (which is the Ack method).

	return messages, nil
}
