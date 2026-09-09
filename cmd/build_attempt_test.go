package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBoundaryBuildFixturesUseCanonicalAuthority200(t *testing.T) {
	cases := []struct {
		file      string
		signature string
		want      []string
	}{
		{
			file:      "ceremony_team_checkin_test.go",
			signature: "func TestPendingDecisionStillRendersFullCheckinCard(",
			want:      []string{"createApprovedAcceptedBuildTestColony(", "root, 5,"},
		},
		{
			file:      "orchestrator_boundary_guidance_test.go",
			signature: "func TestBuildFinalizeAddsOrchestratorBoundaryGuidance(",
			want:      []string{"createApprovedAcceptedBuildTestColony(", "commitTestBuildStartAt(", "ExecutionOwner:", "DispatchMode:"},
		},
		{
			file:      "build_attempt_test.go",
			signature: "func TestResumeDashboard" + "DoesNotRedispatchLiveBuildProcess(",
			want:      []string{"ProcessState: testBuildProcessLive", "ExecutionOwner:", "DispatchMode:"},
		},
	}
	for _, tc := range cases {
		content, err := os.ReadFile(tc.file)
		if err != nil {
			t.Fatalf("read %s: %v", tc.file, err)
		}
		body := extractGoFunctionBody(t, string(content), tc.signature)
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Errorf("%s must contain explicit authority marker %q", tc.signature, want)
			}
		}
	}
}

func TestBuildAttemptPersistsTransitionsAndTerminalEvidence(t *testing.T) {
	saveGlobals(t)
	startedAt := time.Now().UTC()
	dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", TaskID: "1.1", Status: "spawned"}}
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt: startedAt, SelectedTasks: []string{"1.1"}, ExecutionOwner: "go-runtime", Dispatches: dispatches,
		PrepareRoot: func(root string) {
			if err := os.WriteFile(filepath.Join(root, "app.txt"), []byte("attempt evidence\n"), 0644); err != nil {
				t.Fatalf("write attempt artifact: %v", err)
			}
		},
	})
	root, attemptRel := fixture.Root, fixture.AttemptPath
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "dispatch started", dispatches, nil, "real", nil); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	dispatches[0].Status = "completed"
	summary := &codex.ClaimsSummary{
		FilesCreated: []string{"app.txt"},
		TaskClaims:   []codex.TaskClaimsSummary{{TaskID: "1.1", FilesCreated: []string{"app.txt"}}},
	}
	claims, err := recordBuildAttemptTerminal(root, attemptRel, 1, startedAt, dispatches, summary, "real", nil)
	if err != nil {
		t.Fatalf("record terminal attempt: %v", err)
	}
	if len(claims.ArtifactEvidence) != 1 || claims.ArtifactEvidence[0].Path != "app.txt" || claims.ArtifactEvidence[0].SHA256 == "" {
		t.Fatalf("terminal claims lack artifact fingerprint: %+v", claims)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "built committed", dispatches, claims, "real", nil); err != nil {
		t.Fatalf("mark built: %v", err)
	}

	loadedRel, record, ok := loadLatestBuildAttempt(1)
	if !ok || loadedRel != attemptRel {
		t.Fatalf("latest attempt = %q %+v %v, want %q", loadedRel, record, ok, attemptRel)
	}
	if record.Status != buildAttemptBuilt || record.Recoverable || record.RecoveryCommand != "" || record.OriginalStateSHA == "" {
		t.Fatalf("attempt record = %+v", record)
	}
	if len(record.History) != 5 || record.History[0].Status != buildAttemptPrepared || record.History[1].Status != buildAttemptAwaiting || record.History[2].Status != buildAttemptDispatching || record.History[3].Status != buildAttemptTerminal || record.History[4].Status != buildAttemptBuilt {
		t.Fatalf("attempt history = %+v", record.History)
	}
}

