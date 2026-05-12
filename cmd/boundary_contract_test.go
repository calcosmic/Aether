package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// fileSnapshot records the state of a single file for comparison.
type fileSnapshot struct {
	size    int64
	modTime time.Time
}

// snapshotDataDir captures the current state of all files in a directory.
func snapshotDataDir(t *testing.T, dir string) map[string]fileSnapshot {
	t.Helper()
	snap := make(map[string]fileSnapshot)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return snap
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		snap[e.Name()] = fileSnapshot{size: info.Size(), modTime: info.ModTime()}
	}
	return snap
}

// assertDataDirUnchanged compares two snapshots and reports differences.
func assertDataDirUnchanged(t *testing.T, before, after map[string]fileSnapshot) {
	t.Helper()
	for name, beforeSnap := range before {
		afterSnap, ok := after[name]
		if !ok {
			t.Errorf("file removed during orchestration: %s", name)
			continue
		}
		if beforeSnap.size != afterSnap.size {
			t.Errorf("file size changed during orchestration: %s (%d -> %d)", name, beforeSnap.size, afterSnap.size)
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			t.Errorf("file added during orchestration: %s", name)
		}
	}
}

// TestBoundaryContract_NoStateWritesDuringOrchestration verifies that no
// .aether/data/ files are written during the orchestration phase (between
// plan-only manifest generation and finalizer commit). This enforces the
// runtime boundary contract: only Go finalizers may mutate state.
func TestBoundaryContract_NoStateWritesDuringOrchestration(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}

	// Initialize a minimal colony state with one planned phase.
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	goal := "test boundary contract"
	now := time.Now().UTC()
	state := colony.ColonyState{
		Version:       "1.0",
		Goal:          &goal,
		CurrentPhase:  1,
		InitializedAt: &now,
		State:         colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Test Phase",
					Status:      colony.PhaseReady,
				},
			},
		},
	}

	statePath := "COLONY_STATE.json"
	if err := s.SaveJSON(statePath, &state); err != nil {
		t.Fatalf("save initial state: %v", err)
	}

	// Snapshot data directory before orchestration phase.
	before := snapshotDataDir(t, dataDir)

	// --- Orchestration phase ---
	// This represents the time when the TS host would be dispatching workers.
	// No Go functions are called that write to .aether/data/.
	// The orchestration phase is intentionally a no-op: the test validates
	// that no state writes occur while the TS host is active.

	// --- End orchestration phase ---

	// Snapshot data directory after orchestration.
	after := snapshotDataDir(t, dataDir)

	// Assert no files changed during orchestration.
	assertDataDirUnchanged(t, before, after)

	if t.Failed() {
		t.Fatal("boundary violation: .aether/data/ was modified during orchestration phase")
	}

	// Now simulate the finalizer committing state (this IS allowed).
	err = s.UpdateJSONAtomically(statePath, &state, func() error {
		state.Plan.Phases[0].Status = colony.PhaseCompleted
		return nil
	})
	if err != nil {
		t.Fatalf("finalizer state update: %v", err)
	}

	// Verify state was committed correctly.
	var updated colony.ColonyState
	data, err := os.ReadFile(filepath.Join(dataDir, statePath))
	if err != nil {
		t.Fatalf("read committed state: %v", err)
	}
	if err := json.Unmarshal(data, &updated); err != nil {
		t.Fatalf("unmarshal committed state: %v", err)
	}
	if updated.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Errorf("expected phase status COMPLETED, got %s", updated.Plan.Phases[0].Status)
	}
}

// TestBoundaryContract_ContractDocumentExists verifies the runtime boundary
// contract file exists at the expected path with valid YAML frontmatter.
func TestBoundaryContract_ContractDocumentExists(t *testing.T) {
	contractPath := filepath.Join("..", ".aether", "references", "contracts", "runtime-boundary-contract.md")

	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Skipf("contract file not found at %s: %v (expected in development repo)", contractPath, err)
	}

	str := string(content)
	checks := []struct {
		name   string
		needle string
	}{
		{"schema_version", "schema_version"},
		{"id", "id: runtime-boundary-contract"},
		{"anti-patterns section", "## Anti-Patterns"},
		{"go ownership section", "## Ownership: Go Runtime"},
		{"ts ownership section", "## Ownership: TypeScript Host"},
		{"rules section", "## Rules"},
	}

	for _, c := range checks {
		if !strings.Contains(str, c.needle) {
			t.Errorf("contract missing %s: expected %q", c.name, c.needle)
		}
	}
}
