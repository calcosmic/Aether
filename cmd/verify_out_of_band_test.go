package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- Fixtures -----------------------------------------------------------

// setupOutOfBandTest sets up a colony sitting mid-phase (State: EXECUTING,
// the phase itself: PhaseInProgress) with NO build attempt recorded at
// all -- the "nothing was ever dispatched for this phase" shape -- and a
// real, working Go workspace so the ceremony's fresh build/types/lint/tests
// commands are genuine subprocess runs, not stubs.
func setupOutOfBandTest(t *testing.T, phase colony.Phase) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	phase.ID = 1
	phase.Status = colony.PhaseInProgress
	goal := "Reconcile work done outside the build pipeline"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		ColonyDepth:  "standard",
		CurrentPhase: phase.ID,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})
	return root
}

// setupOutOfBandStuckFixture reproduces the field scenario
// (.planning/todos/pending/2026-08-21-no-reentry-path-for-out-of-band-work.md)
// literally: a phase whose build attempt genuinely started -- real
// dispatches were planned via the SAME plan-only build path
// TestForcedRedispatchAfterBuiltIsNotADeadlock (cmd/build_finalize_deadlock_test.go)
// uses -- but no worker ever produced a result. That is the exact "stuck,
// no honest way to close it" shape this whole plan exists to fix.
//
// runCodexBuildPlanOnly (the wrapper-delegated manifest path) deliberately
// does NOT itself commit the colony to EXECUTING/PhaseInProgress -- that
// commit is applyCodexBuildState, called only from build-finalize
// (cmd/codex_build_finalize.go) and the hosted queen-led dispatch path
// (cmd/codex_build.go), neither of which this fixture needs. So after
// plan-only produces a real attempt record with `planned` dispatches, this
// fixture applies the SAME durable transition by hand -- the state a crash
// or an abandoned wrapper dispatch would genuinely leave on disk.
func setupOutOfBandStuckFixture(t *testing.T, phase colony.Phase) (root string, dataDir string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir = setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Reconcile work done outside the build pipeline"
	taskID := "1.1"
	phase.ID = 1
	phase.Status = colony.PhaseReady
	phase.Tasks = []colony.Task{{ID: &taskID, Goal: "Score intake photos", Status: colony.TaskPending}}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan:         colony.Plan{Phases: []colony.Phase{phase}},
	})

	if _, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil); err != nil {
		t.Fatalf("plan-only build to create the stuck attempt: %v", err)
	}

	var stuck colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &stuck, func() error {
		stuck.State = colony.StateEXECUTING
		stuck.CurrentPhase = 1
		stuck.Plan.Phases[0].Status = colony.PhaseInProgress
		return nil
	}); err != nil {
		t.Fatalf("apply the durable stuck-mid-build state: %v", err)
	}
	return root, dataDir
}

func boundPassingOutOfBandPhase() colony.Phase {
	return colony.Phase{
		Name:            "Photo intake",
		SuccessCriteria: []string{"the workspace builds and its tests pass"},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
			Criterion: "the workspace builds and its tests pass",
			Artifacts: []string{"main.go"},
			Checks:    []string{"build", "tests"},
		}},
	}
}

func boundUnsupportedOutOfBandPhase() colony.Phase {
	return colony.Phase{
		Name:            "Photo intake",
		SuccessCriteria: []string{"a watcher reviewed the change"},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{{
			Criterion: "a watcher reviewed the change",
			Checks:    []string{"watcher"},
		}},
	}
}

func legacyOutOfBandPhase() colony.Phase {
	return colony.Phase{
		Name:            "Photo intake",
		SuccessCriteria: []string{"photos are scored correctly, confirmed by hand-reviewed spot checks"},
	}
}

// writeFailingOutOfBandTest overwrites the fixture workspace's test file
// with one that genuinely fails, so the ceremony's fresh `go test ./...`
// run is a real, live failure -- not a mock.
func writeFailingOutOfBandTest(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) { t.Fatal(\"deliberate failure for the verify-out-of-band rubber-stamp probe\") }\n"), 0644); err != nil {
		t.Fatalf("write failing test fixture: %v", err)
	}
}

