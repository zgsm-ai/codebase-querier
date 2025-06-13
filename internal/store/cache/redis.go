package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store interface using Redis with JSON serialization
type RedisStore[T any] struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis cache store
func NewRedisStore[T any](client *redis.Client) Store[T] {
	return &RedisStore[T]{
		client: client,
	}
}

// versionMember 存储版本信息
type versionMember struct {
	Version   int64     `json:"v"` // 版本号
	ExpiresAt time.Time `json:"e"` // 过期时间
}

func (r *RedisStore[T]) Get(ctx context.Context, key string) (*T, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &ErrKeyNotFound{Key: key}
		}
		return nil, err
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *RedisStore[T]) Set(ctx context.Context, key string, value T, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisStore[T]) Delete(ctx context.Context, key string) error {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore[T]) AddVersion(ctx context.Context, key string, version int64, expiration time.Duration) error {
	// 计算过期时间
	expiresAt := time.Now().Add(expiration)

	// 创建版本信息
	member := versionMember{
		Version:   version,
		ExpiresAt: expiresAt,
	}

	// 序列化版本信息
	memberData, err := json.Marshal(member)
	if err != nil {
		return err
	}

	// 使用版本号作为 score，确保排序正确
	return r.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(version),
		Member: string(memberData),
	}).Err()
}

func (r *RedisStore[T]) GetVersions(ctx context.Context, key string) ([]int64, error) {
	// 获取所有版本
	versions, err := r.client.ZRevRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, &ErrVersionNotFound{Key: key}
	}

	now := time.Now()
	var result []int64
	for _, v := range versions {
		var member versionMember
		if err := json.Unmarshal([]byte(v), &member); err != nil {
			return nil, fmt.Errorf("invalid version format: %s", v)
		}

		// 过滤掉过期的版本
		if member.ExpiresAt.After(now) {
			result = append(result, member.Version)
		}
	}

	if len(result) == 0 {
		return nil, &ErrVersionNotFound{Key: key}
	}
	return result, nil
}

func (r *RedisStore[T]) GetLatestVersion(ctx context.Context, key string) (int64, error) {
	versions, err := r.GetVersions(ctx, key)
	if err != nil {
		return 0, err
	}
	return versions[0], nil
}

// CleanExpiredVersions 清理过期的版本
// 建议定期调用此方法，比如每小时执行一次
func (r *RedisStore[T]) CleanExpiredVersions(ctx context.Context, key string) error {
	versions, err := r.client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}

	now := time.Now()
	pipe := r.client.Pipeline()
	for _, v := range versions {
		var member versionMember
		if err := json.Unmarshal([]byte(v), &member); err != nil {
			continue
		}

		// 删除过期的版本
		if member.ExpiresAt.Before(now) {
			pipe.ZRem(ctx, key, v)
		}
	}
	_, err = pipe.Exec(ctx)
	return err
}
