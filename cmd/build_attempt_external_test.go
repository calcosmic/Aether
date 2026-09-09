package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestBuildAttemptExternalFixturesUseCanonicalTransaction200 is the bounded
// migration ratchet for Plan 36's remaining fixture families. The wider
// repository ratchet is added only after these callers are gone; this test
// first gives every migration a fail-first boundary of its own.
func TestBuildAttemptExternalFixturesUseCanonicalTransaction200(t *testing.T) {
	files := []string{
		"build_attempt_external_test.go",
		"coherent_job_retry_test.go",
		"coherent_job_retry_plan_only_test.go",
		"coherent_job_retry_command_test.go",
		"pause_resume_199_test.go",
	}
	forbiddenCalls := map[string]bool{
		"beginBuildAttempt":       true,
		"beginBuildAttemptRecord": true,
		"beginChildBuildAttempt":  true,
	}
	var violations []string
	for _, name := range files {
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, name, nil, 0)
		if err != nil {
			t.Fatalf("parse remaining build-start fixture %s: %v", name, err)
		}
		canonicalCalls := 0
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if identifier.Name == "commitTestBuildStart" {
				canonicalCalls++
			}
			if forbiddenCalls[identifier.Name] {
				violations = append(violations, name+":"+set.Position(call.Pos()).String()+" calls legacy "+identifier.Name)
			}
			return true
		})
		if canonicalCalls == 0 {
			violations = append(violations, name+" has no canonical commitTestBuildStart call")
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			hasAttemptRecord := false
			hasLatestPointer := false
			ast.Inspect(function.Body, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				identifier, ok := literal.Type.(*ast.Ident)
				if !ok {
					return true
				}
				hasAttemptRecord = hasAttemptRecord || identifier.Name == "buildAttemptRecord"
				hasLatestPointer = hasLatestPointer || identifier.Name == "latestBuildAttemptPointer"
				return true
			})
			if hasAttemptRecord && hasLatestPointer {
				violations = append(violations, name+":"+set.Position(function.Pos()).String()+" reconstructs attempt and latest-pointer writes")
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("remaining build-start fixtures bypass the canonical transaction:\n%s", strings.Join(violations, "\n"))
	}
}

func TestBuildAttemptExternalUnboundStartReplayIsByteStable(t *testing.T) {
	generatedAt := time.Date(2026, time.September, 9, 10, 0, 0, 0, time.UTC)
	dispatch := codexBuildDispatch{
		Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-external-replay",
		TaskID: "1.1", CoveredTaskIDs: []string{"1.1"}, Status: "completed",
	}
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartExternalUnbound, GeneratedAt: generatedAt,
		SelectedTasks: []string{"1.1"}, Dispatches: []codexBuildDispatch{dispatch},
		ExecutionOwner: "external-task", DispatchMode: "external-task", MakeLatest: testBuildStartBool(true),
	})
	attemptBefore, err := os.ReadFile(filepath.Join(fixture.DataRoot, filepath.FromSlash(fixture.AttemptPath)))
	if err != nil {
		t.Fatalf("read bound external attempt before replay: %v", err)
	}
	pointerBefore, err := os.ReadFile(filepath.Join(fixture.DataRoot, filepath.FromSlash(latestBuildAttemptPointerPath(fixture.Request.Phase))))
	if err != nil {
		t.Fatalf("read bound external pointer before replay: %v", err)
	}

	replayed, err := commitBuildStart(fixture.Root, fixture.Request, buildStartOptions{})
	if err != nil {
		t.Fatalf("replay already-bound external start: %v", err)
	}
	if replayed.ID != fixture.Receipt.ID || replayed.ContentHash != fixture.Receipt.ContentHash || replayed.TransactionID != fixture.Receipt.TransactionID {
		t.Fatalf("bound replay minted new receipt identity: first=%+v replay=%+v", fixture.Receipt, replayed)
	}
	attemptAfter, err := os.ReadFile(filepath.Join(fixture.DataRoot, filepath.FromSlash(fixture.AttemptPath)))
	if err != nil {
		t.Fatalf("read bound external attempt after replay: %v", err)
	}
	pointerAfter, err := os.ReadFile(filepath.Join(fixture.DataRoot, filepath.FromSlash(latestBuildAttemptPointerPath(fixture.Request.Phase))))
	if err != nil {
		t.Fatalf("read bound external pointer after replay: %v", err)
	}
	if !bytes.Equal(attemptBefore, attemptAfter) || !bytes.Equal(pointerBefore, pointerAfter) {
		t.Fatal("already-bound external replay rewrote attempt or latest-pointer bytes")
	}
}

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

