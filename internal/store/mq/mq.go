package mq

import (
	"context"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Produce Publish 发布消息到指定主题
	Produce(ctx context.Context, topic string, message []byte, opts types.ProduceOptions) error
	// Consume 订阅主题，返回消息通道和错误
	Consume(ctx context.Context, topic string, opts types.ConsumeOptions) (*types.Message, error)
	// CreateTopic 创建主题（如果不支持自动创建）
	CreateTopic(ctx context.Context, topic string, opts types.TopicOptions) error
	// DeleteTopic 删除主题
	DeleteTopic(ctx context.Context, topic string) error
	// Ack 确认消息已被成功处理
	Ack(ctx context.Context, topic, consumerGroup string, msgId string) error
	// Nack 消息处理失败，不确认消息，使其保留在Pending状态以便重试或其他消费者认领
	Nack(ctx context.Context, topic, consumerGroup string, msgId string) error
}

// Status 队列状态
type Status struct {
	Connected    bool          // 是否连接
	Latency      time.Duration // 延迟
	MessageCount int64         // 消息数量
	Error        error         // 错误信息
}

type ProduceOptions struct {
}

type ConsumeOptions struct {
	ReadTimeout time.Duration
}

type TopicOptions struct {
}
