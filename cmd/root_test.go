package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	appconfig "github.com/YewFence/yew-key/internal/config"
	"github.com/YewFence/yew-key/internal/keyringstore"
	"github.com/YewFence/yew-key/internal/provider"
	"github.com/adrg/xdg"
)

func executeTestCommand(t *testing.T, deps runtimeDeps, args ...string) string {
	t.Helper()
	command := newRootCommand(deps)
	buffer := new(bytes.Buffer)
	command.SetOut(buffer)
	command.SetErr(buffer)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("execute command %v: %v\n%s", args, err, buffer.String())
	}
	return buffer.String()
}

func isolateXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	xdg.Reload()
	t.Cleanup(xdg.Reload)
}

func TestCompletionCommand(t *testing.T) {
	buffer := new(bytes.Buffer)

	if err := newRootCommand(runtimeDeps{}).GenBashCompletionV2(buffer, true); err != nil {
		t.Fatalf("generate bash completion: %v", err)
	}

	if got := buffer.String(); !strings.Contains(got, "# bash completion V2 for yewk") {
		t.Fatalf("unexpected completion output: %q", got)
	}
}

func TestHelpIncludesAuthTokenHints(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"sync", "--help"},
	} {
		output := executeTestCommand(t, runtimeDeps{}, args...)
		for _, want := range []string{"INFISICAL_TOKEN", "BAO_TOKEN", "VAULT_TOKEN"} {
			if !strings.Contains(output, want) {
				t.Fatalf("help %v missing %q:\n%s", args, want, output)
			}
		}
	}
}

func TestProfileCommandPrintsConfigPath(t *testing.T) {
	isolateXDG(t)

	output := executeTestCommand(t, runtimeDeps{}, "profile")

	if !strings.Contains(output, "config.toml") || strings.Contains(output, "Manage profiles") {
		t.Fatalf("profile output = %q", output)
	}
}

func TestProfileEditSplitsEditorCommand(t *testing.T) {
	isolateXDG(t)
	binDir := t.TempDir()
	argsPath := filepath.Join(t.TempDir(), "args")
	editorPath := filepath.Join(binDir, "fake-editor")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$YEWK_TEST_EDITOR_ARGS\"\n"
	if err := os.WriteFile(editorPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "fake-editor --wait")
	t.Setenv("YEWK_TEST_EDITOR_ARGS", argsPath)

	executeTestCommand(t, runtimeDeps{}, "profile", "edit")

	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || lines[0] != "--wait" || !strings.HasSuffix(lines[1], filepath.Join("yewk", "config.toml")) {
		t.Fatalf("editor args = %#v", lines)
	}
}

func TestProfileAddSyncAndEnv(t *testing.T) {
	isolateXDG(t)
	store := newMemoryStore()
	deps := runtimeDeps{
		providers: fakeFactory{providerName: "infisical", provider: fakeProvider{
			secrets: []provider.Secret{
				{RemoteKey: "DATABASE_URL", EnvName: "DATABASE_URL", Value: "postgres://local?sslmode=disable", Version: "7"},
				{RemoteKey: "OPENAI_API_KEY", EnvName: "OPENAI_API_KEY", Value: "sk-test value with 'quote'", Version: "2"},
			},
			cursor: provider.SyncCursor{ETag: "abc"},
		}},
		keyrings: fakeOpener{store: store},
	}

	output := executeTestCommand(t, deps,
		"profile", "add",
		"--name", "work",
		"--provider", "infisical",
		"--infisical-project-id", "project",
		"--infisical-environment", "dev",
		"--env", "DATABASE_URL=DATABASE_URL",
		"--env", "OPENAI_API_KEY=OPENAI_API_KEY",
	)
	if !strings.Contains(output, "profile work saved") {
		t.Fatalf("profile add output = %q", output)
	}

	output = executeTestCommand(t, deps, "sync", "work")
	if !strings.Contains(output, "synced 2 variables for profile work") {
		t.Fatalf("sync output = %q", output)
	}

	output = executeTestCommand(t, deps, "env", "work", "--shell", "zsh")
	if strings.Contains(output, "postgres://") || strings.Contains(output, "sk-test") {
		t.Fatalf("env summary leaked secret values:\n%s", output)
	}
	for _, want := range []string{"# DATABASE_URL", "# OPENAI_API_KEY", `# eval "$(yewk env work --shell zsh --reveal)"`} {
		if !strings.Contains(output, want) {
			t.Fatalf("env summary missing %q:\n%s", want, output)
		}
	}

	output = executeTestCommand(t, deps, "env", "work", "--shell", "zsh", "--reveal")
	for _, want := range []string{"export DATABASE_URL=", "postgres://local?sslmode=disable", "export OPENAI_API_KEY=", `sk-test value with '`} {
		if !strings.Contains(output, want) {
			t.Fatalf("env reveal missing %q:\n%s", want, output)
		}
	}
}

