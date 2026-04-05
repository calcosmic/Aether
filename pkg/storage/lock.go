package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// FileLocker provides cross-process file locking using syscall.Flock.
// Lock files are stored in a configurable directory as <sanitized-name>.lock,
// compatible with the force-unlock command that removes .lock files from .aether/locks/.
//
// FileLocker is safe for concurrent use within a process. An internal sync.Mutex
// serializes Lock/Unlock and RLock/RUnlock calls to prevent concurrent flock
// operations on the same file descriptor.
type FileLocker struct {
	locksDir string
	mu       sync.Mutex   // guards fd map access
	fd       map[string]*os.File
}

// NewFileLocker creates a FileLocker that stores lock files in locksDir.
// The directory is created if it does not exist.
func NewFileLocker(locksDir string) (*FileLocker, error) {
	if locksDir == "" {
		return nil, fmt.Errorf("storage: locks directory path must not be empty")
	}
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return nil, fmt.Errorf("storage: create locks dir %q: %w", locksDir, err)
	}
	return &FileLocker{
		locksDir: locksDir,
		fd:       make(map[string]*os.File),
	}, nil
}

// Lock acquires an exclusive (write) lock on the file identified by dataPath.
// The lock file is created at locksDir/<sanitized-name>.lock.
// Lock blocks until the exclusive lock is available.
func (fl *FileLocker) Lock(dataPath string) error {
	return fl.lock(dataPath, syscall.LOCK_EX)
}

// Unlock releases the exclusive lock on the file identified by dataPath.
func (fl *FileLocker) Unlock(dataPath string) error {
	return fl.unlock(dataPath)
}

// RLock acquires a shared (read) lock on the file identified by dataPath.
// Multiple readers can hold shared locks simultaneously.
// RLock blocks until the shared lock is available.
func (fl *FileLocker) RLock(dataPath string) error {
	return fl.lock(dataPath, syscall.LOCK_SH)
}

// RUnlock releases the shared lock on the file identified by dataPath.
func (fl *FileLocker) RUnlock(dataPath string) error {
	return fl.unlock(dataPath)
}

func (fl *FileLocker) lock(dataPath string, how int) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// If already holding a lock for this path, release it first to avoid deadlock.
	// This makes the FileLocker reusable: Lock/Unlock can be called multiple times.
	if f, ok := fl.fd[dataPath]; ok {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		delete(fl.fd, dataPath)
	}

	lockFile := filepath.Join(fl.locksDir, sanitizeName(dataPath)+".lock")

	f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("storage: open lock file %q: %w", lockFile, err)
	}

	if err := syscall.Flock(int(f.Fd()), how); err != nil {
		f.Close()
		return fmt.Errorf("storage: flock %q: %w", lockFile, err)
	}

	fl.fd[dataPath] = f
	return nil
}

func (fl *FileLocker) unlock(dataPath string) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	f, ok := fl.fd[dataPath]
	if !ok {
		// Not holding a lock for this path; nothing to do.
		return nil
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("storage: unlock %q: %w", dataPath, err)
	}
	f.Close()
	delete(fl.fd, dataPath)
	return nil
}

// sanitizeName derives a safe lock file name from a data file path.
// It extracts the base name to avoid path traversal and ensure determinism.
func sanitizeName(path string) string {
	return filepath.Base(path)
}