// setupExternalBuildAttemptTestWithVerifiableWork is a variant of
// setupExternalBuildAttemptTest whose phase wording legitimately scores a
// second caste (architect, via the "design" keyword) above the build spawn
// threshold, so the resulting manifest carries two REAL dispatches from the
// program's own relevance scoring. Watcher cannot be used for this: 193
// (D-08) removed watcher from the build dispatch plans entirely (see
// queenBuildPreWavePlans / queenBuildPostWavePlans), so a watcher can score
// above threshold and still never appear in a real manifest. Architect DOES
// have a build dispatch plan (queenBuildPreWavePlans, wave 2). Unlike a
// post-hoc synthetic dispatch (prepareExternalBuildCompletionWithSecondWorker),
// this stays consistent with the durable build-attempt hash check
// runCodexBuildFinalize enforces, so it is the right fixture for any test
// that runs a real finalize and needs more than one dispatch. The base
// fixture stays single-dispatch (just the builder, after 194-02's floor
// shrink) for every other caller.
func setupExternalBuildAttemptTestWithVerifiableWork(t *testing.T) string {
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
			Description: "Bind one wrapper dispatch to one lifecycle commit; design the boundary first",
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
	return externalBuildCompletionFromPlanOnlyResult(t, root, result)
}

// prepareExternalBuildCompletionWithProposal is prepareExternalBuildCompletion
// for a fixture that needs a genuine SECOND worker in the real build manifest
// (not a synthetic one appended after the fact -- appending post-hoc breaks
// the durable build-attempt hash, since that hash is computed over the
// manifest the finalizer actually received). Plan 194-05 (D-11) removed the
// no-proposal keyword-scoring fallback at build, so a fixture that used to
// get a second dispatch "for free" from its own wording now needs an
// explicit --castes-equivalent proposal to reach the judgement path instead
// of the required-caste-only fallback.
func prepareExternalBuildCompletionWithProposal(t *testing.T, root string, castes []string, why []string) (codexBuildManifest, codexExternalBuildCompletion) {
	t.Helper()
	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{QueenCastes: castes, QueenCasteWhy: why})
	if err != nil {
		t.Fatalf("runCodexBuildPlanOnlyWithOptions returned error: %v", err)
	}
	return externalBuildCompletionFromPlanOnlyResult(t, root, result)
}

