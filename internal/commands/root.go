// Package commands wires the Cobra CLI together. Each subcommand lives in its
// own file; shared adapter plumbing is in adapters.go.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gyver",
		Short: "A universal command layer for Docker, systemd and friends",
		Long: "gyver lets you list, inspect and control services without caring\n" +
			"whether they are managed by Docker, systemd, PM2 or launchd.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newServicesCmd(),
		newLogsCmd(),
		newRestartCmd(),
		newPortsCmd(),
		newHowCmd(),
		newAliasCmd(),
	)
	return root
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
