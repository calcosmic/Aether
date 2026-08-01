package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildCompletionStageMakesWrapperResultResumableWithoutRedispatch(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")

	durablePath, digest, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("stage build completion: %v", err)
	}
	wantPath := displayDataPath(durableBuildCompletionPath(1, manifest.AttemptID))
	if durablePath != wantPath || digest == "" {
		t.Fatalf("staged completion = %q %q, want %q and digest", durablePath, digest, wantPath)
	}
	loaded, err := loadExternalBuildCompletion(filepath.Join(root, filepath.FromSlash(durablePath)))
	if err != nil {
		t.Fatalf("load durable completion through finalizer contract: %v", err)
	}
	loadedDigest, err := jsonSHA256(loaded)
	if err != nil || loadedDigest != digest {
		t.Fatalf("durable completion digest = %q, err=%v, want %q", loadedDigest, err, digest)
	}

	_, attempt, ok := loadLatestBuildAttempt(1)
	if !ok || attempt.CompletionPath != durablePath || attempt.CompletionSHA256 != digest {
		t.Fatalf("attempt did not retain durable completion: %+v", attempt)
	}
	dashboard := buildResumeDashboardResult()
	recovery := dashboard["recovery"].(map[string]interface{})
	wantRecovery := buildFinalizeRecoveryCommand(1, durablePath)
	if recovery["next"] != wantRecovery {
		t.Fatalf("resume recovery next = %v, want %q", recovery["next"], wantRecovery)
	}

	result, state, _, _, err := runCodexBuildFinalize(root, 1, loaded, false)
	if err != nil {
		t.Fatalf("finalize durable completion: %v", err)
	}
	if result["idempotent"] != false || state.State != colony.StateBUILT {
		t.Fatalf("durable completion did not finalize original attempt: result=%+v state=%s", result, state.State)
	}

	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(durablePath)))
	if err != nil {
		t.Fatalf("durable completion disappeared after finalization: %v", err)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil || envelope["result"] == nil {
		t.Fatalf("durable completion packet is not inspectable after finalization: err=%v", err)
	}
}

// TestBuildCompletionStageRejectsChangedPacketWithoutDeletingRecoveryEvidence
// originally asserted that ANY attempt rejects a changed staged packet --
// that "any attempt" contract was the exact behavior D-08 (plan 03) removes:
// a corrected packet may now rebind while the attempt is unsealed. This test
// is retargeted at a `built` (sealed) attempt, where rejection is still the
// correct contract, and keeps proving the sealed rejection does not destroy
// the recovery evidence already on disk. The non-terminal rebind-succeeds
// case this test used to (incorrectly) cover now lives in
// TestStageBuildAttemptCompletionAllowsRebindWhileNonTerminal
// (cmd/build_attempt_test.go).
func TestBuildCompletionStageRejectsChangedPacketWithoutDeletingRecoveryEvidence(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")
	durablePath, _, err := stageBuildAttemptCompletion(attemptRel, completion)
	if err != nil {
		t.Fatalf("stage initial completion: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "simulate finalize commit", nil, nil, "external-task", nil); err != nil {
		t.Fatalf("mark attempt built: %v", err)
	}
	changed := completion
	changed.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	changed.Dispatches[0].Summary += " tampered"
	if _, _, err := stageBuildAttemptCompletion(attemptRel, changed); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("changed staged completion should be rejected once sealed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(durablePath))); err != nil {
		t.Fatalf("rejected restage deleted valid recovery evidence: %v", err)
	}
}

func TestBuildPlanOnlyBindsAndSupersedesExternalAttempt(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	firstRel, first, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("plan-only build did not persist a latest attempt")
	}
	if manifest.AttemptID != first.ID || manifest.AttemptPath != displayDataPath(firstRel) {
		t.Fatalf("manifest attempt identity = %q %q, journal = %q %q", manifest.AttemptID, manifest.AttemptPath, first.ID, displayDataPath(firstRel))
	}
	if first.Status != buildAttemptAwaiting || first.PlanManifest == nil || first.ManifestSHA256 == "" {
		t.Fatalf("attempt was not durably bound to the plan manifest: %+v", first)
	}

	// Re-entry contract (changed after the v1.25 review): a dangling
	// plan-only attempt still awaiting external workers holds no worker
	// output, so a fresh plan-only request supersedes it automatically — the
	// old behavior blocked every /ant-build after an aborted one until the
	// user discovered --force.
	second, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only re-entry over an idle plan-only attempt should supersede automatically, got %v", err)
	}
	secondManifest := second["dispatch_manifest"].(codexBuildManifest)
	if secondManifest.AttemptID == manifest.AttemptID {
		t.Fatalf("plan-only re-entry reused attempt %s instead of opening a fresh one", manifest.AttemptID)
	}
	var superseded buildAttemptRecord
	if err := store.LoadJSON(firstRel, &superseded); err != nil {
		t.Fatalf("reload superseded attempt: %v", err)
	}
	if superseded.Status != buildAttemptInterrupted {
		t.Fatalf("superseded attempt status = %s, want %s", superseded.Status, buildAttemptInterrupted)
	}

	// An attempt that has moved past awaiting_external (workers dispatching)
	// still blocks without --force — automatic supersession applies ONLY to
	// idle plan-only attempts.
	secondRel, secondAttempt, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("second plan-only attempt not persisted")
	}
	secondAttempt.Status = buildAttemptDispatching
	if err := store.SaveJSON(secondRel, &secondAttempt); err != nil {
		t.Fatalf("mark attempt dispatching: %v", err)
	}
	if _, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil); err == nil || !strings.Contains(err.Error(), "active build attempt") {
		t.Fatalf("plan-only over a dispatching attempt should still require --force, got %v", err)
	}
	forced, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{Force: true})
	if err != nil {
		t.Fatalf("forced plan-only redispatch returned error: %v", err)
	}
	forcedManifest := forced["dispatch_manifest"].(codexBuildManifest)
	if forcedManifest.AttemptID == secondManifest.AttemptID {
		t.Fatalf("forced redispatch reused attempt %s", secondManifest.AttemptID)
	}

	staleCompletion := codexExternalBuildCompletion{DispatchManifest: &manifest}
	if _, _, _, _, err := runCodexBuildFinalize(root, 1, staleCompletion, false); err == nil || !strings.Contains(err.Error(), "superseded") {
		t.Fatalf("stale completion should be rejected as superseded, got %v", err)
	}
}

