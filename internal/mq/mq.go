package mq

import "context"

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Close 关闭连接
	Close() error
	// Publish 发布消息到指定主题
	Publish(ctx context.Context, topic string, message []byte, opts ...PublishOption) error
	// Subscribe 订阅主题，返回消息通道和错误
	Subscribe(ctx context.Context, topic string, opts ...SubscribeOption) (<-chan Message, error)
	// CreateTopic 创建主题（如果不支持自动创建）
	CreateTopic(ctx context.Context, topic string, opts ...TopicOption) error
	// DeleteTopic 删除主题
	DeleteTopic(ctx context.Context, topic string) error
	// Status 获取队列状态
	Status(ctx context.Context) (Status, error)
}
