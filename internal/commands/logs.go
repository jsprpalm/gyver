package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
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

			fmt.Printf("== %s logs for %s (%s) ==\n", adapter.Name(), service.Name, service.ID)
			return adapter.Logs(ctx, service)
		},
	}
}
