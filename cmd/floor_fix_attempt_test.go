package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestTimedOutCheckIsAFailedCheck proves a verification step that hits its
// timeout is reported as failed, not skipped, contributes a blocking issue
// naming the check and the timeout duration, and is eligible for the fix
// path -- while a step whose command simply never resolved stays skipped and
// is NOT eligible. The two must be distinguishable.
func TestTimedOutCheckIsAFailedCheck(t *testing.T) {
	t.Run("a step that times out is failed, not skipped", func(t *testing.T) {
		step := runVerificationStep(context.Background(), ".", "tests", false, "sleep 2", 50*time.Millisecond)
		if !step.TimedOut {
			t.Fatalf("expected TimedOut=true, got %+v", step)
		}
		if step.Passed {
			t.Fatalf("expected Passed=false for a timed-out step, got %+v", step)
		}
		if step.Skipped {
			t.Fatalf("expected Skipped=false for a timed-out step, got %+v", step)
		}
		if step.ErrorClass != ErrorClassTimeout {
			t.Fatalf("expected ErrorClass=%q, got %q", ErrorClassTimeout, step.ErrorClass)
		}
		if !strings.Contains(step.Summary, "timed out") {
			t.Fatalf("expected Summary to name the timeout, got %q", step.Summary)
		}
	})

	t.Run("a timed-out step contributes a blocking issue naming the check and duration", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: sh -c 'sleep 2'")
		phase := colony.Phase{ID: 1, Name: "Timeout fixture"}
		floor := runDeterministicFloor(context.Background(), root, phase, codexContinueManifest{}, codexWatcherVerification{}, 50*time.Millisecond)
		if floor.ChecksPassed {
			t.Fatalf("expected a timed-out check to fail the floor, got %+v", floor)
		}
		found := false
		for _, issue := range floor.BlockingIssues {
			if strings.Contains(issue, "tests") && strings.Contains(issue, "timed out") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a blocking issue naming the tests check and the timeout, got %v", floor.BlockingIssues)
		}
	})

	t.Run("a step with no resolved command stays skipped and is not eligible", func(t *testing.T) {
		step := runVerificationStep(context.Background(), ".", "lint", false, "", 5*time.Second)
		if !step.Skipped {
			t.Fatalf("expected Skipped=true when no command resolved, got %+v", step)
		}
		if step.TimedOut {
			t.Fatalf("expected TimedOut=false when no command resolved, got %+v", step)
		}
		if !step.Passed {
			t.Fatalf("expected an optional check with no command to Pass (D-01 enrichment), got %+v", step)
		}
	})
}

// TestFailureIndexIsCompactAndNamesTheImplicatedTasks proves the index for a
// failing check with a long output is bounded in size (a small proportion of
// the raw output, not a fixed byte count pinned to one fixture), contains
// the failing check's name and the first distinct failure locations, and
// names the task whose reported changed files are implicated.
func TestFailureIndexIsCompactAndNamesTheImplicatedTasks(t *testing.T) {
	var raw strings.Builder
	raw.WriteString("cmd/widget.go:42: assertion failed: expected true got false\n")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&raw, "line %d: unrelated failure detail padding padding padding padding\n", i)
	}
	output := raw.String()

	claims := codexBuildClaims{
		TaskClaims: []codexBuildTaskClaim{
			{TaskID: "1.1", FilesModified: []string{"cmd/widget.go"}},
		},
	}
	step := codexVerificationStep{Name: "tests", Command: "go test ./...", ExitCode: 1, Output: output, Summary: "tests failed (exit 1)"}

	index := buildCheckFailureIndex(step, claims, colony.Phase{})

	if index.Check != "tests" {
		t.Fatalf("Check = %q, want %q", index.Check, "tests")
	}
	if len(index.Excerpts) == 0 {
		t.Fatalf("expected at least one excerpt")
	}
	joined := strings.Join(index.Excerpts, "\n")
	// The bound is asserted as a proportion of the raw output, not a fixed
	// byte count -- this must hold regardless of how large a real failure
	// log happens to be.
	if len(joined) >= len(output)/10 {
		t.Fatalf("index is not a small proportion of the raw output: index=%d bytes, output=%d bytes", len(joined), len(output))
	}
	if !index.Truncated {
		t.Fatalf("expected Truncated=true for output far exceeding the excerpt bound")
	}
	if len(index.ImplicatedTaskIDs) != 1 || index.ImplicatedTaskIDs[0] != "1.1" {
		t.Fatalf("ImplicatedTaskIDs = %v, want [1.1]", index.ImplicatedTaskIDs)
	}
}

