// Package storage provides atomic file operations, backup rotation, and path
// resolution for Aether colony data files. It replaces the shell-based
// atomic-write.sh and path resolution logic now provided by the aether Go binary.
package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store provides thread-safe atomic file operations within a base directory.
// All file paths are resolved relative to basePath unless they are absolute.
// File operations are coordinated via FileLocker for cross-process safety.
type Store struct {
	basePath string
	locker   *FileLocker
}

// NewStore creates a new Store rooted at basePath.
// The directory is created if it does not exist.
// A FileLocker is initialized in a sibling "locks" directory.
func NewStore(basePath string) (*Store, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("storage: create base dir %q: %w", basePath, err)
	}
	locksDir := filepath.Join(filepath.Dir(basePath), "locks")
	locker, err := NewFileLocker(locksDir)
	if err != nil {
		return nil, fmt.Errorf("storage: create file locker: %w", err)
	}
	return &Store{basePath: basePath, locker: locker}, nil
}

// BasePath returns the store's root directory.
func (s *Store) BasePath() string {
	return s.basePath
}

// AtomicWrite writes data to path atomically using a temporary file and rename.
// If path ends in .json, the content is validated as valid JSON before writing.
// On error, the temporary file is cleaned up.
func (s *Store) AtomicWrite(path string, data []byte) error {
	if err := s.locker.Lock(path); err != nil {
		return fmt.Errorf("storage: acquire lock for %q: %w", path, err)
	}
	defer s.locker.Unlock(path)

	return s.atomicWriteLocked(path, data)
}

// UpdateFile performs a read-modify-write cycle under a single exclusive lock.
// It is intended for callers that need cross-process safe updates based on the
// current file contents.
func (s *Store) UpdateFile(path string, mutate func(existing []byte) ([]byte, error)) error {
	if err := s.locker.Lock(path); err != nil {
		return fmt.Errorf("storage: acquire lock for %q: %w", path, err)
	}
	defer s.locker.Unlock(path)

	fullPath := s.resolvePath(path)
	existing, err := os.ReadFile(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: read %q: %w", fullPath, err)
	}

	updated, err := mutate(existing)
	if err != nil {
		return err
	}
	return s.atomicWriteLocked(path, updated)
}

func (s *Store) atomicWriteLocked(path string, data []byte) error {

	fullPath := s.resolvePath(path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("storage: create dir %q: %w", dir, err)
	}

	// Write to temp file first with unique suffix for concurrent safety
	rnd := make([]byte, 4)
	rand.Read(rnd)
	tmpPath := fullPath + ".tmp." + fmt.Sprintf("%d-%s", os.Getpid(), hex.EncodeToString(rnd))
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("storage: write temp %q: %w", tmpPath, err)
	}

	// Validate JSON for .json files
	if strings.HasSuffix(fullPath, ".json") {
		if !json.Valid(data) {
			return fmt.Errorf("storage: invalid JSON for %q", fullPath)
		}
	}

	// Atomic rename
	if err := os.Rename(tmpPath, fullPath); err != nil {
		return fmt.Errorf("storage: rename %q -> %q: %w", tmpPath, fullPath, err)
	}

	success = true
	return nil
}

// UpdateJSONAtomically reads the JSON at path, calls mutate on the decoded value,
// and writes the result back atomically. If mutate returns an error, no write occurs.
// The operation is safe for concurrent use.
func (s *Store) UpdateJSONAtomically(path string, ptr interface{}, mutate func() error) error {
	return s.UpdateFile(path, func(existing []byte) ([]byte, error) {
		if len(existing) > 0 {
			if err := json.Unmarshal(existing, ptr); err != nil {
				return nil, fmt.Errorf("unmarshal existing %s: %w", path, err)
			}
		}
		if err := mutate(); err != nil {
			return nil, err
		}
		updated, err := json.MarshalIndent(ptr, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal updated %s: %w", path, err)
		}
		return updated, nil
	})
}

// SaveJSON marshals data as formatted JSON and writes it atomically.
// For COLONY_STATE.json, the events array is capped at 100 entries.
func (s *Store) SaveJSON(path string, data interface{}) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: marshal JSON for %q: %w", path, err)
	}
	encoded = append(encoded, '\n')

	// Enforce event array cap for COLONY_STATE.json
	if path == "COLONY_STATE.json" {
		encoded, err = capEventsArray(encoded)
		if err != nil {
			return fmt.Errorf("storage: cap events array for %q: %w", path, err)
		}
	}

	return s.AtomicWrite(path, encoded)
}

