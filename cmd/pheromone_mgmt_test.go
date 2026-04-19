package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPheromoneSnapshotInjectIsAvailable(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-snapshot-inject"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if cmd.Deprecated != "" {
		t.Fatalf("pheromone-snapshot-inject should not be deprecated, got %q", cmd.Deprecated)
	}
}

func TestPheromoneMergeBackIsAvailable(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"pheromone-merge-back"})
	if err != nil {
		t.Fatalf("command not found: %v", err)
	}
	if cmd.Deprecated != "" {
		t.Fatalf("pheromone-merge-back should not be deprecated, got %q", cmd.Deprecated)
	}
}

func TestPheromoneSnapshotInjectCopiesActiveSignalsBetweenRoots(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	sourceDataDir := setupBuildFlowTest(t)
	sourceRoot := filepath.Dir(filepath.Dir(sourceDataDir))
	targetRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(targetRoot, ".aether", "data"), 0755); err != nil {
		t.Fatalf("mkdir target data dir: %v", err)
	}

	expiresAt := "2026-04-30T12:00:00Z"
	strength := 1.0
	rootSignals := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "focus-1",
				Type:      "FOCUS",
				Priority:  "normal",
				Source:    "root",
				CreatedAt: "2026-04-19T10:00:00Z",
				Active:    true,
				Strength:  &strength,
				Content:   json.RawMessage(`{"text":"security"}`),
			},
			{
				ID:        "feedback-1",
				Type:      "FEEDBACK",
				Priority:  "low",
				Source:    "root",
				CreatedAt: "2026-04-18T10:00:00Z",
				Active:    false,
				ExpiresAt: &expiresAt,
				Content:   json.RawMessage(`{"text":"stale note"}`),
			},
		},
	}
	if err := store.SaveJSON("pheromones.json", rootSignals); err != nil {
		t.Fatalf("save source pheromones: %v", err)
	}

	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"pheromone-snapshot-inject", "--source-root", sourceRoot, "--target-root", targetRoot})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("pheromone-snapshot-inject returned error: %v", err)
	}

	targetStore, _, err := newPheromoneStoreForRoot(targetRoot)
	if err != nil {
		t.Fatalf("target store: %v", err)
	}
	var pf colony.PheromoneFile
	if err := targetStore.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load injected pheromones: %v", err)
	}
	if len(pf.Signals) != 1 {
		t.Fatalf("expected only active signal to be injected, got %d", len(pf.Signals))
	}
	if text := extractText(pf.Signals[0].Content); text != "security" {
		t.Fatalf("injected content = %q, want security", text)
	}
}

func TestMergePheromoneFilesUpdatesExistingSignalByID(t *testing.T) {
	target := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{{
			ID:                 "focus-1",
			Type:               "FOCUS",
			Priority:           "normal",
			Source:             "root",
			CreatedAt:          "2026-04-19T10:00:00Z",
			Active:             true,
			Content:            json.RawMessage(`{"text":"security"}`),
			ReinforcementCount: intPtr(1),
		}},
	}
	source := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{{
			ID:                 "focus-1",
			Type:               "FOCUS",
			Priority:           "normal",
			Source:             "worktree",
			CreatedAt:          "2026-04-19T11:00:00Z",
			Active:             true,
			Content:            json.RawMessage(`{"text":"security"}`),
			ReinforcementCount: intPtr(2),
		}},
	}

	result := mergePheromoneFiles(&target, source, pheromoneSyncOptions{})
	if result.UpdatedSignals != 1 {
		t.Fatalf("updated_signals = %d, want 1", result.UpdatedSignals)
	}
	if len(target.Signals) != 1 {
		t.Fatalf("expected merged signal set, got %d signals", len(target.Signals))
	}
	if count := signalObservationCount(target.Signals[0]); count != 3 {
		t.Fatalf("observation count = %d, want 3", count)
	}
}

func TestMergePheromoneFilesDedupesEquivalentNewSignal(t *testing.T) {
	target := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{{
			ID:        "focus-1",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "root",
			CreatedAt: "2026-04-19T10:00:00Z",
			Active:    true,
			Content:   json.RawMessage(`{"text":"security"}`),
		}},
	}
	source := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{{
			ID:        "focus-2",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "worktree",
			CreatedAt: "2026-04-19T11:00:00Z",
			Active:    true,
			Content:   json.RawMessage(`{"text":"security"}`),
		}},
	}

	result := mergePheromoneFiles(&target, source, pheromoneSyncOptions{})
	if result.DedupedExisting != 1 {
		t.Fatalf("deduped_existing = %d, want 1", result.DedupedExisting)
	}
	if len(target.Signals) != 1 {
		t.Fatalf("expected duplicate signal to merge into existing entry, got %d signals", len(target.Signals))
	}
	if count := signalObservationCount(target.Signals[0]); count != 2 {
		t.Fatalf("observation count = %d, want 2", count)
	}
}

func intPtr(v int) *int {
	return &v
}
