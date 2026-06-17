package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jsprpalm/gyver/internal/core"
)

func sampleItems() []Item {
	return []Item{
		{Service: core.Service{Type: "docker", Name: "caddy", ID: "abc123", Status: "Up 2 hours", Ports: []string{"0.0.0.0:80->80/tcp"}}},
		{Service: core.Service{Type: "systemd", Name: "ollama", ID: "ollama.service", Status: "active (running)"}},
		{Service: core.Service{Type: "docker", Name: "frigate", ID: "def456", Status: "Up 7 days"}},
	}
}

func newTestModel(items []Item) model {
	rows := make([]table.Row, 0, len(items))
	for _, it := range items {
		rows = append(rows, rowFor(it.Service))
	}
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Type", Width: 8},
			{Title: "Name", Width: 26},
			{Title: "ID", Width: 14},
			{Title: "Status", Width: 22},
			{Title: "Ports", Width: 30},
		}),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	return model{table: t, items: items, filtered: items}
}

func send(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestMatches(t *testing.T) {
	caddy := core.Service{Type: "docker", Name: "caddy", Status: "Up 2 hours", Ports: []string{"0.0.0.0:80->80/tcp"}}
	ollama := core.Service{Type: "systemd", Name: "ollama", Status: "active (running)"}

	cases := []struct {
		name   string
		svc    core.Service
		needle string
		want   bool
	}{
		{"name substring", caddy, "cad", true},
		{"status", ollama, "running", true},
		{"ports", caddy, "80", true},
		{"type", caddy, "docker", true},
		{"miss", caddy, "zzz", false},
	}
	for _, c := range cases {
		if got := matches(c.svc, c.needle); got != c.want {
			t.Errorf("%s: matches(%q, %q) = %v, want %v", c.name, c.svc.Name, c.needle, got, c.want)
		}
	}
}

func TestSearchNarrowsAndSelects(t *testing.T) {
	m := newTestModel(sampleItems())

	// "/" enters search mode; typing filters the list.
	m = send(m, runes("/"))
	if !m.search {
		t.Fatal("expected search mode after '/'")
	}
	m = send(m, runes("fri"))

	if len(m.filtered) != 1 {
		t.Fatalf("filtered = %d, want 1: %v", len(m.filtered), m.filter)
	}
	it, ok := m.selected()
	if !ok || it.Service.Name != "frigate" {
		t.Fatalf("selected = %+v ok=%v, want frigate", it.Service, ok)
	}
}

func TestSearchBackspaceAndEscClears(t *testing.T) {
	m := newTestModel(sampleItems())
	m = send(m, runes("/"))
	m = send(m, runes("frix"))
	m = send(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "fri" {
		t.Fatalf("filter after backspace = %q, want %q", m.filter, "fri")
	}

	// Esc cancels the search and restores the full list.
	m = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.search {
		t.Error("esc should leave search mode")
	}
	if m.filter != "" || len(m.filtered) != 3 {
		t.Fatalf("after esc: filter=%q filtered=%d, want empty filter and 3 items", m.filter, len(m.filtered))
	}
}

func TestActionKeysReturnSelection(t *testing.T) {
	m := newTestModel(sampleItems())

	// Filter down to ollama, leave search with Enter, then request a restart.
	m = send(m, runes("/"))
	m = send(m, runes("ollama"))
	m = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.search {
		t.Fatal("enter should leave search mode")
	}
	m = send(m, runes("r"))

	if m.result.Action != ActionRestart {
		t.Fatalf("action = %v, want ActionRestart", m.result.Action)
	}
	if m.result.Item.Service.Name != "ollama" {
		t.Fatalf("restart target = %q, want ollama", m.result.Item.Service.Name)
	}
}

func TestLogsActionOnUnfilteredList(t *testing.T) {
	m := newTestModel(sampleItems())
	// Cursor starts on the first row (caddy).
	m = send(m, runes("l"))
	if m.result.Action != ActionLogs || m.result.Item.Service.Name != "caddy" {
		t.Fatalf("got %v on %q, want ActionLogs on caddy", m.result.Action, m.result.Item.Service.Name)
	}
}

func TestRunesIgnoredWhileNavigating(t *testing.T) {
	// Outside search mode, an unbound rune like "x" must not enter search or set
	// an action — it just falls through to the table.
	m := newTestModel(sampleItems())
	m = send(m, runes("x"))
	if m.search || m.result.Action != ActionNone {
		t.Fatalf("stray key changed state: search=%v action=%v", m.search, m.result.Action)
	}
}
