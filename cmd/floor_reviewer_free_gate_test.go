package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// gateFreeReviewerFixture builds a temp repo + phase + manifest for a
// criterion explicitly bound to the "watcher" check (D-06's compatibility
// case: a plan authored before Phase 193 with required_checks: watcher).
// allGreen controls whether the "tests" shell command passes.
func gateFreeReviewerFixture(t *testing.T, allGreen bool) (root string, phase colony.Phase, manifest codexContinueManifest) {
	t.Helper()
	s, root := newTestStore(t)
	store = s
	testsLine := "- tests: true"
	if !allGreen {
		testsLine = "- tests: false"
	}
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", testsLine)
	criterion := "Reviewer signed off on the change"
	phase = colony.Phase{
		ID:              1,
		Name:            "Reviewer-bound criterion, no reviewer sent",
		SuccessCriteria: []string{criterion},
		EvidenceRequirements: []colony.CriterionEvidenceRequirement{
			{Criterion: criterion, Checks: []string{"watcher"}},
		},
	}
	manifest = codexContinueManifest{
		Present: true,
		Data: codexBuildManifest{
			Phase:                   1,
			CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
			EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
		},
	}
	// runCodexContinueGates' charter_compliance_executed gate (pre-existing,
	// unrelated to this plan) and the owner_confirmation_pending gate's
	// isLastPhaseOfActivePlan both read the active colony state; give them
	// one to read so this fixture exercises the real gate list, not a
	// storage-not-found short-circuit.
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		State: colony.StateEXECUTING,
		Plan:  colony.Plan{Phases: []colony.Phase{phase}},
	}); err != nil {
		t.Fatalf("save colony state: %v", err)
	}
	return root, phase, manifest
}

// TestGateAcceptsDeterministicEvidenceWithoutReviewer proves FLOOR-03 at the
// gate-list level (not just the floor's own ChecksPassed field, which
// 193-01/193-02 already lock): a phase whose criterion is bound to the
// "watcher" check, with no reviewer dispatched, passes the FULL gate list
// (runCodexContinueGates) and the phase advances on BOTH continue lanes when
// every free check is green -- and the same phase still blocks on both
// lanes when a free check fails.
func TestGateAcceptsDeterministicEvidenceWithoutReviewer(t *testing.T) {
	t.Run("all checks green: gate list passes, phase advances, no reviewer dispatched, both lanes", func(t *testing.T) {
		saveGlobals(t)
		root, phase, manifest := gateFreeReviewerFixture(t, true)
		now := time.Now().UTC()

		verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
		if verification.Watcher.Present && verification.Watcher.Status != "skipped" {
			t.Fatalf("expected no reviewer dispatched (in-process lane), got watcher=%+v", verification.Watcher)
		}
		assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{SkipWatchers: true}, now)
		gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
		if !gates.Passed {
			t.Fatalf("in-process lane: expected gate list to pass with no reviewer dispatched and all checks green, got %+v", gates)
		}
		if !assessment.Passed {
			t.Fatalf("in-process lane: expected the phase to advance, got %+v", assessment)
		}

		snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, now, 5*time.Second, true)
		snapAssessment := assessCodexContinue(phase, manifest, snapshot, codexContinueOptions{SkipWatchers: true}, now)
		snapGates := runCodexContinueGates(phase, manifest, snapshot, snapAssessment, now, nil)
		if !snapGates.Passed {
			t.Fatalf("wrapper lane: expected gate list to pass with no reviewer dispatched and all checks green, got %+v", snapGates)
		}
		if !snapAssessment.Passed {
			t.Fatalf("wrapper lane: expected the phase to advance, got %+v", snapAssessment)
		}
	})

	t.Run("a failing check still blocks, both lanes", func(t *testing.T) {
		saveGlobals(t)
		root, phase, manifest := gateFreeReviewerFixture(t, false)
		now := time.Now().UTC()

		verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
		assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{SkipWatchers: true}, now)
		gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
		if gates.Passed {
			t.Fatalf("in-process lane: expected gate list to block on a failing check, got %+v", gates)
		}

		snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, now, 5*time.Second, true)
		snapAssessment := assessCodexContinue(phase, manifest, snapshot, codexContinueOptions{SkipWatchers: true}, now)
		snapGates := runCodexContinueGates(phase, manifest, snapshot, snapAssessment, now, nil)
		if snapGates.Passed {
			t.Fatalf("wrapper lane: expected gate list to block on a failing check, got %+v", snapGates)
		}
	})
}

