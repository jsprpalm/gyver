package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// portRow is a single listening socket, normalized across `ss` and `lsof`.
type portRow struct {
	Proto   string
	Address string
	Process string
}

func newPortsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ports",
		Short: "Show listening ports (cross-platform best effort)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			rows, err := listPorts(ctx)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				fmt.Fprintln(os.Stderr, "no listening ports found")
				return nil
			}
			return printPorts(rows)
		},
	}
}

// listPorts picks the best available tool for the host and parses its output.
func listPorts(ctx context.Context) ([]portRow, error) {
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("ss"); err == nil {
			out, err := exec.CommandContext(ctx, "ss", "-tulpn").Output()
			if err == nil {
				return parseSS(string(out)), nil
			}
		}
	}
	// macOS, or Linux without `ss`: fall back to lsof.
	if _, err := exec.LookPath("lsof"); err == nil {
		out, err := exec.CommandContext(ctx, "lsof", "-i", "-P", "-n").Output()
		if err == nil {
			return parseLsof(string(out)), nil
		}
	}
	return nil, fmt.Errorf("could not list ports: neither `ss` nor `lsof` is available")
}

var ssProcessRe = regexp.MustCompile(`"([^"]+)",pid=(\d+)`)

// parseSS parses `ss -tulpn` output.
func parseSS(out string) []portRow {
	var rows []portRow
	for i, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if i == 0 || len(fields) < 5 {
			continue // header or short line
		}
		// Columns: Netid State Recv-Q Send-Q Local-Address:Port Peer:Port [Process]
		proto := fields[0]
		local := fields[4]
		process := "-"
		if m := ssProcessRe.FindStringSubmatch(line); m != nil {
			process = fmt.Sprintf("%s (pid %s)", m[1], m[2])
		}
		rows = append(rows, portRow{Proto: proto, Address: local, Process: process})
	}
	return rows
}

// parseLsof parses `lsof -i -P -n`, keeping only listening sockets.
func parseLsof(out string) []portRow {
	var rows []portRow
	for i, line := range strings.Split(out, "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		// UDP rows have no "(LISTEN)" marker; keep TCP listeners plus all UDP.
		isUDP := strings.Contains(line, "UDP")
		if !strings.Contains(line, "(LISTEN)") && !isUDP {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		command := fields[0]
		pid := fields[1]
		proto := fields[7] // TCP / UDP
		address := fields[8]
		rows = append(rows, portRow{
			Proto:   proto,
			Address: address,
			Process: fmt.Sprintf("%s (pid %s)", command, pid),
		})
	}
	return rows
}

func printPorts(rows []portRow) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PROTO\tADDRESS\tPROCESS")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Proto, r.Address, r.Process)
	}
	return w.Flush()
}
