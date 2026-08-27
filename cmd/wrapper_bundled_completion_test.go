package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// setupWrapperBundledManifestTest builds a real, plan-only manifest bound to
// a durable build attempt, with one INDEPENDENT task (no DependsOn) per
// taskID. Independent tasks share a wave and are never merged by
// coalesceSequentialDispatches -- see TestMergedDispatchCreditsEveryCoveredTask's
// own comment on this -- so each task becomes its own separate, uncoalesced
// dispatch with no codexBuildDispatch.CoveredTaskIDs pre-set. That is the
// exact shape FIELD-02 is about: the WRAPPER, not the runtime, bundling
// several manifest-listed dispatches into one worker call, which is a
// different case from the runtime-coalesced chain cmd/merged_dispatch_task_credit_test.go
// already locks (191.1-CONTEXT.md D-04, 191.1-PATTERNS.md Pattern 6).
//
// "standard" colony depth also appends its own probe/watcher
// verification-stage dispatches to every plan-only manifest -- unrelated
// noise for this file's covered_task_ids scenarios. The returned
// taskDispatches slice holds only the dispatches matching the fixture's own
// taskIDs, in that order, so callers never mistake a verification dispatch
// for one of the fixture's tasks.
func setupWrapperBundledManifestTest(t *testing.T, taskIDs []string) (string, codexBuildManifest, []codexBuildDispatch) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}

	goal := "Bundle several independent tasks into one honest completion"
	tasks := make([]colony.Task, 0, len(taskIDs))
	for i := range taskIDs {
		id := taskIDs[i]
		tasks = append(tasks, colony.Task{ID: &id, Goal: "Independent step " + id, Status: colony.TaskPending})
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Wrapper-bundled work", Description: "Several independent tasks, no runtime coalescing",
			Status: colony.PhaseReady, Tasks: tasks,
		}}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	byTaskID := make(map[string]codexBuildDispatch, len(taskIDs))
	for _, d := range manifest.Dispatches {
		if taskID := strings.TrimSpace(d.TaskID); taskID != "" {
			byTaskID[taskID] = d
		}
	}
	taskDispatches := make([]codexBuildDispatch, 0, len(taskIDs))
	for _, id := range taskIDs {
		d, ok := byTaskID[id]
		if !ok {
			t.Fatalf("fixture broken: no dispatch in the manifest carries task ID %s; dispatches: %+v", id, manifest.Dispatches)
		}
		// Assert the fixture's own precondition instead of assuming it: if a
		// task dispatch already carries CoveredTaskIDs, the fixture
		// accidentally exercises the DIFFERENT, already-covered
		// runtime-coalesced case and this file would prove nothing about the
		// wrapper-bundled shape it exists for.
		if len(d.CoveredTaskIDs) > 0 {
			t.Fatalf("fixture broken: dispatch %s already carries CoveredTaskIDs %v -- independent tasks must not be runtime-coalesced", d.Name, d.CoveredTaskIDs)
		}
		taskDispatches = append(taskDispatches, d)
	}

	return root, manifest, taskDispatches
}

// resultsExcluding returns a copy of results with every entry whose
// effective name is in exclude removed.
func resultsExcluding(results []codexExternalBuildWorkerResult, exclude ...string) []codexExternalBuildWorkerResult {
	skip := make(map[string]bool, len(exclude))
	for _, name := range exclude {
		skip[name] = true
	}
	out := make([]codexExternalBuildWorkerResult, 0, len(results))
	for _, r := range results {
		if skip[r.effectiveName()] {
			continue
		}
		out = append(out, r)
	}
	return out
}

// withCoveredTaskIDs returns a copy of results with the entry named name
// given coveredTaskIDs.
func withCoveredTaskIDs(results []codexExternalBuildWorkerResult, name string, coveredTaskIDs []string) []codexExternalBuildWorkerResult {
	out := make([]codexExternalBuildWorkerResult, len(results))
	copy(out, results)
	for i := range out {
		if out[i].effectiveName() == name {
			out[i].CoveredTaskIDs = coveredTaskIDs
		}
	}
	return out
}

