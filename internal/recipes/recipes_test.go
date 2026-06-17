package recipes

import (
	"context"
	"testing"
)

func TestLocalProviderMatches(t *testing.T) {
	p := NewLocalProvider()
	ctx := context.Background()

	cases := []struct {
		question string
		wantPart string // substring expected in the suggested command
	}{
		{"how do I list open ports?", "ss"}, // ss or lsof depending on OS; ss on linux
		{"show me the largest files here", "du"},
		{"how much disk space is left", "df"},
		{"what are the running processes", "ps"},
		{"do a dns lookup for a domain", "dig"},
	}

	for _, tc := range cases {
		s, err := p.Suggest(ctx, tc.question)
		if err != nil {
			t.Fatalf("Suggest(%q) error: %v", tc.question, err)
		}
		if s.Command == "" {
			t.Errorf("Suggest(%q) returned no command", tc.question)
			continue
		}
		// On macOS the open-ports recipe is lsof, not ss; accept either there.
		if tc.wantPart == "ss" {
			if !contains(s.Command, "ss") && !contains(s.Command, "lsof") {
				t.Errorf("Suggest(%q) = %q, want ss/lsof", tc.question, s.Command)
			}
			continue
		}
		if !contains(s.Command, tc.wantPart) {
			t.Errorf("Suggest(%q) = %q, want substring %q", tc.question, s.Command, tc.wantPart)
		}
	}
}

func TestLocalProviderNoMatch(t *testing.T) {
	p := NewLocalProvider()
	s, err := p.Suggest(context.Background(), "xyzzy plugh nothing relevant")
	if err != nil {
		t.Fatal(err)
	}
	if s.Command != "" {
		t.Errorf("expected empty command for no match, got %q", s.Command)
	}
	if s.Explanation == "" {
		t.Error("expected a helpful explanation for no match")
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 ||
		(len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
