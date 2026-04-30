package main

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTemplateOrigin(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		want      bool
	}{
		{
			name:      "matches template origin",
			remoteURL: "https://github.com/YewFence/go-cli-template",
			want:      true,
		},
		{
			name:      "matches template origin with git suffix",
			remoteURL: "https://github.com/YewFence/go-cli-template.git\n",
			want:      true,
		},
		{
			name:      "matches template origin with http",
			remoteURL: "http://github.com/YewFence/go-cli-template.git",
			want:      true,
		},
		{
			name:      "matches template origin with ssh",
			remoteURL: "git@github.com:YewFence/go-cli-template.git",
			want:      true,
		},
		{
			name:      "ignores other origin",
			remoteURL: "https://github.com/YewFence/other.git",
			want:      false,
		},
		{
			name:      "ignores other ssh origin",
			remoteURL: "git@github.com:YewFence/other.git",
			want:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := isTemplateOrigin(test.remoteURL)
			if got != test.want {
				t.Fatalf("isTemplateOrigin(%q) = %v, want %v", test.remoteURL, got, test.want)
			}
		})
	}
}

func TestResetGitHistoryCreatesFreshRepository(t *testing.T) {
	requireGit(t)

	directory := t.TempDir()
	t.Chdir(directory)
	if err := os.WriteFile("main.go", []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runTestGit(t, "init", "-b", "main")
	runTestGit(t, "remote", "add", "origin", "https://github.com/YewFence/go-cli-template.git")
	runTestGit(t, "add", ".")
	runTestGit(t, "-c", "user.name=Template", "-c", "user.email=template@YewFence.com", "-c", "commit.gpgsign=false", "commit", "-m", "initial")

	if err := resetGitHistory(); err != nil {
		t.Fatalf("resetGitHistory() error = %v", err)
	}

	if info, err := os.Stat(".git"); err != nil || !info.IsDir() {
		t.Fatalf(".git directory was not recreated")
	}
	if remoteURL, err := runTestGitAllowError("remote", "get-url", "origin"); err == nil {
		t.Fatalf("origin remote = %q, want missing remote", remoteURL)
	}
	if output := runTestGit(t, "branch", "--show-current"); strings.TrimSpace(output) != "main" {
		t.Fatalf("branch = %q, want main", output)
	}
	if output, err := runTestGitAllowError("log", "--oneline"); err == nil {
		t.Fatalf("git log succeeded after history reset with output %q", output)
	}
}

func TestResetGitHistoryRejectsNestedRepository(t *testing.T) {
	requireGit(t)

	directory := t.TempDir()
	t.Chdir(directory)
	runTestGit(t, "init", "-b", "main")

	nestedDirectory := filepath.Join(directory, "nested")
	if err := os.Mkdir(nestedDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nestedDirectory)

	err := resetGitHistory()
	if err == nil {
		t.Fatalf("resetGitHistory() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("resetGitHistory() error = %v, want repository root error", err)
	}
}

func TestTemplateReplacementsSupportExplicitPlaceholders(t *testing.T) {
	config := config{
		module:      "github.com/acme/widget-module",
		name:        "widget",
		owner:       "acme",
		repo:        "widget-repo",
		description: "Manage widgets",
	}
	replacements := templateReplacements(config)

	directory := t.TempDir()
	path := filepath.Join(directory, "README.template.md")
	content := strings.Join([]string{
		"# yewk",
		"a cli to sync your secrets to anywhere, use system keyring to store safely",
		"https://github.com/YewFence/yew-key",
		"github.com/YewFence/yew-key",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := replaceInFile(path, replacements); err != nil {
		t.Fatalf("replaceInFile() error = %v", err)
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(output)
	for _, want := range []string{
		"# widget",
		"Manage widgets",
		"https://github.com/acme/widget-repo",
		"github.com/acme/widget-module",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{{") {
		t.Fatalf("output still contains template placeholders:\n%s", got)
	}
}

func TestRunReplacesReadmeWithReadmeTemplate(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	if err := os.WriteFile("README.md", []byte("# Template repository README\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readmeTemplate := strings.Join([]string{
		"# yewk",
		"a cli to sync your secrets to anywhere, use system keyring to store safely",
		"https://github.com/YewFence/yew-key",
		"github.com/YewFence/yew-key",
	}, "\n")
	if err := os.WriteFile("README.template.md", []byte(readmeTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})
	flag.CommandLine = flag.NewFlagSet("init-template", flag.ContinueOnError)
	os.Args = []string{
		"init-template",
		"--module", "github.com/acme/widget-module",
		"--name", "widget",
		"--owner", "acme",
		"--repo", "widget-repo",
		"--description", "Manage widgets",
	}

	if err := run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat("README.template.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("README.template.md stat error = %v, want not exist", err)
	}
	output, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}

	got := string(output)
	for _, want := range []string{
		"# widget",
		"Manage widgets",
		"https://github.com/acme/widget-repo",
		"github.com/acme/widget-module",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("README.md missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Template repository README") {
		t.Fatalf("README.md still contains the template repository README:\n%s", got)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func runTestGit(t *testing.T, args ...string) string {
	t.Helper()
	output, err := runTestGitAllowError(args...)
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return output
}

func runTestGitAllowError(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	return string(output), err
}
