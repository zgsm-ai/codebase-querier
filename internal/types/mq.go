package types

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

type ProduceOptions struct {
}

type ConsumeOptions struct {
}

type TopicOptions struct {
}
