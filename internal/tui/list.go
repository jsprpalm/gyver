// Package tui holds the Bubble Tea / Lip Gloss interactive views. Today that is
// just the `gyver list` table, but it is the natural home for future dashboards.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jsprpalm/gyver/internal/core"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginLeft(1)

	helpStyle = lipgloss.NewStyle().
			Faint(true).
			MarginLeft(1).
			MarginTop(1)

	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))
)

type model struct {
	table table.Model
	count int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render(fmt.Sprintf("gyver — %d service(s)", m.count))
	help := helpStyle.Render("↑/↓ or j/k to move · q to quit")
	return strings.Join([]string{
		header,
		borderStyle.Render(m.table.View()),
		help,
	}, "\n")
}

// Run launches the interactive table for the given services and blocks until
// the user quits.
func Run(services []core.Service) error {
	columns := []table.Column{
		{Title: "Type", Width: 8},
		{Title: "Name", Width: 26},
		{Title: "ID", Width: 14},
		{Title: "Status", Width: 22},
		{Title: "Ports", Width: 30},
	}

	rows := make([]table.Row, 0, len(services))
	for _, s := range services {
		rows = append(rows, table.Row{
			s.Type,
			truncate(s.Name, 26),
			truncate(s.ID, 14),
			truncate(s.Status, 22),
			truncate(strings.Join(s.Ports, ", "), 30),
		})
	}

	height := len(rows) + 1
	if height > 20 {
		height = 20
	}
	if height < 3 {
		height = 3
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(styles)

	_, err := tea.NewProgram(model{table: t, count: len(services)}).Run()
	return err
}

func truncate(s string, max int) string {
	if max <= 1 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
