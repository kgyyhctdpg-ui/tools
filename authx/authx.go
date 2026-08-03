// Package authx provides common authentication helpers for services.
package authx

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultSessionPrefix = "authx:session:"
	defaultSessionTTL    = 24 * time.Hour
	defaultTokenBytes    = 32
)

var (
	ErrEmptySecret  = errors.New("authx: secret is empty")
	ErrInvalidToken = errors.New("authx: invalid token")
	ErrNilStore     = errors.New("authx: store is nil")
)

// Store is the minimal cache capability required by SessionManager and
// RefreshTokenManager. cache.Store from this repository satisfies it.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string, cost ...int) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", errors.New("authx: password is empty")
	}
	bcryptCost := bcrypt.DefaultCost
	if len(cost) > 0 && cost[0] > 0 {
		bcryptCost = cost[0]
	}
	data, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// VerifyPassword reports whether password matches a bcrypt hash.
func VerifyPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ParseBearerToken extracts a bearer token from an Authorization header.
func ParseBearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if len(header) < len("Bearer ")+1 {
		return "", false
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// Claims is the default JWT claims shape used by authx.
type Claims struct {
	UserID      string   `json:"uid,omitempty"`
	Username    string   `json:"username,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// ClaimsOption configures new claims.
type ClaimsOption func(*Claims)

func WithIssuer(issuer string) ClaimsOption {
	return func(claims *Claims) { claims.Issuer = issuer }
}

func WithAudience(audience ...string) ClaimsOption {
	return func(claims *Claims) { claims.Audience = jwt.ClaimStrings(audience) }
}

func WithUsername(username string) ClaimsOption {
	return func(claims *Claims) { claims.Username = username }
}

func WithRoles(roles ...string) ClaimsOption {
	return func(claims *Claims) { claims.Roles = append([]string(nil), roles...) }
}

func WithPermissions(permissions ...string) ClaimsOption {
	return func(claims *Claims) { claims.Permissions = append([]string(nil), permissions...) }
}

func WithJWTID(id string) ClaimsOption {
	return func(claims *Claims) { claims.ID = id }
}

func WithIssuedAt(now time.Time) ClaimsOption {
	return func(claims *Claims) { claims.IssuedAt = jwt.NewNumericDate(now) }
}

// NewClaims creates standard JWT claims for userID and ttl.
func NewClaims(userID string, ttl time.Duration, opts ...ClaimsOption) Claims {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	if ttl > 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&claims)
		}
	}
	return claims
}

// SignJWT signs claims with HS256.
func SignJWT(secret string, claims Claims) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrEmptySecret
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyJWT verifies an HS256 token and returns its claims.
func VerifyJWT(tokenString, secret string) (*Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrEmptySecret
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("authx: unexpected signing method %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if token == nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GenerateAPIKey creates a URL-safe API key. prefix is optional.
func GenerateAPIKey(prefix string, byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = defaultTokenBytes
	}
	token, err := randomToken(byteLen)
	if err != nil {
		return "", err
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "_")
	if prefix == "" {
		return token, nil
	}
	return prefix + "_" + token, nil
}

// HashAPIKey returns the SHA-256 digest of an API key.
func HashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

// VerifyAPIKey compares an API key with a stored SHA-256 digest.
func VerifyAPIKey(apiKey, digest string) bool {
	actual := HashAPIKey(apiKey)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(digest)) == 1
}

// MaskToken masks a token for logs or UI display.
func MaskToken(token string, visible int) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if visible <= 0 {
		visible = 4
	}
	if len(token) <= visible*2 {
		return strings.Repeat("*", len(token))
	}
	return token[:visible] + strings.Repeat("*", len(token)-visible*2) + token[len(token)-visible:]
}

// Session is the value stored by SessionManager.
type Session struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Values    map[string]string `json:"values,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at,omitempty"`
}

// SessionManager stores sessions in a Store.
type SessionManager struct {
	store      Store
	prefix     string
	ttl        time.Duration
	tokenBytes int
	now        func() time.Time
}

// SessionOption configures a session manager.
type SessionOption func(*SessionManager)

func WithSessionPrefix(prefix string) SessionOption {
	return func(manager *SessionManager) {
		if strings.TrimSpace(prefix) != "" {
			manager.prefix = prefix
		}
	}
}

func WithSessionTTL(ttl time.Duration) SessionOption {
	return func(manager *SessionManager) {
		if ttl > 0 {
			manager.ttl = ttl
		}
	}
}

func WithSessionTokenBytes(tokenBytes int) SessionOption {
	return func(manager *SessionManager) {
		if tokenBytes > 0 {
			manager.tokenBytes = tokenBytes
		}
	}
}

func WithSessionNow(now func() time.Time) SessionOption {
	return func(manager *SessionManager) {
		if now != nil {
			manager.now = now
		}
	}
}

func NewSessionManager(store Store, opts ...SessionOption) *SessionManager {
	manager := &SessionManager{
		store:      store,
		prefix:     defaultSessionPrefix,
		ttl:        defaultSessionTTL,
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

func (manager *SessionManager) Create(ctx context.Context, userID string, values map[string]string) (*Session, error) {
	if manager == nil || manager.store == nil {
		return nil, ErrNilStore
	}
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("authx: user id is empty")
	}
	id, err := randomToken(manager.tokenBytes)
	if err != nil {
		return nil, err
	}
	now := manager.now()
	session := &Session{
		ID:        id,
		UserID:    userID,
		Values:    cloneStringMap(values),
		CreatedAt: now,
		ExpiresAt: now.Add(manager.ttl),
	}
	if err := manager.storeSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (manager *SessionManager) Get(ctx context.Context, id string) (*Session, error) {
	if manager == nil || manager.store == nil {
		return nil, ErrNilStore
	}
	data, err := manager.store.Get(ctx, manager.key(id))
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("authx: decode session: %w", err)
	}
	return &session, nil
}

func (manager *SessionManager) Refresh(ctx context.Context, id string) (*Session, error) {
	session, err := manager.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	session.ExpiresAt = manager.now().Add(manager.ttl)
	if err := manager.storeSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (manager *SessionManager) Destroy(ctx context.Context, id string) error {
	if manager == nil || manager.store == nil {
		return ErrNilStore
	}
	return manager.store.Del(ctx, manager.key(id))
}

func (manager *SessionManager) storeSession(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return manager.store.Set(ctx, manager.key(session.ID), string(data), manager.ttl)
}

func (manager *SessionManager) key(id string) string {
	return manager.prefix + strings.TrimSpace(id)
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func randomToken(byteLen int) (string, error) {
	data := make([]byte, byteLen)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
