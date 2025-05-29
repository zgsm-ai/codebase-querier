package mq

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	redisstore "github.com/zgsm-ai/codebase-indexer/internal/store/redis"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

// redisMQ Redis消息队列实现（基于原生redis/go-redis/v9）
type redisMQ struct {
	logger logx.Logger
	client *redis.Client // 原生Redis客户端
}

// newRedisMQ 创建Redis消息队列实例
func newRedisMQ(ctx context.Context, c config.MessageQueueConf) (MessageQueue, error) {
	// 构建原生Redis客户端配置
	// 测试连接
	client, err := redisstore.NewRedisClient(c)
	if err != nil {
		return nil, err
	}
	return &redisMQ{
		client: client,
		logger: logx.WithContext(ctx),
	}, nil
}

func (r *redisMQ) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *redisMQ) CreateTopic(ctx context.Context, topic string, opts types.TopicOptions) error {
	// Redis中列表自动创建，无需显式创建
	return nil
}

func (r *redisMQ) DeleteTopic(ctx context.Context, topic string) error {
	_, err := r.client.Del(ctx, topic).Result()
	return err
}

// Produce 实现MessageQueue接口的Publish方法
func (r *redisMQ) Produce(ctx context.Context, topic string, message []byte, opts types.ProduceOptions) error {

	// 使用RPush命令将消息推入列表右侧
	_, err := r.client.RPush(ctx, topic, message).Result()
	if err != nil {
		return err
	}
	return nil
}

// Consume  实现MessageQueue接口的Subscribe方法
func (r *redisMQ) Consume(ctx context.Context, topic string, opts types.ConsumeOptions) (*types.Message, error) {
	// 使用BLPop命令实现阻塞读取，设置超时时间
	vals, err := r.client.BLPop(ctx, opts.ReadTimeout, topic).Result()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, redis.Nil) {
			// 超时或无消息，继续循环
			return nil, errs.ReadTimeout
		}
		return nil, err
	}

	// 解析消息（vals[0]=key, vals[1]=value）
	if len(vals) < 2 {
		return nil, fmt.Errorf("invalid message: %v", vals)
	}
	return &types.Message{
		Body:      []byte(vals[1]),
		Topic:     topic,
		Timestamp: time.Now(),
	}, nil

}
