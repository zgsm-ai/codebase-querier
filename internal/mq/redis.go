package mq

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zgsm-ai/codebase-indexer/internal/config"
)

// redisMQ Redis消息队列实现（基于原生redis/go-redis/v9）
type redisMQ struct {
	logger    logx.Logger
	commonCfg config.CommonConfig
	redisCfg  config.RedisConfig // 使用自定义配置结构
	client    *redis.Client      // 原生Redis客户端
}

// newRedisMQ 创建Redis消息队列实例
func newRedisMQ(commonCfg config.CommonConfig, redisCfg config.RedisConfig) (MessageQueue, error) {
	// 构建原生Redis客户端配置
	rdbCfg := redis.Options{
		Addr:         redisCfg.Host,
		Password:     redisCfg.Password,
		DB:           redisCfg.DB,
		PoolSize:     redisCfg.PoolSize,
		MinIdleConns: redisCfg.MinIdleConn,
		DialTimeout:  commonCfg.ConnectTimeout,
		ReadTimeout:  commonCfg.ReadTimeout,
		WriteTimeout: commonCfg.WriteTimeout,
	}

	client := redis.NewClient(&rdbCfg)

	// 测试连接
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &redisMQ{
		commonCfg: commonCfg,
		redisCfg:  redisCfg,
		client:    client,
		logger:    logx.WithContext(context.Background()),
	}, nil
}

func (r *redisMQ) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *redisMQ) CreateTopic(ctx context.Context, topic string, opts ...TopicOption) error {
	// Redis中列表自动创建，无需显式创建
	return nil
}

func (r *redisMQ) DeleteTopic(ctx context.Context, topic string) error {
	_, err := r.client.Del(ctx, topic).Result()
	return err
}

func (r *redisMQ) Status(ctx context.Context) (Status, error) {
	pong, err := r.client.Ping(ctx).Result()
	if err != nil {
		return Status{}, err
	}

	// 计算延迟
	start := time.Now()
	_, err = r.client.Ping(ctx).Result()
	latency := time.Since(start)

	return Status{
		Connected:    pong == "PONG",
		Latency:      latency,
		MessageCount: 0,
		Error:        nil,
	}, nil
}

// Publish 实现MessageQueue接口的Publish方法
func (r *redisMQ) Publish(ctx context.Context, topic string, message []byte, opts ...PublishOption) error {
	ctx, cancel := context.WithTimeout(ctx, r.commonCfg.WriteTimeout)
	defer cancel()

	// 使用RPush命令将消息推入列表右侧
	_, err := r.client.RPush(ctx, topic, message).Result()
	if err != nil {
		return err
	}
	return nil
}

// Subscribe 实现MessageQueue接口的Subscribe方法
func (r *redisMQ) Subscribe(ctx context.Context, topic string, opts ...SubscribeOption) (<-chan Message, error) {
	msgCh := make(chan Message)

	go func() {
		defer close(msgCh)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 使用BLPop命令实现阻塞读取，设置超时时间
				ctxTimeout, cancel := context.WithTimeout(ctx, r.commonCfg.ReadTimeout)
				vals, err := r.client.BLPop(ctxTimeout, r.commonCfg.ReadTimeout, topic).Result()
				cancel()

				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, redis.Nil) {
						// 超时或无消息，继续循环
						continue
					}
					r.logger.Errorf("BLPop from redis error: %v", err)
				}

				// 解析消息（vals[0]=key, vals[1]=value）
				if len(vals) >= 2 {
					msg := Message{
						Body:      []byte(vals[1]),
						Topic:     topic,
						Timestamp: time.Now(),
					}
					select {
					case msgCh <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return msgCh, nil
}
