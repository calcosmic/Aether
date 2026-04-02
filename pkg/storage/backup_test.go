package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestRotateBackups_NoBackups(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Create a regular file but no backup files
	regular := filepath.Join(dir, "data.json")
	if err := os.WriteFile(regular, []byte(`{"a":1}`), 0644); err != nil {
		t.Fatal(err)
	}

	// RotateBackups should do nothing, return nil
	if err := s.RotateBackups(regular); err != nil {
		t.Errorf("RotateBackups with no backups: %v", err)
	}
}

func TestRotateBackups_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	base := filepath.Join(dir, "data.json")
	// Create 2 backup files (under maxBackups=3) with unique suffixes
	for i := 0; i < 2; i++ {
		bp := filepath.Join(dir, fmt.Sprintf("data.json.bak.%s-%d", time.Now().Format("20060102-150405"), i))
		if err := os.WriteFile(bp, []byte(`backup`), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond) // ensure different mtimes
	}

	if err := s.RotateBackups(base); err != nil {
		t.Errorf("RotateBackups under limit: %v", err)
	}

	// Both should still exist
	matches, _ := filepath.Glob(base + ".bak.*")
	if len(matches) != 2 {
		t.Errorf("expected 2 backups, got %d", len(matches))
	}
}

func TestRotateBackups_AtLimit(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	base := filepath.Join(dir, "data.json")
	// Create exactly 3 backup files with unique suffixes
	for i := 0; i < 3; i++ {
		bp := filepath.Join(dir, fmt.Sprintf("data.json.bak.%s-%d", time.Now().Format("20060102-150405"), i))
		if err := os.WriteFile(bp, []byte(`backup`), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := s.RotateBackups(base); err != nil {
		t.Errorf("RotateBackups at limit: %v", err)
	}

	matches, _ := filepath.Glob(base + ".bak.*")
	if len(matches) != 3 {
		t.Errorf("expected 3 backups, got %d", len(matches))
	}
}

func TestRotateBackups_OverLimit(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	base := filepath.Join(dir, "data.json")
	// Create 5 backup files with unique suffixes and different mtimes
	var paths []string
	for i := 0; i < 5; i++ {
		bp := filepath.Join(dir, fmt.Sprintf("data.json.bak.%s-%d", time.Now().Format("20060102-150405"), i))
		if err := os.WriteFile(bp, []byte(`backup`), 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, bp)
		time.Sleep(20 * time.Millisecond) // ensure different mtimes for sort
	}

	if err := s.RotateBackups(base); err != nil {
		t.Errorf("RotateBackups over limit: %v", err)
	}

	matches, _ := filepath.Glob(base + ".bak.*")
	if len(matches) != 3 {
		t.Errorf("expected 3 backups after rotation, got %d", len(matches))
	}

	// Verify the 3 newest were kept (last 3 created)
	sort.Slice(matches, func(i, j int) bool {
		si, _ := os.Stat(matches[i])
		sj, _ := os.Stat(matches[j])
		return si.ModTime().After(sj.ModTime())
	})
	// Newest 3 from original should be kept
	for _, expected := range paths[2:] {
		found := false
		for _, m := range matches {
			if m == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected backup %q to be kept", expected)
		}
	}
}

func TestCreateBackup_NoOriginal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Call CreateBackup on a path that does not exist
	nonexistent := filepath.Join(dir, "nope.json")
	if err := s.CreateBackup(nonexistent); err != nil {
		t.Errorf("CreateBackup on nonexistent file should return nil, got: %v", err)
	}
}

func TestCreateBackup_WithOriginal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	content := []byte(`{"hello":"world"}`)
	original := filepath.Join(dir, "data.json")
	if err := os.WriteFile(original, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.CreateBackup(original); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}

	// Verify a .bak file was created
	matches, err := filepath.Glob(original + ".bak.*")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(matches))
	}

	// Verify backup content matches original
	backupContent, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupContent) != string(content) {
		t.Errorf("backup content mismatch: got %q, want %q", backupContent, content)
	}
}
