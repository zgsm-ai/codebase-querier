package redis

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"sync"
)

var (
	redisCli *redis.Redis
	once     sync.Once
)

func New(c config.Config) (*redis.Redis, error) {
	var err error
	once.Do(func() {
		redisCli, err = redis.NewRedis(c.RedisConf)
	})
	return redisCli, err
}