func TestForceRedispatchMarksActiveAttemptInterrupted(t *testing.T) {
	saveGlobals(t)
	startedAt := time.Now().UTC().Add(-time.Hour)
	fixture := commitTestBuildStart(t, testBuildStartOptions{GeneratedAt: startedAt, ExecutionOwner: "go-runtime"})
	attemptRel := fixture.AttemptPath
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "workers active", nil, nil, "real", nil); err != nil {
		t.Fatalf("mark dispatching: %v", err)
	}
	if err := interruptLatestBuildAttempt(1, "superseded by force redispatch"); err != nil {
		t.Fatalf("interrupt attempt: %v", err)
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.Status != buildAttemptInterrupted || record.CompletedAt == "" || record.Error == "" {
		t.Fatalf("interrupted attempt = %+v, present=%v", record, ok)
	}
}

func TestStatusSurfacesFailedAttemptAfterLifecycleRollback(t *testing.T) {
	saveGlobals(t)
	startedAt := time.Now().UTC()
	fixture := commitTestBuildStart(t, testBuildStartOptions{GeneratedAt: startedAt, ExecutionOwner: "go-runtime"})
	attemptRel, state := fixture.AttemptPath, fixture.State
	if err := transitionBuildAttempt(attemptRel, buildAttemptFailed, "dispatch failed", nil, nil, "real", os.ErrDeadlineExceeded); err != nil {
		t.Fatalf("mark failed attempt: %v", err)
	}
	result := buildStatusResult(state, store)
	summary, ok := result["build_attempt"].(map[string]interface{})
	if !ok || summary["status"] != buildAttemptFailed || summary["recovery_command"] != "aether build 1 --force" {
		t.Fatalf("status build_attempt = %#v", result["build_attempt"])
	}
}

func TestBuildAttemptVisualShowsDurableRunAndWorkerState(t *testing.T) {
	attempt := buildAttemptRecord{
		ID:              "attempt-visual",
		Status:          buildAttemptDispatching,
		RunID:           "run-visual-proof",
		ManifestSHA256:  "1234567890abcdef",
		RecoveryCommand: "aether build 1 --force",
		Dispatches:      []codexBuildDispatch{{Name: "Builder-1"}, {Name: "Watcher-1"}},
		WorkerRuns: []buildAttemptWorkerRun{
			{WorkerName: "Builder-1", Status: buildWorkerCompleted},
			{WorkerName: "Watcher-1", Status: buildWorkerDispatching},
		},
	}

	output := renderBuildAttemptStatus(attempt)
	for _, expected := range []string{
		"run-visual-proof",
		"Manifest: 1234567890ab",
		"1 completed | 1 active",
		"aether build 1 --force",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("build attempt visual missing %q:\n%s", expected, output)
		}
	}
}

// TestStageBuildAttemptCompletionRejectsSemanticViolations proves D-07:
// staging runs the same semantic validation build-finalize runs, before
// anything is written. A packet finalize would reject must never reach the
// durable completion file or the attempt journal's digest/path binding.
func TestStageBuildAttemptCompletionRejectsSemanticViolations(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	corrupted := completion
	corrupted.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	corrupted.Dispatches[0].Handoff.VerificationStatus = "not-a-real-status"
	corrupted.Dispatches[0].FilesModified = []string{"/etc/passwd"}

	_, _, err := stageBuildAttemptCompletion(attemptRel, corrupted)
	if err == nil {
		t.Fatal("staging a semantically invalid packet was accepted")
	}
	var contractErr *completionContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("expected *completionContractError, got %v (%T)", err, err)
	}
	if len(contractErr.Violations) < 2 {
		t.Fatalf("expected a multi-violation rejection (invalid handoff status + absolute path), got %+v", contractErr.Violations)
	}

	completionRel := durableBuildCompletionPath(manifest.Phase, manifest.AttemptID)
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(completionRel))); statErr == nil {
		t.Fatalf("rejected stage wrote a durable completion file at %s", completionRel)
	}
	_, reloaded, ok := loadLatestBuildAttempt(1)
	if !ok || reloaded.CompletionSHA256 != "" || reloaded.CompletionPath != "" {
		t.Fatalf("rejected stage bound completion data onto the attempt journal: %+v", reloaded)
	}
	if reloaded.Status != buildAttemptAwaiting {
		t.Fatalf("rejected stage mutated attempt status: %+v", reloaded)
	}
}