// setupReconcileParityFixture builds a real build-flow-test colony (fresh
// temp store) for one lane of TestFinalizeCountsReconcileTaskAsEvidence: a
// failed builder dispatch, empty builder claims, and no bound success
// criteria (so the deterministic floor is the shell checks alone).
func setupReconcileParityFixture(t *testing.T) (root, taskID string) {
	t.Helper()
	dataDir := setupBuildFlowTest(t)
	root = filepath.Dir(filepath.Dir(dataDir))
	withTestWorkspace(t, root)
	withWorkingDir(t, root)

	goal := "Finalize counts reconcile-task as evidence"
	now := time.Now().UTC()
	taskID = "1.1"
	phase := colony.Phase{
		ID:     1,
		Name:   "Finalize reconcile parity",
		Status: colony.PhaseInProgress,
		Tasks:  []colony.Task{{ID: &taskID, Goal: "Land the fix", Status: colony.TaskInProgress}},
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateBUILT,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan:           colony.Plan{Phases: []colony.Phase{phase}},
	})
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Land the fix", Status: "completed", TaskID: taskID},
	}
	seedContinueBuildPacket(t, dataDir, 1, "Finalize reconcile parity", goal, dispatches)
	if err := store.SaveJSON("last-build-claims.json", codexBuildClaims{BuildPhase: 1, Timestamp: now.Format(time.RFC3339)}); err != nil {
		t.Fatalf("overwrite claims: %v", err)
	}
	return root, taskID
}

// TestFinalizeCountsReconcileTaskAsEvidence proves the wrapper (finalize)
// lane accepts --reconcile-task supplied AT FINALIZE TIME (via
// mergeReconcileTaskIDs, simulating the flag continueFinalizeCmd now
// registers) and reaches the same verdict as the direct `aether continue`
// lane over identical inputs -- closing the 2026-08-01 folded todo.
func TestFinalizeCountsReconcileTaskAsEvidence(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	// Direct lane: --reconcile-task supplied to `aether continue` itself.
	directRoot, directTaskID := setupReconcileParityFixture(t)
	directResult, _, _, _, _, _, err := runCodexContinue(directRoot, codexContinueOptions{
		ReconcileTaskIDs: []string{directTaskID}, LightFlag: true, SkipWatchers: true,
	})
	if err != nil {
		t.Fatalf("runCodexContinue: %v", err)
	}
	directBlocked, _ := directResult["blocked"].(bool)
	directBlockers := stringSliceValue(directResult["blocking_issues"])

	// Wrapper lane: identical inputs, but reconciliation is supplied at
	// finalize time only -- the plan-only manifest below carries NO
	// reconcile task IDs, exactly the gap the folded todo names.
	saveGlobals(t)
	resetRootCmd(t)
	wrapperRoot, wrapperTaskID := setupReconcileParityFixture(t)
	planResult, _, _, _, err := runCodexContinuePlanOnly(wrapperRoot, codexContinueOptions{LightFlag: true, SkipWatchers: true})
	if err != nil {
		t.Fatalf("runCodexContinuePlanOnly: %v", err)
	}
	plan, ok := planResult["continue_manifest"].(codexContinuePlanManifest)
	if !ok {
		t.Fatalf("expected continue_manifest in result, got %#v", planResult["continue_manifest"])
	}
	if len(plan.ReconcileTaskIDs) != 0 {
		t.Fatalf("test premise broken: plan-only manifest already carries reconcile task IDs: %v", plan.ReconcileTaskIDs)
	}
	// This is exactly what continueFinalizeCmd's RunE now does with the
	// --reconcile-task flag it registers (Task 1, action item 1/2).
	plan.ReconcileTaskIDs = mergeReconcileTaskIDs(plan.ReconcileTaskIDs, []string{wrapperTaskID})

	completion := codexExternalContinueCompletion{ContinueManifest: &plan, Dispatches: []codexContinueExternalDispatch{}}
	wrapperResult, _, _, _, _, _, err := runCodexContinueFinalize(wrapperRoot, completion, false, 0, false)
	if err != nil {
		t.Fatalf("runCodexContinueFinalize: %v", err)
	}
	wrapperBlocked, _ := wrapperResult["blocked"].(bool)
	wrapperBlockers := stringSliceValue(wrapperResult["blocking_issues"])

	if directBlocked {
		t.Fatalf("expected the direct lane to advance, got blocked: %v", directResult)
	}
	if wrapperBlocked {
		t.Fatalf("expected the wrapper lane to advance when --reconcile-task is supplied at finalize time, got blocked: %v", wrapperResult)
	}
	if strings.Join(sortedCopy(directBlockers), "|") != strings.Join(sortedCopy(wrapperBlockers), "|") {
		t.Fatalf("lanes disagree on the blocking-issue set: direct=%v wrapper=%v", directBlockers, wrapperBlockers)
	}
}

