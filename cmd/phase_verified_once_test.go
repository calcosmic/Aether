package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
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
		dispatches := testPlannedBuildDispatchesWithJudgement(phase, state, nil, colony.VerificationDepthStandard, nil, "")
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

	dispatches := testPlannedBuildDispatchesWithJudgement(
		phase, state, nil, colony.VerificationDepthStandard,
		[]string{"builder", "watcher"}, "owner asked for an explicit watcher pass",
		map[string]string{"watcher": "owner asked for an explicit watcher pass"},
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

// TestPhaseVerifiedOnce proves ruling D11 rule 4
// (.planning/decisions/2026-08-22-queen-decides-program-checks.md) for the
// specific claim Task 1 of this plan makes true: the watcher is not
// dispatched at both the build boundary and the continue boundary for the
// same phase unless the Queen's proposal explicitly named it. It plans a
// real build manifest (plannedBuildDispatchesWithJudgement) and a real
// continue plan (plannedContinueReviewDispatches + continueWatcherDecision,
// the two functions that decide continue's dispatched castes) for the same
// phase, and asserts on the intersection of the two caste sets as a set --
// not on any stage name or single hardcoded caste name beyond "watcher"
// itself -- so a future change that moves the watcher dispatch to a
// different stage still gets caught.
//
// Measured baseline this exists to prevent (2026-08-22, before this plan):
// a one-task bug fix was sent 8 workers, 3 of them verification on the
// continue side, because the build side ALSO always dispatched a watcher
// under the required-caste floor with no Queen proposal involved.
//
// Scope note (this plan's own frontmatter, "flagged_assumptions": "FLOOR-04
// edge probe row is unclassified -- NOT auto-resolved"): probe, auditor and
// gatekeeper still legitimately double-dispatch today on production/security
// phases with NO explicit Queen proposal on either side -- both the build
// and continue deterministic engines (queenOrchestrate, and for continue at
// non-light depth, cmd/caste_relevance.go isAlwaysRequired's "continue" case)
// independently decide a security-shaped or testable-code phase needs the
// same specialist. That is real, and it is NOT something this plan's Task 1
// touches -- the plan explicitly scopes required-caste floor changes to
// Phase 194 ("Phase 194 moves the required-caste floor; 193 only stops the
// duplicate [watcher] dispatch"). Asserting full intersection-emptiness
// across every caste here would fail on the production and high-risk rows
// for a real, pre-existing reason this plan does not fix -- that would be
// testing Phase 194's claim, not this plan's. This test is scoped to the
// watcher claim Task 1 actually delivers; the wider probe/auditor/gatekeeper
// gap is recorded in 193-02-SUMMARY.md and the WINDOWS.md defect ledger for
// Phase 194 to close.
func TestPhaseVerifiedOnce(t *testing.T) {
	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	production := phaseVerifiedOncePhase("Ship the release", "Production deploy", colony.PhaseModeProduction)

	for _, tc := range []struct {
		name               string
		phase              colony.Phase
		proposedCastes     []string
		casteReason        string
		casteReasons       map[string]string
		wantWatcherOverlap bool
	}{
		{
			name:  "documentation-only, no proposal",
			phase: phaseVerifiedOncePhase("Write the README", "Documentation only, no code changes", colony.PhaseModeMaintenance),
		},
		{
			name:  "prototype, no proposal",
			phase: phaseVerifiedOncePhase("Add a hello endpoint", "Implement the /hello route", colony.PhaseModePrototype),
		},
		{
			name:  "production, no proposal",
			phase: production,
		},
		{
			name:  "high-risk, no proposal",
			phase: phaseVerifiedOncePhase("Password reset", "Let users reset their password via an emailed token", colony.PhaseModeProduction),
		},
		{
			// The row proving this test measures the rule, not mere
			// emptiness: an explicit proposal naming the watcher makes the
			// intersection non-empty, on purpose, on both sides.
			name:               "production, explicit watcher proposal",
			phase:              production,
			proposedCastes:     []string{"builder", "watcher"},
			casteReason:        "owner asked for an explicit watcher pass",
			casteReasons:       map[string]string{"watcher": "owner asked for an explicit watcher pass"},
			wantWatcherOverlap: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{tc.phase}}}

			buildDispatches := testPlannedBuildDispatchesWithJudgement(
				tc.phase, state, nil, colony.VerificationDepthStandard, tc.proposedCastes, tc.casteReason, tc.casteReasons,
			)
			buildCastes := map[string]bool{}
			for _, d := range buildDispatches {
				buildCastes[d.Caste] = true
			}

			continueDispatches := plannedContinueReviewDispatches(
				"/tmp", tc.phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{},
				&codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, tc.proposedCastes, tc.casteReason, tc.casteReasons,
			)
			continueCastes := map[string]bool{}
			for _, d := range continueDispatches {
				continueCastes[d.Caste] = true
			}
			if dispatchWatcher, _ := continueWatcherDecision(state, tc.phase, codexContinueManifest{}, codexWatcherVerification{}, false); dispatchWatcher {
				continueCastes["watcher"] = true
			}

			intersection := map[string]bool{}
			for caste := range buildCastes {
				if continueCastes[caste] {
					intersection[caste] = true
				}
			}

			watcherOverlap := intersection["watcher"]
			if watcherOverlap != tc.wantWatcherOverlap {
				t.Fatalf("watcher in build∩continue = %v, want %v (build=%v continue=%v intersection=%v)",
					watcherOverlap, tc.wantWatcherOverlap, buildCastes, continueCastes, intersection)
			}
		})
	}
}
