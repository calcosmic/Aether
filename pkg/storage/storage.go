// Package storage provides atomic file operations, backup rotation, and path
// resolution for Aether colony data files. It replaces the shell-based
// atomic-write.sh and path resolution logic now provided by the aether Go binary.
package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const repositoryContainmentError = "storage: repository containment refused"

// RepositoryRoot is an open, physically resolved repository authority. The
// repository directory handle remains open for the lifetime of the Store so
// every repository-scoped path can be reopened relative to that trusted root
// without following a replaced intermediate component.
type RepositoryRoot struct {
	physicalRoot   string
	dataPath       string
	dataComponents []string
	lockComponents []string
	rootDir        *os.File
	mu             sync.RWMutex
}

// OpenRepositoryRoot validates a repository-scoped data path without creating
// anything. Existing path components are opened one at a time with no-follow
// semantics; missing components are allowed so a later mutating command can
// create them relative to the verified repository handle.
func OpenRepositoryRoot(repositoryRoot, dataPath string) (*RepositoryRoot, error) {
	rootInput := strings.TrimSpace(repositoryRoot)
	dataInput := strings.TrimSpace(dataPath)
	if rootInput == "" {
		return nil, repositoryContainmentRefusal("repository root path must not be empty", nil)
	}
	if dataInput == "" {
		return nil, repositoryContainmentRefusal("data root path must not be empty", nil)
	}
	if hasDegeneratePathComponent(dataPath) {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("data root %q contains a dot or dot-dot component", dataPath), nil)
	}

	rootAbs, err := filepath.Abs(rootInput)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("resolve repository root %q", repositoryRoot), err)
	}
	rootAbs = filepath.Clean(rootAbs)
	rootPhysical, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("resolve physical repository root %q", rootAbs), err)
	}
	rootPhysical, err = filepath.Abs(rootPhysical)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("normalize physical repository root %q", rootPhysical), err)
	}
	rootInfo, err := os.Lstat(rootPhysical)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("inspect physical repository root %q", rootPhysical), err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("physical repository root %q is not a real directory", rootPhysical), nil)
	}

	dataAbs, err := filepath.Abs(dataInput)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("resolve data root %q", dataPath), err)
	}
	dataAbs = filepath.Clean(dataAbs)
	dataRel, err := filepath.Rel(rootAbs, dataAbs)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("compare data root %q with repository %q", dataAbs, rootAbs), err)
	}
	dataComponents, err := containedPathComponents(dataRel)
	if err != nil {
		// macOS commonly exposes the same temporary directory as both /var and
		// /private/var. Accept the candidate's already-physical spelling only
		// when it is component-contained by the canonical repository itself.
		// We do not EvalSymlinks on the candidate, so links below the repository
		// remain visible to (and rejected by) the no-follow component walk.
		physicalRel, physicalRelErr := filepath.Rel(rootPhysical, dataAbs)
		if physicalRelErr != nil {
			return nil, repositoryContainmentRefusal(fmt.Sprintf("data root %q is outside repository %q", dataAbs, rootAbs), err)
		}
		dataComponents, physicalRelErr = containedPathComponents(physicalRel)
		if physicalRelErr != nil {
			return nil, repositoryContainmentRefusal(fmt.Sprintf("data root %q is outside repository %q", dataAbs, rootAbs), err)
		}
		dataRel = physicalRel
	}
	lockRel := filepath.Join(filepath.Dir(dataRel), "locks")
	lockComponents, err := containedPathComponents(lockRel)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("locks root derived from %q is outside repository %q", dataAbs, rootAbs), err)
	}

	rootDir, err := platformOpenDirectory(rootPhysical)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("open physical repository root %q", rootPhysical), err)
	}
	authority := &RepositoryRoot{
		physicalRoot: rootPhysical,
		// Preserve the caller-visible absolute spelling for BasePath compatibility
		// (macOS commonly spells /private/var as /var). Repository I/O never uses
		// this string as authority; it stays anchored to rootDir below.
		dataPath:       dataAbs,
		dataComponents: dataComponents,
		lockComponents: lockComponents,
		rootDir:        rootDir,
	}
	if err := authority.revalidate(); err != nil {
		_ = authority.Close()
		return nil, err
	}
	return authority, nil
}

// Close releases the repository directory authority. Stores intentionally keep
// it open for their process lifetime; read-only probes close it when no colony
// exists and no Store is constructed.
func (r *RepositoryRoot) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rootDir == nil {
		return nil
	}
	err := r.rootDir.Close()
	r.rootDir = nil
	return err
}

