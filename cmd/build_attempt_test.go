package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestBuildAttemptPersistsTransitionsAndTerminalEvidence(t *testing.T) {
	saveGlobals(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	if err := os.WriteFile(filepath.Join(root, "app.txt"), []byte("attempt evidence\n"), 0644); err != nil {
		t.Fatalf("write attempt artifact: %v", err)
	}
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Journal proof"}
	state := colony.ColonyState{State: colony.StateREADY, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	dispatches := []codexBuildDispatch{{Name: "Mason-1", Caste: "builder", TaskID: "1.1", Status: "spawned"}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, []string{"1.1"}, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", dispatches)
	if err != nil {
		t.Fatalf("begin build attempt: %v", err)
	}
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
	if len(record.History) != 4 || record.History[0].Status != buildAttemptPrepared || record.History[1].Status != buildAttemptDispatching || record.History[2].Status != buildAttemptTerminal || record.History[3].Status != buildAttemptBuilt {
		t.Fatalf("attempt history = %+v", record.History)
	}
}

func TestForceRedispatchMarksActiveAttemptInterrupted(t *testing.T) {
	saveGlobals(t)
	setupBuildFlowTest(t)
	startedAt := time.Now().UTC().Add(-time.Hour)
	phase := colony.Phase{ID: 1, Name: "Interrupted journal"}
	state := colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 1, BuildStartedAt: &startedAt, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil)
	if err != nil {
		t.Fatalf("begin active attempt: %v", err)
	}
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
	setupBuildFlowTest(t)
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Rolled back phase", Status: colony.PhaseReady}
	state := colony.ColonyState{State: colony.StateREADY, CurrentPhase: 0, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	attemptRel, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil)
	if err != nil {
		t.Fatalf("begin failed attempt: %v", err)
	}
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
	setupBuildFlowTest(t)
	goal := "Keep live work isolated"
	startedAt := time.Now().UTC()
	phase := colony.Phase{ID: 1, Name: "Live phase", Status: colony.PhaseInProgress}
	state := colony.ColonyState{
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &startedAt,
		Plan:           colony.Plan{Phases: []colony.Phase{phase}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save live state: %v", err)
	}
	if _, err := beginBuildAttempt(state, 1, phase, startedAt, nil, "checkpoints/pre-build-phase-1.json", "build/phase-1/manifest.json", "last-build-claims.json", "go-runtime", nil); err != nil {
		t.Fatalf("begin live attempt: %v", err)
	}
	result := buildResumeDashboardResult()
	recovery, ok := result["recovery"].(map[string]interface{})
	if !ok || recovery["next"] != "aether watch" {
		t.Fatalf("live build recovery = %#v, want aether watch", result["recovery"])
	}
}
