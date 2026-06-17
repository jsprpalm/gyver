package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ErrCanceled is returned by RunWithSpinner when the user interrupts the work
// (ctrl+c or esc) before it finishes.
var ErrCanceled = errors.New("canceled")

// spinnerStyle matches the green accent used by `gyver how`.
var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

// spinnerDoneMsg signals that the background work has returned.
type spinnerDoneMsg struct{}

type spinnerModel struct {
	spinner  spinner.Model
	label    string
	done     bool
	canceled bool
	cancel   context.CancelFunc
}

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			m.cancel() // unblock the work by canceling its context
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.done || m.canceled {
		return "" // clear the spinner line so the result prints cleanly
	}
	return fmt.Sprintf("  %s %s", m.spinner.View(), spinnerLabelStyle.Render(m.label))
}

var spinnerLabelStyle = lipgloss.NewStyle().Faint(true)

// RunWithSpinner runs work in the background while animating a spinner labelled
// label on stderr, then returns work's result. Rendering on stderr keeps stdout
// clean so `gyver how ... | …` still pipes only the suggestion.
//
// When stderr is not a terminal (output redirected, CI, etc.) it runs work
// directly with no spinner. Pressing ctrl+c or esc cancels work's context and
// returns ErrCanceled.
func RunWithSpinner[T any](label string, work func(context.Context) (T, error)) (T, error) {
	var zero T

	if !isatty.IsTerminal(os.Stderr.Fd()) {
		return work(context.Background())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	p := tea.NewProgram(
		spinnerModel{spinner: sp, label: label, cancel: cancel},
		tea.WithOutput(os.Stderr),
	)

	var (
		res    T
		runErr error
	)
	go func() {
		res, runErr = work(ctx)
		p.Send(spinnerDoneMsg{})
	}()

	final, err := p.Run()
	if err != nil {
		return zero, err
	}
	if fm, ok := final.(spinnerModel); ok && fm.canceled {
		return zero, ErrCanceled
	}
	// The channel send/receive of spinnerDoneMsg happens-before p.Run returns,
	// so res/runErr are safely published to this goroutine here.
	return res, runErr
}