func externalBuildCompletionFromPlanOnlyResult(t *testing.T, root string, result map[string]interface{}) (codexBuildManifest, codexExternalBuildCompletion) {
	t.Helper()
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

// prepareExternalBuildCompletionWithSecondWorker extends
// prepareExternalBuildCompletion with a synthetic second worker dispatch.
// Before plan 194-02 shrank the build floor to the builder alone
// (194-CONTEXT.md D-07), this fixture's phase always produced at least two
// dispatches -- builder plus the unconditionally-required watcher -- with no
// extra setup. That floor is gone, so a test that needs two independent
// workers to prove per-worker (not per-caste-floor) behavior now builds that
// second worker explicitly. "Keen-6" was this fixture's watcher's actual name
// before the floor shrank (see the finalize test's own comment); reused here
// so the fixture reads the same to anyone who remembers it.
func prepareExternalBuildCompletionWithSecondWorker(t *testing.T, root string) (codexBuildManifest, codexExternalBuildCompletion) {
	t.Helper()
	manifest, completion := prepareExternalBuildCompletion(t, root)

	base := manifest.Dispatches[0]
	second := base
	second.Caste = "watcher"
	second.Name = "Keen-6"
	manifest.Dispatches = append(manifest.Dispatches, second)
	completion.DispatchManifest = &manifest

	completion.Dispatches = append(completion.Dispatches, codexExternalBuildWorkerResult{
		Stage:         second.Stage,
		Wave:          second.Wave,
		ExecutionWave: second.ExecutionWave,
		Caste:         second.Caste,
		Name:          second.Name,
		TaskID:        second.TaskID,
		Status:        "completed",
		Summary:       second.Name + " completed externally",
		Handoff: codex.WorkerHandoff{
			CommandsRun:            []string{"go test ./..."},
			VerificationStatus:     "pass",
			NextWorkerInstructions: []string{second.Name + " work is complete"},
		},
	})
	return manifest, completion
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

// TestCommitBuildFinalizeStateDoesNotOverwritePausedState reproduces CR-02
// (188-REVIEW.md): commitBuildFinalizeState's atomic commit used to be
// `store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func()
// error { committedState = params.UpdatedState; return nil })` -- the
// closure discarded UpdateJSONAtomically's own fresh on-disk read
// (`committedState`) and replaced it wholesale with a value built earlier in
// runCodexBuildFinalize, long before the checkpoint save, build-attempt
// transitions, claims write, and (in worktree mode) a full worktree merge.
// Any concurrent write to COLONY_STATE.json during that window -- an
// operator pausing the colony -- was silently discarded.
//
// runCodexBuildFinalize itself has an EARLIER, unrelated safety check
// (validateBuildAttemptManifestBinding's OriginalStateSHA digest comparison)
// that refuses if COLONY_STATE.json changed between the build attempt being
// prepared and runCodexBuildFinalize's own early state load -- which would
// intercept a pause written before runCodexBuildFinalize is even called,
// masking CR-02's own, later window (between that early load and the atomic
// commit many operations later) behind a different, unrelated error. Calling
// commitBuildFinalizeState directly -- the one function that actually
// performs CR-02's write -- reaches the real vulnerability precisely,
// mirroring the direct-build sibling's own concurrent-pause coverage
// (TestBuildFinalizationDoesNotOverwritePausedState, cmd/codex_build_test.go)
// without needing an injectable worker-invocation hook build-finalize does
// not have.
func TestCommitBuildFinalizeStateDoesNotOverwritePausedState(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	_ = root

	goal := "commitBuildFinalizeState does not overwrite a concurrent pause"
	taskID := "1.1"
	seeded := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Pause during build-finalize commit",
			Status: colony.PhaseReady,
			Tasks:  []colony.Task{{ID: &taskID, Goal: "Pause while commit runs", Status: colony.TaskPending}},
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", seeded); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}

	startedAt := time.Now().UTC()
	// updatedState represents what runCodexBuildFinalize would have computed
	// from a `state` loaded before the concurrent pause below -- stale by
	// the time the atomic commit actually runs.
	updatedState := seeded
	updatedState.State = colony.StateBUILT
	updatedState.Plan.Phases[0].Status = colony.PhaseCompleted

	// Simulate a concurrent process pausing the colony after
	// runCodexBuildFinalize's own early state load but before this commit --
	// the exact gap CR-02 closes.
	var paused colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &paused); err != nil {
		t.Fatalf("load state for mutation: %v", err)
	}
	pausedAt := time.Now().UTC().Format(time.RFC3339)
	paused.Paused = true
	paused.PausedAt = &pausedAt
	if err := store.SaveJSON("COLONY_STATE.json", paused); err != nil {
		t.Fatalf("write competing pause: %v", err)
	}

	_, err := commitBuildFinalizeState(buildFinalizeCommitParams{
		PhaseNum:     1,
		StartedAt:    startedAt,
		ReviewDepth:  colony.VerificationDepthLight,
		CompletedAt:  startedAt.Add(time.Minute),
		UpdatedState: updatedState,
	})
	if !errors.Is(err, errRuntimeStateSuperseded) {
		t.Fatalf("expected superseded build-finalize commit error, got %v", err)
	}

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("reload state: %v", err)
	}
	if !after.Paused {
		t.Fatalf("expected paused state to be preserved, got %+v", after)
	}
	if after.State == colony.StateBUILT {
		t.Fatalf("stale build-finalize commit overwrote state to BUILT despite a concurrent pause")
	}
}

