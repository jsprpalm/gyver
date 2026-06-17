package core

import "context"

// Adapter is the contract every backend (Docker, systemd, …) must satisfy.
//
// Implementations should be cheap to construct; all real work happens inside
// the context-aware methods so callers can apply timeouts and cancellation.
type Adapter interface {
	// Name returns the adapter's stable identifier, e.g. "docker".
	Name() string

	// Available reports whether this adapter can be used on the current host
	// right now (correct OS, binary installed, daemon reachable, …). It must
	// never panic and should fail fast.
	Available(ctx context.Context) bool

	// ListServices returns the services this adapter currently knows about.
	ListServices(ctx context.Context) ([]Service, error)

	// Logs streams (or prints recent) logs for the given service to stdout.
	Logs(ctx context.Context, service Service) error

	// Restart restarts the given service.
	Restart(ctx context.Context, service Service) error
}
