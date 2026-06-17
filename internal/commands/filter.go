package commands

import (
	"strings"

	"github.com/jsprpalm/gyver/internal/core"
)

// listFilter describes how `gyver list` narrows the unified service list.
type listFilter struct {
	types   []string // restrict to these adapter types; empty means all
	running bool      // only services that are actively running
	all     bool      // include internal units that are hidden by default
}

// filterServices applies f to services and returns the surviving services plus
// the number of internal units that were hidden *only* because of the default
// "hide internals" rule (i.e. units the user would otherwise have seen). That
// count lets the caller transparently report what was suppressed.
func filterServices(services []core.Service, f listFilter) (kept []core.Service, hiddenInternal int) {
	for _, s := range services {
		if len(f.types) > 0 && !containsFold(f.types, s.Type) {
			continue
		}
		if f.running && !isRunning(s) {
			continue
		}
		if !f.all && isInternal(s) {
			hiddenInternal++
			continue
		}
		kept = append(kept, s)
	}
	return kept, hiddenInternal
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
