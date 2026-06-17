package commands

import (
	"testing"

	"github.com/jsprpalm/gyver/internal/core"
)

func sampleServices() []core.Service {
	return []core.Service{
		{Type: "docker", Name: "caddy", Status: "Up 7 days"},
		{Type: "docker", Name: "frigate", Status: "Up 7 days (healthy)"},
		{Type: "systemd", Name: "ollama", Status: "active (running)"},
		{Type: "systemd", Name: "backup-to-nas", Status: "failed (failed)"},
		{Type: "systemd", Name: "cloud-init", Status: "active (exited)"},
		{Type: "systemd", Name: "systemd-journald", Status: "active (running)"},
		{Type: "systemd", Name: "systemd-udevd", Status: "active (running)"},
	}
}

func names(services []core.Service) []string {
	out := make([]string, len(services))
	for i, s := range services {
		out[i] = s.Name
	}
	return out
}

func TestFilterDefaultHidesInternal(t *testing.T) {
	kept, hidden := filterServices(sampleServices(), listFilter{})
	if hidden != 2 {
		t.Errorf("hiddenInternal = %d, want 2", hidden)
	}
	for _, s := range kept {
		if isInternal(s) {
			t.Errorf("internal unit %q leaked into default output", s.Name)
		}
	}
	if len(kept) != 5 {
		t.Errorf("kept %d, want 5: %v", len(kept), names(kept))
	}
}

func TestFilterAllIncludesInternal(t *testing.T) {
	kept, hidden := filterServices(sampleServices(), listFilter{all: true})
	if hidden != 0 {
		t.Errorf("hiddenInternal = %d, want 0 when --all", hidden)
	}
	if len(kept) != 7 {
		t.Errorf("kept %d, want 7", len(kept))
	}
}

func TestFilterRunning(t *testing.T) {
	// --running hides exited/failed; --all so internals are eligible too.
	kept, _ := filterServices(sampleServices(), listFilter{running: true, all: true})
	got := names(kept)
	want := map[string]bool{
		"caddy": true, "frigate": true, "ollama": true,
		"systemd-journald": true, "systemd-udevd": true,
	}
	if len(got) != len(want) {
		t.Fatalf("running set = %v, want %v", got, want)
	}
	for _, n := range got {
		if !want[n] {
			t.Errorf("unexpected running service %q", n)
		}
	}
}

func TestFilterByType(t *testing.T) {
	kept, _ := filterServices(sampleServices(), listFilter{types: []string{"docker"}, all: true})
	if len(kept) != 2 {
		t.Fatalf("docker-only = %v, want 2", names(kept))
	}
	for _, s := range kept {
		if s.Type != "docker" {
			t.Errorf("non-docker %q leaked through type filter", s.Name)
		}
	}
}

func TestFilterByTypeIsCaseInsensitive(t *testing.T) {
	kept, _ := filterServices(sampleServices(), listFilter{types: []string{"Docker"}, all: true})
	if len(kept) != 2 {
		t.Errorf("case-insensitive type filter = %v, want 2", names(kept))
	}
}

func TestHiddenCountRespectsOtherFilters(t *testing.T) {
	// When restricting to docker, no systemd internals could have shown anyway,
	// so the "hidden" count must be 0 (nothing was actually suppressed).
	_, hidden := filterServices(sampleServices(), listFilter{types: []string{"docker"}})
	if hidden != 0 {
		t.Errorf("hiddenInternal = %d, want 0 when type excludes systemd", hidden)
	}
}
