package keyringstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"time"

	"github.com/99designs/keyring"
	appconfig "github.com/YewFence/yew-key/internal/config"
	"github.com/YewFence/yew-key/internal/provider"
)

type Variable struct {
	EnvName   string    `json:"env_name"`
	RemoteKey string    `json:"remote_key"`
	Provider  string    `json:"provider"`
	Version   string    `json:"version,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Index struct {
	Profile   string     `json:"profile"`
	Variables []Variable `json:"variables"`
}

type Store interface {
	SyncProfile(profile appconfig.Profile, secrets []provider.Secret) (Index, error)
	Index(profile appconfig.Profile) (Index, error)
	ReadValue(profile appconfig.Profile, envName string) (string, error)
}

type Opener interface {
	Open(serviceName string) (Store, error)
}

type DefaultOpener struct{}

func (DefaultOpener) Open(serviceName string) (Store, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:              serviceName,
		KeychainName:             serviceName,
		KeychainTrustApplication: true,
		KeychainSynchronizable:   false,
		LibSecretCollectionName:  "default",
		KeyCtlScope:              "user",
		PassPrefix:               serviceName,
		WinCredPrefix:            serviceName,
		KeychainPasswordFunc:     keyring.TerminalPrompt,
		FilePasswordFunc:         keyring.TerminalPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("open keyring service %q: %w", serviceName, err)
	}
	return StoreBackend{ring: ring}, nil
}

type StoreBackend struct {
	ring keyring.Keyring
	now  func() time.Time
}

func NewStoreBackend(ring keyring.Keyring) StoreBackend {
	return StoreBackend{ring: ring}
}

func (store StoreBackend) SyncProfile(profile appconfig.Profile, secrets []provider.Secret) (Index, error) {
	now := store.timeNow()
	index := Index{
		Profile:   profile.Name,
		Variables: make([]Variable, 0, len(secrets)),
	}
	for _, secret := range secrets {
		itemKey := envItemKey(profile.Name, secret.EnvName)
		if err := store.ring.Set(keyring.Item{
			Key:         itemKey,
			Data:        []byte(secret.Value),
			Label:       secret.EnvName,
			Description: fmt.Sprintf("yewk profile %s env %s", profile.Name, secret.EnvName),
		}); err != nil {
			return Index{}, fmt.Errorf("write secret %s for profile %s: %w", secret.EnvName, profile.Name, err)
		}
		index.Variables = append(index.Variables, Variable{
			EnvName:   secret.EnvName,
			RemoteKey: secret.RemoteKey,
			Provider:  profile.Provider,
			Version:   secret.Version,
			UpdatedAt: now,
		})
	}
	sort.Slice(index.Variables, func(i, j int) bool {
		return index.Variables[i].EnvName < index.Variables[j].EnvName
	})
	data, err := json.Marshal(index)
	if err != nil {
		return Index{}, err
	}
	if err := store.ring.Set(keyring.Item{
		Key:         indexItemKey(profile.Name),
		Data:        data,
		Label:       "yewk index",
		Description: fmt.Sprintf("yewk profile %s metadata index", profile.Name),
	}); err != nil {
		return Index{}, fmt.Errorf("write metadata index for profile %s: %w", profile.Name, err)
	}
	return index, nil
}

func (store StoreBackend) Index(profile appconfig.Profile) (Index, error) {
	item, err := store.ring.Get(indexItemKey(profile.Name))
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return Index{Profile: profile.Name}, nil
	}
	if err != nil {
		return Index{}, fmt.Errorf("read metadata index for profile %s: %w", profile.Name, err)
	}
	var index Index
	if err := json.Unmarshal(item.Data, &index); err != nil {
		return Index{}, fmt.Errorf("parse metadata index for profile %s: %w", profile.Name, err)
	}
	return index, nil
}

func (store StoreBackend) ReadValue(profile appconfig.Profile, envName string) (string, error) {
	item, err := store.ring.Get(envItemKey(profile.Name, envName))
	if err != nil {
		return "", fmt.Errorf("read env %s for profile %s: %w", envName, profile.Name, err)
	}
	return string(item.Data), nil
}

func (store StoreBackend) timeNow() time.Time {
	if store.now != nil {
		return store.now()
	}
	return time.Now().UTC()
}

func indexItemKey(profile string) string {
	return path.Join("profiles", profile, "meta", "index")
}

func envItemKey(profile string, envName string) string {
	return path.Join("profiles", profile, "env", envName)
}