// TestWrapperBundledWorkerCreditsAllCoveredTasks is the fix-direction-(a)+(b)
// lock (191.1-CONTEXT.md D-04): a manifest lists 3 separate, uncoalesced
// task dispatches. One worker's completion result matches only the first
// dispatch's identity but names the other two in covered_task_ids, with real
// file-change evidence. No result is submitted for the other two dispatches
// at all. Both the direct merge function and the full build-finalize
// entrypoint must credit all 3 -- not just the one the wrapper happened to
// name.
func TestWrapperBundledWorkerCreditsAllCoveredTasks(t *testing.T) {
	root, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"1.1", "1.2", "1.3"})
	primary := taskDispatches[0]
	coveredIDs := []string{taskDispatches[1].TaskID, taskDispatches[2].TaskID}

	baseline := externalResultsForManifest(manifest)
	results := resultsExcluding(baseline, taskDispatches[1].Name, taskDispatches[2].Name)
	results = withCoveredTaskIDs(results, primary.Name, coveredIDs)

	// (a) mergeExternalBuildResults directly -- the function this plan edits.
	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	for _, v := range violations {
		if v.Rule == violationRuleResultMissing {
			t.Errorf("unexpected %s violation for %s -- covered_task_ids should have credited it honestly instead of leaving it unreported", violationRuleResultMissing, v.Worker)
		}
	}
	byName := make(map[string]codexBuildDispatch, len(dispatches))
	for _, d := range dispatches {
		byName[d.Name] = d
	}
	for _, want := range taskDispatches {
		got, ok := byName[want.Name]
		if !ok {
			t.Fatalf("dispatch %s missing from mergeExternalBuildResults output", want.Name)
		}
		if got.Status != "completed" {
			t.Errorf("dispatch %s status = %q, want completed -- one worker covered all 3 via covered_task_ids", got.Name, got.Status)
		}
		if len(got.Outputs) == 0 {
			t.Errorf("dispatch %s has no Outputs after being credited -- validateBuildProvenance's file-change requirement would not be honestly satisfied for it", got.Name)
		}
	}

	// (b) the full runCodexBuildFinalize path -- the real entrypoint the
	// wrapper protocol actually calls -- reusing the same manifest and
	// results.
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}
	_, finalState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("runCodexBuildFinalize: %v", err)
	}
	for _, task := range finalState.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("task %s is %q after one worker honestly covered all 3 tasks via covered_task_ids through the real build-finalize path, want %q -- the worker did the work and the receipt lost it", *task.ID, task.Status, colony.TaskCompleted)
		}
	}
}

// TestWrapperBundledWorkerRejectsUnknownCoveredTaskID locks the trust
// boundary Pattern 6 requires: covered_task_ids is the WORKER's own claim, so
// every entry must be validated against the manifest before any credit is
// granted. An unresolvable ID, or two different results claiming the same
// ID, must produce a distinct, named contractViolation -- never silent
// acceptance and never a silently overwritten credit.
func TestWrapperBundledWorkerRejectsUnknownCoveredTaskID(t *testing.T) {
	_, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"1.1", "1.2", "1.3"})
	primary := taskDispatches[0]
	baseline := externalResultsForManifest(manifest)

	t.Run("unknown task ID", func(t *testing.T) {
		results := withCoveredTaskIDs(baseline, primary.Name, []string{"9.9-does-not-exist-anywhere"})
		_, violations, err := mergeExternalBuildResults(manifest, results)
		if err != nil {
			t.Fatalf("mergeExternalBuildResults: %v", err)
		}
		var found *contractViolation
		for i := range violations {
			if violations[i].Rule == violationRuleCoveredTaskUnknown {
				found = &violations[i]
			}
		}
		if found == nil {
			t.Fatalf("no %s violation for an unresolvable covered_task_ids entry; violations: %+v", violationRuleCoveredTaskUnknown, violations)
		}
		if found.Value != "9.9-does-not-exist-anywhere" {
			t.Errorf("violation Value = %q, want the unresolved task ID", found.Value)
		}
		if found.Worker != primary.Name {
			t.Errorf("violation Worker = %q, want the claiming worker %q", found.Worker, primary.Name)
		}
	})

	t.Run("duplicate claim", func(t *testing.T) {
		second := taskDispatches[1]
		contested := taskDispatches[2]

		// The contested dispatch gets no direct result of its own -- the
		// only way it can end up credited is through one of the two
		// covered_task_ids claims below, so the test proves the SECOND
		// claim is rejected rather than merely proving a redundant direct
		// result wins over both.
		results := resultsExcluding(baseline, contested.Name)
		results = withCoveredTaskIDs(results, primary.Name, []string{contested.TaskID})
		results = withCoveredTaskIDs(results, second.Name, []string{contested.TaskID})

		dispatches, violations, err := mergeExternalBuildResults(manifest, results)
		if err != nil {
			t.Fatalf("mergeExternalBuildResults: %v", err)
		}
		duplicateCount := 0
		for _, v := range violations {
			if v.Rule == violationRuleCoveredTaskDuplicate {
				duplicateCount++
			}
			if v.Rule == violationRuleResultMissing {
				t.Errorf("unexpected %s violation for %s in a duplicate-claim case", violationRuleResultMissing, v.Worker)
			}
		}
		if duplicateCount != 1 {
			t.Fatalf("%s violations = %d, want exactly 1 (the second, losing claim); violations: %+v", violationRuleCoveredTaskDuplicate, duplicateCount, violations)
		}
		// The contested dispatch is still credited exactly once (to
		// whichever claim resolved first) -- a rejected second claim must
		// not corrupt or blank out the first, honest credit.
		creditedOnce := 0
		for _, d := range dispatches {
			if d.TaskID == contested.TaskID && d.Status == "completed" {
				creditedOnce++
			}
		}
		if creditedOnce != 1 {
			t.Fatalf("contested dispatch ended completed %d times, want exactly 1", creditedOnce)
		}
	})
}

