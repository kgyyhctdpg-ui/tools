package captcha

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrStoreKeyNotFound = errors.New("captcha: cache key not found")

type memoryItem struct {
	value     string
	expiresAt time.Time
}

// MemoryStore 是一个轻量内存 Store，适合测试、示例和单进程本地场景。
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// NewMemoryStore 创建内存缓存实现。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]memoryItem),
	}
}

func (s *MemoryStore) SetNX(ctx context.Context, key, value string, expiration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if item, ok := s.items[key]; ok && !item.expired(time.Now()) {
		return nil
	}

	expiresAt := time.Time{}
	if expiration > 0 {
		expiresAt = time.Now().Add(expiration)
	}
	s.items[key] = memoryItem{
		value:     value,
		expiresAt: expiresAt,
	}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return "", ErrStoreKeyNotFound
	}

	if item.expired(time.Now()) {
		s.mu.Lock()
		delete(s.items, key)
		s.mu.Unlock()
		return "", ErrStoreKeyNotFound
	}

	return item.value, nil
}

func (item memoryItem) expired(now time.Time) bool {
	return !item.expiresAt.IsZero() && now.After(item.expiresAt)
}
