// Package aliases persists user-saved shell command shortcuts for
// `gyver alias`. An alias is just a name mapped to a command string (often one
// suggested by `gyver how`). Storage is a small JSON file in the user's config
// directory; the Store is path-injectable so it can be exercised in tests.
package aliases

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Alias is a single named command shortcut.
type Alias struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// Store reads and writes aliases to a JSON file.
type Store struct {
	path string
}

// ErrNotFound is returned when an alias name does not exist.
var ErrNotFound = errors.New("alias not found")

// DefaultPath resolves the aliases file location. It honours GYVER_ALIASES_FILE
// for overrides (handy for tests and scripting) and otherwise falls back to
// <user-config-dir>/gyver/aliases.json.
func DefaultPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("GYVER_ALIASES_FILE")); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config dir: %w", err)
	}
	return filepath.Join(dir, "gyver", "aliases.json"), nil
}

// NewStore returns a Store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultStore returns a Store at the default path.
func DefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

// List returns all aliases sorted by name. A missing file is treated as empty.
func (s *Store) List() ([]Alias, error) {
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Alias, 0, len(all))
	for name, cmd := range all {
		out = append(out, Alias{Name: name, Command: cmd})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Get returns the alias with the given name, or ErrNotFound.
func (s *Store) Get(name string) (Alias, error) {
	all, err := s.load()
	if err != nil {
		return Alias{}, err
	}
	cmd, ok := all[name]
	if !ok {
		return Alias{}, ErrNotFound
	}
	return Alias{Name: name, Command: cmd}, nil
}

// Add stores an alias. With replace=false it refuses to overwrite an existing
// name; with replace=true it upserts. The name must be a single token (no
// whitespace) so aliases stay clean identifiers.
func (s *Store) Add(name, command string, replace bool) error {
	name = strings.TrimSpace(name)
	command = strings.TrimSpace(command)
	if name == "" {
		return errors.New("alias name must not be empty")
	}
	if strings.ContainsAny(name, " \t\n") {
		return fmt.Errorf("alias name %q must not contain whitespace", name)
	}
	if command == "" {
		return errors.New("alias command must not be empty")
	}

	all, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := all[name]; exists && !replace {
		return fmt.Errorf("alias %q already exists (use --force to overwrite)", name)
	}
	all[name] = command
	return s.save(all)
}

// Remove deletes an alias, returning ErrNotFound if it does not exist.
func (s *Store) Remove(name string) error {
	all, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := all[name]; !ok {
		return ErrNotFound
	}
	delete(all, name)
	return s.save(all)
}

// load reads the file into a name→command map. A missing file is not an error.
func (s *Store) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return map[string]string{}, nil
	}
	var all map[string]string
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", s.path, err)
	}
	if all == nil {
		all = map[string]string{}
	}
	return all, nil
}

// save writes the map back atomically (temp file + rename) so a crash mid-write
// can't corrupt the existing aliases.
func (s *Store) save(all map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".aliases-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}
	return nil
}
