package captcha

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreSetNXAndGet(t *testing.T) {
	store := NewMemoryStore()

	if err := store.SetNX(context.Background(), "captcha-key", "first", time.Minute); err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if err := store.SetNX(context.Background(), "captcha-key", "second", time.Minute); err != nil {
		t.Fatalf("second SetNX failed: %v", err)
	}

	value, err := store.Get(context.Background(), "captcha-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "first" {
		t.Fatalf("value = %q, want %q", value, "first")
	}
}

func TestMemoryStoreExpires(t *testing.T) {
	store := NewMemoryStore()

	if err := store.SetNX(context.Background(), "captcha-key", "value", 5*time.Millisecond); err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, err := store.Get(context.Background(), "captcha-key")
	if !errors.Is(err, ErrStoreKeyNotFound) {
		t.Fatalf("Get error = %v, want %v", err, ErrStoreKeyNotFound)
	}
}

func TestMemoryStoreHonorsContext(t *testing.T) {
	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.SetNX(ctx, "captcha-key", "value", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("SetNX error = %v, want %v", err, context.Canceled)
	}
	if _, err := store.Get(ctx, "captcha-key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get error = %v, want %v", err, context.Canceled)
	}
}