// TestStageBuildAttemptCompletionAcceptsValidPacket proves the clean path is
// unchanged by D-07: a packet that passes semantic validation is durably
// written and bound exactly as before.
func TestStageBuildAttemptCompletionAcceptsValidPacket(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	displayPath, digest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("stage valid completion: %v", err)
	}
	if displayPath == "" || digest == "" {
		t.Fatalf("stage valid completion returned empty path/digest: %q %q", displayPath, digest)
	}
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(displayPath))); statErr != nil {
		t.Fatalf("durable completion file missing after valid stage: %v", statErr)
	}
}

// TestStageBuildAttemptCompletionRejectsMismatchedIdentity proves identity
// checks still run first (D-07 ordering): a packet for the wrong attempt is
// a routing error, not a contract violation, and must fail before semantic
// validation runs at all.
func TestStageBuildAttemptCompletionRejectsMismatchedIdentity(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	mismatched := completion
	badManifest := manifest
	badManifest.AttemptID = "attempt-does-not-exist"
	mismatched.DispatchManifest = &badManifest

	_, _, err := stageBuildAttemptCompletion(attemptRel, mismatched)
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("mismatched attempt identity should be rejected on identity, got %v", err)
	}
	var contractErr *completionContractError
	if errors.As(err, &contractErr) {
		t.Fatalf("mismatched identity should fail before semantic validation runs, got a contract error: %+v", contractErr.Violations)
	}
}

// TestStageBuildAttemptCompletionAllowsRebindWhileNonTerminal proves D-08:
// a corrected packet may replace the bound one while the attempt has not yet
// finalized successfully. Correcting a packet no longer needs a new attempt.
func TestStageBuildAttemptCompletionAllowsRebindWhileNonTerminal(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "workers dispatching", nil, nil, "external-task", nil); err != nil {
		t.Fatalf("mark attempt dispatching: %v", err)
	}

	_, firstDigest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("stage first completion: %v", err)
	}

	corrected := completion
	corrected.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	corrected.Dispatches[0].Summary += " corrected"

	secondDisplayPath, secondDigest, err := stageBuildAttemptCompletion(attemptRel, corrected)
	if err != nil {
		t.Fatalf("rebind while dispatching (non-terminal) should succeed, got %v", err)
	}
	if secondDigest == firstDigest {
		t.Fatalf("corrected packet produced the same digest as the original")
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.CompletionSHA256 != secondDigest || record.CompletionPath != secondDisplayPath {
		t.Fatalf("attempt did not rebind to the corrected packet: %+v", record)
	}
}

// TestStageBuildAttemptCompletionRebindDeletesStaleDurableFile proves
// T-163.1-13: when a rebind while unsealed points the attempt at a different
// durable path, the superseded file is deleted rather than left orphaned.
// durableBuildCompletionPath is a pure function of phase+attempt ID, so a
// legitimate rebind never actually produces a different path today -- this
// test seeds the record directly to exercise the defensive cleanup.
func TestStageBuildAttemptCompletionRebindDeletesStaleDurableFile(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	staleRel := "build/phase-1/attempts/stale-completion.json"
	staleDisplay := displayDataPath(staleRel)
	if err := store.AtomicWrite(staleRel, []byte("{}")); err != nil {
		t.Fatalf("seed stale completion file: %v", err)
	}
	var seeded buildAttemptRecord
	if err := store.UpdateJSONAtomically(attemptRel, &seeded, func() error {
		seeded.CompletionSHA256 = "stale-digest"
		seeded.CompletionPath = staleDisplay
		return nil
	}); err != nil {
		t.Fatalf("seed stale attempt binding: %v", err)
	}

	displayPath, digest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("rebind to a different durable path while unsealed should succeed, got %v", err)
	}
	if displayPath == staleDisplay {
		t.Fatalf("expected the new completion path to differ from the stale path")
	}
	if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(staleRel))); !os.IsNotExist(statErr) {
		t.Fatalf("stale completion file was not removed after rebind: err=%v", statErr)
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.CompletionSHA256 != digest || record.CompletionPath != displayPath {
		t.Fatalf("attempt did not rebind to the new completion: %+v", record)
	}
}

