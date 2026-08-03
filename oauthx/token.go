package oauthx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RefreshToken exchanges a refresh token for a new access token.
func (config Config) RefreshToken(ctx context.Context, refreshToken string, opts ...ExchangeOption) (*TokenResponse, error) {
	if strings.TrimSpace(config.TokenURL) == "" {
		return nil, errors.New("oauthx: token URL is empty")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, errors.New("oauthx: client id is empty")
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("oauthx: refresh token is empty")
	}
	exchange := exchangeConfig{params: url.Values{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&exchange)
		}
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", config.ClientID)
	if config.ClientSecret != "" {
		form.Set("client_secret", config.ClientSecret)
	}
	for key, values := range exchange.params {
		for _, value := range values {
			form.Add(key, value)
		}
	}
	return config.requestToken(ctx, form)
}

// RevokeToken revokes an access or refresh token when the provider supports a
// revocation endpoint.
func (config Config) RevokeToken(ctx context.Context, token string, tokenTypeHint string) error {
	if strings.TrimSpace(config.RevokeURL) == "" {
		return errors.New("oauthx: revoke URL is empty")
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("oauthx: token is empty")
	}
	form := url.Values{}
	form.Set("token", token)
	if tokenTypeHint != "" {
		form.Set("token_type_hint", tokenTypeHint)
	}
	if config.ClientID != "" {
		form.Set("client_id", config.ClientID)
	}
	if config.ClientSecret != "" {
		form.Set("client_secret", config.ClientSecret)
	}
	_, err := config.doTokenRequest(ctx, config.RevokeURL, form)
	return err
}

// ParseTokenResponse decodes a provider token response into TokenResponse.
func ParseTokenResponse(data []byte) (*TokenResponse, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	token := &TokenResponse{Raw: raw}
	token.AccessToken = firstString(raw, "access_token", "accessToken")
	token.TokenType = firstString(raw, "token_type", "tokenType")
	token.RefreshToken = firstString(raw, "refresh_token", "refreshToken")
	token.Scope = firstString(raw, "scope")
	token.ExpiresIn = firstInt64(raw, "expires_in", "expiresIn", "expire_in", "expireIn")
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if token.AccessToken == "" {
		return nil, errors.New("oauthx: token response missing access_token")
	}
	return token, nil
}

func (config Config) requestToken(ctx context.Context, form url.Values) (*TokenResponse, error) {
	data, err := config.doTokenRequest(ctx, config.TokenURL, form)
	if err != nil {
		return nil, err
	}
	return ParseTokenResponse(data)
}

func (config Config) doTokenRequest(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	mapped := config.mapTokenFields(form)
	method := config.tokenMethod()
	style := config.tokenStyle()
	requestURL := endpoint
	var body io.Reader
	contentType := ""

	switch style {
	case TokenRequestJSON:
		data, err := json.Marshal(valuesMap(mapped))
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	case TokenRequestQuery:
		if method == "" {
			method = http.MethodGet
		}
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}
		query := parsed.Query()
		for key, values := range mapped {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		parsed.RawQuery = query.Encode()
		requestURL = parsed.String()
	default:
		body = strings.NewReader(mapped.Encode())
		contentType = "application/x-www-form-urlencoded"
	}
	if method == "" {
		method = http.MethodPost
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
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
		return nil, fmt.Errorf("oauthx: token endpoint status %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (config Config) mapTokenFields(form url.Values) url.Values {
	mapped := url.Values{}
	for key, values := range form {
		target := key
		if config.TokenFieldMap != nil && config.TokenFieldMap[key] != "" {
			target = config.TokenFieldMap[key]
		}
		for _, value := range values {
			mapped.Add(target, value)
		}
	}
	return mapped
}

func (config Config) tokenMethod() string {
	if config.TokenMethod != "" {
		return config.TokenMethod
	}
	if config.TokenStyle == TokenRequestQuery {
		return http.MethodGet
	}
	return http.MethodPost
}

func (config Config) tokenStyle() TokenRequestStyle {
	if config.TokenStyle != "" {
		return config.TokenStyle
	}
	return TokenRequestForm
}

func (config Config) scopeSeparator() string {
	if config.ScopeSeparator != "" {
		return config.ScopeSeparator
	}
	return " "
}

func valuesMap(values url.Values) map[string]any {
	result := make(map[string]any, len(values))
	for key, items := range values {
		if len(items) == 1 {
			result[key] = items[0]
			continue
		}
		result[key] = append([]string(nil), items...)
	}
	return result
}

func firstInt64(raw map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			return int64Number(value)
		}
	}
	return 0
}
