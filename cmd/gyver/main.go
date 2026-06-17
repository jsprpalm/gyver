// Command gyver is a universal command layer that lets you list, inspect and
// control services regardless of whether they are managed by Docker, systemd,
// or (later) PM2 / launchd.
package main

import "github.com/jsprpalm/gyver/internal/commands"

func main() {
	commands.Execute()
}