// --- Task 1: functional correctness --------------------------------------

// TestVerifyOutOfBandReportOnlyMutatesNothing is CLAUDE.md's dry-run
// corollary, applied to this ceremony: consolidation-phase-end --dry-run
// and consolidation-seal --dry-run both wrote to instincts.json for months
// despite documenting "report without modifying". The default (no --force)
// invocation must leave COLONY_STATE.json byte-identical, not just
// "logically unchanged".
func TestVerifyOutOfBandReportOnlyMutatesNothing(t *testing.T) {
	root := setupOutOfBandTest(t, boundPassingOutOfBandPhase())
	stateFile := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")

	before, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read colony state before: %v", err)
	}
	beforeInfo, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("stat colony state before: %v", err)
	}

	report, result, err := executeVerifyOutOfBand(context.Background(), 1, false, false)
	if err != nil {
		t.Fatalf("report-only invocation returned error: %v", err)
	}
	if result != nil {
		t.Fatalf("report-only invocation returned a close result: %+v", result)
	}
	if !report.Passed {
		t.Fatalf("expected the fixture's bound criteria to pass fresh verification, got blocking issues: %v / steps: %+v", report.BlockingIssues, report.Steps)
	}

	after, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read colony state after: %v", err)
	}
	afterInfo, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("stat colony state after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("report-only invocation mutated COLONY_STATE.json content")
	}
	if !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatalf("report-only invocation mutated COLONY_STATE.json mtime: %v -> %v", beforeInfo.ModTime(), afterInfo.ModTime())
	}
	if _, _, ok := loadLatestBuildAttempt(1); ok {
		t.Fatal("report-only invocation fabricated a build attempt")
	}
}

// TestVerifyOutOfBandClosesFieldReportedStuckState is the field
// reproduction (191.1-PATTERNS.md Pattern 1): the exact stuck shape
// 2026-08-21-no-reentry-path-for-out-of-band-work.md reports -- a real
// build attempt with dispatches still `planned` and no worker ever ran --
// closed honestly through the new ceremony, citing fresh verification
// instead of nine fabricated worker results.
func TestVerifyOutOfBandClosesFieldReportedStuckState(t *testing.T) {
	root, _ := setupOutOfBandStuckFixture(t, boundPassingOutOfBandPhase())
	_ = root

	// Assert the precondition rather than assume it (191.1-PATTERNS.md
	// Pattern 2): this test is only meaningful while the attempt genuinely
	// has no worker results at all.
	attemptRel, attempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("fixture did not produce a build attempt")
	}
	if attempt.CompletionSHA256 != "" || attempt.Claims != nil {
		t.Fatalf("fixture broken: attempt already carries terminal evidence (digest=%q claims=%v)", attempt.CompletionSHA256, attempt.Claims != nil)
	}
	planned := 0
	for _, dispatch := range attempt.Dispatches {
		if strings.TrimSpace(dispatch.Status) == "planned" {
			planned++
		}
	}
	if planned == 0 || planned != len(attempt.Dispatches) {
		t.Fatalf("fixture broken: %d of %d dispatches are planned", planned, len(attempt.Dispatches))
	}
	if len(attempt.WorkerRuns) != 0 {
		t.Fatalf("fixture broken: attempt already has %d worker runs", len(attempt.WorkerRuns))
	}

	report, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err != nil {
		t.Fatalf("ceremony refused a genuinely satisfied phase: %v (blocking: %v)", err, report.BlockingIssues)
	}
	if result == nil || result["advanced"] != true {
		t.Fatalf("ceremony did not report advancement: %+v", result)
	}

	var closed buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &closed); err != nil {
		t.Fatalf("reload closed attempt: %v", err)
	}
	if closed.Status != buildAttemptBuilt {
		t.Fatalf("attempt status = %s, want %s", closed.Status, buildAttemptBuilt)
	}
	if closed.OutOfBandVerification == nil {
		t.Fatal("closed attempt carries no out-of-band verification provenance")
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase status = %s, want %s", state.Plan.Phases[0].Status, colony.PhaseCompleted)
	}
}

