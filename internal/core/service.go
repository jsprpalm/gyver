package core

// Service is the unified representation of anything gyver can manage: a Docker
// container, a systemd unit, and (in the future) a PM2 process or launchd job.
//
// Adapters are responsible for translating their native concepts into this
// shape so that the rest of gyver can treat everything uniformly.
type Service struct {
	// ID is the stable identifier used to address the service with its native
	// tooling (e.g. a Docker container ID, or a systemd unit name).
	ID string

	// Name is the human-friendly name used for matching on the command line.
	Name string

	// Type is the adapter that owns this service, e.g. "docker" or "systemd".
	Type string

	// Status is a short, human-readable status string, e.g. "running" or
	// "active (running)".
	Status string

	// Ports holds any published/listening ports, when the adapter can discover
	// them. May be empty.
	Ports []string
}
