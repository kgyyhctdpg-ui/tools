package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreSetGetAndTTL(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore(WithNow(func() time.Time { return now }))

	if err := store.Set(context.Background(), "name", "gavin", time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, err := store.Get(context.Background(), "name")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "gavin" {
		t.Fatalf("value = %q, want %q", value, "gavin")
	}

	ttl, err := store.TTL(context.Background(), "name")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl != time.Minute {
		t.Fatalf("ttl = %s, want %s", ttl, time.Minute)
	}
}

func TestMemoryStoreExpiration(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore(WithNow(func() time.Time { return now }))

	if err := store.Set(context.Background(), "token", "abc", time.Second); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	now = now.Add(time.Second)

	if _, err := store.Get(context.Background(), "token"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want %v", err, ErrNotFound)
	}
	exists, err := store.Exists(context.Background(), "token")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("expired key should not exist")
	}
}

func TestMemoryStoreSetNX(t *testing.T) {
	store := NewMemoryStore()

	ok, err := store.SetNX(context.Background(), "lock", "first", time.Minute)
	if err != nil {
		t.Fatalf("SetNX failed: %v", err)
	}
	if !ok {
		t.Fatal("first SetNX should store value")
	}

	ok, err = store.SetNX(context.Background(), "lock", "second", time.Minute)
	if err != nil {
		t.Fatalf("second SetNX failed: %v", err)
	}
	if ok {
		t.Fatal("second SetNX should not overwrite existing value")
	}

	value, err := store.Get(context.Background(), "lock")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "first" {
		t.Fatalf("value = %q, want %q", value, "first")
	}
}

func TestMemoryStoreDelAndNoExpiration(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Set(context.Background(), "permanent", "value", 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	ttl, err := store.TTL(context.Background(), "permanent")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl != NoExpiration {
		t.Fatalf("ttl = %s, want NoExpiration", ttl)
	}

	if err := store.Del(context.Background(), "permanent"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if _, err := store.Get(context.Background(), "permanent"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want %v", err, ErrNotFound)
	}
}

func TestRemember(t *testing.T) {
	store := NewMemoryStore()
	calls := 0
	loader := func(ctx context.Context) (string, error) {
		calls++
		return "loaded", nil
	}

	first, err := Remember(context.Background(), store, "key", time.Minute, loader)
	if err != nil {
		t.Fatalf("Remember failed: %v", err)
	}
	second, err := Remember(context.Background(), store, "key", time.Minute, loader)
	if err != nil {
		t.Fatalf("second Remember failed: %v", err)
	}
	if first != "loaded" || second != "loaded" {
		t.Fatalf("values = %q/%q, want loaded/loaded", first, second)
	}
	if calls != 1 {
		t.Fatalf("loader calls = %d, want 1", calls)
	}
}

func TestMemoryStoreContextCanceled(t *testing.T) {
	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Set(ctx, "key", "value", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set error = %v, want %v", err, context.Canceled)
	}
}
