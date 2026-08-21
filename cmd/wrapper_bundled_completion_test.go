package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
