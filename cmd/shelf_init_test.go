package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestLoadActiveShelf(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
		{ID: "shelf_2", Text: "b", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z", PromotedTo: "x"},
		{ID: "shelf_3", Text: "c", Status: colony.ShelfDismissed, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	active, err := loadActiveShelf(s)
	if err != nil {
		t.Fatalf("loadActiveShelf failed: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active entry, got %d", len(active))
	}
	if active[0].ID != "shelf_1" {
		t.Errorf("id = %v, want shelf_1", active[0].ID)
	}
}

func TestPromoteShelfEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	err := promoteShelfEntry(s, "shelf_1", "Build feature X")
	if err != nil {
		t.Fatalf("promoteShelfEntry failed: %v", err)
	}

	var updated colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &updated); err != nil {
		t.Fatalf("failed to load shelf: %v", err)
	}
	if updated.Entries[0].Status != colony.ShelfPromoted {
		t.Errorf("status = %v, want promoted", updated.Entries[0].Status)
	}
	if updated.Entries[0].PromotedTo != "Build feature X" {
		t.Errorf("promoted_to = %v, want Build feature X", updated.Entries[0].PromotedTo)
	}
}

func TestDismissShelfEntry(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	err := dismissShelfEntry(s, "shelf_1")
	if err != nil {
		t.Fatalf("dismissShelfEntry failed: %v", err)
	}

	var updated colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &updated); err != nil {
		t.Fatalf("failed to load shelf: %v", err)
	}
	if updated.Entries[0].Status != colony.ShelfDismissed {
		t.Errorf("status = %v, want dismissed", updated.Entries[0].Status)
	}
}

func TestShelfEntryToTodo(t *testing.T) {
	entry := colony.ShelfEntry{
		Category: colony.ShelfCategoryRedirect,
		Text:     "Avoid global mutable state in workers",
	}
	got := shelfEntryToTodo(entry)
	want := "[shelf:redirect] Avoid global mutable state in workers"
	if got != want {
		t.Errorf("shelfEntryToTodo = %v, want %v", got, want)
	}
}

func TestFormatShelfForInit(t *testing.T) {
	entries := []colony.ShelfEntry{
		{ID: "s1", Text: "idea one", Category: colony.ShelfCategoryInstinct, CreatedAt: "2024-01-01T00:00:00Z", SourcePhase: 1},
		{ID: "s2", Text: "idea two", Category: colony.ShelfCategoryRedirect, CreatedAt: "2024-01-02T00:00:00Z", SourcePhase: 2},
	}
	out := formatShelfForInit(entries)
	if !strings.Contains(out, "idea one") {
		t.Errorf("output missing idea one: %s", out)
	}
	if !strings.Contains(out, "idea two") {
		t.Errorf("output missing idea two: %s", out)
	}
}

func TestLoadActiveShelfEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	active, err := loadActiveShelf(s)
	if err != nil {
		t.Fatalf("loadActiveShelf failed: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected 0 active entries, got %d", len(active))
	}
}

func TestInitShelfBacklogOutput(t *testing.T) {
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

	// Pre-populate shelf
	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "focus on tests", Status: colony.ShelfShelved, Category: colony.ShelfCategoryPheromone, CreatedAt: time.Now().UTC().Format(time.RFC3339)},
	}
	s.SaveJSON("shelf.json", sf)

	rootCmd.SetArgs([]string{"init", "Build feature X"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("init returned error: %v", err)
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
	count, ok := result["shelf_backlog_count"].(float64)
	if !ok {
		t.Fatalf("shelf_backlog_count missing or wrong type: %v", result["shelf_backlog_count"])
	}
	if count != 1 {
		t.Errorf("shelf_backlog_count = %v, want 1", count)
	}
	backlog, ok := result["shelf_backlog"].([]interface{})
	if !ok {
		t.Fatalf("shelf_backlog missing or wrong type: %v", result["shelf_backlog"])
	}
	if len(backlog) != 1 {
		t.Errorf("shelf_backlog len = %v, want 1", len(backlog))
	}
}
