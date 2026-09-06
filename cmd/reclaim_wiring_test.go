package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestDispatchComposesForPhaseCharacter is Phase 166's positive-direction
// complement to Phase 182's suppression tests: the right specialist actually
// shows up for the right work, observable in the composed dispatch — and an
// unrelated specialist does not ride along.
//
// Plan 194-05 (D-11) removed the no-proposal keyword-scoring fallback from
// queenOrchestrate's build path entirely, so these subtests call
// queenCandidateDispatches (the scoring engine itself, unaffected by the
// gate on the no-proposal ENTRY POINT) rather than queenOrchestrate.
func TestDispatchComposesForPhaseCharacter(t *testing.T) {
	t.Run("legacy phase summons the archaeologist, not the includer", func(t *testing.T) {
		phase := colony.Phase{
			ID:          3,
			Name:        "Modernize the legacy billing module",
			Description: "Refactor old legacy payment code; migration of historical billing records.",
			Mode:        colony.PhaseModePrototype,
			Tasks: []colony.Task{
				{Goal: "Map the legacy billing paths before rewriting them"},
				{Goal: "Migrate historical records to the new schema"},
			},
		}
		dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})
		if !HasCaste(dispatches, "archaeologist") {
			t.Errorf("legacy-touching phase did not summon the Archaeologist: %+v", dispatches)
		}
		if HasCaste(dispatches, "includer") {
			t.Errorf("legacy-touching phase summoned the accessibility specialist for no reason: %+v", dispatches)
		}
	})

	t.Run("hardening phase summons chaos", func(t *testing.T) {
		phase := colony.Phase{
			ID:          4,
			Name:        "Resilience hardening",
			Description: "Stress test the queue under failure; crash recovery and error handling robustness.",
			Mode:        colony.PhaseModeProduction,
			Tasks: []colony.Task{
				{Goal: "Stress test the retry path under simulated crash conditions"},
			},
		}
		dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})
		if !HasCaste(dispatches, "chaos") {
			t.Errorf("hardening phase did not summon Chaos: %+v", dispatches)
		}
	})

	t.Run("retrospective close summons the sage", func(t *testing.T) {
		phase := colony.Phase{
			ID:          5,
			Name:        "Milestone retrospective",
			Description: "Synthesize the milestone's learnings into reusable wisdom patterns; retrospective on what the colony learned.",
			Mode:        colony.PhaseModeMaintenance,
			Tasks: []colony.Task{
				{Goal: "Synthesize phase learnings into wisdom patterns for the retrospective"},
			},
		}
		dispatches := queenCandidateDispatches(phase, "build", colony.ColonyState{})
		if !HasCaste(dispatches, "sage") {
			t.Errorf("retrospective phase did not summon the Sage: %+v", dispatches)
		}
	})
}

// TestInitRegistersColonyAndSeedsHive proves the RECLAIM-02/09 wiring: a
// plain `aether init` registers the repo (with detected domains) in the hub
// colony registry and seeds QUEEN.md from hive wisdom — behaviour v5.4.0 had
// and the modern runtime computed for nobody.
func TestInitRegistersColonyAndSeedsHive(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")
	saveGlobals(t)
	resetRootCmd(t)

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)
	if err := os.MkdirAll(filepath.Join(hubDir, "hive"), 0755); err != nil {
		t.Fatalf("mkdir hive: %v", err)
	}
	wisdom := `{"entries":[{"id":"w1","text":"Prefer failing tests before fixes","confidence":0.9}]}`
	if err := os.WriteFile(filepath.Join(hubDir, "hive", "wisdom.json"), []byte(wisdom), 0644); err != nil {
		t.Fatalf("seed wisdom: %v", err)
	}

	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)
	// A go.mod marker so domain detection has something real to find.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"init", "Ship the registry wiring"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Registry entry exists, active, with the detected domain and the goal.
	raw, err := os.ReadFile(filepath.Join(hubDir, "registry", "registry.json"))
	if err != nil {
		t.Fatalf("init did not create the hub registry: %v", err)
	}
	var rd registryData
	if err := json.Unmarshal(raw, &rd); err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	found := false
	for _, entry := range rd.Colonies {
		if entry.RepoPath == root {
			found = true
			if !entry.Active {
				t.Errorf("registry entry not marked active after init")
			}
			if entry.LastGoal != "Ship the registry wiring" {
				t.Errorf("registry goal = %q", entry.LastGoal)
			}
			if len(entry.Domains) == 0 || entry.Domains[0] != "go" {
				t.Errorf("registry domains = %v, want detected [go]", entry.Domains)
			}
		}
	}
	if !found {
		t.Fatalf("init did not register the colony; registry: %+v", rd.Colonies)
	}

	// QUEEN.md gained the hive wisdom entry.
	queenText, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil || !strings.Contains(string(queenText), "Prefer failing tests before fixes") {
		t.Errorf("hive wisdom was not seeded into QUEEN.md (err=%v)", err)
	}

	// The birth close announces the seeding, classic style.
	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "🧠 Hive wisdom: 1 cross-colony pattern(s) seeded into QUEEN.md") {
		t.Errorf("init close missing the hive seed line:\n%s", output)
	}
}

