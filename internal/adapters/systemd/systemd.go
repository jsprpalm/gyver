// Package systemd implements the core.Adapter contract on top of systemctl and
// journalctl. It is only available on Linux hosts running systemd.
package systemd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jsprpalm/gyver/internal/core"
)

// Adapter manages systemd service units.
type Adapter struct{}

// New constructs a systemd adapter.
func New() *Adapter { return &Adapter{} }

// Name implements core.Adapter.
func (a *Adapter) Name() string { return "systemd" }

// Available reports whether we are on Linux with systemctl present. On macOS
// this is always false, which is exactly what we want.
func (a *Adapter) Available(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	// Confirm systemd is actually the init system / reachable.
	cmd := exec.CommandContext(ctx, "systemctl", "is-system-running")
	// is-system-running exits non-zero in "degraded"/"maintenance" states but
	// still prints a word; we only care that the command ran at all.
	out, _ := cmd.Output()
	return len(strings.TrimSpace(string(out))) > 0
}

// ListServices returns the currently loaded service units.
func (a *Adapter) ListServices(ctx context.Context) ([]core.Service, error) {
	cmd := exec.CommandContext(ctx, "systemctl",
		"list-units", "--type=service", "--no-legend", "--no-pager", "--plain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units: %w", err)
	}

	var services []core.Service
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		// Columns: UNIT LOAD ACTIVE SUB DESCRIPTION
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		unit := fields[0]
		services = append(services, core.Service{
			ID:     unit,
			Name:   strings.TrimSuffix(unit, ".service"),
			Type:   a.Name(),
			Status: fmt.Sprintf("%s (%s)", fields[2], fields[3]),
			// systemd does not expose listening ports per-unit; use `gyver ports`.
			Ports: nil,
		})
	}
	return services, nil
}

// Logs prints the most recent journal entries for the unit.
func (a *Adapter) Logs(ctx context.Context, service core.Service) error {
	cmd := exec.CommandContext(ctx, "journalctl", "-u", service.ID, "-n", "200", "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Restart restarts the unit. This may require elevated privileges; any
// permission error from systemctl is surfaced to the caller.
func (a *Adapter) Restart(ctx context.Context, service core.Service) error {
	cmd := exec.CommandContext(ctx, "systemctl", "restart", service.ID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
