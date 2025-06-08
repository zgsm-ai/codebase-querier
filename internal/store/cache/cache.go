package cache

import (
	"context"
	"time"
)

// Store defines a generic interface for cache operations
type Store[T any] interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (*T, error)

	// Set stores a value in cache with optional expiration
	Set(ctx context.Context, key string, value T, expiration time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error
}

// ErrKeyNotFound is returned when a key is not found in the cache
type ErrKeyNotFound struct {
	Key string
}

func (e *ErrKeyNotFound) Error() string {
	return "key not found: " + e.Key
}
