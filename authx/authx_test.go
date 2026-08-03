package authx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("secret", 4)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "secret") {
		t.Fatal("password should verify")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("wrong password should not verify")
	}
}

func TestParseBearerToken(t *testing.T) {
	token, ok := ParseBearerToken("Bearer abc.def")
	if !ok || token != "abc.def" {
		t.Fatalf("unexpected bearer token: token=%q ok=%v", token, ok)
	}
	if _, ok := ParseBearerToken("Basic abc"); ok {
		t.Fatal("basic auth should not parse as bearer")
	}
}

func TestJWTSignAndVerify(t *testing.T) {
	claims := NewClaims("u1", time.Hour, WithIssuer("tools"), WithRoles("admin"), WithPermissions("user:create"))
	token, err := SignJWT("secret", claims)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := VerifyJWT(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.UserID != "u1" || parsed.Issuer != "tools" || len(parsed.Roles) != 1 || parsed.Roles[0] != "admin" {
		t.Fatalf("unexpected claims: %#v", parsed)
	}
	if _, err := VerifyJWT(token, "other"); err == nil {
		t.Fatal("wrong secret should fail")
	}
}

func TestAPIKeyHelpers(t *testing.T) {
	apiKey, err := GenerateAPIKey("sk", 8)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(apiKey, "sk_") {
		t.Fatalf("unexpected API key prefix: %q", apiKey)
	}
	digest := HashAPIKey(apiKey)
	if !VerifyAPIKey(apiKey, digest) || VerifyAPIKey(apiKey+"x", digest) {
		t.Fatal("API key verification failed")
	}
	if masked := MaskToken("abcdefghijkl", 3); masked != "abc******jkl" {
		t.Fatalf("unexpected mask: %q", masked)
	}
}

func TestSessionManager(t *testing.T) {
	store := newTestStore()
	manager := NewSessionManager(store, WithSessionTTL(time.Minute), WithSessionTokenBytes(8))
	session, err := manager.Create(context.Background(), "u1", map[string]string{"ip": "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := manager.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.UserID != "u1" || loaded.Values["ip"] != "127.0.0.1" {
		t.Fatalf("unexpected session: %#v", loaded)
	}
	if _, err := manager.Refresh(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Destroy(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Get(context.Background(), session.ID); !errors.Is(err, errTestStoreNotFound) {
		t.Fatalf("expected session to be removed, got %v", err)
	}
}
