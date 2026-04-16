package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type heldLock struct {
	file   *os.File
	shared bool
	count  int
}

// FileLocker provides cross-process file locking backed by platform-specific
// file lock primitives. Lock files are stored as <sanitized-name>.lock in the
// configured locks directory.
type FileLocker struct {
	locksDir string
	mu       sync.Mutex
	fd       map[string]*heldLock
}

func NewFileLocker(locksDir string) (*FileLocker, error) {
	if locksDir == "" {
		return nil, fmt.Errorf("storage: locks directory path must not be empty")
	}
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return nil, fmt.Errorf("storage: create locks dir %q: %w", locksDir, err)
	}
	return &FileLocker{
		locksDir: locksDir,
		fd:       make(map[string]*heldLock),
	}, nil
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

	if existing, ok := fl.fd[dataPath]; ok {
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

	lockFile := filepath.Join(fl.locksDir, sanitizeName(dataPath)+".lock")
	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("storage: open lock file %q: %w", lockFile, err)
	}

	if err := platformLockFile(f, shared); err != nil {
		_ = f.Close()
		return fmt.Errorf("storage: lock %q: %w", lockFile, err)
	}

	fl.fd[dataPath] = &heldLock{
		file:   f,
		shared: shared,
		count:  1,
	}
	return nil
}

func (fl *FileLocker) unlock(dataPath string, shared bool) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	held, ok := fl.fd[dataPath]
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
	delete(fl.fd, dataPath)
	return nil
}

func sanitizeName(path string) string {
	return filepath.Base(path)
}
