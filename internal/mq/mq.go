package mq

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Close 关闭连接
	Close() error
	// Produce Publish 发布消息到指定主题
	Produce(ctx context.Context, topic string, message []byte, opts types.ProduceOptions) error
	// Consume 订阅主题，返回消息通道和错误
	Consume(ctx context.Context, topic string, opts types.ConsumeOptions) (*types.Message, error)
	// CreateTopic 创建主题（如果不支持自动创建）
	CreateTopic(ctx context.Context, topic string, opts types.TopicOptions) error
	// DeleteTopic 删除主题
	DeleteTopic(ctx context.Context, topic string) error
}
