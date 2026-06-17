package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jsprpalm/gyver/internal/adapters/docker"
	"github.com/jsprpalm/gyver/internal/adapters/systemd"
	"github.com/jsprpalm/gyver/internal/core"
	"github.com/jsprpalm/gyver/internal/tui"
)

// allAdapters is the single place where backends are registered. Adding PM2 or
// launchd later is a one-line change here.
func allAdapters() []core.Adapter {
	return []core.Adapter{
		docker.New(),
		systemd.New(),
	}
}

// availableAdapters returns only the adapters usable on this host right now.
func availableAdapters(ctx context.Context) []core.Adapter {
	var out []core.Adapter
	for _, a := range allAdapters() {
		if a.Available(ctx) {
			out = append(out, a)
		}
	}
	return out
}

// gatherItems collects services from every available adapter, keeping each one
// paired with the adapter that owns it so the TUI can act on a selection
// without a second lookup. Per-adapter errors are reported on stderr but do not
// abort the whole listing.
func gatherItems(ctx context.Context) []tui.Item {
	var items []tui.Item
	for _, a := range availableAdapters(ctx) {
		svcs, err := a.ListServices(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", a.Name(), err)
			continue
		}
		for _, s := range svcs {
			items = append(items, tui.Item{Adapter: a, Service: s})
		}
	}
	return items
}

// findService locates a single service by name across all available adapters.
// Matching is case-insensitive and tolerant of the ".service" suffix. Exact
// matches win; otherwise we fall back to a unique prefix match.
func findService(ctx context.Context, query string) (core.Adapter, core.Service, error) {
	needle := normalize(query)

	type hit struct {
		adapter core.Adapter
		service core.Service
	}
	var exact, prefix []hit

	for _, a := range availableAdapters(ctx) {
		svcs, err := a.ListServices(ctx)
		if err != nil {
			continue
		}
		for _, s := range svcs {
			name := normalize(s.Name)
			id := normalize(s.ID)
			switch {
			case name == needle || id == needle:
				exact = append(exact, hit{a, s})
			case strings.HasPrefix(name, needle) || strings.HasPrefix(id, needle):
				prefix = append(prefix, hit{a, s})
			}
		}
	}

	switch {
	case len(exact) == 1:
		return exact[0].adapter, exact[0].service, nil
	case len(exact) > 1:
		return nil, core.Service{}, ambiguous(query, len(exact))
	case len(prefix) == 1:
		return prefix[0].adapter, prefix[0].service, nil
	case len(prefix) > 1:
		return nil, core.Service{}, ambiguous(query, len(prefix))
	}

	return nil, core.Service{}, fmt.Errorf(
		"no service or container named %q found (try `gyver services`)", query)
}

func ambiguous(query string, n int) error {
	return fmt.Errorf("%q is ambiguous — %d services match; use a more specific name or the exact ID",
		query, n)
}

func normalize(s string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".service")
}
