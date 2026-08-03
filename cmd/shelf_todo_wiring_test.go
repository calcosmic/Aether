package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Phase 165 gap CR-01: a promoted shelf entry must become a durable colony
// todo, written by the Go runtime, and must survive a session refresh. These
// tests prove the full chain: shelf-promote-batch -> aether init ->
// session.json -> session refresh.

func TestShelfPromoteBatchReturnsTodos(t *testing.T) {
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

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
		{ID: "shelf_2", Text: "Stop using global mutexes", Status: colony.ShelfShelved, Category: colony.ShelfCategoryRedirect, CreatedAt: "2024-01-01T00:00:01Z"},
	}
	s.SaveJSON("shelf.json", sf)

	rootCmd.SetArgs([]string{"shelf-promote-batch", "--ids", "shelf_1,shelf_2", "--colony", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("shelf-promote-batch returned error: %v", err)
	}

	output := bytes.TrimSpace(buf.Bytes())
	var envelope map[string]interface{}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	result := envelope["result"].(map[string]interface{})
	todosRaw, ok := result["todos"].([]interface{})
	if !ok {
		t.Fatalf("todos missing or wrong type: %v", result["todos"])
	}
	todos := make([]string, len(todosRaw))
	for i, v := range todosRaw {
		todos[i] = v.(string)
	}
	wantA := "[shelf:user-note] Add rate limiting"
	wantB := "[shelf:redirect] Stop using global mutexes"
	foundA, foundB := false, false
	for _, todo := range todos {
		if todo == wantA {
			foundA = true
		}
		if todo == wantB {
			foundB = true
		}
	}
	if !foundA {
		t.Errorf("todos missing %q: %v", wantA, todos)
	}
	if !foundB {
		t.Errorf("todos missing %q: %v", wantB, todos)
	}
}

func TestInitSeedsSessionTodosFromPromotedShelf(t *testing.T) {
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

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	if err := promoteShelfEntry(s, "shelf_1", "Ship v2"); err != nil {
		t.Fatalf("promoteShelfEntry failed: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("failed to load session.json: %v", err)
	}
	want := "[shelf:user-note] Add rate limiting"
	found := false
	for _, todo := range session.ActiveTodos {
		if todo == want {
			found = true
		}
	}
	if !found {
		t.Errorf("session.json active_todos missing %q: %v", want, session.ActiveTodos)
	}
}

func TestSessionRefreshPreservesShelfTodos(t *testing.T) {
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

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	if err := promoteShelfEntry(s, "shelf_1", "Ship v2"); err != nil {
		t.Fatalf("promoteShelfEntry failed: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("failed to load COLONY_STATE.json: %v", err)
	}

	// Simulate the next session refresh (e.g. from `aether status`), which
	// must not clobber the shelf-seeded todo even though this phase-less
	// post-init state derives zero todos of its own.
	if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
		CommandName:   "status",
		SuggestedNext: "aether plan",
		Summary:       "Session refreshed",
	}); err != nil {
		t.Fatalf("syncColonyArtifacts failed: %v", err)
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("failed to load session.json: %v", err)
	}
	want := "[shelf:user-note] Add rate limiting"
	found := false
	for _, todo := range session.ActiveTodos {
		if todo == want {
			found = true
		}
	}
	if !found {
		t.Errorf("session refresh dropped shelf todo %q: %v", want, session.ActiveTodos)
	}
}

func TestMergeShelfTodosKeepsShelfEntriesAheadOfDerived(t *testing.T) {
	got := mergeShelfTodos(
		[]string{"[shelf:user-note] A", "Old task"},
		[]string{"Phase task 1", "[shelf:user-note] A"},
	)
	want := []string{"[shelf:user-note] A", "Phase task 1"}
	if len(got) != len(want) {
		t.Fatalf("mergeShelfTodos = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mergeShelfTodos[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}
