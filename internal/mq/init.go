package mq

import (
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

const messageQueueRedis = "redis"

// New  根据配置创建消息队列实例
func New(cfg config.MessageQueueConf) (MessageQueue, error) {
	switch cfg.Type {
	case messageQueueRedis:
		if cfg.Redis == nil {
			return nil, errors.New("redis config is required for redis type")
		}
		return newRedisMQ(cfg.Common, *cfg.Redis)
	default:
		return nil, fmt.Errorf("unsupported message queue type: %s", cfg.Type)
	}
}
