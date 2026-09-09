package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newRepositoryStore200(t *testing.T) (*Store, string) {
	t.Helper()

	repositoryRoot := t.TempDir()
	dataPath := filepath.Join(repositoryRoot, ".aether", "data")
	authority, err := OpenRepositoryRoot(repositoryRoot, dataPath)
	if err != nil {
		t.Fatalf("OpenRepositoryRoot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := authority.Close(); err != nil {
			t.Errorf("close repository authority: %v", err)
		}
	})
	store, err := NewRepositoryStore(authority)
	if err != nil {
		t.Fatalf("NewRepositoryStore() error = %v", err)
	}
	return store, dataPath
}

func TestRepositoryStoreMissingErrorIdentity200(t *testing.T) {
	store, _ := newRepositoryStore200(t)

	readers := map[string]func() error{
		"ReadFile missing file": func() error {
			_, err := store.ReadFile("missing.txt")
			return err
		},
		"ReadFile missing parent": func() error {
			_, err := store.ReadFile(filepath.Join("missing-parent", "missing.txt"))
			return err
		},
		"LoadRawJSON": func() error {
			_, err := store.LoadRawJSON("missing.json")
			return err
		},
		"LoadJSON": func() error {
			var value map[string]interface{}
			return store.LoadJSON("missing.json", &value)
		},
		"ReadJSONL": func() error {
			_, err := store.ReadJSONL("missing.jsonl")
			return err
		},
	}

	for name, read := range readers {
		t.Run(name, func(t *testing.T) {
			err := read()
			if err == nil {
				t.Fatal("error = nil, want a missing-file error")
			}
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("errors.Is(err, fs.ErrNotExist) = false for %T: %v", err, err)
			}
		})
	}

	for _, path := range []string{"missing.txt", filepath.Join("missing-parent", "missing.txt")} {
		exists, err := store.FileExists(path)
		if err != nil {
			t.Fatalf("FileExists(%q) error = %v, want nil", path, err)
		}
		if exists {
			t.Fatalf("FileExists(%q) = true, want false", path)
		}
	}

	if err := store.UpdateFile(filepath.Join("fresh", "ledger.txt"), func(existing []byte) ([]byte, error) {
		if len(existing) != 0 {
			t.Fatalf("fresh UpdateFile existing bytes = %q, want empty", existing)
		}
		return []byte("first\n"), nil
	}); err != nil {
		t.Fatalf("UpdateFile() first write error = %v", err)
	}
}

func TestRepositoryStoreRefusesNonAbsentFailures200(t *testing.T) {
	store, dataPath := newRepositoryStore200(t)

	assertRefusedNotAbsent := func(t *testing.T, path string) {
		t.Helper()
		_, err := store.ReadFile(path)
		if err == nil {
			t.Fatalf("ReadFile(%q) error = nil, want refusal", path)
		}
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ReadFile(%q) misclassified refusal as absence: %v", path, err)
		}
	}

	if err := os.Mkdir(filepath.Join(dataPath, "directory.txt"), 0755); err != nil {
		t.Fatalf("create obstructing directory: %v", err)
	}
	assertRefusedNotAbsent(t, "directory.txt")

	target := filepath.Join(dataPath, "target.txt")
	if err := os.WriteFile(target, []byte("outside authority should not be followed"), 0644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dataPath, "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	assertRefusedNotAbsent(t, "linked.txt")

	corruptPath := filepath.Join(dataPath, "corrupt.json")
	if err := os.WriteFile(corruptPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("write corrupt JSON: %v", err)
	}
	var value map[string]interface{}
	err := store.LoadJSON("corrupt.json", &value)
	if err == nil {
		t.Fatal("LoadJSON(corrupt.json) error = nil, want malformed JSON error")
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("LoadJSON(corrupt.json) misclassified corruption as absence: %v", err)
	}

	unreadablePath := filepath.Join(dataPath, "unreadable.txt")
	if err := os.WriteFile(unreadablePath, []byte("protected"), 0600); err != nil {
		t.Fatalf("write unreadable fixture: %v", err)
	}
	if err := os.Chmod(unreadablePath, 0000); err != nil {
		t.Fatalf("chmod unreadable fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadablePath, 0600) })
	if _, err := store.ReadFile("unreadable.txt"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("permission denial misclassified as absence: %v", err)
		}
	} else {
		t.Log("permission denial could not be induced for this process; non-absence cases remain covered by directory, link, and corruption fixtures")
	}
}

func TestReadJSONL_MalformedLineSkipped(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Create a JSONL file with valid, blank, malformed, and valid lines
	path := filepath.Join(dir, "events.jsonl")
	content := `{"a":1}

{bad json}
{"b":2}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(results))
	}

	// Verify first entry
	var first map[string]int
	if err := json.Unmarshal(results[0], &first); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}
	if first["a"] != 1 {
		t.Errorf("first entry: got %v, want a=1", first)
	}

	// Verify second entry
	var second map[string]int
	if err := json.Unmarshal(results[1], &second); err != nil {
		t.Fatalf("unmarshal second: %v", err)
	}
	if second["b"] != 2 {
		t.Errorf("second entry: got %v, want b=2", second)
	}
}

func TestReadJSONL_AllMalformed(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "bad.jsonl")
	content := `{not valid}
{also bad}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL with all malformed: should not error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 entries for all malformed, got %d", len(results))
	}
}