// TestVerifyOutOfBandRefusesWhenFreshVerificationFails is the rubber-stamp
// probe: a phase whose criteria are NOT genuinely met must be refused, not
// passed because a phase whose bound criterion is nominally present.
func TestVerifyOutOfBandRefusesWhenFreshVerificationFails(t *testing.T) {
	root := setupOutOfBandTest(t, boundPassingOutOfBandPhase())
	writeFailingOutOfBandTest(t, root)
	stateFile := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")

	before, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read colony state before: %v", err)
	}

	report, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err == nil {
		t.Fatalf("expected refusal for a phase whose tests genuinely fail, got success: report=%+v result=%+v", report, result)
	}
	if result != nil {
		t.Fatalf("refusal should not return a close result, got %+v", result)
	}
	if report.Passed {
		t.Fatal("report claims Passed=true for a genuinely failing phase")
	}
	if !strings.Contains(err.Error(), "tests") {
		t.Fatalf("refusal should name the failing tests check, got: %v", err)
	}

	after, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read colony state after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("refused ceremony still mutated COLONY_STATE.json")
	}
	if _, _, ok := loadLatestBuildAttempt(1); ok {
		t.Fatal("refused ceremony fabricated a build attempt")
	}
}

// TestVerifyOutOfBandRefusesUnsupportedWorkerEvidenceChecks proves the
// ceremony refuses -- rather than silently passing or silently skipping --
// a bound criterion whose Checks require evidence only a real worker or an
// executed watcher review could honestly supply.
func TestVerifyOutOfBandRefusesUnsupportedWorkerEvidenceChecks(t *testing.T) {
	root := setupOutOfBandTest(t, boundUnsupportedOutOfBandPhase())
	_ = root

	report, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err == nil {
		t.Fatalf("expected refusal for a criterion requiring watcher evidence, got success: %+v", result)
	}
	if report.Passed {
		t.Fatal("report claims Passed=true for an unsupported-evidence criterion")
	}
	if len(report.Criteria) != 1 || !report.Criteria[0].Unsupported {
		t.Fatalf("expected exactly one unsupported criterion in the report, got %+v", report.Criteria)
	}
	found := false
	for _, issue := range report.Criteria[0].BlockingIssues {
		if strings.Contains(issue, "watcher") && strings.Contains(issue, "cannot honestly supply") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the refusal to name the unsupported watcher evidence honestly, got %v", report.Criteria[0].BlockingIssues)
	}
	if _, _, ok := loadLatestBuildAttempt(1); ok {
		t.Fatal("refused ceremony fabricated a build attempt")
	}
}

// TestVerifyOutOfBandRefusesLegacyCriteriaWithoutAcknowledgment proves
// T-191.1-03-04: a phase with only free-prose success criteria cannot be
// closed on the strength of passing shell commands alone. It requires the
// separate, explicit --acknowledge-legacy-criteria flag, and the prose
// criteria are surfaced so the operator is confirming something they were
// shown.
func TestVerifyOutOfBandRefusesLegacyCriteriaWithoutAcknowledgment(t *testing.T) {
	root := setupOutOfBandTest(t, legacyOutOfBandPhase())
	_ = root

	report, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err == nil {
		t.Fatalf("expected refusal without --acknowledge-legacy-criteria, got success: %+v", result)
	}
	if !report.RequiresAcknowledgment || report.Acknowledged {
		t.Fatalf("report acknowledgment state wrong: requires=%v acknowledged=%v", report.RequiresAcknowledgment, report.Acknowledged)
	}
	if len(report.LegacySuccessCriteria) == 0 {
		t.Fatal("report did not surface the phase's prose success criteria")
	}

	report2, result2, err2 := executeVerifyOutOfBand(context.Background(), 1, true, true)
	if err2 != nil {
		t.Fatalf("expected success once legacy criteria are acknowledged and shell checks pass: %v (blocking: %v)", err2, report2.BlockingIssues)
	}
	if result2 == nil || result2["advanced"] != true {
		t.Fatalf("expected the phase to advance, got %+v", result2)
	}
}

