package authx

import (
	"context"
	"errors"
	"time"
)

var errTestStoreNotFound = errors.New("test store: key not found")

type testStore struct {
	items map[string]string
}

func newTestStore() *testStore {
	return &testStore{items: map[string]string{}}
}

func (store *testStore) Get(_ context.Context, key string) (string, error) {
	value, ok := store.items[key]
	if !ok {
		return "", errTestStoreNotFound
	}
	return value, nil
}

func (store *testStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	store.items[key] = value
	return nil
}

func (store *testStore) Del(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(store.items, key)
	}
	return nil
}
