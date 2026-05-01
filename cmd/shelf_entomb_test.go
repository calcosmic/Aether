package cmd

import (
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestCopyShelfToChamber(t *testing.T) {
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
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote},
		{ID: "shelf_2", Text: "b", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, PromotedTo: "x"},
	}
	s.SaveJSON("shelf.json", sf)

	chamberDir := tmpDir + "/chamber"
	os.MkdirAll(chamberDir, 0755)

	err := copyShelfToChamber(s, chamberDir)
	if err != nil {
		t.Fatalf("copyShelfToChamber failed: %v", err)
	}

	if _, err := os.Stat(chamberDir + "/shelf.json"); err != nil {
		t.Fatalf("shelf.json not copied to chamber: %v", err)
	}

	var copied colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &copied); err != nil {
		t.Fatalf("failed to load copied shelf: %v", err)
	}
	if len(copied.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(copied.Entries))
	}
}

func TestCopyShelfToChamberMissing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	chamberDir := tmpDir + "/chamber"
	os.MkdirAll(chamberDir, 0755)

	err := copyShelfToChamber(s, chamberDir)
	if err != nil {
		t.Fatalf("copyShelfToChamber should return nil when shelf missing: %v", err)
	}

	if _, err := os.Stat(chamberDir + "/shelf.json"); err == nil {
		t.Fatalf("shelf.json should not exist when no source shelf")
	}
}

func TestShelfChamberSummary(t *testing.T) {
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
		{ID: "shelf_1", Text: "a", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote},
		{ID: "shelf_2", Text: "b", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, PromotedTo: "x"},
		{ID: "shelf_3", Text: "c", Status: colony.ShelfDismissed, Category: colony.ShelfCategoryUserNote},
	}
	s.SaveJSON("shelf.json", sf)

	summary := shelfChamberSummary(s)
	want := "Shelved ideas: 3 (1 promoted, 1 dismissed, 1 active)"
	if summary != want {
		t.Errorf("summary = %v, want %v", summary, want)
	}
}

func TestShelfChamberSummaryEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	os.Setenv("AETHER_ROOT", tmpDir)
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))

	summary := shelfChamberSummary(s)
	want := "Shelved ideas: 0"
	if summary != want {
		t.Errorf("summary = %v, want %v", summary, want)
	}
}

func TestShelfChamberSummaryAllPromoted(t *testing.T) {
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
		{ID: "shelf_1", Text: "a", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, PromotedTo: "x"},
		{ID: "shelf_2", Text: "b", Status: colony.ShelfPromoted, Category: colony.ShelfCategoryUserNote, PromotedTo: "y"},
	}
	s.SaveJSON("shelf.json", sf)

	summary := shelfChamberSummary(s)
	want := "Shelved ideas: 2 (2 promoted, 0 dismissed, 0 active)"
	if summary != want {
		t.Errorf("summary = %v, want %v", summary, want)
	}
}