// TestCloseOutOfBandCeremonyNeverSealsWithoutAdvancing is the CR-03
// regression lock (191.1-REVIEW.md): closeOutOfBandCeremony used to close
// (irreversibly seal -- Recoverable:false, RecoveryCommand:"") the build
// attempt BEFORE calling advancePhase, so a currency refusal from
// advancePhase (paused colony, stale phase/build identity -- any of the
// same reasons this phase's own supersession machinery exists to catch)
// left the attempt permanently sealed while the phase never advanced at
// all. This reproduces the field shape deterministically (no race
// required): a genuinely satisfied phase, paused directly on disk (an
// entirely ordinary, first-class operational state) between gathering
// fresh evidence and closing.
func TestCloseOutOfBandCeremonyNeverSealsWithoutAdvancing(t *testing.T) {
	root, _ := setupOutOfBandStuckFixture(t, boundPassingOutOfBandPhase())
	_ = root

	attemptRel, before, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("fixture did not produce a build attempt")
	}
	if before.Status != buildAttemptAwaiting {
		t.Fatalf("fixture broken: attempt status = %q, want %q", before.Status, buildAttemptAwaiting)
	}
	if !before.Recoverable || strings.TrimSpace(before.RecoveryCommand) == "" {
		t.Fatalf("fixture broken: attempt is not recoverable before the ceremony runs: recoverable=%v recovery_command=%q", before.Recoverable, before.RecoveryCommand)
	}

	// Confirm fresh evidence genuinely passes BEFORE introducing the pause
	// -- this is a colony whose real, current work is done; only the
	// currency check should refuse it, never the evidence itself.
	report, _, err := executeVerifyOutOfBand(context.Background(), 1, false, false)
	if err != nil {
		t.Fatalf("report-only invocation returned error: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected fresh evidence to genuinely pass before the pause is introduced, got blocking issues: %v", report.BlockingIssues)
	}

	// Pause the colony directly on disk.
	var paused colony.ColonyState
	if updateErr := store.UpdateJSONAtomically("COLONY_STATE.json", &paused, func() error {
		paused.Paused = true
		return nil
	}); updateErr != nil {
		t.Fatalf("pause colony state: %v", updateErr)
	}

	_, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if err == nil {
		t.Fatalf("expected the ceremony to refuse while the colony is paused, got success: %+v", result)
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Fatalf("refusal should name the pause as the reason, got: %v", err)
	}

	var after buildAttemptRecord
	if loadErr := store.LoadJSON(attemptRel, &after); loadErr != nil {
		t.Fatalf("reload attempt after refused ceremony: %v", loadErr)
	}
	if after.Status != buildAttemptAwaiting {
		t.Fatalf("CR-03: attempt status = %q after a refused ceremony, want unchanged %q -- the ceremony sealed the attempt without ever advancing the phase", after.Status, buildAttemptAwaiting)
	}
	if !after.Recoverable {
		t.Fatal("CR-03: attempt Recoverable flipped to false by a refused ceremony -- the only recorded recovery path was destroyed for nothing")
	}
	if strings.TrimSpace(after.RecoveryCommand) == "" {
		t.Fatal("CR-03: attempt RecoveryCommand was cleared by a refused ceremony")
	}
	if after.OutOfBandVerification != nil {
		t.Fatal("CR-03: attempt carries out-of-band provenance despite the ceremony never actually advancing the phase")
	}

	var state colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr != nil {
		t.Fatalf("reload colony state: %v", loadErr)
	}
	if state.Plan.Phases[0].Status == colony.PhaseCompleted {
		t.Fatal("phase advanced despite the ceremony being refused")
	}

	// The colony resuming later must still be able to close it honestly --
	// nothing was left in a state ceremony can no longer recover from.
	rewriteErr := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
		state.Paused = false
		return nil
	})
	if rewriteErr != nil {
		t.Fatalf("resume colony state: %v", rewriteErr)
	}
	_, resumedResult, resumedErr := executeVerifyOutOfBand(context.Background(), 1, true, false)
	if resumedErr != nil {
		t.Fatalf("ceremony refused a genuinely satisfied, resumed phase: %v", resumedErr)
	}
	if resumedResult == nil || resumedResult["advanced"] != true {
		t.Fatalf("ceremony did not advance the phase once resumed: %+v", resumedResult)
	}
}

