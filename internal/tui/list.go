// Package tui holds the Bubble Tea / Lip Gloss interactive views. Today that is
// just the `gyver services` table, but it is the natural home for future dashboards.
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

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			MarginLeft(1)
)

// Action is what the user asked gyver to do with the selected service on the
// way out of the table.
type Action int

const (
	// ActionNone means the user just quit without picking an action.
	ActionNone Action = iota
	// ActionLogs means: show logs for the selected service.
	ActionLogs
	// ActionRestart means: restart the selected service.
	ActionRestart
)

// Item pairs a service with the adapter that owns it, so the table can hand
// back enough context for the caller to run logs/restart without a second
// lookup.
type Item struct {
	Adapter core.Adapter
	Service core.Service
}

// Result reports what the user did: the action requested (ActionNone if they
// just quit) and the item it applies to.
type Result struct {
	Action Action
	Item   Item
}

type model struct {
	table    table.Model
	items    []Item // every item, in original order
	filtered []Item // items currently shown; index-aligned with the table rows
	filter   string // current search query ("" = no filter)
	search   bool   // true while the user is typing a search
	result   Result
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	// While searching, keystrokes edit the query rather than driving the table,
	// so that q/j/k typed as part of a search aren't swallowed as commands.
	if m.search {
		return m.updateSearch(key)
	}

	switch key.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.search = true
		return m, nil
	case "l":
		if it, ok := m.selected(); ok {
			m.result = Result{Action: ActionLogs, Item: it}
			return m, tea.Quit
		}
		return m, nil
	case "r":
		if it, ok := m.selected(); ok {
			m.result = Result{Action: ActionRestart, Item: it}
			return m, tea.Quit
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(key)
	return m, cmd
}

// updateSearch handles keystrokes while the search prompt is active. Esc
// cancels and restores the full list; Enter keeps the filter applied and
// returns to navigation.
func (m model) updateSearch(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.search = false
		m.filter = ""
		m.applyFilter()
	case tea.KeyEnter:
		m.search = false
	case tea.KeyBackspace:
		if r := []rune(m.filter); len(r) > 0 {
			m.filter = string(r[:len(r)-1])
			m.applyFilter()
		}
	case tea.KeySpace:
		m.filter += " "
		m.applyFilter()
	case tea.KeyRunes:
		m.filter += string(key.Runes)
		m.applyFilter()
	}
	return m, nil
}

// selected returns the item under the cursor, or ok=false when the (possibly
// filtered) list is empty.
func (m model) selected() (Item, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.filtered) {
		return Item{}, false
	}
	return m.filtered[i], true
}

// applyFilter recomputes the visible rows from the current query and keeps the
// cursor within range.
func (m *model) applyFilter() {
	needle := strings.ToLower(strings.TrimSpace(m.filter))

	filtered := make([]Item, 0, len(m.items))
	for _, it := range m.items {
		if needle == "" || matches(it.Service, needle) {
			filtered = append(filtered, it)
		}
	}
	m.filtered = filtered

	rows := make([]table.Row, 0, len(filtered))
	for _, it := range filtered {
		rows = append(rows, rowFor(it.Service))
	}
	m.table.SetRows(rows)

	// SetRows doesn't clamp the cursor, so a shrinking list can leave it past
	// the end — pull it back so the highlight stays on a real row.
	if c := m.table.Cursor(); c >= len(rows) {
		if len(rows) == 0 {
			m.table.SetCursor(0)
		} else {
			m.table.SetCursor(len(rows) - 1)
		}
	}
}

func (m model) View() string {
	header := titleStyle.Render(m.headerText())

	var status string
	switch {
	case m.search:
		status = searchStyle.Render("search: " + m.filter + "▌")
	case m.filter != "":
		status = searchStyle.Render(fmt.Sprintf("filter: %q · press / then esc to clear", m.filter))
	}

	help := helpStyle.Render("↑/↓ move · / search · l logs · r restart · q quit")

	parts := []string{header}
	if status != "" {
		parts = append(parts, status)
	}
	parts = append(parts, borderStyle.Render(m.table.View()), help)
	return strings.Join(parts, "\n")
}

func (m model) headerText() string {
	if m.filter != "" {
		return fmt.Sprintf("gyver — %d of %d service(s)", len(m.filtered), len(m.items))
	}
	return fmt.Sprintf("gyver — %d service(s)", len(m.items))
}

// matches reports whether the service contains needle (already lower-cased) in
// any of its visible fields.
func matches(s core.Service, needle string) bool {
	fields := append([]string{s.Type, s.Name, s.ID, s.Status}, s.Ports...)
	return strings.Contains(strings.ToLower(strings.Join(fields, " ")), needle)
}

func rowFor(s core.Service) table.Row {
	return table.Row{
		s.Type,
		truncate(s.Name, 26),
		truncate(s.ID, 14),
		truncate(s.Status, 22),
		truncate(strings.Join(s.Ports, ", "), 30),
	}
}

// Run launches the interactive table for the given items and blocks until the
// user quits. The returned Result reports any action they chose (logs/restart)
// for the caller to carry out.
func Run(items []Item) (Result, error) {
	columns := []table.Column{
		{Title: "Type", Width: 8},
		{Title: "Name", Width: 26},
		{Title: "ID", Width: 14},
		{Title: "Status", Width: 22},
		{Title: "Ports", Width: 30},
	}

	rows := make([]table.Row, 0, len(items))
	for _, it := range items {
		rows = append(rows, rowFor(it.Service))
	}

	height := min(max(len(rows)+1, 3), 20)

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

	final, err := tea.NewProgram(model{
		table:    t,
		items:    items,
		filtered: items,
	}).Run()
	if err != nil {
		return Result{}, err
	}
	return final.(model).result, nil
}

func truncate(s string, max int) string {
	if max <= 1 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