// TestCoveredTaskCreditNotConfusedByLegitimateRetryResubmission is the WR-02
// regression lock (191.1-REVIEW.md): a worker that legitimately resubmits
// under its own identical name -- first timeout, then completed, both
// carrying the same covered_task_ids claim -- must not trip
// violationRuleCoveredTaskDuplicate against itself. Before the fix, the
// coveredBy pass iterated the raw, non-deduplicated results slice instead of
// the same resultByName map (post preferCompletedResultOverTimeout
// resolution) the main dispatch loop already computes two lines above it,
// so this exact resubmission shape was treated as two different claimants
// fighting over the same task.
func TestCoveredTaskCreditNotConfusedByLegitimateRetryResubmission(t *testing.T) {
	_, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"1.1", "1.2"})
	primary := taskDispatches[0]
	covered := taskDispatches[1]

	baseline := externalResultsForManifest(manifest)
	results := resultsExcluding(baseline, covered.Name)
	results = withCoveredTaskIDs(results, primary.Name, []string{covered.TaskID})

	var primaryResult codexExternalBuildWorkerResult
	for _, r := range results {
		if r.effectiveName() == primary.Name {
			primaryResult = r
		}
	}
	// Simulate a wrapper-side resend: the SAME worker name submits a second
	// result, this time timeout, carrying the identical covered_task_ids
	// claim -- a real shape preferCompletedResultOverTimeout already treats
	// as one legitimate resubmission in the main dispatch loop.
	retry := primaryResult
	retry.Status = "timeout"
	results = append(results, retry)

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	for _, v := range violations {
		if v.Rule == violationRuleCoveredTaskDuplicate {
			t.Fatalf("unexpected %s violation for a legitimate same-name resubmission: %+v", violationRuleCoveredTaskDuplicate, v)
		}
	}
	byName := make(map[string]codexBuildDispatch, len(dispatches))
	for _, d := range dispatches {
		byName[d.Name] = d
	}
	got, ok := byName[covered.Name]
	if !ok || got.Status != "completed" {
		t.Fatalf("dispatch %s status = %+v, want completed -- the legitimate resubmission must still credit its covered_task_ids claim", covered.Name, got)
	}
}

