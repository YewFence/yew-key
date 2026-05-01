package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/YewFence/yew-key/internal/app"
	appconfig "github.com/YewFence/yew-key/internal/config"
	"github.com/YewFence/yew-key/internal/keyringstore"
	"github.com/YewFence/yew-key/internal/provider"
	"github.com/YewFence/yew-key/internal/shellenv"
	appstate "github.com/YewFence/yew-key/internal/state"
	"github.com/spf13/cobra"
	shshell "mvdan.cc/sh/v3/shell"
)

var appVersion = "dev"

const authHelp = `yewk does not sign in to remote secret managers or store their auth tokens.
Before running sync, provide the token in the current shell.

Infisical uses INFISICAL_TOKEN.
  export INFISICAL_TOKEN="$(infisical user get token --plain)"

OpenBao uses BAO_TOKEN or VAULT_TOKEN.
  export BAO_TOKEN="..."`

type runtimeDeps struct {
	providers provider.Factory
	keyrings  keyringstore.Opener
	stdin     io.Reader
}

var rootCmd = newRootCommand(runtimeDeps{
	providers: provider.DefaultFactory{},
	keyrings:  keyringstore.DefaultOpener{},
	stdin:     os.Stdin,
})

func newRootCommand(deps runtimeDeps) *cobra.Command {
	if deps.providers == nil {
		deps.providers = provider.DefaultFactory{}
	}
	if deps.keyrings == nil {
		deps.keyrings = keyringstore.DefaultOpener{}
	}
	if deps.stdin == nil {
		deps.stdin = os.Stdin
	}

	command := &cobra.Command{
		Use:   "yewk",
		Short: "a cli to sync your secrets to anywhere, use system keyring to store safely",
		Long:  "a cli to sync your secrets to anywhere, use system keyring to store safely.\n\n" + authHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(newVersionCommand())
	command.AddCommand(newProfileCommand(deps))
	command.AddCommand(newSyncCommand(deps))
	command.AddCommand(newEnvCommand(deps))
	command.AddCommand(newStatusCommand(deps))
	return command
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func SetVersion(version string) {
	appVersion = version
}

func loadProfile(name string) (appconfig.Config, appconfig.Profile, string, error) {
	cfg, path, err := appconfig.Load()
	if err != nil {
		return appconfig.Config{}, appconfig.Profile{}, "", err
	}
	profile, ok := cfg.Profile(name)
	if !ok {
		return appconfig.Config{}, appconfig.Profile{}, path, fmt.Errorf("profile %q not found in %s", name, path)
	}
	if err := appconfig.ValidateProfile(profile); err != nil {
		return appconfig.Config{}, appconfig.Profile{}, path, err
	}
	return cfg, profile, path, nil
}

func openStore(deps runtimeDeps, profile appconfig.Profile) (keyringstore.Store, error) {
	return deps.keyrings.Open(appconfig.DefaultKeyringService(profile))
}

func updateProfileState(profileName string, mutate func(*appstate.ProfileState)) error {
	state, _, err := appstate.Load()
	if err != nil {
		return err
	}
	profileState := state.Profiles[profileName]
	mutate(&profileState)
	state.Profiles[profileName] = profileState
	_, err = appstate.Save(state)
	return err
}

func newProfileCommand(deps runtimeDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := appconfig.Path()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), path)
			return err
		},
	}
	command.AddCommand(newProfileAddCommand(deps))
	command.AddCommand(newProfileEditCommand())
	return command
}

func newProfileAddCommand(deps runtimeDeps) *cobra.Command {
	var profile appconfig.Profile
	var mappings []string
	command := &cobra.Command{
		Use:   "add",
		Short: "Add a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, mapping := range mappings {
				parts := strings.SplitN(mapping, "=", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid env mapping %q, use REMOTE_KEY=ENV_NAME", mapping)
				}
				profile.Env = append(profile.Env, appconfig.EnvMapping{RemoteKey: parts[0], EnvName: parts[1]})
			}
			if profile.Name == "" || profile.Provider == "" || len(profile.Env) == 0 {
				var err error
				profile, err = app.PromptProfile(deps.stdin, cmd.OutOrStdout(), profile)
				if err != nil {
					return err
				}
			}
			profile = appconfig.NormalizeProfile(profile)
			if err := appconfig.ValidateProfile(profile); err != nil {
				return err
			}
			cfg, _, err := appconfig.Load()
			if err != nil {
				return err
			}
			cfg.UpsertProfile(profile)
			path, err := appconfig.Save(cfg)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "profile %s saved to %s\n", profile.Name, path)
			return err
		},
	}
	command.Flags().StringVar(&profile.Name, "name", "", "profile name")
	command.Flags().StringVar(&profile.Provider, "provider", "", "provider, infisical or openbao")
	command.Flags().StringVar(&profile.KeyringService, "keyring-service", "yewk", "keyring service name")
	command.Flags().StringVar(&profile.Infisical.SiteURL, "infisical-site-url", "https://app.infisical.com", "Infisical site URL")
	command.Flags().StringVar(&profile.Infisical.ProjectID, "infisical-project-id", "", "Infisical project ID")
	command.Flags().StringVar(&profile.Infisical.Environment, "infisical-environment", "", "Infisical environment")
	command.Flags().StringVar(&profile.Infisical.SecretPath, "infisical-secret-path", "/", "Infisical secret path")
	command.Flags().BoolVar(&profile.Infisical.Recursive, "infisical-recursive", true, "Infisical recursive listing")
	command.Flags().BoolVar(&profile.Infisical.IncludeImports, "infisical-include-imports", true, "Infisical include imports")
	command.Flags().StringVar(&profile.OpenBao.Address, "openbao-address", "", "OpenBao address")
	command.Flags().StringVar(&profile.OpenBao.Mount, "openbao-mount", "secret", "OpenBao KV mount")
	command.Flags().StringVar(&profile.OpenBao.Path, "openbao-path", "", "OpenBao secret path")
	command.Flags().IntVar(&profile.OpenBao.KVVersion, "openbao-kv-version", 2, "OpenBao KV version")
	command.Flags().StringArrayVar(&mappings, "env", nil, "env mapping as REMOTE_KEY=ENV_NAME")
	return command
}

func newProfileEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Edit the config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := appconfig.Path()
			if err != nil {
				return err
			}
			editor := os.Getenv("VISUAL")
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				editor = defaultEditor()
			}
			if editor == "" {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "config file is %s\n", path)
				return err
			}
			editorFields, err := splitEditorCommand(editor)
			if err != nil {
				return err
			}
			editorArgs := append(editorFields[1:], path)
			editorCommand := exec.Command(editorFields[0], editorArgs...)
			editorCommand.Stdin = os.Stdin
			editorCommand.Stdout = os.Stdout
			editorCommand.Stderr = os.Stderr
			return editorCommand.Run()
		},
	}
}

func newSyncCommand(deps runtimeDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "sync <profile>",
		Short: "Sync secrets into local keyring",
		Long:  "Sync secrets from the configured remote provider into local keyring.\n\n" + authHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, profile, _, err := loadProfile(args[0])
			if err != nil {
				return err
			}
			remoteProvider, err := deps.providers.Provider(profile.Provider)
			if err != nil {
				return err
			}
			secrets, cursor, err := remoteProvider.Fetch(context.Background(), profile)
			if err != nil {
				_ = updateProfileState(profile.Name, func(profileState *appstate.ProfileState) {
					profileState.LastError = err.Error()
				})
				return err
			}
			store, err := openStore(deps, profile)
			if err != nil {
				return err
			}
			index, err := store.SyncProfile(profile, secrets)
			if err != nil {
				return err
			}
			if err := updateProfileState(profile.Name, func(profileState *appstate.ProfileState) {
				profileState.LastSuccess = time.Now().UTC()
				profileState.LastError = ""
				profileState.Cursor = cursor
				profileState.Variables = index.Variables
			}); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "synced %d variables for profile %s\n", len(index.Variables), profile.Name)
			return err
		},
	}
}

func newEnvCommand(deps runtimeDeps) *cobra.Command {
	var shellName string
	var reveal bool
	command := &cobra.Command{
		Use:   "env <profile>",
		Short: "Print shell environment loader",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, profile, _, err := loadProfile(args[0])
			if err != nil {
				return err
			}
			if !shellenv.SupportedShell(shellName) {
				return fmt.Errorf("unsupported shell %q", shellName)
			}
			store, err := openStore(deps, profile)
			if err != nil {
				return err
			}
			index, err := store.Index(profile)
			if err != nil {
				return err
			}
			if !reveal {
				return shellenv.RenderSummary(cmd.OutOrStdout(), profile.Name, shellName, index)
			}
			values := make(map[string]string, len(index.Variables))
			for _, variable := range index.Variables {
				value, err := store.ReadValue(profile, variable.EnvName)
				if err != nil {
					return err
				}
				values[variable.EnvName] = value
			}
			return shellenv.RenderExports(cmd.OutOrStdout(), profile.Name, shellName, values)
		},
	}
	command.Flags().StringVar(&shellName, "shell", "zsh", "shell name")
	command.Flags().BoolVar(&reveal, "reveal", false, "reveal secret values")
	return command
}

func newStatusCommand(deps runtimeDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "status <profile>",
		Short: "Show local sync status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, profile, _, err := loadProfile(args[0])
			if err != nil {
				return err
			}
			state, _, err := appstate.Load()
			if err != nil {
				return err
			}
			profileState := state.Profiles[profile.Name]
			if profileState.LastSuccess.IsZero() {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "profile %s has not synced successfully yet\n", profile.Name); err != nil {
					return err
				}
			} else if _, err := fmt.Fprintf(cmd.OutOrStdout(), "profile %s last synced at %s\n", profile.Name, profileState.LastSuccess.Format(time.RFC3339)); err != nil {
				return err
			}
			if profileState.LastError != "" {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "last error: %s\n", profileState.LastError); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "variables: %d\n", len(profileState.Variables))
			return err
		},
	}
}

func defaultEditor() string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	for _, editor := range []string{"nano", "vi"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}
	return ""
}

func splitEditorCommand(editor string) ([]string, error) {
	fields, err := shshell.Fields(editor, os.Getenv)
	if err != nil {
		return nil, fmt.Errorf("parse editor command %q: %w", editor, err)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("editor command is empty")
	}
	return fields, nil
}