// TestGroupedJobPartialRetryIsAppendOnlyExternal is the external/wrapper-lane
// counterpart of TestGroupedJobPartialRetryIsAppendOnlyDirect: a real
// `aether build-finalize` call for a four-of-six failed grouped completion
// must, per D-10, credit exactly the four proven tasks, leave the colony
// honestly non-BUILT, seal the parent attempt as `partial` (never `built`,
// since it did not finish everything it was responsible for), and create a
// brand-new, parent-linked, append-only retry attempt naming exactly the two
// unfinished tasks -- without rewriting the parent's own dispatches,
// receipts, or claims.
func TestGroupedJobPartialRetryIsAppendOnlyExternal(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "External lane partial credit creates an append-only retry")

	proven := ids[:4]
	pending := ids[4:]
	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touchedFiles := make([]string, 0, len(proven))
	for _, id := range proven {
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

	parentAttemptID := manifest.AttemptID
	result, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("build-finalize should accept a validated partial-credit failure, got error: %v", err)
	}
	if updatedState.State == colony.StateBUILT {
		t.Fatalf("partial credit advanced colony to BUILT: %s", updatedState.State)
	}
	if recovery, _ := result["recovery_job"].(bool); !recovery {
		t.Fatalf("result did not report a D-10 recovery job: %+v", result)
	}
	gotParentID, _ := result["parent_attempt_id"].(string)
	if gotParentID != parentAttemptID {
		t.Fatalf("result parent_attempt_id = %q, want %q", gotParentID, parentAttemptID)
	}
	retryAttemptID, _ := result["retry_attempt_id"].(string)
	if retryAttemptID == "" || retryAttemptID == parentAttemptID {
		t.Fatalf("expected a distinct non-empty retry attempt id, got %q (parent %q)", retryAttemptID, parentAttemptID)
	}
	unfinished, _ := result["unfinished_task_ids"].([]string)
	if len(unfinished) != len(pending) {
		t.Fatalf("unfinished_task_ids = %v, want the two pending tasks %v", unfinished, pending)
	}

	var parentBefore buildAttemptRecord
	parentRel := strings.TrimPrefix(manifest.AttemptPath, ".aether/data/")
	if err := store.LoadJSON(parentRel, &parentBefore); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}
	if parentBefore.Status != buildAttemptPartial {
		t.Fatalf("parent attempt status = %q, want %q (D-10: partial, never a misleading built)", parentBefore.Status, buildAttemptPartial)
	}
	if len(parentBefore.Claims.TaskClaims) == 0 {
		t.Fatalf("parent attempt lost its own task claims: %+v", parentBefore.Claims)
	}

	var child buildAttemptRecord
	found := false
	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ParentAttemptID == parentAttemptID {
			child = record
			found = true
		}
	}
	if !found {
		t.Fatalf("no retry attempt linked to parent %s was found in the phase's attempt journal", parentAttemptID)
	}
	if child.ID != retryAttemptID {
		t.Fatalf("linked child attempt id = %q, want %q", child.ID, retryAttemptID)
	}
	if child.ParentJobName == "" {
		t.Fatalf("child attempt has no ParentJobName recorded: %+v", child)
	}

	// Idempotency at the full entrypoint layer: re-running reconcilePartialBuildRetry
	// directly (the same call build-finalize makes) for the same parent must
	// return the SAME child, never create a second one.
	dispatches, _, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("merge completion results: %v", err)
	}
	dispatches = resolveCoherentJobDispatchReceipts(root, updatedState.Plan.Phases[0], dispatches)
	secondOutcome, err := reconcilePartialBuildRetry(updatedState, 1, updatedState.Plan.Phases[0], parentAttemptID, time.Now().UTC(), dispatches)
	if err != nil {
		t.Fatalf("second reconcilePartialBuildRetry call: %v", err)
	}
	if secondOutcome == nil || secondOutcome.RetryAttemptID != retryAttemptID {
		t.Fatalf("second reconcilePartialBuildRetry call = %+v, want the SAME retry attempt %q", secondOutcome, retryAttemptID)
	}
	childrenLinkedToParent := 0
	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ParentAttemptID == parentAttemptID {
			childrenLinkedToParent++
		}
	}
	if childrenLinkedToParent != 1 {
		t.Fatalf("expected exactly 1 attempt linked to parent %s, found %d (duplicate child created)", parentAttemptID, childrenLinkedToParent)
	}
}