// TestCoveredTaskCreditRefusedFromFailedUnevidencedClaimant is the CR-01
// regression lock (191.1-REVIEW.md): a worker whose OWN submitted result is
// failed, with zero files/outputs, must not be able to grant full completed
// credit -- durably written into COLONY_STATE.json task statuses via
// reconcileCompletedBuildTasks -- to a dispatch it never touched, simply by
// naming it in covered_task_ids. Worse (see the sibling test below), the
// claimant did not even need to correspond to a real dispatch at all before
// this fix.
func TestCoveredTaskCreditRefusedFromFailedUnevidencedClaimant(t *testing.T) {
	root, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"1.1", "1.2"})
	primary := taskDispatches[0]
	other := taskDispatches[1]

	baseline := externalResultsForManifest(manifest)
	results := resultsExcluding(baseline, other.Name)
	for i := range results {
		if results[i].effectiveName() == primary.Name {
			results[i].Status = "failed"
			results[i].FilesModified = nil
			results[i].FilesCreated = nil
			results[i].TestsWritten = nil
			results[i].Outputs = nil
		}
	}
	results = withCoveredTaskIDs(results, primary.Name, []string{other.TaskID})

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	for _, d := range dispatches {
		if d.Name == other.Name && d.Status == "completed" {
			t.Fatalf("CR-01: dispatch %s was credited completed via covered_task_ids from claimant %s, whose own status is failed and carries zero evidence", d.Name, primary.Name)
		}
	}
	found := false
	for _, v := range violations {
		if v.Rule == violationRuleCoveredTaskCreditUnevidenced && v.Value == other.TaskID {
			found = true
		}
	}
	if !found {
		t.Fatalf("no %s violation for a covered_task_ids claim from a failed, unevidenced claimant; violations: %+v", violationRuleCoveredTaskCreditUnevidenced, violations)
	}

	// The full build-finalize entrypoint must refuse the whole packet (D-06
	// whole-packet rejection), not just silently withhold credit.
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}
	_, _, _, _, err = runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatal("expected runCodexBuildFinalize to refuse a packet granting covered_task_ids credit from a failed, unevidenced claimant")
	}
	var contractErr *completionContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("expected a *completionContractError, got %T: %v", err, err)
	}
}

// TestCoveredTaskCreditRefusedFromFabricatedClaimant is the CR-01 regression
// lock (191.1-REVIEW.md): a covered_task_ids claim from a worker name
// matching NO dispatch anywhere in the manifest must be refused, even when
// that fabricated entry's own status is completed and it carries
// plausible-looking evidence.
func TestCoveredTaskCreditRefusedFromFabricatedClaimant(t *testing.T) {
	root, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"2.1", "2.2"})
	other := taskDispatches[1]

	baseline := externalResultsForManifest(manifest)
	results := resultsExcluding(baseline, other.Name)
	results = append(results, codexExternalBuildWorkerResult{
		Name:           "totally-fabricated-ghost-worker",
		Status:         "completed",
		Summary:        "totally-fabricated-ghost-worker completed externally",
		FilesModified:  []string{"external-evidence.txt"},
		CoveredTaskIDs: []string{other.TaskID},
		Handoff: codex.WorkerHandoff{
			CommandsRun:            []string{"go test ./..."},
			VerificationStatus:     "pass",
			NextWorkerInstructions: []string{"ghost done"},
		},
	})

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	for _, d := range dispatches {
		if d.Name == other.Name && d.Status == "completed" {
			t.Fatalf("CR-01: dispatch %s was credited completed via covered_task_ids from a fabricated worker name matching no manifest dispatch", d.Name)
		}
	}
	found := false
	for _, v := range violations {
		if v.Rule == violationRuleCoveredTaskCreditUnevidenced && v.Value == other.TaskID {
			found = true
		}
	}
	if !found {
		t.Fatalf("no %s violation for a covered_task_ids claim from a fabricated claimant; violations: %+v", violationRuleCoveredTaskCreditUnevidenced, violations)
	}

	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}
	_, _, _, _, err = runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatal("expected runCodexBuildFinalize to refuse a packet granting covered_task_ids credit from a fabricated claimant")
	}
	var contractErr *completionContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("expected a *completionContractError, got %T: %v", err, err)
	}
}

