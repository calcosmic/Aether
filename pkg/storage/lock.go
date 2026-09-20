package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type heldLock struct {
	file   *os.File
	shared bool
	count  int
}

// FileLocker provides cross-process file locking backed by platform-specific
// file lock primitives. The actual lock name is a digest of the complete,
// normalized data path so equal basenames in different directories cannot
// alias one another.
type FileLocker struct {
	locksDir           string
	repository         *RepositoryRoot
	lockComponents     []string
	legacySimpleMarker bool
	mu                 sync.Mutex
	fd                 map[string]*heldLock
}

func NewFileLocker(locksDir string) (*FileLocker, error) {
	if locksDir == "" {
		return nil, fmt.Errorf("storage: locks directory path must not be empty")
	}
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return nil, fmt.Errorf("storage: create locks dir %q: %w", locksDir, err)
	}
	return &FileLocker{
		locksDir:           locksDir,
		legacySimpleMarker: true,
		fd:                 make(map[string]*heldLock),
	}, nil
}

func newRepositoryFileLocker(root *RepositoryRoot, components []string) *FileLocker {
	return &FileLocker{
		locksDir:       filepath.Join(append([]string{root.physicalRoot}, components...)...),
		repository:     root,
		lockComponents: append([]string(nil), components...),
		fd:             make(map[string]*heldLock),
	}
}

func (fl *FileLocker) Lock(dataPath string) error {
	return fl.lock(dataPath, false)
}

func (fl *FileLocker) Unlock(dataPath string) error {
	return fl.unlock(dataPath, false)
}

func (fl *FileLocker) RLock(dataPath string) error {
	return fl.lock(dataPath, true)
}

func (fl *FileLocker) RUnlock(dataPath string) error {
	return fl.unlock(dataPath, true)
}

func (fl *FileLocker) lock(dataPath string, shared bool) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	identity, err := fl.normalizedLockIdentity(dataPath)
	if err != nil {
		return err
	}

	if existing, ok := fl.fd[identity]; ok {
		if !shared {
			// Preserve the stronger requested mode in local bookkeeping. The
			// underlying OS lock remains the first-acquired lock, which avoids the
			// prior bug where a same-process second caller dropped the original
			// descriptor out from under the first caller.
			existing.shared = false
		}
		existing.count++
		return nil
	}

	lockName := lockFileName(identity)
	lockFile := filepath.Join(fl.locksDir, lockName)
	f, err := fl.openLockFile(lockName)
	if err != nil {
		return fmt.Errorf("storage: open lock file %q: %w", lockFile, err)
	}

	if err := platformLockFile(f, shared); err != nil {
		_ = f.Close()
		return fmt.Errorf("storage: lock %q: %w", lockFile, err)
	}

	fl.fd[identity] = &heldLock{
		file:   f,
		shared: shared,
		count:  1,
	}
	if fl.legacySimpleMarker && isSimpleLockPath(dataPath) {
		if err := createLegacySimpleLockMarker(fl.locksDir, dataPath); err != nil {
			_ = platformUnlockFile(f)
			_ = f.Close()
			delete(fl.fd, identity)
			return err
		}
	}
	return nil
}

func (fl *FileLocker) unlock(dataPath string, shared bool) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	identity, err := fl.normalizedLockIdentity(dataPath)
	if err != nil {
		return err
	}

	held, ok := fl.fd[identity]
	if !ok {
		return nil
	}
	if held.count > 1 {
		held.count--
		return nil
	}

	if err := platformUnlockFile(held.file); err != nil {
		return fmt.Errorf("storage: unlock %q: %w", dataPath, err)
	}
	if err := held.file.Close(); err != nil {
		return fmt.Errorf("storage: close lock %q: %w", dataPath, err)
	}
	delete(fl.fd, identity)
	return nil
}

func (fl *FileLocker) openLockFile(name string) (*os.File, error) {
	if fl.repository == nil {
		return os.OpenFile(filepath.Join(fl.locksDir, name), os.O_CREATE|os.O_RDWR, 0644)
	}
	locksDir, err := fl.repository.openDirectoryChain(fl.lockComponents, false)
	if err != nil {
		return nil, err
	}
	defer locksDir.Close()
	file, err := platformOpenRegularFileAt(locksDir, name, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, repositoryContainmentRefusal(fmt.Sprintf("open repository lock %q", name), err)
	}
	return file, nil
}

func (fl *FileLocker) normalizedLockIdentity(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("storage: lock data path must not be empty")
	}
	if fl.repository != nil && filepath.IsAbs(trimmed) {
		relative, err := filepath.Rel(fl.repository.dataPath, filepath.Clean(trimmed))
		if err != nil {
			return "", repositoryContainmentRefusal(fmt.Sprintf("compare lock path %q with repository data root", path), err)
		}
		trimmed = relative
	}
	clean := filepath.Clean(trimmed)
	if fl.repository != nil {
		if _, err := containedPathComponents(clean); err != nil {
			return "", repositoryContainmentRefusal(fmt.Sprintf("lock path %q escapes repository data root", path), err)
		}
	}
	if runtime.GOOS == "windows" {
		clean = strings.ToLower(clean)
	}
	return filepath.ToSlash(clean), nil
}

func lockFileName(identity string) string {
	digest := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(digest[:]) + ".lock"
}

func isSimpleLockPath(path string) bool {
	if filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(strings.TrimSpace(path))
	return clean != "." && clean != ".." && filepath.Base(clean) == clean
}

// createLegacySimpleLockMarker preserves the long-standing observable marker
// name used by generic NewFileLocker callers while the actual lock identity is
// always the full-path digest. Repository-authorized lockers never create this
// compatibility marker.
func createLegacySimpleLockMarker(locksDir, dataPath string) error {
	marker := filepath.Join(locksDir, sanitizeName(dataPath)+".lock")
	if info, err := os.Lstat(marker); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("storage: legacy lock marker %q is not a regular file", marker)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("storage: inspect legacy lock marker %q: %w", marker, err)
	}
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("storage: create legacy lock marker %q: %w", marker, err)
	}
	return f.Close()
}

func sanitizeName(path string) string {
	return filepath.Base(path)
}