// DataFileExists checks a repository data file without creating the data path,
// locks directory, or lock file. It is the bootstrap probe used by the first
// read-only status invocation.
func (r *RepositoryRoot) DataFileExists(path string) (bool, error) {
	components, err := containedPathComponents(path)
	if err != nil {
		return false, repositoryContainmentRefusal(fmt.Sprintf("invalid data file path %q", path), err)
	}
	parentComponents := appendPathComponents(r.dataComponents, components[:len(components)-1]...)
	parent, err := r.openDirectoryChain(parentComponents, false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer parent.Close()
	file, err := platformOpenRegularFileAt(parent, components[len(components)-1], os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, repositoryContainmentRefusal(fmt.Sprintf("open data file %q", path), err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, repositoryContainmentRefusal(fmt.Sprintf("inspect data file %q", path), err)
	}
	if !info.Mode().IsRegular() {
		return false, repositoryContainmentRefusal(fmt.Sprintf("data file %q is not a regular file", path), nil)
	}
	return true, nil
}

func (r *RepositoryRoot) revalidate() error {
	for _, components := range [][]string{r.dataComponents, r.lockComponents} {
		dir, err := r.openDirectoryChain(components, false)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		if err := dir.Close(); err != nil {
			return repositoryContainmentRefusal("close validated repository component", err)
		}
	}
	return nil
}

func (r *RepositoryRoot) openDirectoryChain(components []string, create bool) (*os.File, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.rootDir == nil {
		return nil, repositoryContainmentRefusal("repository authority is closed", nil)
	}
	if len(components) == 0 {
		return nil, repositoryContainmentRefusal("repository-relative directory path must not be empty", nil)
	}
	parent := r.rootDir
	var owned *os.File
	for _, component := range components {
		next, err := platformOpenDirectoryAt(parent, component, create)
		if owned != nil {
			_ = owned.Close()
			owned = nil
		}
		if err != nil {
			return nil, repositoryContainmentRefusal(fmt.Sprintf("open repository component %q", component), err)
		}
		owned = next
		parent = next
	}
	return owned, nil
}

func repositoryContainmentRefusal(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%s: %s", repositoryContainmentError, message)
	}
	return fmt.Errorf("%s: %s: %w", repositoryContainmentError, message, cause)
}

func containedPathComponents(path string) ([]string, error) {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || filepath.VolumeName(path) != "" {
		return nil, fmt.Errorf("path must be a non-empty relative path")
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("path escapes its root")
	}
	components := strings.FieldsFunc(clean, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(components) == 0 {
		return nil, fmt.Errorf("path has no components")
	}
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return nil, fmt.Errorf("path contains a degenerate component")
		}
	}
	return components, nil
}

func appendPathComponents(base []string, suffix ...string) []string {
	joined := make([]string, 0, len(base)+len(suffix))
	joined = append(joined, base...)
	joined = append(joined, suffix...)
	return joined
}

func hasDegeneratePathComponent(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return true
	}
	volume := filepath.VolumeName(trimmed)
	trimmed = strings.TrimPrefix(trimmed, volume)
	for _, component := range strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if component == "." || component == ".." {
			return true
		}
	}
	return false
}

