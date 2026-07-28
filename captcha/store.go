package captcha

import (
	"context"
	"errors"
	"time"
)

const DefaultTTL = 10 * time.Minute

var ErrMissingStore = errors.New("captcha: cache store is nil")

// Store 定义验证码缓存需要的最小能力，业务侧可用 Redis、内存缓存或其他存储实现。
type Store interface {
	SetNX(ctx context.Context, key, value string, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
}

// Option 用于配置验证码客户端。
type Option func(*Client)

// WithTTL 设置验证码缓存有效期。
func WithTTL(ttl time.Duration) Option {
	return func(client *Client) {
		if ttl > 0 {
			client.ttl = ttl
		}
	}
}