// TestStageBuildAttemptCompletionAllowsRebindAtTerminalStatus proves the
// rebind window stays open at buildAttemptTerminal -- worker results have
// been recorded but finalize has not yet committed the built lifecycle
// state, so this must NOT be treated as sealed (see buildAttemptCompletionSealed).
func TestStageBuildAttemptCompletionAllowsRebindAtTerminalStatus(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	if _, _, err := stageBuildAttemptCompletion(attemptRel, completion); err != nil {
		t.Fatalf("stage initial completion: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptTerminal, "worker results recorded before finalize completes", nil, nil, "external-task", nil); err != nil {
		t.Fatalf("mark attempt terminal: %v", err)
	}

	corrected := completion
	corrected.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	corrected.Dispatches[0].Summary += " corrected before finalize completed"

	_, digest, err := stageBuildAttemptCompletion(attemptRel, corrected)
	if err != nil {
		t.Fatalf("a terminal (not yet built) attempt should still accept a corrected packet, got %v", err)
	}
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok || record.CompletionSHA256 != digest {
		t.Fatalf("attempt did not accept the corrected packet at terminal status: %+v", record)
	}
}

// TestBuildAttemptRejectsRebindAfterTerminal proves the other half of D-08:
// once an attempt has finalized successfully (status built), the rebind
// window is permanently closed and a non-identical packet is rejected.
func TestBuildAttemptRejectsRebindAfterTerminal(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	if _, _, err := stageBuildAttemptCompletion(attemptRel, completion); err != nil {
		t.Fatalf("stage initial completion: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "simulate finalize commit", nil, nil, "external-task", nil); err != nil {
		t.Fatalf("mark attempt built: %v", err)
	}

	changed := completion
	changed.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	changed.Dispatches[0].Summary += " tampered after seal"

	if _, _, err := stageBuildAttemptCompletion(attemptRel, changed); err == nil || !strings.Contains(err.Error(), "does not match the result already bound") {
		t.Fatalf("sealed attempt should reject a changed packet, got %v", err)
	}
}

// TestBuildAttemptIdempotentResubmit proves a byte-identical resubmit
// against a built (sealed) attempt still succeeds idempotently.
func TestBuildAttemptIdempotentResubmit(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	displayPath, digest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("stage initial completion: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "simulate finalize commit", nil, nil, "external-task", nil); err != nil {
		t.Fatalf("mark attempt built: %v", err)
	}

	resubmitPath, resubmitDigest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("byte-identical resubmit against a built attempt should still succeed, got %v", err)
	}
	if resubmitPath != displayPath || resubmitDigest != digest {
		t.Fatalf("resubmit changed the binding: path=%q digest=%q, want %q %q", resubmitPath, resubmitDigest, displayPath, digest)
	}
}

func TestResumeDashboardDoesNotRedispatchLiveBuildProcess(t *testing.T) {
	saveGlobals(t)
	startedAt := time.Now().UTC()
	commitTestBuildStart(t, testBuildStartOptions{GeneratedAt: startedAt, ExecutionOwner: "go-runtime"})
	result := buildResumeDashboardResult()
	recovery, ok := result["recovery"].(map[string]interface{})
	if !ok || recovery["next"] != "aether watch" {
		t.Fatalf("live build recovery = %#v, want aether watch", result["recovery"])
	}
}

// TestCommittedAttemptRefusalNamesAWorkingPath pins the recovery hint on the
// one refusal that has demonstrably cost real dispatches.
//
// reconcileCommittedExternalBuildAttempt is the idempotency path: re-submitting
// the same completion packet for an already-committed build returns the same
// result. A different packet is refused, correctly — a committed build is not
// superseded by a later one.
//
// The refusal used to read "cannot reconcile its committed state without
// matching terminal evidence", which invites the reader to go and produce
// matching evidence. That is impossible: every redispatch yields a new digest.
// A real session spent six worker dispatches finding that out, re-running the
// phase at four workers, then six, then the full eleven, before concluding it
// could not be done. The fix for its actual situation — files changed after
// sign-off because a reviewer's findings were fixed — was `aether continue`,
// which re-runs verification and accepts amended artifacts when it passes.
//
// An error that names no working path is how a user burns an afternoon, so the
// message must keep naming one.
func TestCommittedAttemptRefusalNamesAWorkingPath(t *testing.T) {
	binding := buildAttemptManifestBinding{
		Bound: true,
		Path:  ".aether/data/build/phase-2/attempts/attempt-1.json",
		Record: buildAttemptRecord{
			ID:               "attempt-1",
			CompletionSHA256: "committed-digest",
			Claims:           &codexBuildClaims{},
		},
	}

	_, _, _, _, err := reconcileCommittedExternalBuildAttempt(colony.ColonyState{}, 2, binding, "a-different-digest")
	if err == nil {
		t.Fatal("expected a committed attempt to refuse a different completion packet")
	}

	msg := err.Error()
	if !strings.Contains(msg, "aether continue") {
		t.Errorf("refusal names no working recovery path:\n  %s", msg)
	}
	if !strings.Contains(msg, "already committed") {
		t.Errorf("refusal does not say what the situation actually is:\n  %s", msg)
	}
	if strings.Contains(msg, "without matching terminal evidence") {
		t.Errorf("refusal still invites the reader to chase evidence that cannot be produced:\n  %s", msg)
	}
}

// coherentJobPartialFailInvoker is a scripted native/direct-lane worker
// invoker that always returns a "failed" terminal result carrying the given
// task receipts and touched files -- the direct-lane counterpart of the
// external-lane fixtures in merged_dispatch_task_credit_test.go
// (receiptForTask/sixChainedTasks), used to drive a genuine four-of-six
// partial-credit dispatch through the real runCodexBuild entrypoint.
type coherentJobPartialFailInvoker struct {
	receipts []codex.TaskReceipt
	touched  []string
}

func (i *coherentJobPartialFailInvoker) Invoke(_ context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName:    config.WorkerName,
		Caste:         config.Caste,
		TaskID:        config.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing four of six steps",
		FilesModified: i.touched,
		TaskReceipts:  i.receipts,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
	}, nil
}

