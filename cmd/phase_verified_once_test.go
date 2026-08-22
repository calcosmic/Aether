package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// This file asserts ruling D11 rule 4 (.planning/decisions/2026-08-22-queen-decides-program-checks.md):
// each phase is verified once. The build-side "verification" stage and the
// follow-up continue review are one pass, not two, and the same caste is not
// dispatched at both boundaries for the same phase unless the Queen asks.
//
// phaseVerifiedOncePhase builds a minimal phase fixture with one task, so the
// dispatch planner has real work to plan around (an empty-task phase takes a
// different, less representative code path).
func phaseVerifiedOncePhase(name, description string, mode colony.PhaseMode) colony.Phase {
	taskID := "1.1"
	return colony.Phase{
		ID:          1,
		Name:        name,
		Description: description,
		Mode:        mode,
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{{
			ID:     &taskID,
			Goal:   description,
			Status: colony.TaskPending,
		}},
	}
}

// TestBuildPlansNoReviewerWithoutAProposal is FLOOR-04's build-side half: a
// build planned with no caste proposal produces zero dispatches whose Stage is
// the verification stage, for a documentation phase, a prototype phase and a
// production phase alike. The program's free checks are the floor; nothing
// implicit reviews the work a second time.
func TestBuildPlansNoReviewerWithoutAProposal(t *testing.T) {
	for _, phase := range []colony.Phase{
		phaseVerifiedOncePhase("Write the README", "Documentation only", colony.PhaseModeMaintenance),
		phaseVerifiedOncePhase("Add a hello endpoint", "Implement the /hello route", colony.PhaseModePrototype),
		phaseVerifiedOncePhase("Ship the release", "Production deploy", colony.PhaseModeProduction),
	} {
		state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}
		dispatches := plannedBuildDispatchesWithJudgement(phase, state, nil, colony.VerificationDepthStandard, nil, "")
		for _, d := range dispatches {
			if d.Stage == "verification" {
				t.Errorf("phase %q (%s): build planned a verification-stage dispatch with no Queen proposal: %+v",
					phase.Name, phase.Mode, d)
			}
		}
	}
}

// TestBuildStillDispatchesAWatcherTheQueenAskedFor proves the owner override
// survives: a production phase whose proposal explicitly names the watcher
// still produces exactly one verification-stage watcher dispatch.
func TestBuildStillDispatchesAWatcherTheQueenAskedFor(t *testing.T) {
	phase := phaseVerifiedOncePhase("Ship the release", "Production deploy", colony.PhaseModeProduction)
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}

	dispatches := plannedBuildDispatchesWithJudgement(
		phase, state, nil, colony.VerificationDepthStandard,
		[]string{"builder", "watcher"}, "owner asked for an explicit watcher pass",
	)

	count := 0
	for _, d := range dispatches {
		if d.Stage == "verification" && d.Caste == "watcher" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one verification-stage watcher dispatch when the Queen's proposal named it, got %d: %+v", count, dispatches)
	}
}

// writePhaseVerifiedOnceVerificationCommands writes a "## Verification
// Commands" CLAUDE.md section into root so a build-finalize fixture can be
// given a passing or a failing shell command set. An empty command string
// omits that line entirely, leaving the check unresolved.
func writePhaseVerifiedOnceVerificationCommands(t *testing.T, root, build, types, lint, tests string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("## Verification Commands\n\n")
	for _, entry := range []struct{ label, command string }{
		{"build", build},
		{"types", types},
		{"lint", lint},
		{"tests", tests},
	} {
		if strings.TrimSpace(entry.command) == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", entry.label, entry.command)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(b.String()), 0644); err != nil {
		t.Fatalf("write CLAUDE.md verification commands: %v", err)
	}
}