// TestFailureIndexNeverCarriesTheWholeLog proves the index for a large
// output is strictly smaller than that output and does not contain its
// tail.
func TestFailureIndexNeverCarriesTheWholeLog(t *testing.T) {
	var raw strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&raw, "unique failure line %d with some padding text here\n", i)
	}
	output := raw.String()
	step := codexVerificationStep{Name: "tests", Command: "go test ./...", ExitCode: 1, Output: output, Summary: "tests failed"}

	index := buildCheckFailureIndex(step, codexBuildClaims{}, colony.Phase{})

	joined := strings.Join(index.Excerpts, "\n")
	if len(joined) >= len(output) {
		t.Fatalf("index (%d bytes) is not strictly smaller than the raw output (%d bytes)", len(joined), len(output))
	}
	tail := "unique failure line 999"
	if strings.Contains(joined, tail) {
		t.Fatalf("index carries the tail of the raw output: %v", index.Excerpts)
	}
	if !index.Truncated {
		t.Fatalf("expected Truncated=true for an output this much larger than the excerpt bound")
	}
}

// TestPlanCheckFixAttemptHonoursNonDefaultClaimsPath proves WR-01
// (193-REVIEW.md): planCheckFixAttempt must thread the real continue
// manifest through to loadRawBuildClaimsForScope, not a zero-value
// codexContinueManifest{}. A manifest naming a non-default ClaimsPath (the
// external/wrapper lane's completion packets can set one) must actually be
// read -- before the fix, this was silently ignored in favor of the
// default last-build-claims.json, which does not exist in this fixture, so
// ImplicatedTaskIDs would come back empty instead of naming the task whose
// claimed changed file the failure implicates.
func TestPlanCheckFixAttemptHonoursNonDefaultClaimsPath(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	taskID := "1.1"
	phase := colony.Phase{
		ID:    1,
		Name:  "Non-default claims path",
		Tasks: []colony.Task{{ID: &taskID, Goal: "Land the fix"}},
	}
	customClaims := codexBuildClaims{
		TaskClaims: []codexBuildTaskClaim{
			{TaskID: taskID, FilesModified: []string{"cmd/widget.go"}},
		},
	}
	claimsRel := "custom/wrapper-claims.json"
	if err := store.SaveJSON(claimsRel, customClaims); err != nil {
		t.Fatalf("save custom claims file: %v", err)
	}
	manifest := codexContinueManifest{
		Present: true,
		Data:    codexBuildManifest{ClaimsPath: ".aether/data/" + claimsRel},
	}
	failingStep := codexVerificationStep{
		Name: "tests", Command: "go test ./...", ExitCode: 1,
		Output: "cmd/widget.go:10: assertion failed", Summary: "tests failed (exit 1)",
	}
	floor := deterministicFloorResult{ChecksPassed: false, Steps: []codexVerificationStep{failingStep}}

	record, ok := planCheckFixAttempt(colony.ColonyState{}, phase, manifest, floor, false)
	if !ok {
		t.Fatalf("expected a fix attempt to be planned")
	}
	if len(record.FailureIndex.ImplicatedTaskIDs) != 1 || record.FailureIndex.ImplicatedTaskIDs[0] != taskID {
		t.Fatalf("expected ImplicatedTaskIDs=[%s] read from the manifest's own non-default ClaimsPath, got %v", taskID, record.FailureIndex.ImplicatedTaskIDs)
	}
}

