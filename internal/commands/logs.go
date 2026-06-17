package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jsprpalm/gyver/internal/core"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <name>",
		Short: "Show recent logs for a service or container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			adapter, service, err := findService(ctx, args[0])
			if err != nil {
				return err
			}
			return runLogs(ctx, adapter, service)
		},
	}
}

// runLogs prints recent logs for a service. Shared by the `logs` subcommand and
// the `l` action in the services table.
func runLogs(ctx context.Context, adapter core.Adapter, service core.Service) error {
	fmt.Printf("== %s logs for %s (%s) ==\n", adapter.Name(), service.Name, service.ID)
	return adapter.Logs(ctx, service)
}
