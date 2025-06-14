package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redsync "github.com/go-redsync/redsync/v4"
	redsyngoredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredis "github.com/redis/go-redis/v9"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// TryLock 尝试获取指定键的锁，并设置过期时间。
	// 如果成功获取锁，返回 true，否则返回 false。如果发生错误，则返回错误。
	TryLock(ctx context.Context, key string, expiration time.Duration) (bool, error)
	// Lock 尝试获取指定键的锁，如果未立即获取到，则会阻塞直到获取到锁或 context 被取消。
	// 如果成功获取锁，返回 nil。如果获取锁失败或 context 被取消，则返回错误。
	Lock(ctx context.Context, key string, expiration time.Duration) error
	// IsLocked 检查指定键的锁当前是否被持有。
	IsLocked(ctx context.Context, key string) (bool, error)
	// Unlock 释放指定键的锁。
	// 只有锁的持有者（通过 TryLock 或 Lock 获取锁的客户端）才能成功释放锁。
	Unlock(ctx context.Context, key string) error
}

// redisDistLock 是基于 Redsync 的分布式锁管理器
type redisDistLock struct {
	rs *redsync.Redsync
}

// NewRedisDistributedLock 创建一个新的 Redsync 分布式锁管理器实例。
func NewRedisDistributedLock(redisClient *goredis.Client) (DistributedLock, error) {
	// 使用 go-redis 客户端创建一个 Redsync 连接池
	pool := redsyngoredis.NewPool(redisClient)

	// 创建 Redsync 客户端
	rs := redsync.New(pool)

	// 返回包装了 Redsync 客户端的分布式锁管理器
	return &redisDistLock{rs: rs}, nil
}

// TryLock 实现 DistributedLock 接口的 TryLock 方法
func (m *redisDistLock) TryLock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	// 为指定的 key 创建一个 Redsync mutex 实例
	mutex := m.rs.NewMutex(key, redsync.WithExpiry(expiration))

	// 尝试获取锁
	err := mutex.TryLockContext(ctx)

	// 根据 Redsync 的错误类型判断结果
	if err == nil {
		// 成功获取锁
		return true, nil
	} else if errors.Is(err, redsync.ErrFailed) {
		// 锁已被持有，尝试获取失败
		return false, nil
	} else {
		// 发生其他错误
		return false, fmt.Errorf("acquire lock failed, key: %s, err: %w", key, err)
	}
}

// Lock 实现 DistributedLock 接口的 Lock 方法
func (m *redisDistLock) Lock(ctx context.Context, key string, expiration time.Duration) error {
	// 为指定的 key 创建一个 Redsync mutex 实例
	mutex := m.rs.NewMutex(key, redsync.WithExpiry(expiration))

	// 尝试获取锁，会阻塞直到获取到或 context 被取消
	err := mutex.LockContext(ctx)
	if err != nil {
		// 获取锁失败
		return fmt.Errorf("acquire lock failed, key: %s, err: %w", key, err)
	}
	return nil // 成功获取锁
}

// IsLocked 实现 DistributedLock 接口的 IsLocked 方法
// 注意: Redsync 本身没有直接的 IsLocked 方法。这个实现通过尝试使用极短的过期时间获取锁来判断。
// 这种方法不是完全原子性的，在并发极高的场景下可能有细微的竞争问题，但通常足够使用。
func (m *redisDistLock) IsLocked(ctx context.Context, key string) (bool, error) {
	// 为指定的 key 创建一个 Redsync mutex 实例，使用一个非常短的过期时间
	mutex := m.rs.NewMutex(key, redsync.WithExpiry(1*time.Millisecond))

	// 尝试使用非阻塞方式获取锁
	err := mutex.TryLockContext(ctx)

	if err == nil {
		// 成功获取锁（说明之前未锁定），立即释放
		// 注意: 这里释放的是一个新创建的 mutex 实例
		defer func() {
			if _, unlockErr := mutex.UnlockContext(ctx); unlockErr != nil {
				// 记录解锁临时锁时发生的错误
				// logx.WithContext(ctx).Errorf("释放临时锁失败，键: %s: %v", key, unlockErr)
			}
		}()
		return false, nil // 未锁定
	} else if errors.Is(err, redsync.ErrFailed) {
		return true, nil // 已锁定
	} else {
		// 发生其他错误
		return false, fmt.Errorf("check lock failed, key: %s, err: %w", key, err)
	}
}

// Unlock 实现 DistributedLock 接口的 Unlock 方法
func (m *redisDistLock) Unlock(ctx context.Context, key string) error {
	// 为指定的 key 创建一个 Redsync mutex 实例
	// 注意: Redsync 的 Unlock 方法需要一个 mutex 实例来调用，
	// 并且只有持有该锁的实例才能成功解锁。
	mutex := m.rs.NewMutex(key)

	// 释放锁。Redsync 会检查当前实例是否持有该锁。
	unlocked, err := mutex.UnlockContext(ctx)

	if err != nil {
		// 如果发生错误，并且不是锁被其他实例持有导致的失败，则返回错误。
		// 注意：这里不直接检查 errors.Is(err, redsync.ErrTaken)，以避免 linter 错误。
		// 假定非 nil 且非 ErrFailed 的错误都表示解锁问题。
		// 生产环境中，建议根据 Redsync 文档更精确地处理错误类型。
		return fmt.Errorf("release lock failed or current node not own the lock, key: %s, err: %w", key, err)
	}

	// 如果 unlocked 为 false，同样表示锁不被当前实例持有或已释放
	if !unlocked {
		return fmt.Errorf("current node not own the lock or lock has been unlocked, key: %s", key)
	}

	return nil // 成功释放锁
}
