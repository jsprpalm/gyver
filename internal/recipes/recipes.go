// Package recipes powers `gyver how`. For the MVP it is a small, offline
// keyword matcher over a handful of hardcoded recipes. The Provider interface
// is deliberately shaped so an LLM-backed provider can be dropped in later
// without touching the command layer.
package recipes

import (
	"context"
	"runtime"
	"sort"
	"strings"
)

// Suggestion is the result of asking "how do I …?".
type Suggestion struct {
	// Command is the shell command we suggest running.
	Command string
	// Explanation is a one or two sentence description of what it does.
	Explanation string
	// Source identifies where the suggestion came from, e.g. "local-recipes"
	// or (later) "anthropic". Useful for UX and debugging.
	Source string
}

// Provider turns a natural-language question into a Suggestion. The local
// recipe matcher implements it today; an LLM client can implement it tomorrow.
type Provider interface {
	Suggest(ctx context.Context, question string) (Suggestion, error)
}

// Recipe is a single hardcoded answer plus the keywords that should match it.
type Recipe struct {
	Name        string
	Keywords    []string
	Command     string
	Explanation string
}

// LocalProvider answers questions using the built-in recipe table only.
type LocalProvider struct {
	recipes []Recipe
}

// NewLocalProvider builds a LocalProvider seeded with the default recipes for
// the current operating system.
func NewLocalProvider() *LocalProvider {
	return &LocalProvider{recipes: defaultRecipes(runtime.GOOS)}
}

// Suggest implements Provider. It scores every recipe by how many of its
// keywords appear in the question and returns the best match.
func (p *LocalProvider) Suggest(_ context.Context, question string) (Suggestion, error) {
	q := strings.ToLower(question)
	words := tokenize(q)

	type scored struct {
		recipe Recipe
		score  int
	}
	var ranked []scored
	for _, r := range p.recipes {
		score := 0
		for _, kw := range r.Keywords {
			kw = strings.ToLower(kw)
			// Phrase match (e.g. "open ports") counts strongest; otherwise
			// fall back to a per-word match.
			if strings.Contains(q, kw) {
				score += 2
			} else if words[kw] {
				score++
			}
		}
		if score > 0 {
			ranked = append(ranked, scored{recipe: r, score: score})
		}
	}

	if len(ranked) == 0 {
		return Suggestion{
			Command: "",
			Explanation: "No local recipe matched that question yet. " +
				"Try keywords like: open ports, largest files, disk usage, running processes, DNS lookup.",
			Source: "local-recipes",
		}, nil
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	best := ranked[0].recipe
	return Suggestion{
		Command:     best.Command,
		Explanation: best.Explanation,
		Source:      "local-recipes",
	}, nil
}

func tokenize(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		out[w] = true
	}
	return out
}

// defaultRecipes returns OS-appropriate recipes. Where the right command
// differs between macOS and Linux, we pick the native one.
func defaultRecipes(goos string) []Recipe {
	openPorts := "ss -tulpn"
	if goos == "darwin" {
		openPorts = "lsof -i -P -n | grep LISTEN"
	}

	return []Recipe{
		{
			Name:        "open-ports",
			Keywords:    []string{"open ports", "listening", "ports", "port", "sockets"},
			Command:     openPorts,
			Explanation: "Lists processes that are listening on TCP/UDP ports. Tip: `gyver ports` does this cross-platform.",
		},
		{
			Name:        "largest-files",
			Keywords:    []string{"largest files", "biggest files", "large files", "biggest", "biggest file"},
			Command:     "du -ah . | sort -rh | head -n 20",
			Explanation: "Shows the 20 largest files and directories under the current path, biggest first.",
		},
		{
			Name:        "disk-usage",
			Keywords:    []string{"disk usage", "disk space", "free space", "disk", "storage", "df"},
			Command:     "df -h",
			Explanation: "Shows free and used space for each mounted filesystem in human-readable units.",
		},
		{
			Name:        "running-processes",
			Keywords:    []string{"running processes", "processes", "process", "top", "cpu usage", "memory usage"},
			Command:     "ps aux --sort=-%cpu | head -n 20",
			Explanation: "Lists the top processes by CPU usage. Use `top` or `htop` for a live view.",
		},
		{
			Name:        "dns-lookup",
			Keywords:    []string{"dns", "dns lookup", "resolve", "nslookup", "dig", "domain", "ip address of"},
			Command:     "dig +short example.com",
			Explanation: "Resolves a domain to its IP addresses. Replace example.com with your domain (use `dig A`/`MX`/`TXT` for record types).",
		},
	}
}
