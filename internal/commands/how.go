package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jsprpalm/gyver/internal/aliases"
	"github.com/jsprpalm/gyver/internal/recipes"
	"github.com/jsprpalm/gyver/internal/tui"
)

var (
	howCmdStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	howLabelStyle = lipgloss.NewStyle().Faint(true)
)

func newHowCmd() *cobra.Command {
	var (
		local bool
		save  string
	)

	cmd := &cobra.Command{
		Use:   "how \"<question>\"",
		Short: "Suggest a shell command for a plain-English question",
		Long: "Ask how to do something and gyver suggests a command.\n\n" +
			"When ANTHROPIC_API_KEY is set, gyver asks Claude for an answer tailored\n" +
			"to your question and OS. Otherwise (or with --local) it falls back to the\n" +
			"built-in offline recipe matcher. Override the model with GYVER_HOW_MODEL.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")

			s, err := suggest(cmd, question, local)
			if err != nil {
				if errors.Is(err, tui.ErrCanceled) {
					fmt.Fprintln(cmd.ErrOrStderr(), "gyver: canceled")
					return nil
				}
				return err
			}

			if s.Command == "" {
				fmt.Println(s.Explanation)
				return nil
			}

			fmt.Println()
			fmt.Println(howLabelStyle.Render("  suggested command:"))
			fmt.Println("  " + howCmdStyle.Render(s.Command))
			fmt.Println()
			fmt.Println(howLabelStyle.Render("  explanation:"))
			fmt.Println("  " + s.Explanation)
			fmt.Println()
			if save != "" {
				store, err := aliases.DefaultStore()
				if err != nil {
					return err
				}
				if err := store.Add(save, s.Command, false); err != nil {
					return err
				}
				fmt.Println(howLabelStyle.Render(
					fmt.Sprintf("  saved as alias %q — run it with: gyver alias run %s", save, save)))
			} else {
				fmt.Println(howLabelStyle.Render(
					fmt.Sprintf("  save as alias: gyver alias add <name> %q", s.Command)))
			}
			fmt.Println(howLabelStyle.Render("  source: " + s.Source))
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&local, "local", false,
		"skip the AI provider and use only the offline recipe matcher")
	cmd.Flags().StringVar(&save, "save", "",
		"save the suggested command as an alias with this name")

	return cmd
}

// suggest picks a provider and returns a suggestion. It uses the AI-backed
// provider when an API key is available and --local was not passed, and falls
// back to the offline recipe matcher if the AI call fails for any reason.
func suggest(cmd *cobra.Command, question string, local bool) (recipes.Suggestion, error) {
	localProvider := recipes.NewLocalProvider()

	if local {
		return localProvider.Suggest(context.Background(), question)
	}

	ai, err := recipes.NewAnthropicProvider()
	if err != nil {
		// No key configured is the normal offline case — stay quiet. Any other
		// error (e.g. the key command failed) is worth surfacing.
		if !errors.Is(err, recipes.ErrNoAPIKey) {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"gyver: could not get API key (%v); falling back to offline recipes\n", err)
		}
		return localProvider.Suggest(context.Background(), question)
	}

	s, err := tui.RunWithSpinner("asking Claude…", func(ctx context.Context) (recipes.Suggestion, error) {
		return ai.Suggest(ctx, question)
	})
	if err != nil {
		// The user aborted — don't quietly fall back, let the caller exit.
		if errors.Is(err, tui.ErrCanceled) {
			return recipes.Suggestion{}, err
		}
		fmt.Fprintf(cmd.ErrOrStderr(),
			"gyver: AI lookup failed (%v); falling back to offline recipes\n", err)
		return localProvider.Suggest(context.Background(), question)
	}
	return s, nil
}
