package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestShelfAddAndRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	rootCmd.SetArgs([]string{"shelf-add", "--text", "focus on error handling", "--category", "instinct"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-add returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}

	result := envelope["result"].(map[string]interface{})
	if result["created"] != true {
		t.Fatalf("expected created:true, got: %v", result["created"])
	}

	entry := result["entry"].(map[string]interface{})
	if entry["text"] != "focus on error handling" {
		t.Errorf("text = %v, want %q", entry["text"], "focus on error handling")
	}

	// Read back via shelf-list
	buf.Reset()
	rootCmd.SetArgs([]string{"shelf-list", "--json"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-list returned error: %v", err)
	}

	output = strings.TrimSpace(buf.String())
	json.Unmarshal([]byte(output), &envelope)
	result = envelope["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestShelfPromote(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// Add an entry
	rootCmd.SetArgs([]string{"shelf-add", "--text", "test entry", "--category", "user-note"})
	rootCmd.Execute()

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	result := envelope["result"].(map[string]interface{})
	entry := result["entry"].(map[string]interface{})
	id := entry["id"].(string)

	buf.Reset()
	rootCmd.SetArgs([]string{"shelf-promote", "--id", id, "--to", "Build feature X"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-promote returned error: %v", err)
	}

	output = strings.TrimSpace(buf.String())
	json.Unmarshal([]byte(output), &envelope)
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}

	// Verify status changed
	var sf colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &sf); err != nil {
		t.Fatalf("failed to load shelf: %v", err)
	}
	if len(sf.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(sf.Entries))
	}
	if sf.Entries[0].Status != colony.ShelfPromoted {
		t.Errorf("status = %v, want %q", sf.Entries[0].Status, colony.ShelfPromoted)
	}
	if sf.Entries[0].PromotedTo != "Build feature X" {
		t.Errorf("promoted_to = %v, want %q", sf.Entries[0].PromotedTo, "Build feature X")
	}
}

func TestShelfDismiss(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// Add an entry
	rootCmd.SetArgs([]string{"shelf-add", "--text", "dismiss me", "--category", "user-note"})
	rootCmd.Execute()

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	result := envelope["result"].(map[string]interface{})
	entry := result["entry"].(map[string]interface{})
	id := entry["id"].(string)

	buf.Reset()
	rootCmd.SetArgs([]string{"shelf-dismiss", "--id", id})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-dismiss returned error: %v", err)
	}

	output = strings.TrimSpace(buf.String())
	json.Unmarshal([]byte(output), &envelope)
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}

	var sf colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &sf); err != nil {
		t.Fatalf("failed to load shelf: %v", err)
	}
	if sf.Entries[0].Status != colony.ShelfDismissed {
		t.Errorf("status = %v, want %q", sf.Entries[0].Status, colony.ShelfDismissed)
	}
}

func TestShelfListFilter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// Add 3 entries with different statuses
	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
		{ID: "shelf_2", Text: "b", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z", PromotedTo: "x"},
		{ID: "shelf_3", Text: "c", Status: colony.ShelfDismissed, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	buf.Reset()
	rootCmd.SetArgs([]string{"shelf-list", "--status", "shelved", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-list returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	result := envelope["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 shelved entry, got %d", len(entries))
	}
}

func TestShelfFileNotExist(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	// List without any shelf file should return empty, no error
	rootCmd.SetArgs([]string{"shelf-list", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("shelf-list returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	result := envelope["result"].(map[string]interface{})
	entries := result["entries"].([]interface{})
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
