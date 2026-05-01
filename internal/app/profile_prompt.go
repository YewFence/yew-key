package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	appconfig "github.com/YewFence/yew-key/internal/config"
)

func PromptProfile(input io.Reader, output io.Writer, initial appconfig.Profile) (appconfig.Profile, error) {
	reader := bufio.NewReader(input)
	var err error
	initial.Name, err = prompt(reader, output, "Profile name", initial.Name)
	if err != nil {
		return appconfig.Profile{}, err
	}
	initial.Provider, err = prompt(reader, output, "Provider infisical or openbao", initial.Provider)
	if err != nil {
		return appconfig.Profile{}, err
	}
	if initial.KeyringService == "" {
		initial.KeyringService = "yewk"
	}
	initial.KeyringService, err = prompt(reader, output, "Local keyring namespace", initial.KeyringService)
	if err != nil {
		return appconfig.Profile{}, err
	}

	switch initial.Provider {
	case "infisical":
		initial.Infisical.SiteURL, err = prompt(reader, output, "Infisical site url", defaultString(initial.Infisical.SiteURL, "https://app.infisical.com"))
		if err != nil {
			return appconfig.Profile{}, err
		}
		initial.Infisical.ProjectID, err = prompt(reader, output, "Infisical project id", initial.Infisical.ProjectID)
		if err != nil {
			return appconfig.Profile{}, err
		}
		initial.Infisical.Environment, err = prompt(reader, output, "Infisical environment", initial.Infisical.Environment)
		if err != nil {
			return appconfig.Profile{}, err
		}
		initial.Infisical.SecretPath, err = prompt(reader, output, "Infisical secret path", defaultString(initial.Infisical.SecretPath, "/"))
		if err != nil {
			return appconfig.Profile{}, err
		}
	case "openbao":
		initial.OpenBao.Address, err = prompt(reader, output, "OpenBao address", initial.OpenBao.Address)
		if err != nil {
			return appconfig.Profile{}, err
		}
		initial.OpenBao.Mount, err = prompt(reader, output, "OpenBao mount", defaultString(initial.OpenBao.Mount, "secret"))
		if err != nil {
			return appconfig.Profile{}, err
		}
		initial.OpenBao.Path, err = prompt(reader, output, "OpenBao path", initial.OpenBao.Path)
		if err != nil {
			return appconfig.Profile{}, err
		}
		if initial.OpenBao.KVVersion == 0 {
			initial.OpenBao.KVVersion = 2
		}
	default:
		return appconfig.Profile{}, fmt.Errorf("unsupported provider %q", initial.Provider)
	}

	if len(initial.Env) == 0 {
		initial.Env, err = promptMappings(reader, output)
		if err != nil {
			return appconfig.Profile{}, err
		}
	}
	return appconfig.NormalizeProfile(initial), nil
}

func prompt(reader *bufio.Reader, output io.Writer, label string, defaultValue string) (string, error) {
	if defaultValue == "" {
		if _, err := fmt.Fprintf(output, "%s: ", label); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(output, "%s [%s]: ", label, defaultValue); err != nil {
			return "", err
		}
	}
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	return value, nil
}

func promptMappings(reader *bufio.Reader, output io.Writer) ([]appconfig.EnvMapping, error) {
	var mappings []appconfig.EnvMapping
	for {
		remoteKey, err := prompt(reader, output, "Remote secret key, press Enter when done", "")
		if err != nil {
			return nil, err
		}
		if remoteKey == "" {
			break
		}
		envName, err := prompt(reader, output, "Local Env name", remoteKey)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, appconfig.EnvMapping{RemoteKey: remoteKey, EnvName: envName})
	}
	return mappings, nil
}

func defaultString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
