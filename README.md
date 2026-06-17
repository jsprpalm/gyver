```
┌─────────────────────────────────────────────┐
│   ██████╗ ██╗   ██╗██╗   ██╗███████╗██████╗ │
│  ██╔════╝ ╚██╗ ██╔╝██║   ██║██╔════╝██╔══██╗│
│  ██║  ███╗ ╚████╔╝ ██║   ██║█████╗  ██████╔╝│
│  ██║   ██║  ╚██╔╝  ╚██╗ ██╔╝██╔══╝  ██╔══██╗│
│  ╚██████╔╝   ██║    ╚████╔╝ ███████╗██║  ██║│
│   ╚═════╝    ╚═╝     ╚═══╝  ╚══════╝╚═╝  ╚═╝│
├─────────────────────────────────────────────┤
│  one command. many backends. zero panic.    │
└─────────────────────────────────────────────┘
```

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24%2B-00ADD8.svg)](https://go.dev/)

> A universal command layer for developers, sysadmins and homelab users.

Stop remembering whether a thing is managed by **Docker**, **systemd**, **PM2**
or **launchd**. Just talk to `gyver`:

```bash
gyver list
gyver logs adguard
gyver restart caddy
gyver ports
gyver how "how do I list open ports?"
```

Gyver detects which backends ("adapters") are available on the current host and
presents everything through one consistent interface. The MVP ships with the
**Docker** and **systemd** adapters; the adapter architecture is built so PM2,
launchd and others can be added without touching the command layer.

## Status

MVP. Supported today:

| Command          | What it does                                                            |
| ---------------- | ---------------------------------------------------------------------- |
| `gyver list`     | Unified table of containers + units (TUI or `--plain`); filter with `--all`/`--running`/`--type`. |
| `gyver logs`     | Recent logs via `docker logs` / `journalctl -u`.                       |
| `gyver restart`  | Restart with confirmation via `docker restart` / `systemctl restart`.  |
| `gyver ports`    | Listening ports via `ss` (Linux) or `lsof` (macOS / fallback).         |
| `gyver how`      | AI-backed (Claude) suggestion of a shell command for a question, with an offline recipe matcher as fallback. |
| `gyver alias`    | Save, list, run and remove command shortcuts (e.g. the ones `gyver how` suggests). |

Works on **macOS** and **Linux**. On macOS the systemd adapter simply reports
itself unavailable. If Docker (or systemd) isn't present, gyver degrades
gracefully and uses whatever it can.

## Install

Requires Go 1.24+.

```bash
git clone https://github.com/jsprpalm/gyver.git
cd gyver
make tidy          # fetch dependencies & generate go.sum (run once)
make build         # -> ./bin/gyver
make install-local # -> ~/.local/bin/gyver  (make sure it's on your PATH)
```

Or run without installing:

```bash
make run ARGS="list --plain"
```

## Usage

### `gyver list`

Interactive Bubble Tea table by default; `↑/↓` (or `j/k`) to move, `q` to quit.

```bash
gyver list
```

Internal systemd units (`systemd-*` — journald, udevd, logind, …) are **hidden
by default**; gyver prints a one-line note on stderr saying how many were
suppressed. Flags let you slice the list:

```bash
gyver list --all          # -a: include the hidden internal systemd-* units
gyver list --running      # -r: only services that are actively running
gyver list --type docker  # -t: only one adapter (repeatable: --type docker --type systemd)
gyver list -r -t systemd  # combine freely
```

Script-friendly output for pipes and `grep`:

```bash
gyver list --plain
# TYPE     NAME      ID            STATUS           PORTS
# docker   adguard   3f9a1c2b7d8e  Up 2 hours       0.0.0.0:3000->3000/tcp
# systemd  caddy     caddy         active (running) -
```

> `gyver list` automatically falls back to plain text when stdout is not a
> terminal, so piping always works even without `--plain`. The "N internal
> units hidden" note goes to stderr, so it never pollutes a piped list.

### `gyver logs <name>`

```bash
gyver logs adguard     # docker logs --tail 200 <container>
gyver logs caddy       # journalctl -u caddy.service -n 200
```

Names are matched case-insensitively across all adapters, and the `.service`
suffix is optional for systemd units.

### `gyver restart <name>`

Asks before doing anything:

```bash
gyver restart caddy
# Restart systemd service "caddy" (caddy.service)? [y/N]
```

Skip the prompt in scripts with `-y/--yes`:

```bash
gyver restart caddy --yes
```

> systemd restarts may require privileges — run with `sudo` if `systemctl`
> reports a permission error.

### `gyver ports`

```bash
gyver ports
# PROTO  ADDRESS         PROCESS
# tcp    0.0.0.0:22      sshd (pid 812)
# tcp    127.0.0.1:3000  docker-proxy (pid 4410)
```

- **Linux:** prefers `ss -tulpn`, falls back to `lsof -i -P -n`.
- **macOS:** uses `lsof -i -P -n`.

### `gyver how "<question>"`

Ask in plain English and gyver suggests a command:

```bash
gyver how "how do I list open ports?"
#   suggested command:
#   ss -tulpn
#
#   explanation:
#   Lists processes that are listening on TCP/UDP ports. Tip: `gyver ports` does this cross-platform.
#
#   save as alias: gyver alias add <name> "ss -tulpn"
#   source: anthropic:claude-opus-4-8
```

**Two providers**, chosen automatically:

- **AI (Claude)** — used when `ANTHROPIC_API_KEY` is set. gyver sends the
  question and your OS to the Claude Messages API and returns a tailored
  command. The `source:` line shows the model used.
- **Offline recipes** — a built-in keyword matcher with a handful of recipes
  (**open ports**, **largest files**, **disk usage**, **running processes**,
  **DNS lookup**). Used when no API key is set, when you pass `--local`, or
  automatically if the AI call fails (a one-line note goes to stderr, so the
  suggestion on stdout stays pipe-clean).

```bash
export ANTHROPIC_API_KEY=sk-ant-...      # enable the AI provider
gyver how "find files changed in the last 10 minutes"

gyver how --local "how do I list open ports?"   # force the offline matcher
```

Configuration:

| Variable                       | Purpose                                                                 |
| ------------------------------ | ----------------------------------------------------------------------- |
| `GYVER_ANTHROPIC_API_KEY`      | A gyver-specific API key. Use this to give gyver its own credential without touching a shared `ANTHROPIC_API_KEY`. |
| `GYVER_ANTHROPIC_API_KEY_CMD`  | A shell command whose stdout is the key — pull it from a secret manager instead of an env var. |
| `ANTHROPIC_API_KEY`            | The standard key variable, used as a fallback.                          |
| `GYVER_HOW_MODEL`              | Override the model. Defaults to `claude-opus-4-8`; set e.g. `claude-haiku-4-5` for faster, cheaper answers. |

**Providing the key without a global `export`.** gyver resolves the key in the
order above (first match wins), so you have a few options that don't pollute
your shell environment:

```bash
# Per-invocation — set it only for this one command:
ANTHROPIC_API_KEY=sk-ant-... gyver how "list open ports"

# Per-project — scope it to a directory with direnv (.envrc) or a Makefile:
echo 'export GYVER_ANTHROPIC_API_KEY=sk-ant-...' >> .envrc && direnv allow

# Secret manager — no key on disk or in the environment at all:
export GYVER_ANTHROPIC_API_KEY_CMD='pass show anthropic/gyver'
export GYVER_ANTHROPIC_API_KEY_CMD='op read op://Personal/gyver/credential'
export GYVER_ANTHROPIC_API_KEY_CMD='security find-generic-password -s gyver -w'  # macOS Keychain
```

The `_CMD` variant runs through your shell each time a key is needed, so its
output never has to be stored in plaintext. Because gyver reads its own
`GYVER_ANTHROPIC_*` variables first, you can keep different keys for different
tools without them clobbering each other.

The `recipes.Provider` interface backs both providers, so either can be used
anywhere a provider is expected.

### `gyver alias`

Save command shortcuts — including the ones `gyver how` suggests — and run them
later. Aliases are stored as JSON under your config dir
(`<user-config-dir>/gyver/aliases.json`; override with `GYVER_ALIASES_FILE`).

```bash
gyver alias add ports "ss -tulpn"   # quote the command so it's one argument
gyver alias list                    # NAME / COMMAND table (ls works too)
gyver alias run ports               # runs it via $SHELL -c
gyver alias run ports -n            # extra args are appended to the command
gyver alias remove ports            # rm works too
```

`gyver alias add` refuses to overwrite an existing name unless you pass
`--force`. The fastest way to save a suggestion is straight from `how`:

```bash
gyver how --save ports "how do I list open ports?"
# ...
#   saved as alias "ports" — run it with: gyver alias run ports
```

> `gyver alias run` executes saved commands through your shell (`$SHELL -c`, or
> `sh` if unset) because they're shell snippets that may contain pipes. Only run
> aliases you trust — they are arbitrary shell commands.

## Architecture

```
cmd/gyver/main.go                  # entrypoint
internal/core/service.go           # unified Service type
internal/core/adapter.go           # Adapter interface
internal/adapters/docker/docker.go # Docker adapter (docker CLI)
internal/adapters/systemd/systemd.go # systemd adapter (systemctl/journalctl)
internal/commands/                 # one file per Cobra subcommand
  ├─ root.go      # command wiring
  ├─ adapters.go  # adapter registry + service lookup helpers
  ├─ list.go logs.go restart.go ports.go how.go alias.go
internal/tui/list.go               # Bubble Tea + Lip Gloss table
internal/recipes/recipes.go        # `how` Provider interface + offline recipe matcher
internal/recipes/anthropic.go      # `how` Claude-backed Provider (Anthropic Go SDK)
internal/aliases/aliases.go        # `alias` JSON-backed shortcut store
```

Everything an adapter can do is captured by one interface:

```go
type Adapter interface {
    Name() string
    Available(ctx context.Context) bool
    ListServices(ctx context.Context) ([]Service, error)
    Logs(ctx context.Context, service Service) error
    Restart(ctx context.Context, service Service) error
}
```

### Adding a new adapter

1. Create `internal/adapters/<name>/<name>.go` implementing `core.Adapter`.
2. Register it in `allAdapters()` in `internal/commands/adapters.go`.

That's it — `list`, `logs`, `restart` and name-matching pick it up for free.

## Design notes

- Shells out with `exec.CommandContext` (every call is cancellable/timeout-able).
- Avoids `sh -c`; commands run directly with explicit arguments.
- Adapters fail fast in `Available()` so an absent backend is invisible.

## Development

```bash
make test   # go test ./...
make fmt    # go fmt ./...
make vet    # go vet ./...
```

## License

[MIT](LICENSE) © 2026 Jesper Palm.
