package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

var rootCmd = &cobra.Command{
	Use:   "yewk",
	Short: "a cli to sync your secrets to anywhere, use system keyring to store safely",
	Long:  "a cli to sync your secrets to anywhere, use system keyring to store safely.",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "Hello from yewk")
		return err
	},
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