// TestGroupedJobPartialRetryIsIdempotent proves D-10's "repeating the same
// completion cannot create duplicate children" contract directly against
// reconcilePartialBuildRetry: calling it twice with the same parent and the
// same partially-credited dispatches must return the identical retry
// attempt, and the phase's attempt journal must end up with exactly one
// child linked to that parent.
func TestGroupedJobPartialRetryIsIdempotent(t *testing.T) {
	goal := "reconcilePartialBuildRetry is idempotent per parent attempt"
	tasks, ids := sixChainedTasks()
	phase := colony.Phase{
		ID: 1, Name: "Idempotent retry chain", Description: "One worker, six dependent steps",
		Status: colony.PhaseReady, Tasks: tasks,
	}
	state := colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			EvidencePolicy:   colony.PlanEvidenceNotRequired,
			Phases:           []colony.Phase{phase},
		},
	}

	proven := ids[:4]
	touched := make([]string, 0, len(proven))
	for _, id := range proven {
		touched = append(touched, taskFileName(id))
	}
	dispatch := codexBuildDispatch{
		Name: "Mason-1", Caste: "builder", TaskID: ids[0], CoveredTaskIDs: ids,
		Status: "failed", CompletedTaskIDs: append([]string{}, proven...),
	}

	parentStartedAt := time.Now().UTC()
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Variant: buildStartDirect, GeneratedAt: parentStartedAt,
		SelectedTasks: ids, Dispatches: []codexBuildDispatch{dispatch},
		ExecutionOwner: "go-runtime", DispatchMode: "direct", MakeLatest: testBuildStartBool(true),
		PrepareRoot: func(root string) {
			createTestColonyState(t, filepath.Join(root, ".aether", "data"), state)
			for _, id := range proven {
				if err := os.WriteFile(filepath.Join(root, taskFileName(id)), []byte("package fixture\n"), 0o644); err != nil {
					t.Fatalf("write fixture file for task %s: %v", id, err)
				}
			}
		},
	})
	state, phase, parentRel := fixture.State, fixture.Phase, fixture.AttemptPath
	if err := transitionBuildAttempt(parentRel, buildAttemptFailed, "crashed after finishing four of six", []codexBuildDispatch{dispatch}, nil, "real", nil); err != nil {
		t.Fatalf("transition parent to failed: %v", err)
	}
	var parent buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parent); err != nil {
		t.Fatalf("load parent attempt: %v", err)
	}

	first, err := reconcilePartialBuildRetry(state, 1, phase, parent.ID, parentStartedAt.Add(time.Second), []codexBuildDispatch{dispatch})
	if err != nil {
		t.Fatalf("first reconcilePartialBuildRetry: %v", err)
	}
	if first == nil {
		t.Fatal("expected a retry outcome for a genuine partial dispatch")
	}
	second, err := reconcilePartialBuildRetry(state, 1, phase, parent.ID, parentStartedAt.Add(2*time.Second), []codexBuildDispatch{dispatch})
	if err != nil {
		t.Fatalf("second reconcilePartialBuildRetry: %v", err)
	}
	if second == nil || second.RetryAttemptID != first.RetryAttemptID {
		t.Fatalf("second call = %+v, want the identical retry attempt %q", second, first.RetryAttemptID)
	}

	linked := 0
	for _, record := range listBuildAttemptsForPhase(1) {
		if record.ParentAttemptID == parent.ID {
			linked++
		}
	}
	if linked != 1 {
		t.Fatalf("expected exactly 1 attempt linked to parent %s after two reconcile calls, found %d", parent.ID, linked)
	}

	var parentAfter buildAttemptRecord
	if err := store.LoadJSON(parentRel, &parentAfter); err != nil {
		t.Fatalf("reload parent attempt: %v", err)
	}
	if parentAfter.Status != buildAttemptFailed {
		t.Fatalf("parent attempt status changed across idempotent retry calls: %+v", parentAfter)
	}
}
