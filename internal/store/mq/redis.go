package mq

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// redisMQ Redis消息队列实现（基于原生redis/go-redis/v9 和 Streams）
type redisMQ struct {
	logger        logx.Logger
	client        *redis.Client // 原生Redis客户端
	consumerGroup string        // 消费者组名称
}

// NewRedisMQ 创建Redis消息队列实例
// consumerGroup 参数指定消费者组的名称
func NewRedisMQ(ctx context.Context, client *redis.Client, consumerGroup string) (MessageQueue, error) {
	// 可以在这里检查或创建消费者组，但为了简化，我们在Consume中按需创建
	if consumerGroup == "" {
		return nil, errors.New("consumerGroup cannot be empty")
	}
	return &redisMQ{
		client:        client,
		logger:        logx.WithContext(ctx),
		consumerGroup: consumerGroup,
	}, nil
}

func (r *redisMQ) Close() error {
	return nil // 对于go-redis客户端，通常由上层管理连接生命周期
}

func (r *redisMQ) CreateTopic(ctx context.Context, topic string, opts types.TopicOptions) error {
	// Redis Streams 在第一次XAdd时自动创建，无需显式创建
	// 可以在这里选择预先创建消费者组，以便在Consume时少一步检查
	// 为了简化，我们仍然在Consume时按需创建消费者组
	return nil
}

func (r *redisMQ) DeleteTopic(ctx context.Context, topic string) error {
	// 删除Stream使用DEL命令
	_, err := r.client.Del(ctx, topic).Result()
	return err
}

// Produce 实现MessageQueue接口的Publish方法
// 使用 XAdd 命令将消息添加到 Stream
func (r *redisMQ) Produce(ctx context.Context, topic string, message []byte, opts types.ProduceOptions) error {
	// 将消息体和时间戳作为 Streams entry 的字段存储
	values := map[string]interface{}{
		"body":      message,
		"timestamp": time.Now().UnixNano(), // 存储纳秒时间戳
	}

	// 使用 XAdd 添加消息到 Stream
	// "*" 表示让 Redis 自动生成消息 ID
	_, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: values,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to XAdd message to stream %s: %w", topic, err)
	}
	return nil
}

// Consume 实现MessageQueue接口的Subscribe方法
// 使用 XReadGroup 命令从 Streams 消费者组中读取消息
// 并使用 XAck 确认消息
func (r *redisMQ) Consume(ctx context.Context, topic string, opts types.ConsumeOptions) (*types.Message, error) {
	// 确保消费者组存在，如果不存在则创建
	// 使用 MKSTREAM 选项可以在 Stream 不存在时同时创建 Stream
	err := r.client.XGroupCreateMkStream(ctx, topic, r.consumerGroup, "0").Err()
	if err != nil && !errors.Is(err, redis.Nil) && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		// 忽略 BUSYGROUP 错误，表示组已存在
		return nil, fmt.Errorf("failed to create consumer group %s for stream %s: %w", r.consumerGroup, topic, err)
	}

	// 使用 XReadGroup 从消费者组读取消息
	// ">" 表示从该消费者组上次读取的位置之后开始读
	// Count 设置每次最多读取一条消息
	// Block 设置阻塞时间，与 opts.ReadTimeout 对应
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.consumerGroup,
		Consumer: "consumer-" + r.client.ClientID(ctx).String(), // 使用客户端ID作为消费者名称
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
		return nil, fmt.Errorf("failed to XReadGroup from stream %s, group %s: %w", topic, r.consumerGroup, err)
	}

	// 检查是否读取到消息
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, errs.ReadTimeout // 没有新消息
	}

	// 取出第一条消息（因为 Count=1）
	msg := streams[0].Messages[0]

	// 解析消息体 ("body" 字段)
	body, ok := msg.Values["body"].(string)
	if !ok {
		// TODO: Handle malformed message - maybe move to a dead-letter stream?
		// For now, we continue without acknowledging.
		r.logger.Errorf("received message %s with invalid body format from stream %s, group %s: %v", msg.ID, topic, r.consumerGroup, msg.Values["body"])
		return nil, fmt.Errorf("received message %s with invalid body format", msg.ID)
	}

	// 解析时间戳 ("timestamp" 字段)
	timestampVal, ok := msg.Values["timestamp"].(string)
	if !ok {
		// 如果时间戳字段不存在或格式错误，记录错误并使用当前时间
		r.logger.Errorf("Received message %s without valid timestamp field from stream %s, group %s", msg.ID, topic, r.consumerGroup)
		// 继续处理消息，Timestamp 使用当前时间
		return &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     topic,
			Timestamp: time.Now(),
		}, nil
	}

	nanoTimestamp, err := strconv.ParseInt(timestampVal, 10, 64)
	if err != nil {
		// 如果时间戳解析失败，记录错误并使用当前时间
		r.logger.Errorf("Failed to parse timestamp for message %s from stream %s, group %s: %v", msg.ID, topic, r.consumerGroup, err)
		// 继续处理消息，Timestamp 使用当前时间
		return &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     topic,
			Timestamp: time.Now(),
		}, nil
	}

	return &types.Message{
		ID:        msg.ID, // Streams 消息有唯一的 ID
		Body:      []byte(body),
		Topic:     topic,
		Timestamp: time.Unix(0, nanoTimestamp), // 从纳秒时间戳转换
	}, nil
}

