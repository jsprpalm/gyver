// Package docker implements the core.Adapter contract on top of the `docker`
// CLI. It shells out rather than using the Docker SDK to keep dependencies and
// the binary small, and to behave exactly like a human at the terminal.
package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jsprpalm/gyver/internal/core"
)

// Adapter manages Docker containers via the docker CLI.
type Adapter struct{}

// New constructs a Docker adapter.
func New() *Adapter { return &Adapter{} }

// Name implements core.Adapter.
func (a *Adapter) Name() string { return "docker" }

// Available reports whether the docker binary exists and the daemon answers.
func (a *Adapter) Available(ctx context.Context) bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	// `docker version` against the *server* fails fast when the daemon is down,
	// without the noisy output of `docker info`.
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	return cmd.Run() == nil
}

// ListServices returns the running containers as unified services.
func (a *Adapter) ListServices(ctx context.Context) ([]core.Service, error) {
	// Tab-separated so we can parse robustly even when a status contains spaces.
	const format = "{{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}"
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", format)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}

	var services []core.Service
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		services = append(services, core.Service{
			ID:     fields[0],
			Name:   fields[1],
			Type:   a.Name(),
			Status: fields[2],
			Ports:  parsePorts(fields[3]),
		})
	}
	return services, nil
}

// Logs prints the most recent logs for the container.
func (a *Adapter) Logs(ctx context.Context, service core.Service) error {
	cmd := exec.CommandContext(ctx, "docker", "logs", "--tail", "200", service.ID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Restart restarts the container.
func (a *Adapter) Restart(ctx context.Context, service core.Service) error {
	cmd := exec.CommandContext(ctx, "docker", "restart", service.ID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// parsePorts turns docker's port string, e.g.
//
//	"0.0.0.0:8080->80/tcp, :::8080->80/tcp"
//
// into a deduplicated slice of human-readable mappings.
func parsePorts(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var ports []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		ports = append(ports, p)
	}
	return ports
}
