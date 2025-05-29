package mq

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const messageQueueRedis = "redis"

// New  根据配置创建消息队列实例
func New(ctx context.Context, cfg config.MessageQueueConf) (MessageQueue, error) {
	switch cfg.Type {
	case messageQueueRedis:
		if cfg.Redis.Host == types.EmptyString {
			return nil, errors.New("redis config is required for redis type")
		}
		return newRedisMQ(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported message queue type: %s", cfg.Type)
	}
}