// Ack 实现MessageQueue接口的Ack方法
// 使用 XAck 命令确认 Streams 消息
func (r *redisMQ) Ack(ctx context.Context, stream, group string, id string) error {
	_, err := r.client.XAck(ctx, stream, group, id).Result()
	if err != nil {
		return fmt.Errorf("failed to XAck message %s in stream %s, group %s: %w", id, stream, group, err)
	}
	return nil
}

// Nack 实现MessageQueue接口的Nack方法
// 在Streams实现中，Nack表示不确认消息，使其保留在Pending状态。
// 这实际上不需要执行任何Redis命令，因为消息读取后默认就在Pending状态，除非被XACK。
func (r *redisMQ) Nack(ctx context.Context, stream, group string, id string) error {
	// No operation needed for Streams Nack, as messages remain in PEL until ACKed
	// You might add logging here if needed.
	r.logger.Infof("Nacked message %s in stream %s, group %s. It remains in PEL.", id, stream, group)
	return nil
}

// ReclaimPendingMessages 可以用于处理消费者崩溃后未 ACK 的消息
// 通常需要在消费者启动时调用，或者通过一个独立的监控进程来处理
func (r *redisMQ) ReclaimPendingMessages(ctx context.Context, stream string, idleTime time.Duration, count int) ([]*types.Message, error) {
	// 查找处于 Pending 상태且超时未处理的消息
	// 使用 XPENDING 命令获取 Pending 消息的详细信息
	// 注意：XPENDING 不直接返回消息体，需要通过 XCLAIM 或 XAUTOCLAIM 获取

	// 转移这些 Pending 消息的所有权给当前消费者实例，并获取消息体
	claimedStreams, _, err := r.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    r.consumerGroup,
		MinIdle:  idleTime, // 使用 MinIdle 字段
		Start:    "0-0",    // 从 Streams 的开始位置查找
		Count:    int64(count),
		Consumer: "consumer-" + r.client.ClientID(ctx).String(), // 将消息转移给当前消费者
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to XAutoClaim messages for stream %s, group %s: %w", stream, r.consumerGroup, err)
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
			// r.logger.Errorf("Failed to XAck malformed reclaimed message %s in stream %s, group %s: %v", msg.ID, stream, r.consumerGroup, ackErr)
			// }
			r.logger.Errorf("Received reclaimed message %s with invalid body format from stream %s, group %s: %v", msg.ID, stream, r.consumerGroup, msg.Values["body"])
			continue
		}

		// 解析时间戳 ("timestamp" 字段)
		timestampVal, ok := msg.Values["timestamp"].(string)
		if !ok {
			r.logger.Errorf("Received reclaimed message %s without valid timestamp field from stream %s, group %s", msg.ID, stream, r.consumerGroup)
			messages = append(messages, &types.Message{
				ID:        msg.ID,
				Body:      []byte(body),
				Topic:     stream,
				Timestamp: time.Now(),
			})
			continue
		}

		nanoTimestamp, err := strconv.ParseInt(timestampVal, 10, 64)
		if err != nil {
			r.logger.Errorf("Failed to parse timestamp for reclaimed message %s from stream %s, group %s: %v", msg.ID, stream, r.consumerGroup, err)
			messages = append(messages, &types.Message{
				ID:        msg.ID,
				Body:      []byte(body),
				Topic:     stream,
				Timestamp: time.Now(),
			})
			continue
		}

		messages = append(messages, &types.Message{
			ID:        msg.ID,
			Body:      []byte(body),
			Topic:     stream,
			Timestamp: time.Unix(0, nanoTimestamp),
		})
	}

	// Note: The reclaimed messages are now owned by the current consumer instance.
	// The consumer needs to process these messages and then call XAck (which is the Ack method).

	return messages, nil
}