// TestSealMarksRegistryEntryInactive: sealing flips the colony's registry
// entry to inactive with its goal preserved — the registry reads as a true
// history, not a list of ghosts marked forever active.
func TestSealMarksRegistryEntryInactive(t *testing.T) {
	saveGlobals(t)
	s, repo := newTestStore(t)
	store = s

	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	if _, err := upsertColonyRegistryEntry(repo, "the goal", []string{"go"}, true); err != nil {
		t.Fatalf("seed entry: %v", err)
	}
	if updated, err := upsertColonyRegistryEntry(repo, "the goal", nil, false); err != nil || !updated {
		t.Fatalf("seal-side upsert: updated=%v err=%v", updated, err)
	}

	raw, err := os.ReadFile(filepath.Join(hubDir, "registry", "registry.json"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var rd registryData
	if err := json.Unmarshal(raw, &rd); err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	if len(rd.Colonies) != 1 {
		t.Fatalf("expected one entry, got %d", len(rd.Colonies))
	}
	entry := rd.Colonies[0]
	if entry.Active {
		t.Errorf("entry still active after the inactive upsert")
	}
	if len(entry.Domains) != 1 || entry.Domains[0] != "go" {
		t.Errorf("inactive upsert lost the domains: %v", entry.Domains)
	}
	if entry.LastGoal != "the goal" {
		t.Errorf("inactive upsert lost the goal: %q", entry.LastGoal)
	}
}

// TestMedicFixCreatesRollbackCheckpoint (RECLAIM-01): before medic applies
// repairs, a rollback-able state checkpoint exists and the repair result
// names it — the undo command is on screen, not in source code.
func TestMedicFixCreatesRollbackCheckpoint(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "checkpoint fixture"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateREADY}); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	scan := &ScannerResult{Issues: []HealthIssue{}}
	result, err := performRepairs(scan, MedicOptions{Fix: true}, s.BasePath())
	if err != nil {
		t.Fatalf("performRepairs: %v", err)
	}
	if result.Checkpoint == "" {
		t.Fatalf("repair result carries no checkpoint name")
	}
	if _, err := os.Stat(filepath.Join(s.BasePath(), "checkpoints", result.Checkpoint+".json")); err != nil {
		t.Fatalf("checkpoint file missing: %v", err)
	}
}

// TestAutofixRollbackRestoresStateNotEnvelope pins the rollback fix: the
// original implementation wrote the checkpoint ENVELOPE over
// COLONY_STATE.json — corrupting exactly what it claimed to restore. It
// shipped broken and was never caught because nothing called it.
func TestAutofixRollbackRestoresStateNotEnvelope(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "the original goal"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateREADY}); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	name, _, err := createAutofixCheckpoint("test issue")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	// Simulate a bad repair mangling the state.
	mangled := "the mangled goal"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Version: "3.0", Goal: &mangled, State: colony.StateREADY}); err != nil {
		t.Fatalf("mangle state: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"autofix-rollback", "--checkpoint-id", name})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var restored colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &restored); err != nil {
		t.Fatalf("rolled-back state is not parseable colony state (the envelope bug): %v", err)
	}
	if restored.Goal == nil || *restored.Goal != "the original goal" {
		t.Fatalf("rollback did not restore the checkpointed state; goal = %v", restored.Goal)
	}
}
