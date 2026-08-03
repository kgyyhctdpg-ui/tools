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
	now   func() time.Time
}

// NewMemoryStore 创建内存缓存实现。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		items: make(map[string]memoryItem),
		now:   time.Now,
	}
}

func (s *MemoryStore) SetNX(ctx context.Context, key, value string, expiration time.Duration) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if s == nil {
		return ErrMissingStore
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.items[key]; ok && !item.expired(s.now()) {
		return nil
	}
	s.items[key] = memoryItem{
		value:     value,
		expiresAt: expiresAt(s.now(), expiration),
	}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
	if s == nil {
		return "", ErrStoreKeyNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok || item.expired(s.now()) {
		delete(s.items, key)
		return "", ErrStoreKeyNotFound
	}
	return item.value, nil
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

func expiresAt(now time.Time, ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

func (item memoryItem) expired(now time.Time) bool {
	return !item.expiresAt.IsZero() && !now.Before(item.expiresAt)
}
