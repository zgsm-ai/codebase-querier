package mq

import "time"

// Message 消息结构
type Message struct {
	ID        string            // 消息ID
	Body      []byte            // 消息内容
	Topic     string            // 主题
	Timestamp time.Time         // 时间戳
	Metadata  map[string]string // 元数据
}

// Status 队列状态
type Status struct {
	Connected    bool          // 是否连接
	Latency      time.Duration // 延迟
	MessageCount int64         // 消息数量
	Error        error         // 错误信息
}

// PublishOption 发布选项接口
type PublishOption interface {
	ApplyPublish(*PublishOptions)
}

// SubscribeOption 订阅选项接口
type SubscribeOption interface {
	ApplySubscribe(*SubscribeOptions)
}

// TopicOption 主题选项接口
type TopicOption interface {
	applyTopic(*topicOptions)
}

// PublishOptions 发布选项的实现
type PublishOptions struct {
	// 通用选项
	timeout      time.Duration
	headers      map[string]string
	retryCount   int
	retryBackoff time.Duration

	// Redis特定选项
	RedisExpiry time.Duration

	// Kafka特定选项
	kafkaKey       []byte
	kafkaPartition int32

	// Pulsar特定选项
	pulsarSequenceID int64
}

// SubscribeOptions 订阅选项的实现
type SubscribeOptions struct {
	// 通用选项
	groupID    string
	bufferSize int
	autoAck    bool

	// Redis特定选项
	RedisBlock time.Duration

	// Kafka特定选项
	kafkaOffset int64

	// Pulsar特定选项
	pulsarSubscriptionType string
}

// 主题选项的实现
type topicOptions struct {
	// 通用选项
	replicationFactor int

	// Kafka特定选项
	kafkaPartitions int

	// Pulsar特定选项
	pulsarRetentionPolicy string
}
