package config

import "time"

// MessageQueueConf 消息队列配置
type MessageQueueConf struct {
	Type                 string        // 类型: redis, kafka, pulsar
	DeadLetterTopic      string        `json:",default=codebase_indexer:mq:dead_letter"`
	SingleMsgQueueMaxLen int64         `json:",default=1000000"` // one million
	DeadLetterInterval   time.Duration `json:",default=1m"`
}
