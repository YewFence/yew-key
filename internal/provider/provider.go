package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	appconfig "github.com/YewFence/yew-key/internal/config"
	infisical "github.com/infisical/go-sdk"
	openbao "github.com/openbao/openbao/api/v2"
)

type Secret struct {
	RemoteKey string
	EnvName   string
	Value     string
	Version   string
	Source    string
}

type SyncCursor struct {
	ETag string `json:"etag,omitempty"`
}

type Provider interface {
	Fetch(ctx context.Context, profile appconfig.Profile) ([]Secret, SyncCursor, error)
}

type Factory interface {
	Provider(name string) (Provider, error)
}

type DefaultFactory struct{}

func (DefaultFactory) Provider(name string) (Provider, error) {
	switch name {
	case "infisical":
		return InfisicalProvider{}, nil
	case "openbao":
		return OpenBaoProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", name)
	}
}

type InfisicalProvider struct{}

func (InfisicalProvider) Fetch(ctx context.Context, profile appconfig.Profile) ([]Secret, SyncCursor, error) {
	token := strings.TrimSpace(os.Getenv("INFISICAL_TOKEN"))
	if token == "" {
		return nil, SyncCursor{}, errors.New("INFISICAL_TOKEN is required for infisical sync")
	}
	if profile.Infisical.ProjectID == "" {
		return nil, SyncCursor{}, fmt.Errorf("profile %q is missing infisical project_id", profile.Name)
	}
	if profile.Infisical.Environment == "" {
		return nil, SyncCursor{}, fmt.Errorf("profile %q is missing infisical environment", profile.Name)
	}

	siteURL := profile.Infisical.SiteURL
	if siteURL == "" {
		siteURL = "https://app.infisical.com"
	}
	secretPath := profile.Infisical.SecretPath
	if secretPath == "" {
		secretPath = "/"
	}

	config, err := infisicalClientConfig(siteURL)
	if err != nil {
		return nil, SyncCursor{}, err
	}
	client := infisical.NewInfisicalClient(ctx, config)
	client.Auth().SetAccessToken(token)

	result, err := client.Secrets().ListSecrets(infisical.ListSecretsOptions{
		ProjectID:              profile.Infisical.ProjectID,
		Environment:            profile.Infisical.Environment,
		SecretPath:             secretPath,
		Recursive:              profile.Infisical.Recursive,
		IncludeImports:         profile.Infisical.IncludeImports,
		ExpandSecretReferences: true,
		AttachToProcessEnv:     false,
	})
	if err != nil {
		return nil, SyncCursor{}, fmt.Errorf("fetch infisical secrets for profile %q: %w", profile.Name, err)
	}

	remote := make(map[string]Secret, len(result.Secrets))
	for _, secret := range result.Secrets {
		remote[secret.SecretKey] = Secret{
			RemoteKey: secret.SecretKey,
			Value:     secret.SecretValue,
			Version:   fmt.Sprint(secret.Version),
			Source:    secret.SecretPath,
		}
	}
	secrets, err := applyMappings(profile, remote)
	if err != nil {
		return nil, SyncCursor{}, err
	}
	return secrets, SyncCursor{ETag: result.ETag}, nil
}

func infisicalClientConfig(siteURL string) (infisical.Config, error) {
	customHeaders, err := parseInfisicalCustomHeaders(os.Getenv("INFISICAL_CUSTOM_HEADERS"))
	if err != nil {
		return infisical.Config{}, err
	}
	return infisical.Config{
		SiteUrl:          siteURL,
		AutoTokenRefresh: false,
		SilentMode:       true,
		CustomHeaders:    customHeaders,
	}, nil
}