// TestFailedCheckSendsExactlyOneBuilderFixAttempt proves a failing check with
// no reviewer dispatched produces exactly one builder dispatch whose reason
// names the failing check, and no reviewer dispatch of any kind.
func TestFailedCheckSendsExactlyOneBuilderFixAttempt(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "One bounded fix attempt"}
	manifest := codexContinueManifest{}

	verification, watcherFlow := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)

	if watcherFlow != nil {
		t.Fatalf("expected no reviewer dispatch of any kind, got watcherFlow=%+v", watcherFlow)
	}
	if verification.Watcher.Present && verification.Watcher.Status != "skipped" {
		t.Fatalf("expected no reviewer dispatched, got watcher=%+v", verification.Watcher)
	}
	if verification.CheckFixAttempt == nil {
		t.Fatalf("expected a check fix attempt to have run")
	}
	if verification.CheckFixAttempt.Check != "tests" {
		t.Fatalf("CheckFixAttempt.Check = %q, want %q", verification.CheckFixAttempt.Check, "tests")
	}
	if !strings.Contains(verification.CheckFixAttempt.Reason, "tests") {
		t.Fatalf("CheckFixAttempt.Reason = %q, want it to name the tests check", verification.CheckFixAttempt.Reason)
	}

	records := listBuildAttemptsForPhase(phase.ID)
	var builderDispatchCount, fixRecords int
	for _, r := range records {
		if r.CheckFix != nil {
			fixRecords++
		}
		for _, d := range r.Dispatches {
			if d.Caste == "builder" {
				builderDispatchCount++
			}
			if d.Caste == "watcher" {
				t.Fatalf("expected no watcher dispatch of any kind, found one in %+v", d)
			}
		}
	}
	if fixRecords != 1 {
		t.Fatalf("expected exactly 1 fix-attempt journal record, got %d", fixRecords)
	}
	if builderDispatchCount != 1 {
		t.Fatalf("expected exactly 1 builder dispatch, got %d", builderDispatchCount)
	}
}

// TestSecondFailureBlocksAndNamesTheCommand proves that after the fix
// attempt, a still-failing check blocks continue and the reported recovery
// carries one exact command to re-run the builder by hand.
func TestSecondFailureBlocksAndNamesTheCommand(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "Still failing after the fix attempt"}
	manifest := codexContinueManifest{}

	verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	if verification.ChecksPassed {
		t.Fatalf("expected the phase to still be blocked after the fix attempt (FakeInvoker does not repair the repo), got %+v", verification)
	}
	if verification.CheckFixAttempt == nil || verification.CheckFixAttempt.Outcome != "still_failing" {
		t.Fatalf("expected CheckFixAttempt.Outcome=still_failing, got %+v", verification.CheckFixAttempt)
	}

	now := time.Now().UTC()
	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{SkipWatchers: true}, now)
	if assessment.Passed {
		t.Fatalf("expected continue to block, got %+v", assessment)
	}
	if strings.TrimSpace(assessment.Recovery.CheckFixCommand) == "" {
		t.Fatalf("expected Recovery.CheckFixCommand to name one exact command, got empty")
	}

	gates := runCodexContinueGates(phase, manifest, verification, assessment, now, nil)
	var fixGate *gateCheck
	for i := range gates.Checks {
		if gates.Checks[i].Name == "check_fix_attempt" {
			fixGate = &gates.Checks[i]
		}
	}
	if fixGate == nil {
		t.Fatalf("expected a check_fix_attempt gate entry, got %+v", gates.Checks)
	}
	if fixGate.Passed {
		t.Fatalf("expected check_fix_attempt gate to fail, got %+v", fixGate)
	}
	if len(fixGate.RecoveryOptions) != 1 {
		t.Fatalf("expected exactly one recovery command, got %v", fixGate.RecoveryOptions)
	}
}

