package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- FileLocker construction tests ---

func TestNewFileLocker_CreatesLocksDir(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")

	fl, err := NewFileLocker(locksDir)
	if err != nil {
		t.Fatalf("NewFileLocker: %v", err)
	}

	if _, err := os.Stat(locksDir); os.IsNotExist(err) {
		t.Error("NewFileLocker should create the locks directory")
	}
	_ = fl
}

func TestNewFileLocker_EmptyPath(t *testing.T) {
	_, err := NewFileLocker("")
	if err == nil {
		t.Fatal("NewFileLocker with empty path should error")
	}
}

// --- Exclusive lock tests ---

func TestFileLocker_LockUnlock(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")
	fl, err := NewFileLocker(locksDir)
	if err != nil {
		t.Fatal(err)
	}

	dataPath := "test-data.json"

	if err := fl.Lock(dataPath); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	// Verify the lock file was created
	lockFile := filepath.Join(locksDir, sanitizeName(dataPath)+".lock")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("lock file should exist after Lock")
	}

	if err := fl.Unlock(dataPath); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

func TestFileLocker_LockThenUnlock_ReleasesLock(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")
	fl, err := NewFileLocker(locksDir)
	if err != nil {
		t.Fatal(err)
	}

	dataPath := "releasable.json"

	if err := fl.Lock(dataPath); err != nil {
		t.Fatal(err)
	}
	if err := fl.Unlock(dataPath); err != nil {
		t.Fatal(err)
	}

	// A second locker should be able to acquire the same lock
	fl2, err := NewFileLocker(locksDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := fl2.Lock(dataPath); err != nil {
		t.Fatalf("second locker should acquire lock after release: %v", err)
	}
	fl2.Unlock(dataPath)
}

// --- Shared lock tests ---

func TestFileLocker_RLockRUnlock(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")
	fl, err := NewFileLocker(locksDir)
	if err != nil {
		t.Fatal(err)
	}

	dataPath := "shared-data.json"

	if err := fl.RLock(dataPath); err != nil {
		t.Fatalf("RLock: %v", err)
	}

	// Verify the lock file was created
	lockFile := filepath.Join(locksDir, sanitizeName(dataPath)+".lock")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("lock file should exist after RLock")
	}

	if err := fl.RUnlock(dataPath); err != nil {
		t.Fatalf("RUnlock: %v", err)
	}
}

func TestFileLocker_MultipleSharedReaders(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")

	dataPath := "multi-reader.json"

	// Multiple lockers should be able to hold shared locks concurrently
	lockers := make([]*FileLocker, 3)
	for i := range lockers {
		fl, err := NewFileLocker(locksDir)
		if err != nil {
			t.Fatal(err)
		}
		lockers[i] = fl
	}

	for _, fl := range lockers {
		if err := fl.RLock(dataPath); err != nil {
			t.Fatalf("RLock from concurrent reader: %v", err)
		}
	}

	// All should succeed in releasing
	for _, fl := range lockers {
		if err := fl.RUnlock(dataPath); err != nil {
			t.Fatalf("RUnlock: %v", err)
		}
	}
}

// --- Concurrent within-process safety ---

func TestFileLocker_ConcurrentExclusiveLocks(t *testing.T) {
	dir := t.TempDir()
	locksDir := filepath.Join(dir, "locks")

	dataPath := "concurrent-exclusive.json"

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Multiple goroutines trying exclusive lock on the same path
	// Only one should succeed at a time
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fl, err := NewFileLocker(locksDir)
			if err != nil {
				errors <- err
				return
			}
			if err := fl.Lock(dataPath); err != nil {
				errors <- err
				return
			}
			// Hold briefly to ensure contention
			time.Sleep(time.Millisecond)
			if err := fl.Unlock(dataPath); err != nil {
				errors <- err
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent exclusive lock error: %v", err)
	}
}

// --- sanitizeName tests ---

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple.json", "simple.json"},
		{"path/to/file.json", "file.json"},
		{"../escape.json", "escape.json"},
		{"/absolute/path.json", "path.json"},
		{"file", "file"},
	}

	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