func parseInfisicalCustomHeaders(raw string) (map[string]string, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return nil, nil
	}

	headers := make(map[string]string, len(fields))
	for index, field := range fields {
		name, value, ok := strings.Cut(field, "=")
		if !ok {
			return nil, fmt.Errorf("INFISICAL_CUSTOM_HEADERS entry %d must use header=value format", index+1)
		}
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("INFISICAL_CUSTOM_HEADERS entry %d is missing a header name", index+1)
		}
		if value == "" {
			return nil, fmt.Errorf("INFISICAL_CUSTOM_HEADERS entry %d is missing a header value", index+1)
		}
		headers[name] = value
	}
	return headers, nil
}

type OpenBaoProvider struct{}

func (OpenBaoProvider) Fetch(ctx context.Context, profile appconfig.Profile) ([]Secret, SyncCursor, error) {
	token := strings.TrimSpace(os.Getenv("BAO_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("VAULT_TOKEN"))
	}
	if token == "" {
		return nil, SyncCursor{}, errors.New("BAO_TOKEN or VAULT_TOKEN is required for openbao sync")
	}

	address := profile.OpenBao.Address
	if address == "" {
		address = strings.TrimSpace(os.Getenv("BAO_ADDR"))
	}
	if address == "" {
		address = strings.TrimSpace(os.Getenv("VAULT_ADDR"))
	}
	if address == "" {
		return nil, SyncCursor{}, fmt.Errorf("profile %q is missing openbao address", profile.Name)
	}

	mount := profile.OpenBao.Mount
	if mount == "" {
		mount = "secret"
	}
	path := profile.OpenBao.Path
	if path == "" {
		return nil, SyncCursor{}, fmt.Errorf("profile %q is missing openbao path", profile.Name)
	}

	config := openbao.DefaultConfig()
	config.Address = address
	client, err := openbao.NewClient(config)
	if err != nil {
		return nil, SyncCursor{}, fmt.Errorf("create openbao client for profile %q: %w", profile.Name, err)
	}
	client.SetToken(token)

	kvVersion := profile.OpenBao.KVVersion
	if kvVersion == 0 {
		kvVersion = 2
	}

	var data map[string]interface{}
	var version string
	switch kvVersion {
	case 1:
		secret, err := client.KVv1(mount).Get(ctx, path)
		if err != nil {
			return nil, SyncCursor{}, fmt.Errorf("fetch openbao kv v1 secrets for profile %q: %w", profile.Name, err)
		}
		data = secret.Data
	case 2:
		secret, err := client.KVv2(mount).Get(ctx, path)
		if err != nil {
			return nil, SyncCursor{}, fmt.Errorf("fetch openbao kv v2 secrets for profile %q: %w", profile.Name, err)
		}
		data = secret.Data
		if secret.VersionMetadata != nil {
			version = fmt.Sprint(secret.VersionMetadata.Version)
		}
	default:
		return nil, SyncCursor{}, fmt.Errorf("profile %q uses unsupported openbao kv_version %d", profile.Name, kvVersion)
	}

	remote := make(map[string]Secret, len(data))
	for key, rawValue := range data {
		value, ok := rawValue.(string)
		if !ok {
			return nil, SyncCursor{}, fmt.Errorf("openbao remote key %q in profile %q is not a string", key, profile.Name)
		}
		remote[key] = Secret{
			RemoteKey: key,
			Value:     value,
			Version:   version,
			Source:    path,
		}
	}
	secrets, err := applyMappings(profile, remote)
	if err != nil {
		return nil, SyncCursor{}, err
	}
	return secrets, SyncCursor{}, nil
}

func applyMappings(profile appconfig.Profile, remote map[string]Secret) ([]Secret, error) {
	secrets := make([]Secret, 0, len(profile.Env))
	var missing []string
	for _, mapping := range profile.Env {
		secret, ok := remote[mapping.RemoteKey]
		if !ok {
			missing = append(missing, mapping.RemoteKey)
			continue
		}
		secret.EnvName = mapping.EnvName
		secrets = append(secrets, secret)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("profile %q is missing remote keys %s", profile.Name, strings.Join(missing, ", "))
	}
	sort.Slice(secrets, func(i, j int) bool {
		return secrets[i].EnvName < secrets[j].EnvName
	})
	return secrets, nil
}
