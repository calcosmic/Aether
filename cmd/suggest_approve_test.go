package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupSuggestApproveTest creates a temp dir with .aether/data, sets up store
// and stdout capture. Returns the temp dir and bytes buffer.
func setupSuggestApproveTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	// Create a minimal colony state with a goal so loadActiveColonyState succeeds.
	goal := "test goal"
	cs := colony.ColonyState{
		Version: "1.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	data, _ := json.Marshal(cs)
	_ = store.AtomicWrite("COLONY_STATE.json", data)

	return tmpDir, &buf
}

// writePendingSuggestions writes a list of pending suggestions to COLONY_STATE.json.
func writePendingSuggestions(t *testing.T, suggestions []colony.PendingSuggestion) {
	t.Helper()
	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		t.Fatalf("failed to load colony state: %v", err)
	}
	cs.PendingSuggestions = &suggestions
	data, _ := json.Marshal(cs)
	_ = store.AtomicWrite("COLONY_STATE.json", data)
}

// ---------------------------------------------------------------------------
// Test 1: suggest-approve with no pending suggestions returns ok:true with empty list
// ---------------------------------------------------------------------------

func TestSuggestApprove_EmptySuggestions(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	rootCmd.SetArgs([]string{"suggest-approve"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	suggestions, ok := result["suggestions"].([]interface{})
	if !ok {
		t.Fatalf("expected suggestions array, got %T", result["suggestions"])
	}
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(suggestions))
	}
}

// ---------------------------------------------------------------------------
// Test 2: suggest-approve with pending suggestions returns them
// ---------------------------------------------------------------------------

func TestSuggestApprove_ListPendingSuggestions(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_1_test",
			Type:        "FOCUS",
			Content:     "pay attention to error handling",
			Reason:      "pattern: missing error checks",
			ContentHash: "sha256:abc123",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
		{
			ID:          "sig_2_test",
			Type:        "REDIRECT",
			Content:     "avoid global variables",
			Reason:      "pattern: global state detected",
			ContentHash: "sha256:def456",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
	}
	writePendingSuggestions(t, pending)

	rootCmd.SetArgs([]string{"suggest-approve"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	if len(suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(suggestions))
	}

	// Verify first suggestion content
	first := suggestions[0].(map[string]interface{})
	if first["type"] != "FOCUS" {
		t.Errorf("expected first suggestion type FOCUS, got %v", first["type"])
	}
	if first["content"] != "pay attention to error handling" {
		t.Errorf("expected first suggestion content, got %v", first["content"])
	}
}

// ---------------------------------------------------------------------------
// Test 3: suggest-approve --approve writes pheromone signal and removes from pending
// ---------------------------------------------------------------------------

func TestSuggestApprove_ApproveSuggestion(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_approve_test",
			Type:        "FOCUS",
			Content:     "focus on test coverage",
			Reason:      "pattern: low test coverage",
			ContentHash: "sha256:approve123",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
	}
	writePendingSuggestions(t, pending)

	rootCmd.SetArgs([]string{"suggest-approve", "--approve", "sig_approve_test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["approved"] != true {
		t.Errorf("expected approved=true, got %v", result["approved"])
	}

	// Verify pheromones.json has a new signal
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("failed to load pheromones.json: %v", err)
	}
	if len(pf.Signals) == 0 {
		t.Fatal("expected at least one signal in pheromones.json after approval")
	}
	found := false
	for _, sig := range pf.Signals {
		if sig.Type == "FOCUS" && sig.Source == "aether-suggest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected FOCUS signal with source 'aether-suggest' in pheromones.json")
	}

	// Verify suggestion was removed from pending
	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if reloaded.PendingSuggestions != nil && len(*reloaded.PendingSuggestions) > 0 {
		t.Errorf("expected 0 pending suggestions after approval, got %d", len(*reloaded.PendingSuggestions))
	}
}

// ---------------------------------------------------------------------------
// Test 4: suggest-approve --dismiss marks suggestion and hides it
// ---------------------------------------------------------------------------

