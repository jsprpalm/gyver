package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jsprpalm/gyver/internal/core"
)

func newRestartCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a service or container (asks for confirmation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			adapter, service, err := findService(ctx, args[0])
			if err != nil {
				return err
			}
			return runRestart(ctx, adapter, service, yes)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

// runRestart restarts a service, asking for confirmation unless skipConfirm is
// set. Shared by the `restart` subcommand and the `r` action in the services
// table.
func runRestart(ctx context.Context, adapter core.Adapter, service core.Service, skipConfirm bool) error {
	if !skipConfirm {
		ok, err := confirm(fmt.Sprintf(
			"Restart %s service %q (%s)?", adapter.Name(), service.Name, service.ID))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("aborted")
			return nil
		}
	}

	fmt.Printf("Restarting %s…\n", service.Name)
	if err := adapter.Restart(ctx, service); err != nil {
		return err
	}
	fmt.Printf("Restarted %s\n", service.Name)
	return nil
}

// confirm prompts the user on stdin for a yes/no answer, defaulting to no.
func confirm(prompt string) (bool, error) {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, nil // e.g. EOF / no TTY → treat as "no"
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
