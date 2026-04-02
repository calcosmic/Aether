package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
