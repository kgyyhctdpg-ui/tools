package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryOption configures a MemoryStore.
type MemoryOption func(*MemoryStore)

// WithNow overrides the clock used by MemoryStore. It is mainly useful in
// tests.
func WithNow(now func() time.Time) MemoryOption {
	return func(store *MemoryStore) {
		if now != nil {
			store.now = now
		}
	}
}

type memoryItem struct {
	value     string
	expiresAt time.Time
}

// MemoryStore is a concurrency-safe in-process Store implementation.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryItem
	now   func() time.Time
}

// NewMemoryStore creates an empty in-memory cache store.
func NewMemoryStore(opts ...MemoryOption) *MemoryStore {
	store := &MemoryStore{
		items: make(map[string]memoryItem),
		now:   time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(store)
		}
	}
	return store
}

// Get returns a value by key.
func (store *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	if err := checkKey(key); err != nil {
		return "", err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	item, ok := store.items[key]
	if !ok || item.expired(store.now()) {
		delete(store.items, key)
		return "", ErrNotFound
	}
	return item.value, nil
}

// Set stores a value. ttl <= 0 means the value does not expire.
func (store *MemoryStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := checkKey(key); err != nil {
		return err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	store.items[key] = memoryItem{
		value:     value,
		expiresAt: expiresAt(store.now(), ttl),
	}
	return nil
}

// SetNX stores a value only when the key does not exist or has expired.
func (store *MemoryStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	if err := checkKey(key); err != nil {
		return false, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if item, ok := store.items[key]; ok && !item.expired(store.now()) {
		return false, nil
	}
	store.items[key] = memoryItem{
		value:     value,
		expiresAt: expiresAt(store.now(), ttl),
	}
	return true, nil
}

// Del removes keys. Missing keys are ignored.
func (store *MemoryStore) Del(ctx context.Context, keys ...string) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := checkKey(key); err != nil {
			return err
		}
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	for _, key := range keys {
		delete(store.items, key)
	}
	return nil
}

// Exists reports whether a key exists and has not expired.
func (store *MemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	if err := checkContext(ctx); err != nil {
		return false, err
	}
	if err := checkKey(key); err != nil {
		return false, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	item, ok := store.items[key]
	if !ok || item.expired(store.now()) {
		delete(store.items, key)
		return false, nil
	}
	return true, nil
}

// TTL returns the remaining lifetime of a key. Keys without expiration return
// NoExpiration.
func (store *MemoryStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := checkContext(ctx); err != nil {
		return 0, err
	}
	if err := checkKey(key); err != nil {
		return 0, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	now := store.now()
	item, ok := store.items[key]
	if !ok || item.expired(now) {
		delete(store.items, key)
		return 0, ErrNotFound
	}
	if item.expiresAt.IsZero() {
		return NoExpiration, nil
	}
	return item.expiresAt.Sub(now), nil
}

// Len returns the number of non-expired entries in the store.
func (store *MemoryStore) Len() int {
	store.Cleanup()

	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.items)
}

// Cleanup removes expired entries.
func (store *MemoryStore) Cleanup() {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := store.now()
	for key, item := range store.items {
		if item.expired(now) {
			delete(store.items, key)
		}
	}
}

func expiresAt(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

func (item memoryItem) expired(now time.Time) bool {
	return !item.expiresAt.IsZero() && !now.Before(item.expiresAt)
}