func TestCleanRemovesCachedSecretsAndState(t *testing.T) {
	isolateXDG(t)
	store := newMemoryStore()
	deps := runtimeDeps{
		providers: fakeFactory{providerName: "infisical", provider: fakeProvider{
			secrets: []provider.Secret{
				{RemoteKey: "DATABASE_URL", EnvName: "DATABASE_URL", Value: "postgres://local?sslmode=disable", Version: "7"},
				{RemoteKey: "OPENAI_API_KEY", EnvName: "OPENAI_API_KEY", Value: "sk-test", Version: "2"},
			},
		}},
		keyrings: fakeOpener{store: store},
	}

	executeTestCommand(t, deps,
		"profile", "add",
		"--name", "work",
		"--provider", "infisical",
		"--infisical-project-id", "project",
		"--infisical-environment", "dev",
		"--env", "DATABASE_URL=DATABASE_URL",
		"--env", "OPENAI_API_KEY=OPENAI_API_KEY",
	)
	executeTestCommand(t, deps, "sync", "work")

	output := executeTestCommand(t, deps, "clean", "work")
	if !strings.Contains(output, "cleaned 2 cached variables for profile work") {
		t.Fatalf("clean output = %q", output)
	}
	if len(store.values) != 0 || len(store.index.Variables) != 0 {
		t.Fatalf("clean left cached values: %#v %#v", store.values, store.index)
	}

	output = executeTestCommand(t, deps, "env", "work", "--shell", "zsh", "--reveal")
	if strings.Contains(output, "postgres://") || strings.Contains(output, "sk-test") || strings.Contains(output, "export DATABASE_URL=") {
		t.Fatalf("env reveal after clean leaked cached values:\n%s", output)
	}

	output = executeTestCommand(t, deps, "status", "work")
	for _, want := range []string{"profile work has not synced successfully yet", "variables: 0"} {
		if !strings.Contains(output, want) {
			t.Fatalf("status after clean missing %q:\n%s", want, output)
		}
	}
}

func TestProfileAddInteractive(t *testing.T) {
	isolateXDG(t)
	deps := runtimeDeps{
		keyrings: fakeOpener{store: newMemoryStore()},
		stdin: strings.NewReader(strings.Join([]string{
			"bao-dev",
			"openbao",
			"yewk",
			"https://bao.example.com",
			"secret",
			"apps/api",
			"DATABASE_URL",
			"",
			"",
			"",
		}, "\n")),
	}

	output := executeTestCommand(t, deps, "profile", "add")
	if !strings.Contains(output, "profile bao-dev saved") {
		t.Fatalf("profile add output = %q", output)
	}

	cfg, _, err := appconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	profile, ok := cfg.Profile("bao-dev")
	if !ok {
		t.Fatalf("interactive profile was not saved")
	}
	if profile.Provider != "openbao" || profile.OpenBao.Path != "apps/api" || len(profile.Env) != 1 {
		t.Fatalf("unexpected interactive profile: %#v", profile)
	}
}

type fakeFactory struct {
	providerName string
	provider     provider.Provider
}

func (factory fakeFactory) Provider(name string) (provider.Provider, error) {
	if name != factory.providerName {
		return nil, assertError("unexpected provider " + name)
	}
	return factory.provider, nil
}

type fakeProvider struct {
	secrets []provider.Secret
	cursor  provider.SyncCursor
}

func (fake fakeProvider) Fetch(context.Context, appconfig.Profile) ([]provider.Secret, provider.SyncCursor, error) {
	return fake.secrets, fake.cursor, nil
}

type fakeOpener struct {
	store keyringstore.Store
}

func (opener fakeOpener) Open(string) (keyringstore.Store, error) {
	return opener.store, nil
}

type memoryStore struct {
	index  keyringstore.Index
	values map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{values: map[string]string{}}
}

func (store *memoryStore) SyncProfile(profile appconfig.Profile, secrets []provider.Secret) (keyringstore.Index, error) {
	index := keyringstore.Index{Profile: profile.Name}
	for _, secret := range secrets {
		store.values[secret.EnvName] = secret.Value
		index.Variables = append(index.Variables, keyringstore.Variable{
			EnvName:   secret.EnvName,
			RemoteKey: secret.RemoteKey,
			Provider:  profile.Provider,
			Version:   secret.Version,
		})
	}
	store.index = index
	return index, nil
}

func (store *memoryStore) CleanProfile(profile appconfig.Profile) (int, error) {
	removed := len(store.values)
	store.values = map[string]string{}
	store.index = keyringstore.Index{Profile: profile.Name}
	return removed, nil
}

func (store *memoryStore) Index(appconfig.Profile) (keyringstore.Index, error) {
	return store.index, nil
}

func (store *memoryStore) ReadValue(_ appconfig.Profile, envName string) (string, error) {
	return store.values[envName], nil
}

type assertError string

func (err assertError) Error() string {
	return string(err)
}
