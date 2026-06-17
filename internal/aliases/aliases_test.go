package aliases

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "aliases.json"))
}

func TestAddListGet(t *testing.T) {
	s := newTestStore(t)

	if err := s.Add("ports", "ss -tulpn", false); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add("disk", "df -h", false); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List len = %d, want 2", len(got))
	}
	// Sorted by name: disk before ports.
	if got[0].Name != "disk" || got[1].Name != "ports" {
		t.Errorf("List order = %v, want [disk ports]", []string{got[0].Name, got[1].Name})
	}

	a, err := s.Get("ports")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if a.Command != "ss -tulpn" {
		t.Errorf("Get command = %q, want %q", a.Command, "ss -tulpn")
	}
}

func TestAddDuplicateRequiresForce(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("ports", "ss -tulpn", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("ports", "lsof -i", false); err == nil {
		t.Fatal("expected error overwriting without force")
	}
	if err := s.Add("ports", "lsof -i", true); err != nil {
		t.Fatalf("Add with force: %v", err)
	}
	a, _ := s.Get("ports")
	if a.Command != "lsof -i" {
		t.Errorf("after force = %q, want %q", a.Command, "lsof -i")
	}
}

func TestAddValidation(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("", "df -h", false); err == nil {
		t.Error("expected error for empty name")
	}
	if err := s.Add("bad name", "df -h", false); err == nil {
		t.Error("expected error for name with whitespace")
	}
	if err := s.Add("ok", "   ", false); err == nil {
		t.Error("expected error for empty command")
	}
}

func TestRemove(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("ports", "ss -tulpn", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("ports"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := s.Remove("ports"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove missing = %v, want ErrNotFound", err)
	}
}

func TestListEmptyMissingFile(t *testing.T) {
	s := newTestStore(t) // file never created
	got, err := s.List()
	if err != nil {
		t.Fatalf("List on missing file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List len = %d, want 0", len(got))
	}
}

func TestGetMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get missing = %v, want ErrNotFound", err)
	}
}