func TestSuggestApprove_DismissSuggestion(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_dismiss_test",
			Type:        "FEEDBACK",
			Content:     "consider using interfaces",
			Reason:      "pattern: concrete types everywhere",
			ContentHash: "sha256:dismiss123",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
	}
	writePendingSuggestions(t, pending)

	rootCmd.SetArgs([]string{"suggest-approve", "--dismiss", "sig_dismiss_test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["dismissed"] != true {
		t.Errorf("expected dismissed=true, got %v", result["dismissed"])
	}

	// Verify suggestion is marked as dismissed in state
	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if reloaded.PendingSuggestions == nil || len(*reloaded.PendingSuggestions) == 0 {
		t.Fatal("expected suggestion still in pending_suggestions but dismissed")
	}
	if (*reloaded.PendingSuggestions)[0].Dismissed != true {
		t.Error("expected Dismissed=true on the suggestion")
	}

	// Verify dismissed suggestion is not returned in subsequent list call
	var buf2 bytes.Buffer
	stdout = &buf2
	resetRootCmd(t)

	rootCmd.SetArgs([]string{"suggest-approve"})
	err2 := rootCmd.Execute()
	if err2 != nil {
		t.Fatalf("unexpected error on second call: %v", err2)
	}

	env2 := parseEnvelope(t, buf2.String())
	result2 := env2["result"].(map[string]interface{})
	suggestions2, ok2 := result2["suggestions"].([]interface{})
	if !ok2 || suggestions2 == nil {
		// suggestions may be nil or empty -- both acceptable for "no visible suggestions"
		if result2["suggestions"] != nil {
			t.Fatalf("expected suggestions array, got %T", result2["suggestions"])
		}
	} else if len(suggestions2) != 0 {
		t.Errorf("expected 0 visible suggestions after dismiss, got %d", len(suggestions2))
	}
}

// ---------------------------------------------------------------------------
// Test 5: suggest-approve --approve for non-existent id returns not_found
// ---------------------------------------------------------------------------

func TestSuggestApprove_ApproveNonExistent(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	rootCmd.SetArgs([]string{"suggest-approve", "--approve", "sig_nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["not_found"] != true {
		t.Errorf("expected not_found=true, got %v", result["not_found"])
	}
}

// ---------------------------------------------------------------------------
// Test 6: dismissed suggestions not returned in list calls
// ---------------------------------------------------------------------------

func TestSuggestApprove_DismissedNotReturned(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_visible",
			Type:        "FOCUS",
			Content:     "visible suggestion",
			Reason:      "pattern: visible",
			ContentHash: "sha256:visible",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
		{
			ID:          "sig_hidden",
			Type:        "FEEDBACK",
			Content:     "hidden suggestion",
			Reason:      "pattern: hidden",
			ContentHash: "sha256:hidden",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   true,
		},
	}
	writePendingSuggestions(t, pending)

	rootCmd.SetArgs([]string{"suggest-approve"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	suggestions := result["suggestions"].([]interface{})

	if len(suggestions) != 1 {
		t.Fatalf("expected 1 visible suggestion, got %d", len(suggestions))
	}
	first := suggestions[0].(map[string]interface{})
	if first["id"] != "sig_visible" {
		t.Errorf("expected visible suggestion ID 'sig_visible', got %v", first["id"])
	}
}

// ---------------------------------------------------------------------------
// Test 7: suggest-approve --dry-run shows what would happen without persisting
// ---------------------------------------------------------------------------

func TestSuggestApprove_DryRun(t *testing.T) {
	tmpDir, buf := setupSuggestApproveTest(t)
	defer os.RemoveAll(tmpDir)

	pending := []colony.PendingSuggestion{
		{
			ID:          "sig_dryrun_test",
			Type:        "REDIRECT",
			Content:     "avoid hardcoded secrets",
			Reason:      "pattern: secret detected",
			ContentHash: "sha256:dryrun123",
			CreatedAt:   "2026-04-29T00:00:00Z",
			Dismissed:   false,
		},
	}
	writePendingSuggestions(t, pending)

	rootCmd.SetArgs([]string{"suggest-approve", "--approve", "sig_dryrun_test", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})

	if result["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", result["dry_run"])
	}
	if result["would_approve"] != true {
		t.Errorf("expected would_approve=true, got %v", result["would_approve"])
	}

	// Verify pheromones.json was NOT created (dry-run)
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		// Expected: file doesn't exist in dry-run
	} else if len(pf.Signals) > 0 {
		t.Error("expected no signals in pheromones.json after dry-run approval")
	}

	// Verify suggestion still in pending
	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("failed to reload colony state: %v", err)
	}
	if reloaded.PendingSuggestions == nil || len(*reloaded.PendingSuggestions) == 0 {
		t.Error("expected suggestion still in pending_suggestions after dry-run")
	}
}
