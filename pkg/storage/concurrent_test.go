package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestConcurrentAppendJSONL_NoInterleaving launches 20 goroutines that each
// append a unique, content-rich JSON entry via AppendJSONL, then verifies:
//   - every line is valid JSON (no partial or interleaved writes)
//   - every expected entry is present (no lost writes)
//   - the total line count matches the goroutine count
func TestConcurrentAppendJSONL_NoInterleaving(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	const numGoroutines = 20
	path := "concurrent-no-interleave.jsonl"

	type Entry struct {
		WorkerID int    `json:"worker_id"`
		Message  string `json:"message"`
		Sequence int    `json:"sequence"`
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entry := Entry{
				WorkerID: n,
				Message:  fmt.Sprintf("payload from worker %d with some extra text to make it longer", n),
				Sequence: n * 100,
			}
			if err := s.AppendJSONL(path, entry); err != nil {
				errors <- fmt.Errorf("worker %d AppendJSONL: %w", n, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent append error: %v", err)
	}

	// Read back all lines
	lines, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}

	if len(lines) != numGoroutines {
		t.Fatalf("expected %d lines, got %d", numGoroutines, len(lines))
	}

	// Verify every line is valid JSON and parse into Entry struct
	found := make(map[int]bool, numGoroutines)
	for i, raw := range lines {
		if !json.Valid(raw) {
			t.Errorf("line %d is not valid JSON: %s", i, string(raw))
			continue
		}
		var entry Entry
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Errorf("line %d failed to unmarshal: %v (raw: %s)", i, err, string(raw))
			continue
		}
		// Verify the fields are internally consistent
		expectedSeq := entry.WorkerID * 100
		if entry.Sequence != expectedSeq {
			t.Errorf("line %d: worker %d has sequence %d, want %d (possible interleaving)",
				i, entry.WorkerID, entry.Sequence, expectedSeq)
		}
		found[entry.WorkerID] = true
	}

	// Every worker's entry must be present
	for i := 0; i < numGoroutines; i++ {
		if !found[i] {
			t.Errorf("missing entry from worker %d", i)
		}
	}
}

// TestConcurrentAppendJSONL_HighContention uses 50 goroutines with larger
// payloads to stress-test the locking under high contention.
func TestConcurrentAppendJSONL_HighContention(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	const numGoroutines = 50
	path := "high-contention.jsonl"

	type BigEntry struct {
		ID      int    `json:"id"`
		Padding string `json:"padding"`
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			// 200-byte padding to make writes more likely to interleave without locking
			padding := make([]byte, 200)
			for j := range padding {
				padding[j] = byte('A' + (n+j)%26)
			}
			entry := BigEntry{
				ID:      n,
				Padding: string(padding),
			}
			if err := s.AppendJSONL(path, entry); err != nil {
				errors <- fmt.Errorf("worker %d: %w", n, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("high contention append error: %v", err)
	}

	lines, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}

	if len(lines) != numGoroutines {
		t.Fatalf("expected %d lines, got %d", numGoroutines, len(lines))
	}

	// Verify all lines are valid JSON with correct structure
	found := make(map[int]bool, numGoroutines)
	for i, raw := range lines {
		if !json.Valid(raw) {
			t.Errorf("line %d is not valid JSON", i)
			continue
		}
		var entry BigEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Errorf("line %d failed to unmarshal: %v", i, err)
			continue
		}
		if len(entry.Padding) != 200 {
			t.Errorf("line %d: padding length %d, want 200 (possible truncation)", i, len(entry.Padding))
		}
		found[entry.ID] = true
	}

	for i := 0; i < numGoroutines; i++ {
		if !found[i] {
			t.Errorf("missing entry with id %d", i)
		}
	}
}

// TestConcurrentReadWriteJSONL verifies that concurrent readers always see
// complete, valid JSONL data -- never partial writes. One writer goroutine
// continuously appends entries while multiple readers verify that whatever
// they read is always well-formed.
func TestConcurrentReadWriteJSONL(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := "concurrent-rw.jsonl"

	const numWriters = 5
	const numReaders = 10
	const writesPerWriter = 20

	var wg sync.WaitGroup
	writeErrors := make(chan error, numWriters)
	readErrors := make(chan error, numReaders*writesPerWriter)

	// Seed a few entries so readers have something to read immediately
	for i := 0; i < 3; i++ {
		if err := s.AppendJSONL(path, map[string]int{"seed": i}); err != nil {
			t.Fatalf("seed AppendJSONL: %v", err)
		}
	}

	// Writers: each writes writesPerWriter entries
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < writesPerWriter; i++ {
				entry := map[string]interface{}{
					"writer": writerID,
					"seq":    i,
					"data":   fmt.Sprintf("writer-%d-seq-%d-payload", writerID, i),
				}
				if err := s.AppendJSONL(path, entry); err != nil {
					writeErrors <- fmt.Errorf("writer %d seq %d: %w", writerID, i, err)
				}
			}
		}(w)
	}

	// Readers: continuously read and validate while writes are in progress
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < writesPerWriter; i++ {
				lines, err := s.ReadJSONL(path)
				if err != nil {
					readErrors <- fmt.Errorf("reader %d iteration %d: %w", readerID, i, err)
					continue
				}
				// Every line must be valid JSON
				for li, raw := range lines {
					if !json.Valid(raw) {
						readErrors <- fmt.Errorf("reader %d iteration %d line %d: invalid JSON: %s",
							readerID, i, li, string(raw))
					}
				}
				// Small sleep to spread reads across the write timeline
				time.Sleep(time.Microsecond * 100)
			}
		}(r)
	}

	// Wait with a timeout to catch deadlocks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(15 * time.Second):
		t.Fatal("deadlock detected: concurrent read/write did not complete within 15s")
	}

	close(writeErrors)
	for err := range writeErrors {
		t.Errorf("write error: %v", err)
	}

	close(readErrors)
	for err := range readErrors {
		t.Errorf("read error: %v", err)
	}

	// Final validation: file should have exactly seed + writers * writesPerWriter lines
	expectedLines := 3 + numWriters*writesPerWriter
	lines, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("final ReadJSONL: %v", err)
	}
	if len(lines) != expectedLines {
		t.Errorf("expected %d total lines, got %d", expectedLines, len(lines))
	}
}
