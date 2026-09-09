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

	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("COLONY_DATA_DIR", dataDir)

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

	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("COLONY_DATA_DIR", dataDir)

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

	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("COLONY_DATA_DIR", dataDir)

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
	originalRoot, hadRoot := os.LookupEnv("AETHER_ROOT")
	originalDataDir, hadDataDir := os.LookupEnv("COLONY_DATA_DIR")
	t.Cleanup(func() {
		if hadRoot {
			_ = os.Setenv("AETHER_ROOT", originalRoot)
		} else {
			_ = os.Unsetenv("AETHER_ROOT")
		}
		if hadDataDir {
			_ = os.Setenv("COLONY_DATA_DIR", originalDataDir)
		} else {
			_ = os.Unsetenv("COLONY_DATA_DIR")
		}
	})

	_ = os.Unsetenv("AETHER_ROOT")
	_ = os.Unsetenv("COLONY_DATA_DIR")
	store = nil
	tracer = nil

	var fixtureRoot string
	var summary string
	t.Run("isolated shelf fixture", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		fixtureRoot = t.TempDir()
		dataDir := fixtureRoot + "/.aether/data"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatal(err)
		}
		s, err := storage.NewStore(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		store = s

		t.Setenv("AETHER_ROOT", fixtureRoot)
		t.Setenv("COLONY_DATA_DIR", dataDir)

		summary = shelfChamberSummary(s)
	})

	want := "Shelved ideas: 0"
	if summary != want {
		t.Errorf("summary = %v, want %v", summary, want)
	}
	if got, ok := os.LookupEnv("AETHER_ROOT"); ok {
		t.Errorf("AETHER_ROOT survived deleted shelf fixture: %q", got)
	}
	if got, ok := os.LookupEnv("COLONY_DATA_DIR"); ok {
		t.Errorf("COLONY_DATA_DIR survived deleted shelf fixture: %q", got)
	}
	if store != nil {
		t.Error("repository store survived deleted shelf fixture")
	}
	if tracer != nil {
		t.Error("repository tracer survived deleted shelf fixture")
	}
	if _, err := os.Stat(fixtureRoot); !os.IsNotExist(err) {
		t.Errorf("fixture root still exists after subtest cleanup: %v", err)
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

	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("COLONY_DATA_DIR", dataDir)

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