// TestFinalizeSuspectsBundledWorkOnOneOfNShape is the fix-direction-(b) lock
// (191.1-CONTEXT.md D-04): a genuine 1-completed-of-N-dispatches shape with
// real evidence and no covered_task_ids must still be rejected (the packet
// really is incomplete), but finalize must append in-band guidance naming
// the concrete covered_task_ids repair alongside the still-real
// missing-result violations, rather than leaving only a dead-end refusal.
func TestFinalizeSuspectsBundledWorkOnOneOfNShape(t *testing.T) {
	root, manifest, taskDispatches := setupWrapperBundledManifestTest(t, []string{"1.1", "1.2", "1.3", "1.4"})
	primary := taskDispatches[0]
	missing := taskDispatches[1:]
	missingNames := make([]string, 0, len(missing))
	for _, d := range missing {
		missingNames = append(missingNames, d.Name)
	}

	baseline := externalResultsForManifest(manifest)
	results := resultsExcluding(baseline, missingNames...)

	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}
	_, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatal("expected a completion-packet rejection for a 1-of-4 dispatch shape with no covered_task_ids -- the packet really is incomplete")
	}
	var contractErr *completionContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("expected a *completionContractError, got %T: %v", err, err)
	}

	missingCount := 0
	var guidance *contractViolation
	for i := range contractErr.Violations {
		v := &contractErr.Violations[i]
		switch v.Rule {
		case violationRuleResultMissing:
			missingCount++
		case violationRuleBundledWorkSuspected:
			guidance = v
		}
	}
	if missingCount != len(missing) {
		t.Fatalf("%s violations = %d, want %d (one per unreported task dispatch) -- the guidance must never remove or replace the genuine violations; violations: %+v", violationRuleResultMissing, missingCount, len(missing), contractErr.Violations)
	}
	if guidance == nil {
		t.Fatalf("no %s violation for a real, evidenced 1-of-4 completion; violations: %+v", violationRuleBundledWorkSuspected, contractErr.Violations)
	}
	if guidance.Worker != primary.Name {
		t.Errorf("guidance names worker %q, want the completed worker %q", guidance.Worker, primary.Name)
	}
	for _, d := range missing {
		if !strings.Contains(guidance.Message, d.Name) {
			t.Errorf("guidance message does not name missing dispatch %s: %q", d.Name, guidance.Message)
		}
		if !strings.Contains(guidance.Message, d.TaskID) {
			t.Errorf("guidance message does not name missing task ID %s: %q", d.TaskID, guidance.Message)
		}
	}
	if !strings.Contains(guidance.Message, "covered_task_ids") {
		t.Errorf("guidance message does not state the covered_task_ids repair: %q", guidance.Message)
	}

	// Negative case: the SAME 1-of-4 shape, but the "completed" result
	// carries no real file-change evidence at all -- nothing to honestly
	// point at, so the guidance must not fire, avoiding a false-positive
	// suggestion. Only the plain missing-result violations remain.
	root2, manifest2, taskDispatches2 := setupWrapperBundledManifestTest(t, []string{"2.1", "2.2", "2.3", "2.4"})
	primary2 := taskDispatches2[0]
	missing2 := taskDispatches2[1:]
	missingNames2 := make([]string, 0, len(missing2))
	for _, d := range missing2 {
		missingNames2 = append(missingNames2, d.Name)
	}
	baseline2 := externalResultsForManifest(manifest2)
	results2 := resultsExcluding(baseline2, missingNames2...)
	// Strip the one real evidence source: the primary result's own
	// FilesModified, set by externalResultsForManifest for builder-caste
	// dispatches.
	for i := range results2 {
		if results2[i].effectiveName() == primary2.Name {
			results2[i].FilesModified = nil
		}
	}
	completion2 := codexExternalBuildCompletion{DispatchManifest: &manifest2, Dispatches: results2}
	_, _, _, _, err = runCodexBuildFinalize(root2, 1, completion2, false)
	if err == nil {
		t.Fatal("expected a completion-packet rejection for a 1-of-4 dispatch shape with no evidence at all")
	}
	var contractErr2 *completionContractError
	if !errors.As(err, &contractErr2) {
		t.Fatalf("expected a *completionContractError, got %T: %v", err, err)
	}
	for _, v := range contractErr2.Violations {
		if v.Rule == violationRuleBundledWorkSuspected {
			t.Fatalf("%s guidance fired for a completed result with no file-change evidence at all; there is nothing honest to point at: %+v", violationRuleBundledWorkSuspected, v)
		}
	}
}

// ---------------------------------------------------------------------------
// 195-06 Task 1: the external/wrapper lane admits task_receipts through the
// EXACT SAME shared stage-one contract (admitCoherentJobTaskReceipts,
// cmd/coherent_job_receipts.go) the native lane uses -- never a second,
// external-only validator (195-CONTEXT.md D-08/D-09).
// ---------------------------------------------------------------------------