// TestReconcileIsNotABypass proves reconciliation supplies evidence, never a
// pass: a reconciled task whose deterministic floor genuinely FAILS still
// blocks on both continue lanes.
func TestReconcileIsNotABypass(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	taskID := "1.1"
	phase := colony.Phase{
		ID:    1,
		Name:  "Reconcile does not bypass a failing floor",
		Tasks: []colony.Task{{ID: &taskID, Goal: "Land the fix"}},
	}
	manifest := codexContinueManifest{}
	now := time.Now().UTC()

	verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{ReconcileTaskIDs: []string{taskID}}, now)
	if assessment.Passed {
		t.Fatalf("in-process lane: expected reconcile NOT to bypass a failing deterministic floor, got %+v", assessment)
	}
	if len(assessment.BlockingIssues) == 0 {
		t.Fatalf("in-process lane: expected non-empty blocking issues, got %+v", assessment)
	}

	snapshot := runCodexContinueVerificationSnapshot(root, phase, manifest, now, 5*time.Second, true)
	snapAssessment := assessCodexContinue(phase, manifest, snapshot, codexContinueOptions{ReconcileTaskIDs: []string{taskID}}, now)
	if snapAssessment.Passed {
		t.Fatalf("wrapper lane: expected reconcile NOT to bypass a failing deterministic floor, got %+v", snapAssessment)
	}
	if len(snapAssessment.BlockingIssues) == 0 {
		t.Fatalf("wrapper lane: expected non-empty blocking issues, got %+v", snapAssessment)
	}
}

// writeWorkerHandoffRecords seeds .aether/data/handoffs/worker-handoffs.json
// directly -- the same file persistDispatchWorkerHandoff writes -- so
// reRunBuilderReportedEvidence has real, persisted handoffs to read.
func writeWorkerHandoffRecords(t *testing.T, records ...workerHandoffRecord) {
	t.Helper()
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: records}); err != nil {
		t.Fatalf("write worker handoffs: %v", err)
	}
}

