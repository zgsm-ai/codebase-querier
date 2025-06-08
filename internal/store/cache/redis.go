package cache

import (
	"context"
	"encoding/json"
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

func (r *RedisStore[T]) Get(ctx context.Context, key string) (*T, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
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
	return r.client.Del(ctx, key).Err()
}
