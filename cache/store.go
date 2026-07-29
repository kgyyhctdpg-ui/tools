// Package cache defines a small cache abstraction and in-memory store.
package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a key does not exist or has expired.
	ErrNotFound = errors.New("cache: key not found")

	// ErrEmptyKey is returned when an operation receives an empty key.
	ErrEmptyKey = errors.New("cache: key is empty")
)

// NoExpiration is returned by Store.TTL for keys that never expire.
const NoExpiration time.Duration = -1

// Store is the common cache interface used by tools that need simple key-value
// storage without binding to a concrete backend.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// Loader loads a cache value on miss.
type Loader func(ctx context.Context) (string, error)

// Remember returns a cached value when present, otherwise it calls loader,
// stores the returned value, and returns it.
func Remember(ctx context.Context, store Store, key string, ttl time.Duration, loader Loader) (string, error) {
	if store == nil {
		return "", errors.New("cache: store is nil")
	}
	if loader == nil {
		return "", errors.New("cache: loader is nil")
	}

	value, err := store.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return "", err
	}

	value, err = loader(ctx)
	if err != nil {
		return "", err
	}
	if err := store.Set(ctx, key, value, ttl); err != nil {
		return "", err
	}
	return value, nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func checkKey(key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	return nil
}