// TestBuilderReportedCommandIsReRunByTheProgram proves D-04: the program
// itself re-executes a command a builder's handoff reported having run --
// never trusting the handoff's own word -- and the re-run can fail.
func TestBuilderReportedCommandIsReRunByTheProgram(t *testing.T) {
	t.Run("a passing re-run supplies evidence citing the program's own re-run", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		phase := colony.Phase{ID: 1, Name: "Builder evidence re-run"}
		writeWorkerHandoffRecords(t, workerHandoffRecord{
			ID: "build:1.1:Forge-1:1", Workflow: "build", Phase: 1, TaskID: "1.1", WorkerName: "Forge-1",
			CommandsRun: []string{"true"},
		})

		evidence := reRunBuilderReportedEvidence(context.Background(), root, phase, 5*time.Second)
		ok, text := evidence.satisfied()
		if !ok {
			t.Fatalf("expected builder evidence to be satisfied, got %+v", evidence)
		}
		if !strings.Contains(text, "re-ran") {
			t.Fatalf("expected evidence text to cite the program's own re-run, got %q", text)
		}
		if len(evidence.Commands) != 1 || !evidence.Commands[0].Passed || evidence.Commands[0].Unresolvable {
			t.Fatalf("expected the command to be recorded as genuinely passed, got %+v", evidence.Commands)
		}
	})

	t.Run("a failing re-run yields a blocking issue naming the failure", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		phase := colony.Phase{ID: 1, Name: "Builder evidence re-run failure"}
		writeWorkerHandoffRecords(t, workerHandoffRecord{
			ID: "build:1.1:Forge-1:1", Workflow: "build", Phase: 1, TaskID: "1.1", WorkerName: "Forge-1",
			CommandsRun: []string{"false"},
		})

		evidence := reRunBuilderReportedEvidence(context.Background(), root, phase, 5*time.Second)
		ok, _ := evidence.satisfied()
		if ok {
			t.Fatalf("expected a failing re-run NOT to satisfy the check, got %+v", evidence)
		}
		detail := evidence.blockingDetail()
		if !strings.Contains(detail, "failed") {
			t.Fatalf("expected a blocking detail naming the failure, got %q", detail)
		}
		if len(evidence.Commands) != 1 || evidence.Commands[0].Passed || evidence.Commands[0].Unresolvable {
			t.Fatalf("expected the command to be recorded as a genuine failure (not unresolvable), got %+v", evidence.Commands)
		}
	})
}

