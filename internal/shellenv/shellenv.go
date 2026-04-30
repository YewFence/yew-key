package shellenv

import (
	"fmt"
	"io"
	"regexp"
	"sort"

	"github.com/YewFence/yew-key/internal/keyringstore"
	"mvdan.cc/sh/v3/syntax"
)

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func RenderSummary(writer io.Writer, profileName string, shellName string, index keyringstore.Index) error {
	variables := variableNames(index)
	if _, err := fmt.Fprintf(writer, "# yewk profile %s has %d variables:\n", profileName, len(variables)); err != nil {
		return err
	}
	for _, name := range variables {
		if _, err := fmt.Fprintf(writer, "# %s\n", name); err != nil {
			return err
		}
	}
	return writeHint(writer, profileName, shellName)
}

func RenderExports(writer io.Writer, profileName string, shellName string, values map[string]string) error {
	names := make([]string, 0, len(values))
	for name := range values {
		if !envNamePattern.MatchString(name) {
			return fmt.Errorf("env name %q is not shell compatible", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		quoted, err := syntax.Quote(values[name], syntax.LangBash)
		if err != nil {
			return fmt.Errorf("quote env %s: %w", name, err)
		}
		if _, err := fmt.Fprintf(writer, "export %s=%s\n", name, quoted); err != nil {
			return err
		}
	}
	return writeHint(writer, profileName, shellName)
}

func SupportedShell(shellName string) bool {
	return shellName == "zsh" || shellName == "bash"
}

func variableNames(index keyringstore.Index) []string {
	variables := make([]string, 0, len(index.Variables))
	for _, variable := range index.Variables {
		variables = append(variables, variable.EnvName)
	}
	sort.Strings(variables)
	return variables
}

func writeHint(writer io.Writer, profileName string, shellName string) error {
	if _, err := fmt.Fprintf(writer, "\n# To load secret values in %s, add the following line to your shell startup file:\n", shellName); err != nil {
		return err
	}
	_, err := fmt.Fprintf(writer, "# eval \"$(yewk env %s --shell %s --reveal)\"\n", profileName, shellName)
	return err
}