// setupCoherentJobWrapperTest builds a real, plan-only manifest for six
// dependent tasks the runtime coalesces into ONE merged dispatch
// (coalesceSequentialDispatches) -- the D-08/D-09 partial-credit shape --
// and returns root, the phase (with its full Task list, needed by
// admitCoherentJobTaskReceipts), the manifest, the merged chain dispatch,
// and the six task IDs in order.
func setupCoherentJobWrapperTest(t *testing.T, goal string) (string, colony.Phase, codexBuildManifest, codexBuildDispatch, []string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	tasks, ids := sixChainedTasks()
	phase := colony.Phase{
		ID: 1, Name: "Merged chain", Description: "One worker, six dependent steps",
		Status: colony.PhaseReady, Tasks: tasks,
	}
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{phase}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	var chain codexBuildDispatch
	for _, dispatch := range manifest.Dispatches {
		if len(dispatch.CoveredTaskIDs) > 1 {
			chain = dispatch
			break
		}
	}
	if chain.Name == "" || len(chain.CoveredTaskIDs) != len(ids) {
		t.Fatalf("fixture did not produce one merged dispatch covering all six tasks; dispatches: %+v", manifest.Dispatches)
	}
	return root, phase, manifest, chain, ids
}

// codexBuildFinalizeSourcePath returns the absolute path to
// codex_build_finalize.go, resolved via this test file's own location
// (runtime.Caller) rather than a relative "codex_build_finalize.go" path --
// several tests in this package call withWorkingDir, which changes the
// process cwd away from the package directory, so a bare relative read would
// silently break depending on test ordering.
func codexBuildFinalizeSourcePath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "codex_build_finalize.go")
}

// extractGoFunctionBody returns the substring of src starting at
// signaturePrefix and ending at (and including) that function's matching
// closing brace, found via simple, depth-counted brace matching. Sufficient
// for this package's own source, which has no unbalanced braces inside
// string/rune literals within the functions this test inspects.
func extractGoFunctionBody(t *testing.T, src, signaturePrefix string) string {
	t.Helper()
	start := strings.Index(src, signaturePrefix)
	if start < 0 {
		t.Fatalf("could not find function signature %q in source", signaturePrefix)
	}
	depth := 0
	started := false
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
			started = true
		case '}':
			depth--
			if started && depth == 0 {
				return src[start : i+1]
			}
		}
	}
	t.Fatalf("could not find matching closing brace for function %q", signaturePrefix)
	return ""
}

// assertMergeExternalBuildResultsGrantsNoCredit is a source-behavior
// assertion (195-06 Task 1 acceptance criterion): mergeExternalBuildResults
// must never itself call attachBuildArtifactEvidence (a root-evidence read)
// or assign dispatch.CompletedTaskIDs (completion credit) -- both belong to
// finalizeCoherentJobTaskReceiptEvidence alone (cmd/coherent_job_receipts.go).
func assertMergeExternalBuildResultsGrantsNoCredit(t *testing.T) {
	t.Helper()
	src, err := os.ReadFile(codexBuildFinalizeSourcePath(t))
	if err != nil {
		t.Fatalf("read codex_build_finalize.go for source assertion: %v", err)
	}
	body := extractGoFunctionBody(t, string(src), "func mergeExternalBuildResults(")
	if strings.Contains(body, "attachBuildArtifactEvidence") {
		t.Fatal("mergeExternalBuildResults must not call attachBuildArtifactEvidence -- that root-evidence read belongs to finalizeCoherentJobTaskReceiptEvidence alone")
	}
	// CR-03 (195-REVIEW.md): the ONLY writes this function may make to the
	// runtime-owned credit fields are the defensive clears that strip an
	// inbound manifest's self-asserted verdict. Prove both clears are present,
	// then prove nothing else assigns either field.
	for _, clear := range []string{"dispatch.CompletedTaskIDs = nil", "dispatch.TaskClaims = nil"} {
		if !strings.Contains(body, clear) {
			t.Fatalf("mergeExternalBuildResults must defensively clear inbound task credit -- %q is missing, so a wrapper-authored manifest can carry its own verdict past the receipt boundary", clear)
		}
	}
	stripped := strings.ReplaceAll(body, "dispatch.CompletedTaskIDs = nil", "")
	stripped = strings.ReplaceAll(stripped, "dispatch.TaskClaims = nil", "")
	if strings.Contains(stripped, ".CompletedTaskIDs =") || strings.Contains(stripped, ".CompletedTaskIDs=") {
		t.Fatal("mergeExternalBuildResults must not assign CompletedTaskIDs -- that completion credit belongs to finalizeCoherentJobTaskReceiptEvidence alone")
	}
	if strings.Contains(stripped, ".TaskClaims =") || strings.Contains(stripped, ".TaskClaims=") {
		t.Fatal("mergeExternalBuildResults must not assign TaskClaims -- per-task claims belong to finalizeCoherentJobTaskReceiptEvidence alone")
	}
}

