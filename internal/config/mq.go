package config

import (
	"time"
)

// MessageQueueConf 消息队列配置
type MessageQueueConf struct {
	Type   string       // 类型: redis, kafka, pulsar
	Common CommonConfig // 通用配置
	Topic  string
	// 特定实现的配置
	Redis *RedisConfig
}

// CommonConfig 通用配置
type CommonConfig struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxRetries     int
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host        string
	Password    string
	DB          int
	PoolSize    int
	MinIdleConn int
}
