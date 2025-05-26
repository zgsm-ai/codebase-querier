package redis

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

func New(c config.Config) (*redis.Redis, error) {
	redisCli, err := redis.NewRedis(c.RedisConf)
	return redisCli, err
}