// capEventsArray trims the "events" array to at most 100 entries using a
// generic map[string]interface{} approach to avoid importing pkg/colony.
// It returns the re-marshaled JSON bytes and logs a warning when trimming.
func capEventsArray(data []byte) ([]byte, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	eventsRaw, ok := root["events"]
	if !ok {
		return data, nil
	}

	events, ok := eventsRaw.([]interface{})
	if !ok {
		return data, nil
	}

	const cap = 100
	if len(events) <= cap {
		return data, nil
	}

	dropped := len(events) - cap
	root["events"] = events[dropped:]

	fmt.Fprintf(os.Stderr, "warning: event array capped at %d (dropped %d old events)\n", cap, dropped)

	trimmed, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	trimmed = append(trimmed, '\n')
	return trimmed, nil
}

// LoadJSON reads and unmarshals a JSON file.
func (s *Store) LoadJSON(path string, dest interface{}) error {
	if err := s.locker.RLock(path); err != nil {
		return fmt.Errorf("storage: acquire read lock for %q: %w", path, err)
	}
	defer s.locker.RUnlock(path)

	fullPath := s.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("storage: read %q: %w", fullPath, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("storage: unmarshal %q: %w", fullPath, err)
	}
	return nil
}

// LoadRawJSON reads a JSON file and returns raw bytes.
func (s *Store) LoadRawJSON(path string) ([]byte, error) {
	if err := s.locker.RLock(path); err != nil {
		return nil, fmt.Errorf("storage: acquire read lock for %q: %w", path, err)
	}
	defer s.locker.RUnlock(path)

	fullPath := s.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("storage: read %q: %w", fullPath, err)
	}
	return data, nil
}

// SaveRawJSON writes raw bytes to a JSON file atomically.
func (s *Store) SaveRawJSON(path string, data []byte) error {
	return s.AtomicWrite(path, data)
}

// AppendJSONL appends a JSON entry as a single line to a JSONL file.
func (s *Store) AppendJSONL(path string, entry interface{}) error {
	if err := s.locker.Lock(path); err != nil {
		return fmt.Errorf("storage: acquire lock for %q: %w", path, err)
	}
	defer s.locker.Unlock(path)

	fullPath := s.resolvePath(path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("storage: create dir %q: %w", dir, err)
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("storage: marshal JSONL entry: %w", err)
	}

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("storage: open JSONL %q: %w", fullPath, err)
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("storage: write JSONL entry: %w", err)
	}
	return nil
}

// ReadJSONL reads all valid JSON lines from a JSONL file.
// Blank lines are skipped. Malformed lines are logged and skipped (not errored).
func (s *Store) ReadJSONL(path string) ([]json.RawMessage, error) {
	if err := s.locker.RLock(path); err != nil {
		return nil, fmt.Errorf("storage: acquire read lock for %q: %w", path, err)
	}
	defer s.locker.RUnlock(path)

	fullPath := s.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("storage: read JSONL %q: %w", fullPath, err)
	}

	var results []json.RawMessage
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if !json.Valid(json.RawMessage(trimmed)) {
			logMalformedLine(fullPath, string(trimmed))
			continue
		}
		results = append(results, json.RawMessage(trimmed))
	}
	return results, nil
}

// ReadFile reads raw file content from the store.
func (s *Store) ReadFile(path string) ([]byte, error) {
	if err := s.locker.RLock(path); err != nil {
		return nil, fmt.Errorf("storage: acquire read lock for %q: %w", path, err)
	}
	defer s.locker.RUnlock(path)

	fullPath := s.resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("storage: read %q: %w", fullPath, err)
	}
	return data, nil
}

// logMalformedLine logs a malformed JSONL line.
// Extracted as a function for testability.
func logMalformedLine(path, line string) {
	fmt.Fprintf(os.Stderr, "storage: skipping malformed JSONL line in %q: %s\n", path, line)
}

// resolvePath resolves a path relative to the store's base path.
// Absolute paths are returned as-is.
func (s *Store) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.basePath, path)
}