// TestExternalTaskReceiptsUseSharedAdmission proves the external/wrapper lane
// reuses admitCoherentJobTaskReceipts -- the EXACT stage-1 admission function
// the native lane uses -- rather than a second, external-only validator. A
// four-of-six failed grouped dispatch's merged intermediate carries four
// admitted candidate claims and their normalized sync paths, but
// mergeExternalBuildResults itself never sets CompletedTaskIDs or reads root.
func TestExternalTaskReceiptsUseSharedAdmission(t *testing.T) {
	root, phase, manifest, chain, ids := setupCoherentJobWrapperTest(t, "Shared admission for external receipts")

	receiptedIDs := ids[:4]
	receipts := make([]codex.TaskReceipt, 0, len(receiptedIDs))
	touchedFiles := make([]string, 0, len(receiptedIDs))
	for _, id := range receiptedIDs {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing four of six steps",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("unexpected contract violations merging a failed dispatch with valid task receipts: %+v", violations)
	}

	var merged codexBuildDispatch
	for _, d := range dispatches {
		if d.Name == chain.Name {
			merged = d
			break
		}
	}
	if len(merged.CompletedTaskIDs) != 0 || len(merged.TaskClaims) != 0 {
		t.Fatalf("mergeExternalBuildResults must never itself grant completion credit -- got CompletedTaskIDs=%v TaskClaims=%+v", merged.CompletedTaskIDs, merged.TaskClaims)
	}
	if len(merged.TaskReceipts) != len(receiptedIDs) {
		t.Fatalf("merged dispatch should carry all %d submitted receipts unchanged, got %d", len(receiptedIDs), len(merged.TaskReceipts))
	}

	admission, admitViolations := admitCoherentJobTaskReceipts(root, phase, merged, merged.Outputs, merged.TaskReceipts)
	if len(admitViolations) > 0 {
		t.Fatalf("unexpected admission violations for %d well-formed receipts: %+v", len(receiptedIDs), admitViolations)
	}
	if len(admission.Candidates) != len(receiptedIDs) {
		t.Fatalf("admission produced %d candidates, want %d", len(admission.Candidates), len(receiptedIDs))
	}
	if len(admission.SyncPaths) == 0 {
		t.Fatal("admission should carry normalized sync paths for the worktree lane (195-08) to consume later")
	}

	assertMergeExternalBuildResultsGrantsNoCredit(t)
}

// TestWrapperGroupedFailureCreditsFourOfSix is the shipped external-lane
// counterpart of TestFailedGroupedDispatchCreditsExactlyReceiptedTasks
// (cmd/merged_dispatch_task_credit_test.go, native lane, 195-04), run
// through the REAL `aether build-finalize` entrypoint: a single external
// worker covering six chained tasks that failed after finishing four, with
// valid per-task receipts for those four, must credit exactly those four,
// leave the other two pending, never report the colony as BUILT, and name
// both the credited and unfinished task IDs in its result diagnostics
// (195-CONTEXT.md D-08/D-09).
func TestWrapperGroupedFailureCreditsFourOfSix(t *testing.T) {
	root, _, manifest, chain, ids := setupCoherentJobWrapperTest(t, "Credit exactly the receipted tasks, external lane")

	receiptedIDs := ids[:4]
	pendingIDs := ids[4:]
	receipts := make([]codex.TaskReceipt, 0, len(receiptedIDs))
	touchedFiles := make([]string, 0, len(receiptedIDs))
	for _, id := range receiptedIDs {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing four of six steps",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	buildResult, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("build-finalize should accept a failed dispatch with genuine partial receipts, got error: %v", err)
	}
	if updatedState.State == colony.StateBUILT {
		t.Fatalf("colony state = %s after only 4 of 6 tasks were credited, want anything but BUILT (honest partial state)", updatedState.State)
	}
	if buildResult["result_collection"] == nil {
		t.Fatalf("expected a result_collection path in the finalize result, got %+v", buildResult)
	}

	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	statusByID := map[string]string{}
	for _, task := range reloaded.Plan.Phases[0].Tasks {
		statusByID[*task.ID] = string(task.Status)
	}
	for _, id := range receiptedIDs {
		if statusByID[id] != string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want %q", id, statusByID[id], colony.TaskCompleted)
		}
	}
	for _, id := range pendingIDs {
		if statusByID[id] == string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want it to remain pending", id, statusByID[id])
		}
	}

	var report codexResultCollectionReport
	if err := store.LoadJSON(filepath.Join("build", "phase-1", "result-collection.json"), &report); err != nil {
		t.Fatalf("reload result-collection.json: %v", err)
	}
	gotCredited := append([]string{}, report.CreditedTaskIDs...)
	sort.Strings(gotCredited)
	if fmt.Sprint(gotCredited) != fmt.Sprint(receiptedIDs) {
		t.Fatalf("result-collection CreditedTaskIDs = %v, want %v", gotCredited, receiptedIDs)
	}
	gotUnfinished := append([]string{}, report.UnfinishedTaskIDs...)
	sort.Strings(gotUnfinished)
	if fmt.Sprint(gotUnfinished) != fmt.Sprint(pendingIDs) {
		t.Fatalf("result-collection UnfinishedTaskIDs = %v, want %v", gotUnfinished, pendingIDs)
	}

	var claims codexBuildClaims
	if err := store.LoadJSON("last-build-claims.json", &claims); err != nil {
		t.Fatalf("reload last-build-claims.json: %v", err)
	}
	gotClaimIDs := make([]string, 0, len(claims.TaskClaims))
	for _, c := range claims.TaskClaims {
		gotClaimIDs = append(gotClaimIDs, c.TaskID)
	}
	sort.Strings(gotClaimIDs)
	if fmt.Sprint(gotClaimIDs) != fmt.Sprint(receiptedIDs) {
		t.Fatalf("persisted claims cover tasks %v, want exactly %v", gotClaimIDs, receiptedIDs)
	}
}