// TestFixAttemptNeverOverwritesTheFirstResult proves the attempt journal
// after the fix attempt contains both the original attempt and the fix
// attempt as separate entries, and the original's dispatches, claims and
// status are byte-identical to before.
func TestFixAttemptNeverOverwritesTheFirstResult(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "Original attempt preserved"}

	originalDispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Original work", Status: "completed"},
	}
	attemptRel, err := beginBuildAttempt(colony.ColonyState{}, phase.ID, phase, time.Now().UTC(), nil, "", "", "", "test-owner", originalDispatches)
	if err != nil {
		t.Fatalf("begin original build attempt: %v", err)
	}
	claims := &codexBuildClaims{BuildPhase: phase.ID}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "original build complete", originalDispatches, claims, "real", nil); err != nil {
		t.Fatalf("transition original build attempt: %v", err)
	}
	var before buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &before); err != nil {
		t.Fatalf("load original attempt: %v", err)
	}

	manifest := codexContinueManifest{}
	verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	if verification.CheckFixAttempt == nil {
		t.Fatalf("expected a fix attempt to have run")
	}

	var after buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &after); err != nil {
		t.Fatalf("reload original attempt: %v", err)
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("original attempt was mutated by the fix attempt:\nbefore=%s\nafter=%s", beforeJSON, afterJSON)
	}

	records := listBuildAttemptsForPhase(phase.ID)
	if len(records) != 2 {
		t.Fatalf("expected 2 separate attempt records (original + fix), got %d: %+v", len(records), records)
	}
	fixCount := 0
	for _, r := range records {
		if r.CheckFix != nil {
			fixCount++
			if r.CheckFix.ParentAttemptID != before.ID {
				t.Fatalf("fix attempt ParentAttemptID = %q, want %q", r.CheckFix.ParentAttemptID, before.ID)
			}
		}
	}
	if fixCount != 1 {
		t.Fatalf("expected exactly 1 record carrying CheckFix, got %d", fixCount)
	}
}

// TestNoSecondAutomaticFixAttempt proves that with the fix attempt already
// recorded for this phase and check, a further continue run does not send
// another builder automatically.
func TestNoSecondAutomaticFixAttempt(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "No second automatic attempt"}
	manifest := codexContinueManifest{}

	first, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	if first.CheckFixAttempt == nil {
		t.Fatalf("expected the first run to draw a fix attempt")
	}
	recordsAfterFirst := listBuildAttemptsForPhase(phase.ID)

	second, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	if second.CheckFixAttempt != nil {
		t.Fatalf("expected the second run to send no further automatic fix attempt, got %+v", second.CheckFixAttempt)
	}
	recordsAfterSecond := listBuildAttemptsForPhase(phase.ID)
	if len(recordsAfterSecond) != len(recordsAfterFirst) {
		t.Fatalf("expected no new attempt record from the second run: before=%d after=%d", len(recordsAfterFirst), len(recordsAfterSecond))
	}
}

// TestFixAttemptIsCountedSeparately proves the fix attempt appears in the
// dispatch count the team card and cost line read, distinguishable from the
// original build's workers.
func TestFixAttemptIsCountedSeparately(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return &codex.FakeInvoker{} }
	writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "Fix attempt counted separately"}

	originalDispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-1", Task: "Original work", Status: "completed"},
		{Stage: "wave", Wave: 1, Caste: "watcher", Name: "Keen-1", Task: "Verify", Status: "completed"},
	}
	attemptRel, err := beginBuildAttempt(colony.ColonyState{}, phase.ID, phase, time.Now().UTC(), nil, "", "", "", "test-owner", originalDispatches)
	if err != nil {
		t.Fatalf("begin original build attempt: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "original build complete", originalDispatches, nil, "real", nil); err != nil {
		t.Fatalf("transition original build attempt: %v", err)
	}

	manifest := codexContinueManifest{}
	verification, _ := runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)
	if verification.CheckFixAttempt == nil {
		t.Fatalf("expected a fix attempt to have run")
	}

	var original buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &original); err != nil {
		t.Fatalf("load original attempt: %v", err)
	}
	if len(original.Dispatches) != 2 {
		t.Fatalf("original attempt's own dispatch count changed: got %d, want 2", len(original.Dispatches))
	}

	records := listBuildAttemptsForPhase(phase.ID)
	var fixRecord *buildAttemptRecord
	for i := range records {
		if records[i].CheckFix != nil {
			fixRecord = &records[i]
		}
	}
	if fixRecord == nil {
		t.Fatalf("expected a fix-attempt record in the journal")
	}
	if len(fixRecord.Dispatches) != 1 || fixRecord.Dispatches[0].Caste != "builder" {
		t.Fatalf("expected the fix attempt's own dispatch count to be exactly 1 builder, got %+v", fixRecord.Dispatches)
	}
	if fixRecord.ID == original.ID {
		t.Fatalf("fix attempt shares an ID with the original attempt, not counted separately")
	}
}
