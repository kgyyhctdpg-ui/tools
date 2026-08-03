package oauthx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthCodeURLWithPKCE(t *testing.T) {
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	config := Config{
		ClientID:    "client",
		RedirectURL: "https://app.example.com/callback",
		AuthURL:     "https://provider.example.com/oauth/authorize",
		Scopes:      []string{"openid", "email"},
	}
	authURL, err := config.AuthCodeURL(AuthURLOptions{
		State:         "state",
		CodeChallenge: CodeChallengeS256(verifier),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"response_type=code",
		"client_id=client",
		"redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback",
		"scope=openid+email",
		"state=state",
		"code_challenge_method=S256",
	} {
		if !strings.Contains(authURL, expected) {
			t.Fatalf("auth URL missing %q: %s", expected, authURL)
		}
	}
}

func TestStateStoreVerifiesOnce(t *testing.T) {
	store := newTestStateStore()
	if err := StoreState(context.Background(), store, "state", time.Minute); err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyState(context.Background(), store, "state")
	if err != nil || !ok {
		t.Fatalf("state should verify once, ok=%v err=%v", ok, err)
	}
	ok, err = VerifyState(context.Background(), store, "state")
	if err != nil || ok {
		t.Fatalf("state should not verify twice, ok=%v err=%v", ok, err)
	}
}

func TestExchangeCodeAndUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected token method: %s", request.Method)
			}
			if request.FormValue("grant_type") != "authorization_code" ||
				request.FormValue("client_id") != "client" ||
				request.FormValue("client_secret") != "secret" ||
				request.FormValue("code") != "code" ||
				request.FormValue("code_verifier") != "verifier" {
				t.Fatalf("unexpected token form: %v", request.Form)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"access_token":  "access",
				"token_type":    "Bearer",
				"refresh_token": "refresh",
				"expires_in":    3600,
			})
		case "/userinfo":
			if request.Header.Get("Authorization") != "Bearer access" {
				t.Fatalf("unexpected authorization: %q", request.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"id":         "u1",
				"email":      "u1@example.com",
				"avatar_url": "https://cdn.example.com/u1.png",
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	config := Config{
		Provider:     "test",
		ClientID:     "client",
		ClientSecret: "secret",
		TokenURL:     server.URL + "/token",
		UserInfoURL:  server.URL + "/userinfo",
	}
	token, err := config.ExchangeCode(context.Background(), "code", WithCodeVerifier("verifier"))
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" || token.ExpiresIn != 3600 || token.ExpiresAt.IsZero() {
		t.Fatalf("unexpected token response: %#v", token)
	}
	raw, err := config.UserInfo(context.Background(), token.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	user := MapUser(config.Provider, raw)
	if user.Provider != "test" || user.ID != "u1" || user.Email != "u1@example.com" || user.Avatar == "" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.FormValue("grant_type") != "refresh_token" ||
			request.FormValue("refresh_token") != "refresh" ||
			request.FormValue("client_id") != "client" {
			t.Fatalf("unexpected refresh form: %v", request.Form)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"access_token": "access2",
			"expires_in":   json.Number("7200"),
		})
	}))
	defer server.Close()

	config := Config{
		ClientID:     "client",
		ClientSecret: "secret",
		TokenURL:     server.URL,
	}
	token, err := config.RefreshToken(context.Background(), "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access2" || token.ExpiresIn != 7200 {
		t.Fatalf("unexpected refresh token response: %#v", token)
	}
}

func TestJSONTokenStyleWithProviderFieldMap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		if contentType := request.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Fatalf("unexpected content type: %s", contentType)
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["grantType"] != "authorization_code" ||
			body["clientId"] != "client" ||
			body["clientSecret"] != "secret" ||
			body["code"] != "code" {
			t.Fatalf("unexpected json body: %#v", body)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"accessToken":  "access",
			"refreshToken": "refresh",
			"expireIn":     3600,
		})
	}))
	defer server.Close()

	config := DingTalk("client", "secret", "https://app.example.com/callback")
	config.TokenURL = server.URL
	token, err := config.ExchangeCode(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" || token.ExpiresIn != 3600 {
		t.Fatalf("unexpected token: %#v", token)
	}
}

func TestQueryUserInfoAndProviderConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("access_token") != "access" ||
			request.URL.Query().Get("openid") != "openid" {
			t.Fatalf("unexpected query: %s", request.URL.RawQuery)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"openid":   "openid",
			"nickname": "gavin",
		})
	}))
	defer server.Close()

	config, err := ProviderConfig(ProviderWechatOpen, "client", "secret", "https://app.example.com/callback")
	if err != nil {
		t.Fatal(err)
	}
	authURL, err := config.AuthCodeURL(AuthURLOptions{State: "state"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(authURL, "appid=client") {
		t.Fatalf("wechat auth URL should use appid: %s", authURL)
	}
	config.UserInfoURL = server.URL
	raw, err := config.UserInfoWithOptions(context.Background(), "access", UserInfoOptions{
		Params: mapValues("openid", "openid"),
	})
	if err != nil {
		t.Fatal(err)
	}
	user := MapUser(config.Provider, raw)
	if user.OpenID != "openid" || user.Nickname != "gavin" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func mapValues(key, value string) url.Values {
	return url.Values{key: []string{value}}
}
