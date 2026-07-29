package utils

import (
	"net/http"
	"testing"
)

func TestSubstr(t *testing.T) {
	if got := Substr("你好world", 7); got != "你好w" {
		t.Fatalf("Substr = %q, want 你好w", got)
	}
	if got := Substr("你好", 100); got != "你好" {
		t.Fatalf("Substr long = %q, want 你好", got)
	}
	if got := Substr("abc", 0); got != "" {
		t.Fatalf("Substr zero = %q, want empty", got)
	}
}

func TestGetClientIp(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	if got := GetClientIp(req); got != "127.0.0.1" {
		t.Fatalf("GetClientIp remote = %q, want 127.0.0.1", got)
	}

	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	if got := GetClientIp(req); got != "10.0.0.1" {
		t.Fatalf("GetClientIp forwarded = %q, want 10.0.0.1", got)
	}
}
