package config

// MessageQueueConf 消息队列配置
type MessageQueueConf struct {
	Type          string // 类型: redis, kafka, pulsar
	ConsumerGroup string `json:",default=codebase_indexer"` // 消费者组名称（用于Streams）
}
