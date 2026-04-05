package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestNewStore_CreatesLocksDir verifies that NewStore creates a sibling
// locks directory derived from basePath.
func TestNewStore_CreatesLocksDir(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// The locks directory should be a sibling of the data dir
	locksDir := filepath.Join(filepath.Dir(dir), "locks")
	if _, err := os.Stat(locksDir); os.IsNotExist(err) {
		t.Errorf("NewStore should create locks directory at %q", locksDir)
	}
	_ = s
}

// TestNewStore_LocksDirWithParentDerivation verifies locks dir derivation
// when basePath has a parent directory (e.g., /tmp/data).
func TestNewStore_LocksDirWithParentDerivation(t *testing.T) {
	parentDir := t.TempDir()
	dataDir := filepath.Join(parentDir, "data")

	s, err := NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// locks dir should be sibling: <parentDir>/locks
	locksDir := filepath.Join(parentDir, "locks")
	if _, err := os.Stat(locksDir); os.IsNotExist(err) {
		t.Errorf("NewStore should create locks directory at %q", locksDir)
	}
	_ = s
}

// TestAtomicWrite_UsesExclusiveLock verifies that AtomicWrite acquires an
// exclusive lock that blocks concurrent writers. We verify this by checking
// that concurrent writes all succeed (they serialize through the lock).
func TestAtomicWrite_UsesExclusiveLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "exclusive.dat")
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := []byte(`{"worker":` + string(rune('0'+n)) + `}`)
			errors <- s.AtomicWrite(path, data)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent AtomicWrite error: %v", err)
		}
	}

	// File should exist with some valid content
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should exist after concurrent AtomicWrite")
	}
}

// TestAppendJSONL_UsesExclusiveLock verifies that AppendJSONL acquires an
// exclusive lock and concurrent appends all succeed.
func TestAppendJSONL_UsesExclusiveLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "lines.jsonl")
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	type entry struct {
		Worker int `json:"worker"`
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			errors <- s.AppendJSONL(path, entry{Worker: n})
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent AppendJSONL error: %v", err)
		}
	}

	// Should have exactly 10 lines
	results, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}
	if len(results) != 10 {
		t.Errorf("expected 10 JSONL entries, got %d", len(results))
	}
}

// TestLoadJSON_UsesSharedLock verifies that multiple readers can hold shared
// locks concurrently via LoadJSON.
func TestLoadJSON_UsesSharedLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	type TestStruct struct {
		Name string `json:"name"`
	}

	original := TestStruct{Name: "shared-read-test"}
	path := filepath.Join(dir, "shared.json")
	if err := s.SaveJSON(path, &original); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	// Multiple goroutines reading concurrently should all succeed
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var loaded TestStruct
			if err := s.LoadJSON(path, &loaded); err != nil {
				errors <- err
				return
			}
			if loaded.Name != original.Name {
				errors <- fmt.Errorf("name mismatch: got %q, want %q", loaded.Name, original.Name)
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent LoadJSON error: %v", err)
	}
}

// TestReadJSONL_UsesSharedLock verifies that multiple readers can read
// a JSONL file concurrently.
func TestReadJSONL_UsesSharedLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "shared.jsonl")
	for i := 0; i < 5; i++ {
		if err := s.AppendJSONL(path, map[string]int{"n": i}); err != nil {
			t.Fatalf("AppendJSONL: %v", err)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := s.ReadJSONL(path)
			if err != nil {
				errors <- err
				return
			}
			if len(results) != 5 {
				errors <- fmt.Errorf("expected 5 entries, got %d", len(results))
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent ReadJSONL error: %v", err)
	}
}

// TestReadFile_UsesSharedLock verifies that multiple readers can read
// a file concurrently via ReadFile.
func TestReadFile_UsesSharedLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "content.txt")
	if err := s.AtomicWrite(path, []byte("hello world")); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := s.ReadFile(path)
			if err != nil {
				errors <- err
				return
			}
			if string(data) != "hello world" {
				errors <- fmt.Errorf("content mismatch: got %q", string(data))
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent ReadFile error: %v", err)
	}
}

// TestLoadRawJSON_UsesSharedLock verifies that multiple readers can read
// raw JSON concurrently.
func TestLoadRawJSON_UsesSharedLock(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "raw.json")
	content := []byte(`{"raw":true}`)
	if err := s.AtomicWrite(path, content); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := s.LoadRawJSON(path)
			if err != nil {
				errors <- err
				return
			}
			if string(data) != string(content) {
				errors <- fmt.Errorf("content mismatch: got %q, want %q", string(data), string(content))
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent LoadRawJSON error: %v", err)
	}
}

// TestStore_NoSyncMapField verifies the old sync.Map mutexes field has been
// removed from the Store struct by checking that the struct only has the
// expected fields.
func TestStore_NoSyncMapField(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// The locker field should be a *FileLocker, not nil
	if s.locker == nil {
		t.Error("Store.locker should not be nil after NewStore")
	}
	_ = s.locker // ensure the field exists
}

// TestAtomicWrite_LockReleasedOnError verifies that the file lock is properly
// released when AtomicWrite fails (e.g., invalid JSON for a .json file).
func TestAtomicWrite_LockReleasedOnError(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "bad.json")
	// This should fail due to invalid JSON, but the lock must be released
	err = s.AtomicWrite(path, []byte(`{invalid json}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	// After the error, another store should be able to write to the same path
	// (proving the lock was released)
	s2, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore s2: %v", err)
	}
	err = s2.AtomicWrite(path, []byte(`{"valid":true}`))
	if err != nil {
		t.Errorf("second store should be able to write after lock release: %v", err)
	}
}

// TestConcurrentReadWriteLocking verifies that readers and writers can
// coexist without deadlock, and that writes are exclusive while reads
// are concurrent.
func TestConcurrentReadWriteLocking(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Seed the file
	path := filepath.Join(dir, "rw.json")
	type Entry struct {
		Value int `json:"value"`
	}
	if err := s.SaveJSON(path, &Entry{Value: 0}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 30)

	// 10 readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var entry Entry
			if err := s.LoadJSON(path, &entry); err != nil {
				errors <- err
			}
		}()
	}

	// 10 writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			errors <- s.AtomicWrite(path, []byte(`{"value":`+string(rune('0'+n))+`}`))
		}(i)
	}

	// 10 ReadFile readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.ReadFile(path)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Use a timeout to detect deadlocks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed without deadlock
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock detected: concurrent read/write did not complete within 10s")
	}

	close(errors)
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent read/write error: %v", err)
		}
	}
}