// --- Task 2: the honesty ratchet ------------------------------------------

// TestVerifyOutOfBandNeverSynthesizesWorkerReceipts is the honesty ratchet
// T-191.1-03-02 requires (191.1-PATTERNS.md Pattern 4): the ceremony's
// closing path may never fabricate a worker dispatch or worker run. This is
// checked as a STRUCTURAL invariant -- Dispatches and WorkerRuns are
// byte-identical (via JSON comparison) before and after the ceremony closes
// an attempt -- not by checking for the presence of an honest-sounding
// field name, which a ceremony that also happened to write synthetic
// entries under the hood would still satisfy.
func TestVerifyOutOfBandNeverSynthesizesWorkerReceipts(t *testing.T) {
	t.Run("existing stuck attempt", func(t *testing.T) {
		root, _ := setupOutOfBandStuckFixture(t, boundPassingOutOfBandPhase())
		_ = root

		attemptRel, before, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatal("fixture did not produce a build attempt")
		}
		if len(before.Dispatches) == 0 {
			t.Fatal("fixture broken: attempt has no dispatches to protect")
		}
		if len(before.WorkerRuns) != 0 {
			t.Fatalf("fixture broken: attempt already has %d worker runs", len(before.WorkerRuns))
		}
		beforeDispatches, err := json.Marshal(before.Dispatches)
		if err != nil {
			t.Fatalf("marshal before dispatches: %v", err)
		}
		beforeWorkerRuns, err := json.Marshal(before.WorkerRuns)
		if err != nil {
			t.Fatalf("marshal before worker runs: %v", err)
		}

		_, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
		if err != nil {
			t.Fatalf("ceremony refused a genuinely satisfied phase: %v", err)
		}
		if result == nil {
			t.Fatal("ceremony did not report a close result")
		}

		var after buildAttemptRecord
		if err := store.LoadJSON(attemptRel, &after); err != nil {
			t.Fatalf("reload attempt: %v", err)
		}
		afterDispatches, err := json.Marshal(after.Dispatches)
		if err != nil {
			t.Fatalf("marshal after dispatches: %v", err)
		}
		afterWorkerRuns, err := json.Marshal(after.WorkerRuns)
		if err != nil {
			t.Fatalf("marshal after worker runs: %v", err)
		}
		if string(beforeDispatches) != string(afterDispatches) {
			t.Fatalf("Dispatches changed by the ceremony:\nbefore: %s\nafter:  %s", beforeDispatches, afterDispatches)
		}
		if string(beforeWorkerRuns) != string(afterWorkerRuns) {
			t.Fatalf("WorkerRuns changed by the ceremony:\nbefore: %s\nafter:  %s", beforeWorkerRuns, afterWorkerRuns)
		}
		if after.OutOfBandVerification == nil {
			t.Fatal("closed attempt carries no out-of-band provenance -- cannot distinguish it from a genuine worker-verified close")
		}
		if after.Status != buildAttemptBuilt {
			t.Fatalf("attempt status = %s, want %s", after.Status, buildAttemptBuilt)
		}
	})

	t.Run("no attempt exists at all", func(t *testing.T) {
		root := setupOutOfBandTest(t, boundPassingOutOfBandPhase())
		_ = root

		if _, _, ok := loadLatestBuildAttempt(1); ok {
			t.Fatal("fixture broken: an attempt already exists")
		}

		_, result, err := executeVerifyOutOfBand(context.Background(), 1, true, false)
		if err != nil {
			t.Fatalf("ceremony refused a genuinely satisfied phase with no prior attempt: %v", err)
		}
		if result == nil || result["advanced"] != true {
			t.Fatalf("ceremony did not advance the phase: %+v", result)
		}

		if _, _, ok := loadLatestBuildAttempt(1); ok {
			t.Fatal("ceremony fabricated a build attempt where none existed")
		}
	})
}