// Store provides thread-safe atomic file operations within a base directory.
// All file paths are resolved relative to basePath unless they are absolute.
// File operations are coordinated via FileLocker for cross-process safety.
type Store struct {
	basePath   string
	locker     *FileLocker
	repository *RepositoryRoot
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

// NewRepositoryStore creates a Store exclusively through an already-open
// repository authority. Validation is repeated before any directory creation
// so a component replaced after bootstrap validation is refused rather than
// followed. The authority remains attached to the Store for all later I/O.
func NewRepositoryStore(root *RepositoryRoot) (*Store, error) {
	if root == nil {
		return nil, repositoryContainmentRefusal("repository authority is required", nil)
	}
	if err := root.revalidate(); err != nil {
		return nil, err
	}
	dataDir, err := root.openDirectoryChain(root.dataComponents, true)
	if err != nil {
		return nil, err
	}
	if err := dataDir.Close(); err != nil {
		return nil, repositoryContainmentRefusal("close verified data directory", err)
	}
	locksDir, err := root.openDirectoryChain(root.lockComponents, true)
	if err != nil {
		return nil, err
	}
	if err := locksDir.Close(); err != nil {
		return nil, repositoryContainmentRefusal("close verified locks directory", err)
	}
	locker := newRepositoryFileLocker(root, root.lockComponents)
	return &Store{basePath: root.dataPath, locker: locker, repository: root}, nil
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

	existing, err := s.readFileUnlocked(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage: read %q: %w", s.resolvePath(path), err)
	}

	updated, err := mutate(existing)
	if err != nil {
		return err
	}
	return s.atomicWriteLocked(path, updated)
}

func (s *Store) atomicWriteLocked(path string, data []byte) error {
	if s.repository != nil {
		return s.atomicWriteRepository(path, data)
	}

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

func (s *Store) atomicWriteRepository(path string, data []byte) error {
	parent, name, relative, err := s.repositoryFileParent(path, true)
	if err != nil {
		return err
	}
	defer parent.Close()
	if strings.HasSuffix(strings.ToLower(relative), ".json") && !json.Valid(data) {
		return fmt.Errorf("storage: invalid JSON for %q", s.resolvePath(path))
	}
	if target, openErr := platformOpenRegularFileAt(parent, name, os.O_RDONLY, 0); openErr == nil {
		if closeErr := target.Close(); closeErr != nil {
			return fmt.Errorf("storage: close existing target %q: %w", s.resolvePath(path), closeErr)
		}
	} else if !errors.Is(openErr, os.ErrNotExist) {
		return repositoryContainmentRefusal(fmt.Sprintf("refuse non-regular or linked target %q", relative), openErr)
	}

	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return fmt.Errorf("storage: generate temporary name for %q: %w", s.resolvePath(path), err)
	}
	tmpName := fmt.Sprintf(".%s.tmp.%d-%s", name, os.Getpid(), hex.EncodeToString(rnd))
	tmp, err := platformOpenRegularFileAt(parent, tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("storage: create repository temp for %q: %w", s.resolvePath(path), err)
	}
	keepTemp := true
	defer func() {
		_ = tmp.Close()
		if keepTemp {
			_ = platformUnlinkAt(parent, tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("storage: write repository temp for %q: %w", s.resolvePath(path), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: close repository temp for %q: %w", s.resolvePath(path), err)
	}
	if err := platformRenameAt(parent, tmpName, parent, name); err != nil {
		return fmt.Errorf("storage: rename repository temp for %q: %w", s.resolvePath(path), err)
	}
	keepTemp = false
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
	data, err := s.readFileUnlocked(path)
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
	data, err := s.readFileUnlocked(path)
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

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("storage: marshal JSONL entry: %w", err)
	}

	fullPath := s.resolvePath(path)
	var f *os.File
	if s.repository != nil {
		parent, name, _, parentErr := s.repositoryFileParent(path, true)
		if parentErr != nil {
			return parentErr
		}
		defer parent.Close()
		f, err = platformOpenRegularFileAt(parent, name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, err = f.Seek(0, io.SeekEnd)
		}
	} else {
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("storage: create dir %q: %w", dir, err)
		}
		f, err = os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
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
	data, err := s.readFileUnlocked(path)
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
	data, err := s.readFileUnlocked(path)
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

// FileExists reports whether a regular store file exists. Repository-backed
// stores use their open root authority, so this never follows intermediate
// links or creates a lock merely to inspect a path.
func (s *Store) FileExists(path string) (bool, error) {
	if s.repository != nil {
		return s.repository.DataFileExists(path)
	}
	info, err := os.Stat(s.resolvePath(path))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

func (s *Store) readFileUnlocked(path string) ([]byte, error) {
	if s.repository == nil {
		return os.ReadFile(s.resolvePath(path))
	}
	parent, name, relative, err := s.repositoryFileParent(path, false)
	if err != nil {
		return nil, err
	}
	defer parent.Close()
	file, err := platformOpenRegularFileAt(parent, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("storage: open repository file %q: %w", relative, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("storage: inspect repository file %q: %w", relative, err)
	}
	if !info.Mode().IsRegular() {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("repository file %q is not regular", relative), nil)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("storage: read repository file %q: %w", relative, err)
	}
	return data, nil
}

func (s *Store) repositoryFileParent(path string, create bool) (*os.File, string, string, error) {
	relative, err := s.repositoryRelativePath(path)
	if err != nil {
		return nil, "", "", err
	}
	components, err := containedPathComponents(relative)
	if err != nil {
		return nil, "", "", repositoryContainmentRefusal(fmt.Sprintf("invalid store path %q", path), err)
	}
	parents := appendPathComponents(s.repository.dataComponents, components[:len(components)-1]...)
	parent, err := s.repository.openDirectoryChain(parents, create)
	if err != nil {
		return nil, "", "", err
	}
	return parent, components[len(components)-1], relative, nil
}

func (s *Store) repositoryRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", repositoryContainmentRefusal("store path must not be empty", nil)
	}
	if filepath.IsAbs(trimmed) {
		relative, err := filepath.Rel(s.basePath, filepath.Clean(trimmed))
		if err != nil {
			return "", repositoryContainmentRefusal(fmt.Sprintf("compare store path %q with data root", path), err)
		}
		trimmed = relative
	}
	clean := filepath.Clean(trimmed)
	if _, err := containedPathComponents(clean); err != nil {
		return "", repositoryContainmentRefusal(fmt.Sprintf("store path %q escapes repository data root", path), err)
	}
	return clean, nil
}
