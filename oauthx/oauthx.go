// Package oauthx provides small OAuth2 helpers with PKCE and state storage.
package oauthx

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultStatePrefix = "oauthx:state:"
	defaultStateTTL    = 10 * time.Minute
	defaultTokenBytes  = 32
)

var (
	ErrInvalidState = errors.New("oauthx: invalid state")
	ErrNilStore     = errors.New("oauthx: store is nil")
)

// StateStore is the minimal cache capability required for OAuth state replay
// protection. cache.Store from this repository satisfies it.
type StateStore interface {
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// TokenRequestStyle controls how token requests are encoded.
type TokenRequestStyle string

const (
	TokenRequestForm  TokenRequestStyle = "form"
	TokenRequestJSON  TokenRequestStyle = "json"
	TokenRequestQuery TokenRequestStyle = "query"
)

// UserInfoAuthStyle controls where the access token is sent for user info.
type UserInfoAuthStyle string

const (
	UserInfoAuthBearer UserInfoAuthStyle = "bearer"
	UserInfoAuthQuery  UserInfoAuthStyle = "query"
)

// Config describes a generic OAuth2 provider.
type Config struct {
	Provider           string
	ClientID           string
	ClientSecret       string
	RedirectURL        string
	AuthURL            string
	TokenURL           string
	RevokeURL          string
	UserInfoURL        string
	Scopes             []string
	ScopeSeparator     string
	AuthClientIDParam  string
	TokenMethod        string
	TokenStyle         TokenRequestStyle
	TokenFieldMap      map[string]string
	UserInfoAuthStyle  UserInfoAuthStyle
	UserInfoTokenParam string
	HTTPClient         *http.Client
}

// AuthURLOptions controls authorization URL generation.
type AuthURLOptions struct {
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	AccessType          string
	Prompt              string
	Params              url.Values
}

// AuthCodeURL returns an OAuth2 authorization URL.
func (config Config) AuthCodeURL(options AuthURLOptions) (string, error) {
	if strings.TrimSpace(config.AuthURL) == "" {
		return "", errors.New("oauthx: auth URL is empty")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return "", errors.New("oauthx: client id is empty")
	}
	parsed, err := url.Parse(config.AuthURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("response_type", "code")
	clientIDParam := config.AuthClientIDParam
	if clientIDParam == "" {
		clientIDParam = "client_id"
	}
	query.Set(clientIDParam, config.ClientID)
	if config.RedirectURL != "" {
		query.Set("redirect_uri", config.RedirectURL)
	}
	if len(config.Scopes) > 0 {
		query.Set("scope", strings.Join(config.Scopes, config.scopeSeparator()))
	}
	if options.State != "" {
		query.Set("state", options.State)
	}
	if options.CodeChallenge != "" {
		query.Set("code_challenge", options.CodeChallenge)
		method := options.CodeChallengeMethod
		if method == "" {
			method = "S256"
		}
		query.Set("code_challenge_method", method)
	}
	if options.AccessType != "" {
		query.Set("access_type", options.AccessType)
	}
	if options.Prompt != "" {
		query.Set("prompt", options.Prompt)
	}
	for key, values := range options.Params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// TokenResponse is a normalized OAuth2 token response.
type TokenResponse struct {
	AccessToken  string         `json:"access_token"`
	TokenType    string         `json:"token_type,omitempty"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	Scope        string         `json:"scope,omitempty"`
	ExpiresIn    int64          `json:"expires_in,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at,omitempty"`
	Raw          map[string]any `json:"raw,omitempty"`
}

type exchangeConfig struct {
	codeVerifier string
	params       url.Values
}

// ExchangeOption configures code exchange.
type ExchangeOption func(*exchangeConfig)

func WithCodeVerifier(verifier string) ExchangeOption {
	return func(config *exchangeConfig) { config.codeVerifier = verifier }
}

func WithTokenParam(key, value string) ExchangeOption {
	return func(config *exchangeConfig) {
		if config.params == nil {
			config.params = url.Values{}
		}
		config.params.Add(key, value)
	}
}

// ExchangeCode exchanges an authorization code for tokens.
func (config Config) ExchangeCode(ctx context.Context, code string, opts ...ExchangeOption) (*TokenResponse, error) {
	if strings.TrimSpace(config.TokenURL) == "" {
		return nil, errors.New("oauthx: token URL is empty")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, errors.New("oauthx: client id is empty")
	}
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("oauthx: authorization code is empty")
	}
	exchange := exchangeConfig{params: url.Values{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&exchange)
		}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", config.ClientID)
	if config.ClientSecret != "" {
		form.Set("client_secret", config.ClientSecret)
	}
	if config.RedirectURL != "" {
		form.Set("redirect_uri", config.RedirectURL)
	}
	if exchange.codeVerifier != "" {
		form.Set("code_verifier", exchange.codeVerifier)
	}
	for key, values := range exchange.params {
		for _, value := range values {
			form.Add(key, value)
		}
	}
	return config.requestToken(ctx, form)
}

// OAuthUser is a normalized user info shape.
type OAuthUser struct {
	Provider string         `json:"provider,omitempty"`
	ID       string         `json:"id,omitempty"`
	OpenID   string         `json:"openid,omitempty"`
	UnionID  string         `json:"unionid,omitempty"`
	Email    string         `json:"email,omitempty"`
	Username string         `json:"username,omitempty"`
	Nickname string         `json:"nickname,omitempty"`
	Avatar   string         `json:"avatar,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}

// UserInfo fetches raw user info with a bearer access token.
func (config Config) UserInfo(ctx context.Context, accessToken string) (map[string]any, error) {
	return config.UserInfoWithOptions(ctx, accessToken, UserInfoOptions{})
}

// UserInfoOptions configures a user info request.
type UserInfoOptions struct {
	Params     url.Values
	Headers    http.Header
	AuthStyle  UserInfoAuthStyle
	TokenParam string
}

// UserInfoWithOptions fetches raw user info with custom token placement and
// query parameters.
func (config Config) UserInfoWithOptions(ctx context.Context, accessToken string, options UserInfoOptions) (map[string]any, error) {
	if strings.TrimSpace(config.UserInfoURL) == "" {
		return nil, errors.New("oauthx: user info URL is empty")
	}
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("oauthx: access token is empty")
	}
	parsed, err := url.Parse(config.UserInfoURL)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	for key, values := range options.Params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	authStyle := options.AuthStyle
	if authStyle == "" {
		authStyle = config.UserInfoAuthStyle
	}
	if authStyle == "" {
		authStyle = UserInfoAuthBearer
	}
	tokenParam := options.TokenParam
	if tokenParam == "" {
		tokenParam = config.UserInfoTokenParam
	}
	if tokenParam == "" {
		tokenParam = "access_token"
	}
	if authStyle == UserInfoAuthQuery {
		query.Set(tokenParam, accessToken)
	}
	parsed.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	if authStyle == UserInfoAuthBearer {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	request.Header.Set("Accept", "application/json")
	for key, values := range options.Headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	response, err := httpClient(config.HTTPClient).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("oauthx: user info endpoint status %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// MapUser maps a raw user info response into OAuthUser using common keys.
func MapUser(provider string, raw map[string]any) OAuthUser {
	return OAuthUser{
		Provider: provider,
		ID:       firstString(raw, "id", "sub", "uid", "userid", "user_id"),
		OpenID:   firstString(raw, "openid", "open_id"),
		UnionID:  firstString(raw, "unionid", "union_id"),
		Email:    firstString(raw, "email"),
		Username: firstString(raw, "login", "username", "name"),
		Nickname: firstString(raw, "nickname", "display_name", "name"),
		Avatar:   firstString(raw, "avatar_url", "picture", "avatar"),
		Raw:      raw,
	}
}

// GenerateState returns a URL-safe random state value.
func GenerateState(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = defaultTokenBytes
	}
	return randomToken(byteLen)
}

func StoreState(ctx context.Context, store StateStore, state string, ttl time.Duration) error {
	if store == nil {
		return ErrNilStore
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return ErrInvalidState
	}
	if ttl <= 0 {
		ttl = defaultStateTTL
	}
	return store.Set(ctx, StateKey("", state), "1", ttl)
}

// VerifyState verifies a state value once and deletes it to prevent replay.
func VerifyState(ctx context.Context, store StateStore, state string) (bool, error) {
	if store == nil {
		return false, ErrNilStore
	}
	state = strings.TrimSpace(state)
	if state == "" {
		return false, nil
	}
	key := StateKey("", state)
	exists, err := store.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	if err := store.Del(ctx, key); err != nil {
		return false, err
	}
	return true, nil
}

func StateKey(prefix, state string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultStatePrefix
	}
	return prefix + state
}

// GenerateCodeVerifier returns a PKCE code verifier.
func GenerateCodeVerifier() (string, error) {
	return randomToken(defaultTokenBytes)
}

// CodeChallengeS256 returns the PKCE S256 code challenge for verifier.
func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func httpClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return http.DefaultClient
}

func int64Number(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		number, _ := typed.Int64()
		return number
	default:
		return 0
	}
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func randomToken(byteLen int) (string, error) {
	data := make([]byte, byteLen)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
