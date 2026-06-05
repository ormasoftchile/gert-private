package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gert",
	Short: "GERT — GERT Execution and Runbook Toolchain",
	Long: `gert is the command-line interface for the GERT runbook execution toolchain.

It provides subcommands for validating, migrating, and inspecting runbook schemas.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(migrateExprCmd())
}
