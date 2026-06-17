package commands

import (
	"strings"

	"github.com/jsprpalm/gyver/internal/core"
	"github.com/jsprpalm/gyver/internal/tui"
)

// listFilter describes how `gyver services` narrows the unified service list.
type listFilter struct {
	types   []string // restrict to these adapter types; empty means all
	running bool     // only services that are actively running
	all     bool     // include internal units that are hidden by default
}

// filterServices applies f to services and returns the surviving services plus
// the number of internal units that were hidden *only* because of the default
// "hide internals" rule (i.e. units the user would otherwise have seen). That
// count lets the caller transparently report what was suppressed.
func filterServices(services []core.Service, f listFilter) (kept []core.Service, hiddenInternal int) {
	for _, s := range services {
		keep, hidden := keepService(s, f)
		if hidden {
			hiddenInternal++
		}
		if keep {
			kept = append(kept, s)
		}
	}
	return kept, hiddenInternal
}

// filterItems is the item-carrying counterpart of filterServices: it applies
// the same rules but preserves the adapter paired with each service so the TUI
// can act on a selection.
func filterItems(items []tui.Item, f listFilter) (kept []tui.Item, hiddenInternal int) {
	for _, it := range items {
		keep, hidden := keepService(it.Service, f)
		if hidden {
			hiddenInternal++
		}
		if keep {
			kept = append(kept, it)
		}
	}
	return kept, hiddenInternal
}

// keepService decides a single service against the filter. hiddenInternal is
// true only when the service was dropped *purely* by the default "hide
// internals" rule (i.e. the user would otherwise have seen it), so callers can
// transparently report what was suppressed.
func keepService(s core.Service, f listFilter) (keep, hiddenInternal bool) {
	if len(f.types) > 0 && !containsFold(f.types, s.Type) {
		return false, false
	}
	if f.running && !isRunning(s) {
		return false, false
	}
	if !f.all && isInternal(s) {
		return false, true
	}
	return true, false
}

// isInternal reports whether a service is part of the init system's own
// plumbing rather than something the user deployed. Today that means systemd's
// own "systemd-*" units (journald, udevd, logind, resolved, …).
func isInternal(s core.Service) bool {
	return s.Type == "systemd" && strings.HasPrefix(strings.ToLower(s.Name), "systemd-")
}

// isRunning reports whether a service is currently up, interpreting each
// adapter's native status string.
func isRunning(s core.Service) bool {
	st := strings.ToLower(s.Status)
	switch s.Type {
	case "docker":
		return strings.HasPrefix(st, "up")
	case "systemd":
		return strings.Contains(st, "active (running)")
	default:
		return strings.Contains(st, "run") || strings.Contains(st, "active")
	}
}

func containsFold(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(strings.TrimSpace(h), needle) {
			return true
		}
	}
	return false
}
