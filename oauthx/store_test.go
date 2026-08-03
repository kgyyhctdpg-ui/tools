package oauthx

import (
	"context"
	"time"
)

type testStateStore struct {
	items map[string]string
}

func newTestStateStore() *testStateStore {
	return &testStateStore{items: map[string]string{}}
}

func (store *testStateStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	store.items[key] = value
	return nil
}

func (store *testStateStore) Del(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(store.items, key)
	}
	return nil
}

func (store *testStateStore) Exists(_ context.Context, key string) (bool, error) {
	_, ok := store.items[key]
	return ok, nil
}