func TestBuildFinalizeIsIdempotentForBoundCompletion(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)

	firstResult, firstState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("first build-finalize returned error: %v", err)
	}
	if firstResult["idempotent"] != false || firstState.State != colony.StateBUILT {
		t.Fatalf("first finalization result = %+v, state = %s", firstResult, firstState.State)
	}
	attemptRel, firstAttempt, ok := loadLatestBuildAttempt(1)
	if !ok || firstAttempt.ID != manifest.AttemptID || firstAttempt.Status != buildAttemptBuilt || firstAttempt.CompletionSHA256 == "" {
		t.Fatalf("first finalization did not commit the bound attempt: %+v", firstAttempt)
	}
	firstHistoryLen := len(firstAttempt.History)
	firstEventLen := len(firstState.Events)

	secondResult, secondState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("idempotent build-finalize retry returned error: %v", err)
	}
	if secondResult["idempotent"] != true {
		t.Fatalf("retry did not report idempotent success: %+v", secondResult)
	}
	if len(secondState.Events) != firstEventLen {
		t.Fatalf("idempotent retry appended lifecycle events: %d -> %d", firstEventLen, len(secondState.Events))
	}
	var secondAttempt buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &secondAttempt); err != nil {
		t.Fatalf("reload attempt after retry: %v", err)
	}
	if len(secondAttempt.History) != firstHistoryLen {
		t.Fatalf("idempotent retry appended attempt transitions: %d -> %d", firstHistoryLen, len(secondAttempt.History))
	}

	changed := completion
	changed.Dispatches = append([]codexExternalBuildWorkerResult{}, completion.Dispatches...)
	changed.Dispatches[0].Summary += " changed"
	if _, _, _, _, err := runCodexBuildFinalize(root, 1, changed, false); err == nil || !strings.Contains(err.Error(), "does not match the result already bound") {
		t.Fatalf("changed completion replay should be rejected, got %v", err)
	}
}

