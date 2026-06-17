// This file adds an LLM-backed Provider for `gyver how`. It implements the
// same recipes.Provider interface as the offline LocalProvider, so the command
// layer can swap between them without changes. It calls the Claude Messages API
// (Anthropic Go SDK) and asks for a single shell command plus a short
// explanation, returned as strict JSON that we parse into a Suggestion.
package recipes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// DefaultModel is the Claude model used when GYVER_HOW_MODEL is not set.
const DefaultModel = anthropic.ModelClaudeOpus4_8

// AnthropicProvider answers questions with a single Claude Messages API call.
type AnthropicProvider struct {
	client anthropic.Client
	model  anthropic.Model
	goos   string
}

// ErrNoAPIKey indicates that no Anthropic API key was configured through any of
// the supported mechanisms. The command layer treats this as "AI unavailable,
// use the offline matcher" rather than a hard error.
var ErrNoAPIKey = errors.New("no Anthropic API key configured")

// NewAnthropicProvider builds a provider for the current OS, resolving the API
// key via ResolveAPIKey. It returns ErrNoAPIKey when no key is configured. The
// model can be overridden with GYVER_HOW_MODEL (e.g. "claude-haiku-4-5" for a
// faster, cheaper answer).
func NewAnthropicProvider() (*AnthropicProvider, error) {
	key, err := ResolveAPIKey()
	if err != nil {
		return nil, err
	}

	model := anthropic.Model(DefaultModel)
	if m := strings.TrimSpace(os.Getenv("GYVER_HOW_MODEL")); m != "" {
		model = anthropic.Model(m)
	}
	return &AnthropicProvider{
		client: anthropic.NewClient(option.WithAPIKey(key)),
		model:  model,
		goos:   runtime.GOOS,
	}, nil
}

// ResolveAPIKey finds the Anthropic API key, in order of precedence:
//
//  1. GYVER_ANTHROPIC_API_KEY     — a gyver-specific key, so you can give gyver
//     its own credential without touching a shared ANTHROPIC_API_KEY.
//  2. GYVER_ANTHROPIC_API_KEY_CMD — a shell command whose stdout is the key
//     (e.g. "pass show anthropic/gyver", "op read op://vault/gyver/key",
//     "security find-generic-password -s gyver -w"). Keeps the key out of the
//     environment and out of plaintext config — pull it from a secret manager.
//  3. ANTHROPIC_API_KEY           — the standard variable, as a fallback.
//
// It returns ErrNoAPIKey if none are set.
func ResolveAPIKey() (string, error) {
	if k := strings.TrimSpace(os.Getenv("GYVER_ANTHROPIC_API_KEY")); k != "" {
		return k, nil
	}
	if cmd := strings.TrimSpace(os.Getenv("GYVER_ANTHROPIC_API_KEY_CMD")); cmd != "" {
		key, err := runKeyCmd(cmd)
		if err != nil {
			return "", err
		}
		if key == "" {
			return "", fmt.Errorf("GYVER_ANTHROPIC_API_KEY_CMD (%s) produced no output", cmd)
		}
		return key, nil
	}
	if k := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); k != "" {
		return k, nil
	}
	return "", ErrNoAPIKey
}

// runKeyCmd executes the key-fetch command through a shell (so users can write
// pipes/flags freely) and returns its trimmed stdout. A short timeout keeps a
// hung secret-manager from blocking the CLI indefinitely.
func runKeyCmd(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	out, err := exec.CommandContext(ctx, shell, "-c", command).Output()
	if err != nil {
		return "", fmt.Errorf("running GYVER_ANTHROPIC_API_KEY_CMD (%s): %w", command, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// suggestion is the JSON shape we ask Claude to return.
type llmSuggestion struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
}

const howSystemPrompt = `You are gyver, a command-line assistant for developers, sysadmins and homelab users.
Given a plain-language question, reply with the single best shell command that answers it, plus a one or two sentence explanation.

Rules:
- Target the user's operating system (given below). Prefer commands that ship with the OS.
- Return exactly ONE command. If several steps are needed, combine them with pipes.
- Do not wrap the command in markdown, backticks, or prose.
- If the question is not about running a shell command, set "command" to an empty string and explain why in "explanation".
- Respond with ONLY a JSON object of the form {"command": "...", "explanation": "..."} and nothing else.`

// Suggest implements Provider using the Claude Messages API.
func (p *AnthropicProvider) Suggest(ctx context.Context, question string) (Suggestion, error) {
	osName := osDisplayName(p.goos)
	userMsg := fmt.Sprintf("Operating system: %s\nQuestion: %s", osName, question)

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: howSystemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	}, option.WithRequestTimeout(0))
	if err != nil {
		return Suggestion{}, fmt.Errorf("anthropic request failed: %w", err)
	}

	var raw strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			raw.WriteString(t.Text)
		}
	}

	parsed, err := parseLLMSuggestion(raw.String())
	if err != nil {
		return Suggestion{}, err
	}

	return Suggestion{
		Command:     parsed.Command,
		Explanation: parsed.Explanation,
		Source:      "anthropic:" + string(p.model),
	}, nil
}

// parseLLMSuggestion extracts the JSON object from the model's reply. The
// system prompt asks for bare JSON, but we tolerate a stray markdown fence or
// surrounding prose by slicing to the outermost braces.
func parseLLMSuggestion(text string) (llmSuggestion, error) {
	s := strings.TrimSpace(text)
	if i := strings.IndexByte(s, '{'); i >= 0 {
		if j := strings.LastIndexByte(s, '}'); j >= i {
			s = s[i : j+1]
		}
	}

	var out llmSuggestion
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return llmSuggestion{}, fmt.Errorf("could not parse model response as JSON: %w", err)
	}
	return out, nil
}

func osDisplayName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS (darwin)"
	case "linux":
		return "Linux"
	default:
		return goos
	}
}