func (i *coherentJobPartialFailInvoker) IsAvailable(context.Context) bool { return true }
func (i *coherentJobPartialFailInvoker) ValidateAgent(string) error       { return nil }

// TestGroupedJobPartialRetryIsAppendOnlyDirect is the direct/native-lane
// counterpart of the external-lane fixture: a single Builder covering six
// chained tasks crashes after finishing four (proven by real, root-backed
// task receipts). D-10 requires this to bypass the ordinary all-or-nothing
// rollback, credit exactly the four proven tasks, and create a brand-new,
// parent-linked, append-only retry attempt for the other two -- never
// rewriting the first attempt's own dispatch, receipts, or claims.
func TestGroupedJobPartialRetryIsAppendOnlyDirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Direct lane partial credit creates an append-only retry"
	tasks, ids := sixChainedTasks()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Direct partial chain", Description: "One worker, six dependent steps",
			Status: colony.PhaseReady, Tasks: tasks,
		}}},
	})

	proven := ids[:4]
	pending := ids[4:]
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touched := make([]string, 0, len(proven))
	for _, id := range proven {
		receipts = append(receipts, receiptForTask(t, root, id))
		touched = append(touched, taskFileName(id))
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		return &coherentJobPartialFailInvoker{receipts: receipts, touched: touched}
	}
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild should accept a validated partial-credit failure, got error: %v", err)
	}
	if recovery, _ := result["recovery_job"].(bool); !recovery {
		t.Fatalf("result did not report a D-10 recovery job: %+v", result)
	}
	parentAttemptID, _ := result["parent_attempt_id"].(string)
	retryAttemptID, _ := result["retry_attempt_id"].(string)
	if parentAttemptID == "" || retryAttemptID == "" || parentAttemptID == retryAttemptID {
		t.Fatalf("expected distinct non-empty parent/retry attempt IDs, got parent=%q retry=%q", parentAttemptID, retryAttemptID)
	}
	unfinished, _ := result["unfinished_task_ids"].([]string)
	if len(unfinished) != 2 {
		t.Fatalf("unfinished_task_ids = %v, want exactly the two pending tasks", unfinished)
	}
	if cmd, _ := result["recovery_command"].(string); !strings.Contains(cmd, "aether build 1") {
		t.Fatalf("recovery_command = %q, does not name a working redispatch for phase 1", cmd)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.State == colony.StateBUILT {
		t.Fatalf("partial credit advanced colony to BUILT: %+v", state)
	}
	statusByID := map[string]string{}
	for _, task := range state.Plan.Phases[0].Tasks {
		statusByID[*task.ID] = string(task.Status)
	}
	for _, id := range proven {
		if statusByID[id] != string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want %q", id, statusByID[id], colony.TaskCompleted)
		}
	}
	for _, id := range pending {
		if statusByID[id] == string(colony.TaskCompleted) {
			t.Fatalf("task %s (unfinished) is %q, want it to remain pending", id, statusByID[id])
		}
	}

	// Parent attempt: found via listBuildAttemptsForPhase, which is the only
	// reader the recovery record ever had. The phase's "latest attempt" marker
	// deliberately still points at the PARENT (WR-03, 195-REVIEW.md): a
	// recovery record dispatches nothing, and pointing "latest" at it made the
	// runtime believe the phase still had live workers, so the very
	// plan-only call the owner's recovery command makes was refused.
	var parent, child buildAttemptRecord
	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ID == parentAttemptID {
			parent = record
		}
		if record.ID == retryAttemptID {
			child = record
		}
	}
	if parent.ID == "" {
		t.Fatalf("parent attempt %s not found in the phase's attempt journal", parentAttemptID)
	}
	// WR-04 (195-REVIEW.md): the direct lane used to leave this reading
	// `failed`, whose documented meaning is "nothing of this attempt was
	// credited", while real task credit sat in colony state. A partially
	// credited attempt is recorded as `partial` on BOTH lanes.
	if parent.Status != buildAttemptPartial {
		t.Fatalf("parent attempt status = %q, want %q (it credited four of six tasks)", parent.Status, buildAttemptPartial)
	}
	if parent.ParentAttemptID != "" {
		t.Fatalf("parent attempt unexpectedly carries its own ParentAttemptID: %+v", parent)
	}
	if child.ID == "" {
		t.Fatalf("retry attempt %s not found in the phase's attempt journal", retryAttemptID)
	}
	if child.ParentAttemptID != parentAttemptID {
		t.Fatalf("child ParentAttemptID = %q, want %q", child.ParentAttemptID, parentAttemptID)
	}

	latestRel, latest, ok := loadLatestBuildAttempt(1)
	if !ok || latest.ID != parentAttemptID {
		t.Fatalf("latest attempt pointer = %q %+v, want the parent attempt %s -- a recovery record must never occupy the phase's live-attempt slot (WR-03)", latestRel, latest, parentAttemptID)
	}
}