// TestWrapperTaskReceiptRefusalsDoNotEraseValidReceipts proves a
// duplicate/unknown receipt entry mixed into the SAME external result is
// refused by name (the shared admission rules) without erasing the other,
// valid receipts in that same submission (195-CONTEXT.md D-08/D-09).
func TestWrapperTaskReceiptRefusalsDoNotEraseValidReceipts(t *testing.T) {
	root, phase, manifest, chain, ids := setupCoherentJobWrapperTest(t, "Refusals do not erase valid receipts")

	validIDs := ids[:3]
	receipts := make([]codex.TaskReceipt, 0, 5)
	touchedFiles := make([]string, 0, 4)
	for _, id := range validIDs {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}
	// A duplicate receipt for an already-claimed task ID -- neither
	// occurrence beyond the first may be admitted, but the first must stand.
	receipts = append(receipts, receiptForTask(t, root, validIDs[0]))
	// A receipt naming a task ID this phase has never heard of.
	receipts = append(receipts, codex.TaskReceipt{
		TaskID: "9.9", Status: codex.TaskReceiptStatusCompleted, Summary: "bogus",
		Handoff: codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: []string{"go test ./..."}},
	})

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "crashed with a mixed batch of valid and invalid receipts",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}

	dispatches, mergeViolations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(mergeViolations) > 0 {
		t.Fatalf("dispatch-level merge should not itself reject on receipt-level problems, got %+v", mergeViolations)
	}

	var merged codexBuildDispatch
	for _, d := range dispatches {
		if d.Name == chain.Name {
			merged = d
			break
		}
	}

	admission, violations := admitCoherentJobTaskReceipts(root, phase, merged, merged.Outputs, merged.TaskReceipts)
	if len(admission.Candidates) != len(validIDs) {
		gotIDs := make([]string, 0, len(admission.Candidates))
		for _, c := range admission.Candidates {
			gotIDs = append(gotIDs, c.TaskID)
		}
		t.Fatalf("admission credited %v, want exactly the %d valid receipts %v -- refusals must not erase them", gotIDs, len(validIDs), validIDs)
	}
	gotRules := map[string]bool{}
	for _, v := range violations {
		gotRules[v.Rule] = true
	}
	if !gotRules[violationRuleTaskReceiptDuplicate] {
		t.Errorf("expected a %s violation for the duplicate receipt, got %+v", violationRuleTaskReceiptDuplicate, violations)
	}
	if !gotRules[violationRuleTaskReceiptUnknown] {
		t.Errorf("expected a %s violation for the unknown task ID, got %+v", violationRuleTaskReceiptUnknown, violations)
	}
}
