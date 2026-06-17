package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jsprpalm/gyver/internal/core"
	"github.com/jsprpalm/gyver/internal/tui"
)

func newServicesCmd() *cobra.Command {
	var (
		plain   bool
		all     bool
		running bool
		types   []string
	)

	cmd := &cobra.Command{
		Use:   "services",
		Short: "List all services and containers in one unified view",
		Long: "List services and containers from every available adapter.\n\n" +
			"Internal systemd units (systemd-*) are hidden by default; pass --all\n" +
			"to include them.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			items := gatherItems(ctx)

			filtered, hiddenInternal := filterItems(items, listFilter{
				types:   types,
				running: running,
				all:     all,
			})

			// Be transparent about what the default rule suppressed.
			if hiddenInternal > 0 {
				fmt.Fprintf(os.Stderr,
					"(%d internal systemd unit(s) hidden — use --all to show)\n", hiddenInternal)
			}

			if len(filtered) == 0 {
				fmt.Fprintln(os.Stderr,
					"no services match (try --all, relax --type/--running, or check Docker/systemd)")
				return nil
			}

			// Plain output, or a non-interactive terminal, prints script-friendly text.
			if plain || !isInteractive() {
				return printPlain(servicesOf(filtered))
			}

			result, err := tui.Run(filtered)
			if err != nil {
				return err
			}
			// The table blocks for as long as the user keeps it open, so the 10s
			// listing context above is long gone — run any chosen action against
			// a fresh one derived from the command's parent context.
			return runSelected(cmd.Context(), result)
		},
	}

	cmd.Flags().BoolVar(&plain, "plain", false, "print script-friendly plain text instead of the TUI")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "include internal systemd units (systemd-*) hidden by default")
	cmd.Flags().BoolVarP(&running, "running", "r", false, "only show services that are actively running")
	cmd.Flags().StringSliceVarP(&types, "type", "t", nil, "only show these adapter types (e.g. --type docker)")
	return cmd
}

// servicesOf drops the adapter pairing for paths (like --plain) that only need
// the bare services.
func servicesOf(items []tui.Item) []core.Service {
	out := make([]core.Service, len(items))
	for i, it := range items {
		out[i] = it.Service
	}
	return out
}

// runSelected carries out the action the user picked in the table, if any. The
// table has been torn down by now, so logs print and the restart confirmation
// prompt run in the normal terminal exactly like the standalone subcommands.
func runSelected(parent context.Context, r tui.Result) error {
	switch r.Action {
	case tui.ActionLogs:
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		return runLogs(ctx, r.Item.Adapter, r.Item.Service)
	case tui.ActionRestart:
		ctx, cancel := context.WithTimeout(parent, 60*time.Second)
		defer cancel()
		return runRestart(ctx, r.Item.Adapter, r.Item.Service, false)
	default:
		return nil
	}
}

func printPlain(services []core.Service) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tNAME\tID\tSTATUS\tPORTS")
	for _, s := range services {
		ports := strings.Join(s.Ports, ",")
		if ports == "" {
			ports = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Type, s.Name, s.ID, s.Status, ports)
	}
	return w.Flush()
}

// isInteractive reports whether stdout is a terminal. When piped or redirected
// we fall back to plain text so `gyver services | grep …` just works.
func isInteractive() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