// TestGroupedJobRetryNeverReassignsCreditedTasks proves the negative half of
// D-10 directly against the retry attempt's own dispatch: none of the four
// credited task IDs may appear anywhere in the child's covered tasks -- a
// fresh worker dispatched from the retry attempt would otherwise redo proven
// work.
func TestGroupedJobRetryNeverReassignsCreditedTasks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "A retry dispatch never re-lists a credited task"
	tasks, ids := sixChainedTasks()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "No re-credit chain", Description: "One worker, six dependent steps",
			Status: colony.PhaseReady, Tasks: tasks,
		}}},
	})

	proven := ids[:4]
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touched := make([]string, 0, len(proven))
	for _, id := range proven {
		receipts = append(receipts, receiptForTask(t, root, id))
		touched = append(touched, taskFileName(id))
	}

	originalInvoker := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker {
		return &coherentJobPartialFailInvoker{receipts: receipts, touched: touched}
	}
	t.Cleanup(func() { newCodexWorkerInvoker = originalInvoker })

	result, err := runCodexBuild(root, 1, nil, false)
	if err != nil {
		t.Fatalf("runCodexBuild: %v", err)
	}
	retryAttemptID, _ := result["retry_attempt_id"].(string)

	var child buildAttemptRecord
	found := false
	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ID == retryAttemptID {
			child = record
			found = true
		}
	}
	if !found {
		t.Fatalf("retry attempt %s not found", retryAttemptID)
	}
	credited := stringSet(proven)
	for _, dispatch := range child.Dispatches {
		for _, taskID := range dispatchCoveredTaskIDs(dispatch) {
			if credited[taskID] {
				t.Fatalf("retry dispatch %+v re-lists credited task %s", dispatch, taskID)
			}
		}
	}
}
