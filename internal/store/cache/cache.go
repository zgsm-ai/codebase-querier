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

	// AddVersion adds a version to the key's version set with optional expiration
	AddVersion(ctx context.Context, key string, version int64, expiration time.Duration) error

	// GetVersions gets all versions for a key, sorted by version number (desc)
	// Only returns non-expired versions
	GetVersions(ctx context.Context, key string) ([]int64, error)

	// GetLatestVersion gets the latest version for a key
	// Returns ErrVersionNotFound if no versions exist
	GetLatestVersion(ctx context.Context, key string) (int64, error)

	// CleanExpiredVersions removes expired versions for a key
	CleanExpiredVersions(ctx context.Context, key string) error
}

// ErrKeyNotFound is returned when a key is not found in the cache
type ErrKeyNotFound struct {
	Key string
}

func (e *ErrKeyNotFound) Error() string {
	return "key not found: " + e.Key
}

// ErrVersionNotFound is returned when no versions exist for a key
type ErrVersionNotFound struct {
	Key string
}

func (e *ErrVersionNotFound) Error() string {
	return "no versions found for key: " + e.Key
}
