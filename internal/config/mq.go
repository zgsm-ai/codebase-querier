package config

import "time"

// MessageQueueConf 消息队列配置
type MessageQueueConf struct {
	Type           string        // 类型: redis, kafka, pulsar
	ConnectTimeout time.Duration `json:",default=10s"`
	ReadTimeout    time.Duration `json:",default=10s"`
	WriteTimeout   time.Duration `json:",default=10s"`
	// 特定实现的配置
	Redis RedisConfig
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host        string
	Password    string `json:",optional"`
	DB          int    `json:",default=0"`
	PoolSize    int    `json:",default=10"`
	MinIdleConn int    `json:",default=10"`
}
