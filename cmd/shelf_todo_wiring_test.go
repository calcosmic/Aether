package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

	t.Setenv("AETHER_ROOT", tmpDir)

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

	t.Setenv("AETHER_ROOT", tmpDir)

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

	t.Setenv("AETHER_ROOT", tmpDir)

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

	// Review WR-03: the merged todo list must also reach the human-facing
	// recovery document, not only session.json.
	contextData, err := os.ReadFile(filepath.Join(tmpDir, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("failed to read CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextData), want) {
		t.Errorf("CONTEXT.md missing shelf todo %q", want)
	}
}

// TestInitCeremonySeedsSessionTodosFromPromotedShelf proves the Codex-facing
// colony-creation path (init-ceremony) also seeds session todos from promoted
// shelf entries, closing review WR-02: the capability must not exist on one
// colony-creation path and be silently absent on the other.
func TestInitCeremonySeedsSessionTodosFromPromotedShelf(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	t.Setenv("AETHER_ROOT", tmpDir)

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	if err := promoteShelfEntry(s, "shelf_1", "Ship v2"); err != nil {
		t.Fatalf("promoteShelfEntry failed: %v", err)
	}

	if err := createCeremonyColony("Ship v2", colony.ScopeProject, colony.ColonyModeColony, colony.Charter{}); err != nil {
		t.Fatalf("createCeremonyColony returned error: %v", err)
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

// TestInitPromotesShelfEntriesAtomically proves the promotion mutation now
// lives inside the init transaction: `--promote-shelf` on `aether init`
// promotes the entry under the actual init goal and seeds the todo, with no
// separate shelf-promote-batch call required (Phase 165 gap CR-01 / review
// CR-01).
func TestInitPromotesShelfEntriesAtomically(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	t.Setenv("AETHER_ROOT", tmpDir)

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	rootCmd.SetArgs([]string{"init", "--promote-shelf", "shelf_1", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	var updated colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &updated); err != nil {
		t.Fatalf("failed to load shelf.json: %v", err)
	}
	found := false
	for _, e := range updated.Entries {
		if e.ID == "shelf_1" {
			found = true
			if e.Status != colony.ShelfPromoted {
				t.Errorf("shelf_1 status = %q, want promoted", e.Status)
			}
			if e.PromotedTo != "Ship v2" {
				t.Errorf("shelf_1 PromotedTo = %q, want %q", e.PromotedTo, "Ship v2")
			}
		}
	}
	if !found {
		t.Fatalf("shelf_1 not found in shelf.json after init")
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("failed to load session.json: %v", err)
	}
	want := "[shelf:user-note] Add rate limiting"
	foundTodo := false
	for _, todo := range session.ActiveTodos {
		if todo == want {
			foundTodo = true
		}
	}
	if !foundTodo {
		t.Errorf("session.json active_todos missing %q: %v", want, session.ActiveTodos)
	}
}

// TestFailedInitLeavesShelfEntriesShelved is the cancel/failed-init fence the
// verification asks for: a colony-creation attempt that refuses before any
// state write must leave the shelf backlog exactly as it was.
func TestFailedInitLeavesShelfEntriesShelved(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var stdoutBuf, stderrBuf bytes.Buffer
	stdout = &stdoutBuf
	stderr = &stderrBuf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	t.Setenv("AETHER_ROOT", tmpDir)

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	existingGoal := "Existing colony"
	existingState := colony.ColonyState{
		Version:      "3.0",
		Goal:         &existingGoal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
	}
	s.SaveJSON("COLONY_STATE.json", existingState)

	rootCmd.SetArgs([]string{"init", "--promote-shelf", "shelf_1", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rootCmd.Execute returned unexpected error: %v", err)
	}

	env := parseEnvelope(t, stderrBuf.String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false for already-initialized colony, got: %v", env["ok"])
	}

	var updated colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &updated); err != nil {
		t.Fatalf("failed to load shelf.json: %v", err)
	}
	for _, e := range updated.Entries {
		if e.ID == "shelf_1" {
			if e.Status != colony.ShelfShelved {
				t.Errorf("shelf_1 status = %q, want shelved (a failed init must not strand a promotion)", e.Status)
			}
			if e.PromotedTo != "" {
				t.Errorf("shelf_1 PromotedTo = %q, want empty", e.PromotedTo)
			}
		}
	}
}

// TestInitPromotesUnderRevisedGoal proves a goal string that only exists
// because the user revised it at the Approval stage cannot diverge from the
// promotion goal: they are the same variable inside the init transaction.
func TestInitPromotesUnderRevisedGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	t.Setenv("AETHER_ROOT", tmpDir)

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	rootCmd.SetArgs([]string{"init", "--promote-shelf", "shelf_1", "Ship v2 with billing"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	var updated colony.ShelfFile
	if err := s.LoadJSON("shelf.json", &updated); err != nil {
		t.Fatalf("failed to load shelf.json: %v", err)
	}
	for _, e := range updated.Entries {
		if e.ID == "shelf_1" && e.PromotedTo != "Ship v2 with billing" {
			t.Errorf("shelf_1 PromotedTo = %q, want %q", e.PromotedTo, "Ship v2 with billing")
		}
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

// TestInitReportsUnpromotableShelfIDs proves a bad shelf ID never blocks
// colony creation, and that the caller can still see which IDs failed.
func TestInitReportsUnpromotableShelfIDs(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	s, _ := storage.NewStore(dataDir)
	store = s

	t.Setenv("AETHER_ROOT", tmpDir)

	sf := colony.NewShelfFile()
	sf.Entries = []colony.ShelfEntry{
		{ID: "shelf_1", Text: "Add rate limiting", Status: colony.ShelfShelved, Category: colony.ShelfCategoryUserNote, CreatedAt: "2024-01-01T00:00:00Z"},
	}
	s.SaveJSON("shelf.json", sf)

	rootCmd.SetArgs([]string{"init", "--promote-shelf", "shelf_1,does_not_exist", "Ship v2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true (a bad shelf ID must never block colony creation), got: %v", env["ok"])
	}
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result missing or wrong type: %v", env["result"])
	}
	promotedRaw, _ := result["shelf_promoted"].([]interface{})
	failedRaw, _ := result["shelf_failed"].([]interface{})

	foundPromoted := false
	for _, v := range promotedRaw {
		if v == "shelf_1" {
			foundPromoted = true
		}
	}
	if !foundPromoted {
		t.Errorf("shelf_promoted missing %q: %v", "shelf_1", promotedRaw)
	}

	foundFailed := false
	for _, v := range failedRaw {
		if v == "does_not_exist" {
			foundFailed = true
		}
	}
	if !foundFailed {
		t.Errorf("shelf_failed missing %q: %v", "does_not_exist", failedRaw)
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