// TestBuilderReportedFilesMustExistOnDisk proves the second half of D-04: a
// handoff naming a changed file that is not on disk right now yields a
// blocking issue naming that file, even when the command itself passed.
func TestBuilderReportedFilesMustExistOnDisk(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	if err := os.WriteFile(filepath.Join(root, "exists.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	phase := colony.Phase{ID: 1, Name: "Builder evidence file check"}
	writeWorkerHandoffRecords(t, workerHandoffRecord{
		ID: "build:1.1:Forge-1:1", Workflow: "build", Phase: 1, TaskID: "1.1", WorkerName: "Forge-1",
		CommandsRun:  []string{"true"},
		ChangedFiles: []string{"exists.txt", "missing.txt"},
	})

	evidence := reRunBuilderReportedEvidence(context.Background(), root, phase, 5*time.Second)
	ok, _ := evidence.satisfied()
	if ok {
		t.Fatalf("expected builder evidence NOT satisfied when a reported file is missing, got %+v", evidence)
	}
	detail := evidence.blockingDetail()
	if !strings.Contains(detail, "missing.txt") {
		t.Fatalf("expected the blocking detail to name the missing file, got %q", detail)
	}
	foundExists, foundMissing := false, false
	for _, f := range evidence.Files {
		if f.Path == "exists.txt" && f.Exists {
			foundExists = true
		}
		if f.Path == "missing.txt" && !f.Exists {
			foundMissing = true
		}
	}
	if !foundExists || !foundMissing {
		t.Fatalf("expected per-file exists/missing verdicts for both reported files, got %+v", evidence.Files)
	}
}

// TestEvidenceReRunNeverFabricatesAWorkerReceipt asserts, on the stored
// records themselves (not a flag), that reRunBuilderReportedEvidence creates
// no build dispatch, no claims record, and no reviewer verdict -- it only
// reads the phase's already-persisted worker handoffs and re-executes/
// re-checks what they name.
func TestEvidenceReRunNeverFabricatesAWorkerReceipt(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Never fabricates a worker receipt"}
	writeWorkerHandoffRecords(t, workerHandoffRecord{
		ID: "build:1.1:Forge-1:1", Workflow: "build", Phase: 1, TaskID: "1.1", WorkerName: "Forge-1",
		CommandsRun: []string{"true"},
	})

	_ = reRunBuilderReportedEvidence(context.Background(), root, phase, 5*time.Second)

	manifestRel := filepath.ToSlash(filepath.Join("build", "phase-1", "manifest.json"))
	if _, err := store.ReadFile(manifestRel); err == nil {
		t.Fatalf("expected no build manifest to exist after a builder-evidence re-run")
	}
	if _, err := store.ReadFile("last-build-claims.json"); err == nil {
		t.Fatalf("expected no claims file to exist after a builder-evidence re-run")
	}
	records, err := loadWorkerHandoffRecords()
	if err != nil {
		t.Fatalf("load worker handoffs: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected exactly the one seeded handoff record (nothing new fabricated), got %d: %+v", len(records), records)
	}
}

// TestUnprovableCriterionAdvancesButMarksOwnerConfirmation proves D-05: a
// criterion no deterministic source can satisfy and no reviewer was
// dispatched for is recorded needs_owner_confirmation, the phase still
// advances, and no reviewer worker is dispatched because of it -- while a
// check that genuinely ran and FAILED (a dispatched-and-failed watcher)
// still blocks rather than becoming an owner-confirmation item.
func TestUnprovableCriterionAdvancesButMarksOwnerConfirmation(t *testing.T) {
	t.Run("no reviewer dispatched, no deterministic proof: needs_owner_confirmation, phase advances", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: true")
		criterion := "Reviewer signed off on the change"
		phase := colony.Phase{
			ID:              1,
			Name:            "Genuinely unprovable criterion",
			SuccessCriteria: []string{criterion},
			EvidenceRequirements: []colony.CriterionEvidenceRequirement{
				{Criterion: criterion, Checks: []string{"watcher"}},
			},
		}
		manifest := codexContinueManifest{
			Present: true,
			Data: codexBuildManifest{
				Phase:                   1,
				CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
				EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
				// A real builder dispatch makes manifestRequiresBuilderClaims
				// true, and deliberately NO last-build-claims.json exists --
				// deterministicFloorSatisfies (cmd/criterion_evidence.go)
				// requires claims.Passed, so without this the fixture's
				// shell checks passing alone would ALSO satisfy the
				// "watcher" check via the D-06 compatibility fallback, which
				// is not what this test is proving. This is the genuinely
				// unprovable case: shell checks green, but claims required
				// and absent, so nothing can substitute for a reviewer.
				Dispatches: []codexBuildDispatch{
					{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Land the fix", Status: "completed"},
				},
			},
		}
		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
			State: colony.StateEXECUTING,
			Plan:  colony.Plan{Phases: []colony.Phase{phase}},
		}); err != nil {
			t.Fatalf("save colony state: %v", err)
		}

		verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
		if !verification.ChecksPassed {
			t.Fatalf("expected the phase to still advance, got ChecksPassed=false: %+v", verification)
		}
		if len(verification.Criteria) != 1 {
			t.Fatalf("expected exactly one criterion result, got %+v", verification.Criteria)
		}
		criterionResult := verification.Criteria[0]
		if criterionResult.State != criterionStateNeedsOwnerConfirmation {
			t.Fatalf("expected criterion State=%q, got %+v", criterionStateNeedsOwnerConfirmation, criterionResult)
		}
		if !criterionResult.Passed {
			t.Fatalf("expected criterion Passed:true (the phase still advances), got %+v", criterionResult)
		}
		if verification.Watcher.Present && verification.Watcher.Status != "skipped" {
			t.Fatalf("expected no reviewer dispatched because of the owner-confirmation criterion, got watcher=%+v", verification.Watcher)
		}
	})

	t.Run("a dispatched-and-failed watcher still blocks, not owner-confirmation", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: true")
		criterion := "Reviewer signed off on the change"
		phase := colony.Phase{
			ID:              1,
			SuccessCriteria: []string{criterion},
			EvidenceRequirements: []colony.CriterionEvidenceRequirement{
				{Criterion: criterion, Checks: []string{"watcher"}},
			},
		}
		manifest := codexContinueManifest{
			Present: true,
			Data: codexBuildManifest{
				Phase:                   1,
				CriterionEvidencePolicy: criterionEvidencePolicyBoundV1,
				EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(phase),
			},
		}
		failedWatcher := codexWatcherVerification{Present: true, Passed: false, Status: "failed", Worker: "Keen-1", Summary: "watcher found a real problem"}

		floor := runDeterministicFloor(context.Background(), root, phase, manifest, failedWatcher, 5*time.Second)
		if floor.ChecksPassed {
			t.Fatalf("expected a dispatched-and-failed watcher to block the floor, got ChecksPassed=true: %+v", floor)
		}
		if len(floor.Criteria.Criteria) != 1 {
			t.Fatalf("expected exactly one criterion result, got %+v", floor.Criteria.Criteria)
		}
		result := floor.Criteria.Criteria[0]
		if result.State == criterionStateNeedsOwnerConfirmation {
			t.Fatalf("a check that ran and failed must still block, not read needs_owner_confirmation: %+v", result)
		}
		if result.Passed {
			t.Fatalf("expected the criterion to fail, got %+v", result)
		}
	})
}