func TestBuildFinalizeRecoversBoundTerminalAttemptWithoutRedispatch(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	dispatches, violations, err := mergeExternalBuildResults(manifest, completion.workerResults())
	if err != nil {
		t.Fatalf("merge completion results: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
	startedAt := parseManifestGeneratedAt(manifest)
	claims, err := completion.claimsOrAggregate(root, 1, startedAt, dispatches)
	if err != nil {
		t.Fatalf("aggregate completion claims: %v", err)
	}
	attemptRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")
	if _, err := bindBuildAttemptCompletion(attemptRel, completion); err != nil {
		t.Fatalf("bind completion before simulated interruption: %v", err)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptTerminal, "simulated interruption after terminal results", dispatches, &claims, "external-task", nil); err != nil {
		t.Fatalf("record terminal attempt: %v", err)
	}

	result, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("recovery build-finalize returned error: %v", err)
	}
	if result["attempt"] != manifest.AttemptPath || state.State != colony.StateBUILT {
		t.Fatalf("recovery did not finish the original attempt: result=%+v state=%s", result, state.State)
	}
	latestRel, latest, ok := loadLatestBuildAttempt(1)
	if !ok || latestRel != attemptRel || latest.ID != manifest.AttemptID || latest.Status != buildAttemptBuilt {
		t.Fatalf("recovery created or left the wrong attempt: rel=%q record=%+v", latestRel, latest)
	}
}

func TestBuildFinalizeReconcilesJournalAfterBuiltStateCommit(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("initial build-finalize returned error: %v", err)
	}
	attemptRel, attempt, ok := loadLatestBuildAttempt(1)
	if !ok || attempt.Claims == nil {
		t.Fatalf("load built attempt: %+v", attempt)
	}
	if err := transitionBuildAttempt(attemptRel, buildAttemptTerminal, "simulate journal write lag after state commit", attempt.Dispatches, attempt.Claims, "external-task", nil); err != nil {
		t.Fatalf("simulate terminal journal lag: %v", err)
	}

	result, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("journal reconciliation returned error: %v", err)
	}
	if result["idempotent"] != true || result["attempt"] != manifest.AttemptPath || state.State != colony.StateBUILT {
		t.Fatalf("journal reconciliation result=%+v state=%s", result, state.State)
	}
	_, reconciled, ok := loadLatestBuildAttempt(1)
	if !ok || reconciled.Status != buildAttemptBuilt {
		t.Fatalf("journal status after reconciliation = %+v", reconciled)
	}
}

func setupExternalBuildAttemptTest(t *testing.T) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	goal := "Make external build finalization durable"
	taskID := "1.1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "standard",
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:          1,
			Name:        "External attempt",
			Description: "Bind one wrapper dispatch to one lifecycle commit",
			Status:      colony.PhaseReady,
			Tasks:       []colony.Task{{ID: &taskID, Goal: "Write durable evidence", Status: colony.TaskPending}},
		}}},
	})
	return root
}

func prepareExternalBuildCompletion(t *testing.T, root string) (codexBuildManifest, codexExternalBuildCompletion) {
	t.Helper()
	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnly returned error: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)
	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}
	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed externally",
			// Completed workers must relay a non-empty handoff; the finalizer
			// rejects content-free records so the next phase inherits context.
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{dispatch.Name + " work is complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesModified = []string{"external-evidence.txt"}
		}
		results = append(results, worker)
	}
	return manifest, codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}
}

// TestBuildFinalizeRejectsCompletedWorkerWithoutHandoff locks in the
// mandatory-handoff contract. Handoffs are the memory the next phase's workers
// receive; before this rule the store filled with content-free records — the
// chain was "written but empty, read but not delivered."
func TestBuildFinalizeRejectsCompletedWorkerWithoutHandoff(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletion(t, root)
	_ = manifest

	// Strip every handoff — simulating the pre-contract worker output.
	for i := range completion.Dispatches {
		completion.Dispatches[i].Handoff = codex.WorkerHandoff{}
	}

	_, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatal("completed worker without a handoff was accepted; the finalizer must reject content-free relays")
	}
	if !strings.Contains(err.Error(), "handoff") {
		t.Fatalf("rejection should name the missing handoff, got: %v", err)
	}

	// A freshness-only handoff is still content-free and must also be rejected.
	for i := range completion.Dispatches {
		completion.Dispatches[i].Handoff = codex.WorkerHandoff{Freshness: "not-run"}
	}
	_, _, _, _, err = runCodexBuildFinalize(root, 1, completion, false)
	if err == nil {
		t.Fatal("freshness-only handoff was accepted; a timestamp alone relays nothing")
	}
}
