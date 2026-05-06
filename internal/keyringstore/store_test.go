package keyringstore

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/99designs/keyring"
	appconfig "github.com/YewFence/yew-key/internal/config"
	"github.com/YewFence/yew-key/internal/provider"
)

func TestCleanProfileRemovesCachedSecretsAndIndex(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewStoreBackend(ring)
	profile := appconfig.Profile{Name: "work", Provider: "infisical"}

	if _, err := store.SyncProfile(profile, []provider.Secret{
		{RemoteKey: "DATABASE_URL", EnvName: "DATABASE_URL", Value: "postgres://local"},
		{RemoteKey: "OPENAI_API_KEY", EnvName: "OPENAI_API_KEY", Value: "sk-test"},
	}); err != nil {
		t.Fatalf("sync profile: %v", err)
	}
	if err := ring.Set(keyring.Item{Key: envItemKey(profile.Name, "LEGACY_TOKEN"), Data: []byte("legacy")}); err != nil {
		t.Fatalf("seed legacy key: %v", err)
	}

	removed, err := store.CleanProfile(profile)
	if err != nil {
		t.Fatalf("clean profile: %v", err)
	}
	if removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	for _, itemKey := range []string{
		envItemKey(profile.Name, "DATABASE_URL"),
		envItemKey(profile.Name, "OPENAI_API_KEY"),
		envItemKey(profile.Name, "LEGACY_TOKEN"),
		indexItemKey(profile.Name),
	} {
		if _, err := ring.Get(itemKey); !errors.Is(err, keyring.ErrKeyNotFound) {
			t.Fatalf("key %s still exists or returned unexpected error: %v", itemKey, err)
		}
	}
}

type memoryKeyring struct {
	items map[string]keyring.Item
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{items: map[string]keyring.Item{}}
}

func (ring *memoryKeyring) Get(key string) (keyring.Item, error) {
	item, ok := ring.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (ring *memoryKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	item, ok := ring.items[key]
	if !ok {
		return keyring.Metadata{}, keyring.ErrKeyNotFound
	}
	item.Data = nil
	return keyring.Metadata{Item: &item, ModificationTime: time.Now().UTC()}, nil
}

func (ring *memoryKeyring) Set(item keyring.Item) error {
	data := make([]byte, len(item.Data))
	copy(data, item.Data)
	item.Data = data
	ring.items[item.Key] = item
	return nil
}

func (ring *memoryKeyring) Remove(key string) error {
	if _, ok := ring.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(ring.items, key)
	return nil
}

func (ring *memoryKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(ring.items))
	for key := range ring.items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}
