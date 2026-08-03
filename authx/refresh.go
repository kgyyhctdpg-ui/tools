package authx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultRefreshPrefix = "authx:refresh:"
	defaultRefreshTTL    = 30 * 24 * time.Hour
)

// RefreshToken is metadata for a refresh token. Token contains the raw token
// only when returned by Issue, Verify or Rotate.
type RefreshToken struct {
	Token     string            `json:"-"`
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Values    map[string]string `json:"values,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at,omitempty"`
}

// RefreshTokenManager stores hashed refresh tokens in a Store.
type RefreshTokenManager struct {
	store      Store
	prefix     string
	ttl        time.Duration
	tokenBytes int
	now        func() time.Time
}

// RefreshTokenOption configures a refresh token manager.
type RefreshTokenOption func(*RefreshTokenManager)

func WithRefreshPrefix(prefix string) RefreshTokenOption {
	return func(manager *RefreshTokenManager) {
		if strings.TrimSpace(prefix) != "" {
			manager.prefix = prefix
		}
	}
}

func WithRefreshTTL(ttl time.Duration) RefreshTokenOption {
	return func(manager *RefreshTokenManager) {
		if ttl > 0 {
			manager.ttl = ttl
		}
	}
}

func WithRefreshTokenBytes(tokenBytes int) RefreshTokenOption {
	return func(manager *RefreshTokenManager) {
		if tokenBytes > 0 {
			manager.tokenBytes = tokenBytes
		}
	}
}

func WithRefreshNow(now func() time.Time) RefreshTokenOption {
	return func(manager *RefreshTokenManager) {
		if now != nil {
			manager.now = now
		}
	}
}

func NewRefreshTokenManager(store Store, opts ...RefreshTokenOption) *RefreshTokenManager {
	manager := &RefreshTokenManager{
		store:      store,
		prefix:     defaultRefreshPrefix,
		ttl:        defaultRefreshTTL,
		tokenBytes: defaultTokenBytes,
		now:        time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(manager)
		}
	}
	return manager
}

// Issue creates and stores a refresh token for userID.
func (manager *RefreshTokenManager) Issue(ctx context.Context, userID string, values map[string]string) (*RefreshToken, error) {
	if manager == nil || manager.store == nil {
		return nil, ErrNilStore
	}
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("authx: user id is empty")
	}
	token, err := randomToken(manager.tokenBytes)
	if err != nil {
		return nil, err
	}
	id := HashAPIKey(token)
	now := manager.now()
	refreshToken := &RefreshToken{
		Token:     token,
		ID:        id,
		UserID:    userID,
		Values:    cloneStringMap(values),
		CreatedAt: now,
		ExpiresAt: now.Add(manager.ttl),
	}
	if err := manager.storeRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}
	return refreshToken, nil
}

// Verify validates a raw refresh token and returns its metadata.
func (manager *RefreshTokenManager) Verify(ctx context.Context, token string) (*RefreshToken, error) {
	if manager == nil || manager.store == nil {
		return nil, ErrNilStore
	}
	key, err := manager.key(token)
	if err != nil {
		return nil, err
	}
	data, err := manager.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var refreshToken RefreshToken
	if err := json.Unmarshal([]byte(data), &refreshToken); err != nil {
		return nil, fmt.Errorf("authx: decode refresh token: %w", err)
	}
	if !refreshToken.ExpiresAt.IsZero() && !manager.now().Before(refreshToken.ExpiresAt) {
		_ = manager.store.Del(ctx, key)
		return nil, ErrInvalidToken
	}
	refreshToken.Token = strings.TrimSpace(token)
	return &refreshToken, nil
}

// Rotate revokes token and issues a replacement for the same user.
func (manager *RefreshTokenManager) Rotate(ctx context.Context, token string, values map[string]string) (*RefreshToken, error) {
	current, err := manager.Verify(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := manager.Revoke(ctx, token); err != nil {
		return nil, err
	}
	if values == nil {
		values = current.Values
	}
	return manager.Issue(ctx, current.UserID, values)
}

// Revoke deletes a refresh token. Missing tokens are ignored by cache stores.
func (manager *RefreshTokenManager) Revoke(ctx context.Context, token string) error {
	if manager == nil || manager.store == nil {
		return ErrNilStore
	}
	key, err := manager.key(token)
	if err != nil {
		return err
	}
	return manager.store.Del(ctx, key)
}

func (manager *RefreshTokenManager) storeRefreshToken(ctx context.Context, refreshToken *RefreshToken) error {
	data, err := json.Marshal(refreshToken)
	if err != nil {
		return err
	}
	return manager.store.Set(ctx, manager.prefix+refreshToken.ID, string(data), manager.ttl)
}

func (manager *RefreshTokenManager) key(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrInvalidToken
	}
	return manager.prefix + HashAPIKey(token), nil
}
