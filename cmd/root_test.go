package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func configureRootCommand(buffer *bytes.Buffer, args ...string) {
	rootCmd.SetOut(buffer)
	rootCmd.SetErr(buffer)
	rootCmd.SetArgs(args)
	for _, command := range rootCmd.Commands() {
		command.SetOut(buffer)
		command.SetErr(buffer)
	}
}

func resetRootCommand() {
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	rootCmd.SetArgs(nil)
	for _, command := range rootCmd.Commands() {
		command.SetOut(nil)
		command.SetErr(nil)
	}
}

func TestRootCommand(t *testing.T) {
	buffer := new(bytes.Buffer)
	configureRootCommand(buffer)

	t.Cleanup(resetRootCommand)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	if got := buffer.String(); got != "Hello from yewk\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestCompletionCommand(t *testing.T) {
	buffer := new(bytes.Buffer)

	if err := rootCmd.GenBashCompletionV2(buffer, true); err != nil {
		t.Fatalf("generate bash completion: %v", err)
	}

	if got := buffer.String(); !strings.Contains(got, "# bash completion V2 for yewk") {
		t.Fatalf("unexpected completion output: %q", got)
	}
}
