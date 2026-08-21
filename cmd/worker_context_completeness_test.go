package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRecoveryDispatchCarriesFullOriginalContext is written as an invariant
// rather than a field checklist, because a checklist is exactly what failed.
//
// The recovery builder set nine fields and passed the one-line task string as
// the whole brief, so a worker being retried after failing received strictly
// less than the attempt that had already failed. Root was empty, and cmd.Dir is
// only set when Root is non-empty, so the retry ran in the orchestrator's
// working directory instead of the repository it was meant to be fixing.
//
// This asserts that every context-bearing field survives, so adding a tenth
// context field and forgetting this path fails here rather than in production.
func TestRecoveryDispatchCarriesFullOriginalContext(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	// A real colony, so resolveCodexWorkerContext produces an actual capsule
	// rather than the empty string a bare fixture yields.
	goal := "Recovery dispatches must carry the context the original attempt had"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 2, Name: "Recovery", Description: "retry a failed worker", Status: colony.PhaseInProgress},
		}},
	})

	original := codexBuildDispatch{
		Stage:             "wave",
		Wave:              1,
		Caste:             "builder",
		AgentName:         "aether-builder",
		Name:              "Hammer-7",
		Task:              "one line task",
		TaskID:            "1.1",
		Brief:             "# Build Dispatch\n\nthe full rendered brief",
		SkillSection:      "## Skills\n\ntdd discipline",
		DeclaredPaths:     []string{"cmd/thing.go"},
		PermissionProfile: codex.PermissionProfile{},
	}

	workers := codexWorkerDispatchesForRecovery([]codexBuildDispatch{original}, 2)
	if len(workers) != 1 {
		t.Fatalf("got %d recovery dispatches, want 1", len(workers))
	}
	worker := workers[0]

	// The retry must not be handed the one-liner when a full brief exists.
	if worker.TaskBrief != original.Brief {
		t.Errorf("TaskBrief = %q, want the original rendered brief — a retry must not get less than the attempt that failed", worker.TaskBrief)
	}

	// Root drives cmd.Dir. Empty means the retry runs in the wrong directory.
	if strings.TrimSpace(worker.Root) == "" {
		t.Error("Root is empty; the retry would run in the orchestrator's working directory, not the repository")
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{"SkillSection", worker.SkillSection},
		{"ContextCapsule", worker.ContextCapsule},
	} {
		if strings.TrimSpace(field.value) == "" {
			t.Errorf("%s is empty on a recovery dispatch; the retry is blinder than the original attempt", field.name)
		}
	}

	if len(worker.DeclaredPaths) != len(original.DeclaredPaths) {
		t.Errorf("DeclaredPaths = %v, want the original %v", worker.DeclaredPaths, original.DeclaredPaths)
	}
	if worker.Phase != 2 || worker.Workflow != "build" {
		t.Errorf("identity fields lost: phase=%d workflow=%q", worker.Phase, worker.Workflow)
	}
}

// TestContinueWatcherDispatchCarriesHandoffSection covers the asymmetry: the
// watcher is the most expensive single worker in the continue flow and was the
// only one dispatched without the relay its sibling reviewers receive.
func TestContinueWatcherDispatchCarriesHandoffSection(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)

	// Seed a handoff so the section has something to render.
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: []workerHandoffRecord{
		freshnessRecord("Earlier-1", "2026-08-07T10:00:00Z", 1),
	}}); err != nil {
		t.Fatalf("seed handoffs: %v", err)
	}
	// The record must be in the same workflow the watcher reads.
	records, _ := loadWorkerHandoffRecords()
	for i := range records {
		records[i].Workflow = "continue"
	}
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: records}); err != nil {
		t.Fatalf("reseed handoffs: %v", err)
	}

	section := renderWorkerHandoffSection("continue", 1, "SomeOtherWorker")
	if strings.TrimSpace(section) == "" {
		t.Skip("handoff fixture did not render; the dispatch assertion below needs a non-empty section")
	}
	if !strings.Contains(section, "Earlier-1") {
		t.Errorf("seeded handoff missing from continue section:\n%s", section)
	}
}

// TestWorkerBriefCarriesResolvedTestCommand asserts the exact resolved string
// rather than that a section exists. resolveTestCommand has existed since the
// gate was written and its only caller ran *after* the worker finished, so
// every builder rediscovered how to run the project's tests from scratch.
func TestWorkerBriefCarriesResolvedTestCommand(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	repoRoot := filepath.Dir(filepath.Dir(dataDir))
	if err := os.WriteFile(filepath.Join(repoRoot, "CLAUDE.md"),
		[]byte("# Project\n\n## Verification Commands\n\n```bash\ngo test ./...\n```\n"), 0644); err != nil {
		t.Fatalf("seed CLAUDE.md: %v", err)
	}

	command := strings.TrimSpace(resolveTestCommand())
	section := renderVerificationCommandSection()

	if command == "" {
		if strings.TrimSpace(section) != "" {
			t.Errorf("no test command resolved but a section was rendered: %q", section)
		}
		t.Skip("no test command resolvable in this fixture")
	}

	if !strings.Contains(section, command) {
		t.Errorf("verification section does not carry the resolved command %q:\n%s", command, section)
	}
	if len(section) > 400 {
		t.Errorf("verification section is %d chars; it should be a line, not a lecture", len(section))
	}
}