func TestReadJSONL_SkipsBlankLines(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "blanks.jsonl")
	content := `{"x":1}

{"y":2}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := s.ReadJSONL(path)
	if err != nil {
		t.Fatalf("ReadJSONL: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 entries, got %d", len(results))
	}
}

func TestAtomicWrite_NoTmpFileLeft(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "data.json")
	if err := s.AtomicWrite(path, []byte(`{"test":true}`)); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	// Verify the target file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("target file should exist")
	}

	// Verify no .tmp files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWrite_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "bad.json")
	err = s.AtomicWrite(path, []byte(`{not valid json}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON in .json file")
	}

	// Verify target file does NOT exist
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("target file should not exist after failed write")
	}

	// Verify no .tmp files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temp file left behind after error: %s", e.Name())
		}
	}
}

func TestAtomicWrite_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	path := filepath.Join(dir, "valid.json")
	content := []byte(`{"hello":"world","num":42}`)
	if err := s.AtomicWrite(path, content); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", data, content)
	}
}

func TestConcurrentWrites_NoRace(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Use .dat extension to avoid JSON validation, testing atomic write mechanism
	path := filepath.Join(dir, "concurrent.dat")
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			content := []byte(fmt.Sprintf("worker-%d-data", n))
			done <- s.AtomicWrite(path, content)
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent write %d: %v", i, err)
		}
	}
}

func TestUpdateJSONAtomically_RollsBackOnMutateError(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "original", Value: 1}
	path := filepath.Join(dir, "state.json")
	if err := s.SaveJSON(path, &original); err != nil {
		t.Fatalf("SaveJSON initial: %v", err)
	}

	var state TestStruct
	err = s.UpdateJSONAtomically(path, &state, func() error {
		state.Name = "mutated"
		state.Value = 99
		return fmt.Errorf("deliberate mutation failure")
	})
	if err == nil {
		t.Fatal("expected error from UpdateJSONAtomically when mutate fails")
	}
	if err.Error() != "deliberate mutation failure" {
		t.Fatalf("expected deliberate error, got: %v", err)
	}

	// Verify file still contains original data
	var reloaded TestStruct
	if err := s.LoadJSON(path, &reloaded); err != nil {
		t.Fatalf("LoadJSON after rollback: %v", err)
	}
	if reloaded.Name != "original" || reloaded.Value != 1 {
		t.Errorf("file was modified despite mutate error: got %+v, want original", reloaded)
	}
}

func TestUpdateJSONAtomically_CommitsOnSuccess(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "original", Value: 1}
	path := filepath.Join(dir, "state.json")
	if err := s.SaveJSON(path, &original); err != nil {
		t.Fatalf("SaveJSON initial: %v", err)
	}

	var state TestStruct
	if err := s.UpdateJSONAtomically(path, &state, func() error {
		state.Name = "updated"
		state.Value = 42
		return nil
	}); err != nil {
		t.Fatalf("UpdateJSONAtomically: %v", err)
	}

	var reloaded TestStruct
	if err := s.LoadJSON(path, &reloaded); err != nil {
		t.Fatalf("LoadJSON after commit: %v", err)
	}
	if reloaded.Name != "updated" || reloaded.Value != 42 {
		t.Errorf("file not updated: got %+v, want updated", reloaded)
	}
}

func TestSaveAndLoadJSON(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestStruct{Name: "test", Value: 42}
	path := filepath.Join(dir, "obj.json")
	if err := s.SaveJSON(path, &original); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded TestStruct
	if err := s.LoadJSON(path, &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	if loaded.Name != original.Name || loaded.Value != original.Value {
		t.Errorf("round-trip mismatch: got %+v, want %+v", loaded, original)
	}
}

func TestSaveJSON_CapsEventsArray(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := map[string]interface{}{
		"version": "3.0",
		"events":  makeEvents(120),
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("COLONY_STATE.json", &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	events, ok := loaded["events"].([]interface{})
	if !ok {
		t.Fatalf("events is not an array")
	}
	if len(events) != 100 {
		t.Errorf("events len = %d, want 100", len(events))
	}

	// Verify the last event is preserved (event-119)
	lastEvent, ok := events[len(events)-1].(string)
	if !ok || lastEvent != "event-119" {
		t.Errorf("last event = %v, want event-119", lastEvent)
	}

	// Verify the first event is dropped (should be event-20, not event-0)
	firstEvent, ok := events[0].(string)
	if !ok || firstEvent != "event-20" {
		t.Errorf("first event = %v, want event-20", firstEvent)
	}
}

func TestSaveJSON_NoCapForOtherFiles(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	data := map[string]interface{}{
		"items": makeEvents(120),
	}

	if err := s.SaveJSON("other-file.json", data); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("other-file.json", &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	items, ok := loaded["items"].([]interface{})
	if !ok {
		t.Fatalf("items is not an array")
	}
	if len(items) != 120 {
		t.Errorf("items len = %d, want 120 (no cap)", len(items))
	}
}

func TestSaveJSON_NoCapWhenUnderLimit(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	state := map[string]interface{}{
		"version": "3.0",
		"events":  makeEvents(50),
	}

	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded map[string]interface{}
	if err := s.LoadJSON("COLONY_STATE.json", &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}

	events, ok := loaded["events"].([]interface{})
	if !ok {
		t.Fatalf("events is not an array")
	}
	if len(events) != 50 {
		t.Errorf("events len = %d, want 50", len(events))
	}
}

func makeEvents(n int) []string {
	events := make([]string, n)
	for i := 0; i < n; i++ {
		events[i] = fmt.Sprintf("event-%d", i)
	}
	return events
}
