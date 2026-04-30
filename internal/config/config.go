package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml/v2"
)

const relativeConfigPath = "yewk/config.toml"

type Config struct {
	Profiles []Profile `toml:"profiles,omitempty"`
}

type Profile struct {
	Name           string           `toml:"name"`
	Provider       string           `toml:"provider"`
	KeyringService string           `toml:"keyring_service,omitempty"`
	Infisical      InfisicalProfile `toml:"infisical,omitempty"`
	OpenBao        OpenBaoProfile   `toml:"openbao,omitempty"`
	Env            []EnvMapping     `toml:"env,omitempty"`
}

type InfisicalProfile struct {
	SiteURL        string `toml:"site_url,omitempty"`
	ProjectID      string `toml:"project_id,omitempty"`
	Environment    string `toml:"environment,omitempty"`
	SecretPath     string `toml:"secret_path,omitempty"`
	Recursive      bool   `toml:"recursive,omitempty"`
	IncludeImports bool   `toml:"include_imports,omitempty"`
}

type OpenBaoProfile struct {
	Address   string `toml:"address,omitempty"`
	Mount     string `toml:"mount,omitempty"`
	Path      string `toml:"path,omitempty"`
	KVVersion int    `toml:"kv_version,omitempty"`
}

type EnvMapping struct {
	RemoteKey string `toml:"remote_key"`
	EnvName   string `toml:"env_name"`
}

func Path() (string, error) {
	return xdg.ConfigFile(relativeConfigPath)
}

func Load() (Config, string, error) {
	path, err := Path()
	if err != nil {
		return Config{}, "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, path, nil
	}
	if err != nil {
		return Config{}, path, err
	}
	if len(data) == 0 {
		return Config{}, path, nil
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, path, fmt.Errorf("read config %s: %w", path, err)
	}
	return cfg, path, nil
}

func Save(cfg Config) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (cfg Config) Profile(name string) (Profile, bool) {
	for _, profile := range cfg.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}

func (cfg *Config) UpsertProfile(profile Profile) {
	for index := range cfg.Profiles {
		if cfg.Profiles[index].Name == profile.Name {
			cfg.Profiles[index] = profile
			return
		}
	}
	cfg.Profiles = append(cfg.Profiles, profile)
}

func DefaultKeyringService(profile Profile) string {
	if profile.KeyringService != "" {
		return profile.KeyringService
	}
	return "yewk"
}

func ValidateProfile(profile Profile) error {
	if profile.Name == "" {
		return errors.New("profile name is required")
	}
	if profile.Provider != "infisical" && profile.Provider != "openbao" {
		return fmt.Errorf("profile %q has unsupported provider %q", profile.Name, profile.Provider)
	}
	if len(profile.Env) == 0 {
		return fmt.Errorf("profile %q must define at least one env mapping", profile.Name)
	}
	for _, mapping := range profile.Env {
		if mapping.RemoteKey == "" || mapping.EnvName == "" {
			return fmt.Errorf("profile %q has an incomplete env mapping", profile.Name)
		}
	}
	return nil
}

func NormalizeProfile(profile Profile) Profile {
	switch profile.Provider {
	case "infisical":
		if profile.Infisical.SiteURL == "" {
			profile.Infisical.SiteURL = "https://app.infisical.com"
		}
		if profile.Infisical.SecretPath == "" {
			profile.Infisical.SecretPath = "/"
		}
		profile.OpenBao = OpenBaoProfile{}
	case "openbao":
		if profile.OpenBao.Mount == "" {
			profile.OpenBao.Mount = "secret"
		}
		if profile.OpenBao.KVVersion == 0 {
			profile.OpenBao.KVVersion = 2
		}
		profile.Infisical = InfisicalProfile{}
	}
	if profile.KeyringService == "" {
		profile.KeyringService = "yewk"
	}
	return profile
}