// TestBuildFinalizeRecordsFreeChecksAsAReport is Task 2's build half of
// FLOOR-04: build finalize records the deterministic free checks (build,
// types, lint, tests) as a report on the attempt journal.
func TestBuildFinalizeRecordsFreeChecksAsAReport(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	writePhaseVerifiedOnceVerificationCommands(t, root, "true", "true", "true", "true")
	_, completion := prepareExternalBuildCompletion(t, root)

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("runCodexBuildFinalize returned error: %v", err)
	}

	_, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("expected a build attempt record after finalize")
	}
	if attempt.FreeChecks == nil {
		t.Fatal("expected the attempt record to carry a free-check report")
	}
	if !attempt.FreeChecks.Passed {
		t.Fatalf("expected the free-check report to pass with all-green commands: %+v", attempt.FreeChecks)
	}
	for _, want := range []string{"build", "types", "lint", "tests"} {
		if !containsString(attempt.FreeChecks.ChecksRun, want) {
			t.Errorf("free-check report ChecksRun missing %q: %+v", want, attempt.FreeChecks.ChecksRun)
		}
	}
	if strings.TrimSpace(attempt.FreeChecks.Summary) == "" {
		t.Error("free-check report Summary should not be empty")
	}
}

// TestBuildFinalizeFreeChecksDoNotAdvanceThePhase is Task 2's D-08 half: the
// free-check report changes, but advancement (phase/task status) does not --
// that decision stays `continue`'s job either way.
func TestBuildFinalizeFreeChecksDoNotAdvanceThePhase(t *testing.T) {
	passRoot := setupExternalBuildAttemptTest(t)
	writePhaseVerifiedOnceVerificationCommands(t, passRoot, "true", "true", "true", "true")
	_, passCompletion := prepareExternalBuildCompletion(t, passRoot)
	_, passState, _, _, err := runCodexBuildFinalize(passRoot, 1, passCompletion, false)
	if err != nil {
		t.Fatalf("finalize (passing checks) returned error: %v", err)
	}
	_, passAttempt, ok := loadLatestBuildAttempt(1)
	if !ok || passAttempt.FreeChecks == nil {
		t.Fatal("expected a free-check report on the passing-checks attempt")
	}
	if !passAttempt.FreeChecks.Passed {
		t.Fatalf("expected the passing-checks report to pass: %+v", passAttempt.FreeChecks)
	}

	failRoot := setupExternalBuildAttemptTest(t)
	writePhaseVerifiedOnceVerificationCommands(t, failRoot, "true", "true", "true", "false")
	_, failCompletion := prepareExternalBuildCompletion(t, failRoot)
	_, failState, _, _, err := runCodexBuildFinalize(failRoot, 1, failCompletion, false)
	if err != nil {
		t.Fatalf("finalize (failing checks) returned error: %v", err)
	}
	_, failAttempt, ok := loadLatestBuildAttempt(1)
	if !ok || failAttempt.FreeChecks == nil {
		t.Fatal("expected a free-check report on the failing-checks attempt")
	}
	if failAttempt.FreeChecks.Passed {
		t.Fatalf("expected the failing-checks report to fail: %+v", failAttempt.FreeChecks)
	}

	// The report differs (proven above). The advancement decision must not.
	if passState.Plan.Phases[0].Status != failState.Plan.Phases[0].Status {
		t.Fatalf("phase status differs by free-check outcome: pass=%s fail=%s -- advancement must stay continue's job",
			passState.Plan.Phases[0].Status, failState.Plan.Phases[0].Status)
	}
	if len(passState.Plan.Phases[0].Tasks) != len(failState.Plan.Phases[0].Tasks) {
		t.Fatalf("task count differs between pass and fail runs: %d vs %d", len(passState.Plan.Phases[0].Tasks), len(failState.Plan.Phases[0].Tasks))
	}
	for i := range passState.Plan.Phases[0].Tasks {
		if passState.Plan.Phases[0].Tasks[i].Status != failState.Plan.Phases[0].Tasks[i].Status {
			t.Errorf("task %d status differs by free-check outcome: pass=%s fail=%s",
				i, passState.Plan.Phases[0].Tasks[i].Status, failState.Plan.Phases[0].Tasks[i].Status)
		}
	}
}
