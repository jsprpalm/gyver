package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jsprpalm/gyver/internal/aliases"
)

func newAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Save, list and run command shortcuts",
		Long: "Manage named shortcuts for shell commands — for example the ones\n" +
			"suggested by `gyver how`. Aliases live in a JSON file under your\n" +
			"config dir (override with GYVER_ALIASES_FILE).",
	}
	cmd.AddCommand(
		newAliasAddCmd(),
		newAliasListCmd(),
		newAliasRemoveCmd(),
		newAliasRunCmd(),
	)
	return cmd
}

func newAliasAddCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "add <name> <command>",
		Short: "Save a command under a name",
		Long: "Save a command under a name.\n\n" +
			"Quote the command so the shell passes it as one argument:\n" +
			"  gyver alias add ports \"ss -tulpn\"",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := aliases.DefaultStore()
			if err != nil {
				return err
			}
			name := args[0]
			command := strings.Join(args[1:], " ")
			if err := store.Add(name, command, force); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved alias %q → %s\n", name, command)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing alias")
	return cmd
}

func newAliasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List saved aliases",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := aliases.DefaultStore()
			if err != nil {
				return err
			}
			all, err := store.List()
			if err != nil {
				return err
			}
			if len(all) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(),
					"no aliases yet — add one with: gyver alias add <name> \"<command>\"")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tCOMMAND")
			for _, a := range all {
				fmt.Fprintf(w, "%s\t%s\n", a.Name, a.Command)
			}
			return w.Flush()
		},
	}
}

func newAliasRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a saved alias",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := aliases.DefaultStore()
			if err != nil {
				return err
			}
			name := args[0]
			if err := store.Remove(name); err != nil {
				if errors.Is(err, aliases.ErrNotFound) {
					return fmt.Errorf("no alias named %q", name)
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed alias %q\n", name)
			return nil
		},
	}
}

func newAliasRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name> [args...]",
		Short: "Run a saved alias (extra args are appended)",
		Long: "Run a saved alias through your shell. Saved commands are full shell\n" +
			"snippets (they may contain pipes), so they run via $SHELL -c. Any extra\n" +
			"arguments are appended to the command before it runs.",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true, // let extra args (e.g. -n 5) pass through untouched
		RunE: func(cmd *cobra.Command, args []string) error {
			// DisableFlagParsing means we still have to handle -h ourselves.
			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}

			store, err := aliases.DefaultStore()
			if err != nil {
				return err
			}
			name := args[0]
			alias, err := store.Get(name)
			if err != nil {
				if errors.Is(err, aliases.ErrNotFound) {
					return fmt.Errorf("no alias named %q", name)
				}
				return err
			}

			command := alias.Command
			if extra := args[1:]; len(extra) > 0 {
				command += " " + strings.Join(extra, " ")
			}

			shell := os.Getenv("SHELL")
			if shell == "" {
				shell = "sh"
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "+ %s\n", command)

			c := exec.CommandContext(cmd.Context(), shell, "-c", command)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			return c.Run()
		},
	}
	return cmd
}
