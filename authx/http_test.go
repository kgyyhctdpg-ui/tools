package authx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTMiddlewareAndPermissions(t *testing.T) {
	claims := NewClaims("u1", time.Hour, WithRoles("admin"), WithPermissions("user:*"))
	token, err := SignJWT("secret", claims)
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userID, ok := UserIDFromContext(request.Context())
		if !ok || userID != "u1" {
			t.Fatalf("unexpected user id: %q ok=%v", userID, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	protected := JWTMiddleware("secret")(RequireAnyRole("admin")(RequireAllPermissions("user:read")(next)))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	protected.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status: %d", response.Code)
	}

	userToken, err := SignJWT("secret", NewClaims("u2", time.Hour, WithRoles("user")))
	if err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+userToken)
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("wrong role status: %d", response.Code)
	}
}

func TestClaimsPermissionMatching(t *testing.T) {
	claims := &Claims{Permissions: []string{"user:*", "order:read", "report-*:*"}}
	tests := []struct {
		name       string
		permission string
		want       bool
	}{
		{name: "resource wildcard", permission: "user:delete", want: true},
		{name: "exact match", permission: "order:read", want: true},
		{name: "path wildcard", permission: "report-sales:export", want: true},
		{name: "missing", permission: "order:write", want: false},
		{name: "empty", permission: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := claims.HasPermission(tt.permission); got != tt.want {
				t.Fatalf("HasPermission(%q)=%v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	protected := APIKeyMiddleware(func(_ context.Context, key string) (bool, error) {
		return key == "secret-key", nil
	})(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		apiKey, ok := APIKeyFromContext(request.Context())
		if !ok || apiKey != "secret-key" {
			t.Fatalf("unexpected api key: %q ok=%v", apiKey, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-API-Key", "secret-key")
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-API-Key", "wrong")
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status: %d", response.Code)
	}
}

func TestRefreshTokenManager(t *testing.T) {
	store := newTestStore()
	manager := NewRefreshTokenManager(store, WithRefreshTTL(time.Minute), WithRefreshTokenBytes(8))
	token, err := manager.Issue(context.Background(), "u1", map[string]string{"device": "ios"})
	if err != nil {
		t.Fatal(err)
	}
	if token.Token == "" || token.ID == "" {
		t.Fatalf("unexpected refresh token: %#v", token)
	}
	verified, err := manager.Verify(context.Background(), token.Token)
	if err != nil {
		t.Fatal(err)
	}
	if verified.UserID != "u1" || verified.Values["device"] != "ios" {
		t.Fatalf("unexpected verified token: %#v", verified)
	}
	rotated, err := manager.Rotate(context.Background(), token.Token, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Token == token.Token {
		t.Fatal("rotated token should be new")
	}
	if _, err := manager.Verify(context.Background(), token.Token); err == nil {
		t.Fatal("old token should be revoked")
	}
	if err := manager.Revoke(context.Background(), rotated.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Verify(context.Background(), rotated.Token); err == nil {
		t.Fatal("revoked token should not verify")
	}
}
