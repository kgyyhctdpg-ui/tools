package captcha

import (
	"context"
	"errors"
	"time"

	"github.com/scoming-dev/tools/cache"
)

var ErrStoreKeyNotFound = errors.New("captcha: cache key not found")

// MemoryStore 是一个轻量内存 Store，适合测试、示例和单进程本地场景。
type MemoryStore struct {
	store *cache.MemoryStore
}

// NewMemoryStore 创建内存缓存实现。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: cache.NewMemoryStore(),
	}
}

func (s *MemoryStore) SetNX(ctx context.Context, key, value string, expiration time.Duration) error {
	_, err := s.store.SetNX(ctx, key, value, expiration)
	return err
}

func (s *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	value, err := s.store.Get(ctx, key)
	if errors.Is(err, cache.ErrNotFound) {
		return "", ErrStoreKeyNotFound
	}
	return value, err
}