// TestSealBlocksOnUnconfirmedCriterion proves D-05's seal half: with an
// outstanding needs_owner_confirmation criterion, `aether seal` refuses and
// names the criterion in plain English; after the owner answers it through
// the existing decision-answer path, seal proceeds.
func TestSealBlocksOnUnconfirmedCriterion(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	criterion := "A human judged this looks right"
	phaseID := 1
	report := codexContinueVerificationReport{
		Phase:       phaseID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Criteria: []codexCriterionVerification{
			{Criterion: criterion, Passed: true, State: criterionStateNeedsOwnerConfirmation},
		},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(phaseID, "verification.json"), report); err != nil {
		t.Fatalf("save verification report: %v", err)
	}
	goal := "Seal blocks on unconfirmed criterion"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Plan:  colony.Plan{Phases: []colony.Phase{{ID: phaseID, Name: "Final phase", Status: colony.PhaseCompleted}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	blockers, _ := checkSealBlockers(store, state)
	if len(blockers) == 0 {
		t.Fatalf("expected an owner-confirmation seal blocker")
	}
	found := false
	for _, b := range blockers {
		if strings.Contains(b.Description, criterion) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the blocker to name the criterion in plain English, got %+v", blockers)
	}

	if _, _, err := validateSealReady(false); err == nil {
		t.Fatalf("expected aether seal to refuse with an outstanding owner confirmation")
	} else if !strings.Contains(err.Error(), criterion) {
		t.Fatalf("expected the refusal to name the criterion, got: %v", err)
	}

	// The owner answers through the existing decision-answer path.
	if _, err := recordDecisionAnswer(ownerConfirmationQuestionText(phaseID, "", criterion), "confirmed", phaseID, "owner"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	blockers, _ = checkSealBlockers(store, state)
	if len(blockers) != 0 {
		t.Fatalf("expected no seal blockers after the owner answered, got %+v", blockers)
	}
	if _, _, err := validateSealReady(false); err != nil {
		t.Fatalf("expected aether seal to proceed after the owner answered, got: %v", err)
	}
}

// TestSealForceStillOverridesOwnerConfirmation proves the existing
// owner-override contract is unchanged: --force still overrides an
// outstanding owner-confirmation blocker exactly as it overrides any other
// seal blocker.
func TestSealForceStillOverridesOwnerConfirmation(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	criterion := "A human judged this looks right"
	phaseID := 1
	report := codexContinueVerificationReport{
		Phase: phaseID,
		Criteria: []codexCriterionVerification{
			{Criterion: criterion, Passed: true, State: criterionStateNeedsOwnerConfirmation},
		},
	}
	if err := store.SaveJSON(continuePlanArtifactsPath(phaseID, "verification.json"), report); err != nil {
		t.Fatalf("save verification report: %v", err)
	}
	goal := "Seal force overrides owner confirmation"
	state := colony.ColonyState{
		Goal:  &goal,
		State: colony.StateEXECUTING,
		Plan:  colony.Plan{Phases: []colony.Phase{{ID: phaseID, Name: "Final phase", Status: colony.PhaseCompleted}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	if _, _, err := validateSealReady(false); err == nil {
		t.Fatalf("expected validateSealReady(false) to refuse with an outstanding owner confirmation")
	}
	if _, _, err := validateSealReady(true); err != nil {
		t.Fatalf("expected --force to override the owner-confirmation blocker, got %v", err)
	}
}
